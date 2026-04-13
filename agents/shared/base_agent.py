import asyncio
import json
import logging
import os
import signal
import uuid
from abc import ABC, abstractmethod
from datetime import datetime

import httpx
import redis

from .config import AgentConfig
from .schemas import AgentRegistration, HeartbeatMessage, TaskMessage

logger = logging.getLogger(__name__)


class BaseAgent(ABC):
    def __init__(self, config: AgentConfig):
        self.config = config
        if not config.agent_id:
            self.config.agent_id = f"{config.role}-{uuid.uuid4().hex[:8]}"
        self.redis = redis.Redis.from_url(config.redis_url, decode_responses=True)
        self.running = False
        self.current_task_id: str | None = None
        self._consumer_group = f"{config.role}-group"
        self._consumer_name = self.config.agent_id

    def register(self):
        """Register with the Control Plane registry."""
        reg = AgentRegistration(
            agent_id=self.config.agent_id,
            role=self.config.role,
            model=self.config.model,
            session=os.getenv("TMUX_SESSION", ""),
            pid=os.getpid(),
        )
        try:
            resp = httpx.post(
                f"{self.config.registry_url}/registry/agents",
                json=reg.model_dump(),
                timeout=5.0,
            )
            resp.raise_for_status()
            logger.info(f"Registered as {self.config.agent_id}")
        except Exception as e:
            logger.warning(f"Failed to register: {e}")

    def heartbeat(self):
        """Publish heartbeat to Redis stream."""
        msg = HeartbeatMessage(
            agent_id=self.config.agent_id,
            role=self.config.role,
            current_task_id=self.current_task_id,
        )
        self.redis.xadd("agents.heartbeat", msg.model_dump(mode="json"))

    def claim_task(self, task_id: str):
        """Announce task claim on tasks.claimed stream."""
        self.current_task_id = task_id
        self.redis.xadd("tasks.claimed", {
            "task_id": task_id,
            "agent_id": self.config.agent_id,
            "timestamp": datetime.utcnow().isoformat(),
        })

    def submit_artifact(self, task_id: str, content: str, confidence: float = 1.0):
        """Submit artifact for review."""
        from .schemas import ArtifactMessage, ArtifactMetadata
        artifact = ArtifactMessage(
            artifact_id=uuid.uuid4().hex,
            task_id=task_id,
            agent_id=self.config.agent_id,
            content=content,
            metadata=ArtifactMetadata(confidence=confidence),
        )
        self.redis.xadd("artifacts.review", artifact.model_dump(mode="json"))
        self.current_task_id = None
        return artifact.artifact_id

    @abstractmethod
    def on_task(self, task: TaskMessage) -> None:
        """Handle a received task. Subclasses must implement."""
        ...

    def _ensure_consumer_group(self, stream: str):
        try:
            self.redis.xgroup_create(stream, self._consumer_group, id="0", mkstream=True)
        except redis.exceptions.ResponseError as e:
            if "BUSYGROUP" not in str(e):
                raise

    def start(self):
        """Main loop: register, heartbeat, consume tasks."""
        self.running = True
        signal.signal(signal.SIGTERM, lambda *_: self.stop())
        signal.signal(signal.SIGINT, lambda *_: self.stop())

        self.register()

        stream = "tasks.pending"
        self._ensure_consumer_group(stream)

        logger.info(f"Agent {self.config.agent_id} started, listening on {stream}")

        heartbeat_counter = 0
        while self.running:
            # Heartbeat every N iterations (approx heartbeat_interval seconds)
            heartbeat_counter += 1
            if heartbeat_counter >= self.config.heartbeat_interval:
                self.heartbeat()
                heartbeat_counter = 0

            # Read from stream
            try:
                messages = self.redis.xreadgroup(
                    groupname=self._consumer_group,
                    consumername=self._consumer_name,
                    streams={stream: ">"},
                    count=1,
                    block=1000,  # 1 second block
                )
            except Exception as e:
                logger.error(f"Error reading stream: {e}")
                continue

            if not messages:
                continue

            for stream_name, entries in messages:
                for msg_id, data in entries:
                    # Check if this task is for our role
                    role_required = data.get("role_required", "")
                    if role_required and role_required != self.config.role:
                        continue

                    try:
                        # Parse the nested JSON fields
                        if "spec" in data and isinstance(data["spec"], str):
                            data["spec"] = json.loads(data["spec"])
                        if "dag_position" in data and isinstance(data["dag_position"], str):
                            data["dag_position"] = json.loads(data["dag_position"])
                        if "context_refs" in data and isinstance(data["context_refs"], str):
                            data["context_refs"] = json.loads(data["context_refs"])

                        task = TaskMessage.model_validate(data)
                        self.claim_task(task.task_id)
                        logger.info(f"Processing task {task.task_id}")
                        self.on_task(task)
                        self.redis.xack(stream, self._consumer_group, msg_id)
                    except Exception as e:
                        logger.error(f"Error processing task: {e}")

    def stop(self):
        self.running = False
        logger.info(f"Agent {self.config.agent_id} stopping")
