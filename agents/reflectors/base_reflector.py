import json
import logging
from shared.base_agent import BaseAgent
from shared.config import AgentConfig
from shared.inference import InferenceClient, InferenceConfig
from shared.schemas import TaskMessage, ArtifactMessage, ArtifactMetadata
from .schema_validator import validate_schema
from .rag_crossref import crossref_rag
from .test_runner import run_tests
from .llm_auditor import audit_artifact

logger = logging.getLogger(__name__)

class ReflectorAgent(BaseAgent):
    def __init__(self, config: AgentConfig, inference_config: InferenceConfig):
        super().__init__(config)
        self.inference = InferenceClient(inference_config)
        self._revision_counts: dict[str, int] = {}
        self._consumer_group = "reflector-group"

    def start(self):
        """Override to listen on artifacts.review instead of tasks.pending."""
        import signal
        self.running = True
        signal.signal(signal.SIGTERM, lambda *_: self.stop())
        signal.signal(signal.SIGINT, lambda *_: self.stop())
        self.register()

        stream = "artifacts.review"
        self._ensure_consumer_group(stream)
        logger.info(f"Reflector {self.config.agent_id} started, listening on {stream}")

        heartbeat_counter = 0
        while self.running:
            heartbeat_counter += 1
            if heartbeat_counter >= self.config.heartbeat_interval:
                self.heartbeat()
                heartbeat_counter = 0

            try:
                messages = self.redis.xreadgroup(
                    groupname=self._consumer_group,
                    consumername=self.config.agent_id,
                    streams={stream: ">"},
                    count=1,
                    block=1000,
                )
            except Exception as e:
                logger.error(f"Error reading stream: {e}")
                continue

            if not messages:
                continue

            for stream_name, entries in messages:
                for msg_id, data in entries:
                    try:
                        # Parse metadata if it's a string
                        if "metadata" in data and isinstance(data["metadata"], str):
                            data["metadata"] = json.loads(data["metadata"])
                        artifact = ArtifactMessage.model_validate(data)
                        self._review_artifact(artifact)
                        self.redis.xack(stream, self._consumer_group, msg_id)
                    except Exception as e:
                        logger.error(f"Error reviewing artifact: {e}")

    def _review_artifact(self, artifact: ArtifactMessage):
        task_id = artifact.task_id

        # Stage 1: Schema validation
        schema_result = validate_schema(artifact)

        # Stage 2: RAG cross-reference (no memory client yet)
        rag_result = crossref_rag(artifact.content)

        # Stage 3: Test runner (no worktree path yet)
        test_result = run_tests("")

        # Short-circuit on hard failures
        if not schema_result.passed:
            self._publish_verdict(artifact, "REJECT", f"Schema validation failed: {schema_result.details}")
            return

        # Stage 4: LLM audit
        revision_notes = ""
        count = self._revision_counts.get(task_id, 0)

        audit = audit_artifact(
            self.inference, task_id, artifact.content,
            schema_result, rag_result, test_result, revision_notes,
        )

        verdict = audit.get("verdict", "REVISE")

        if verdict == "APPROVE":
            self._publish_verdict(artifact, "APPROVE", audit.get("rationale", ""))
        elif verdict == "REJECT" or count >= 2:
            self._publish_verdict(artifact, "REJECT", audit.get("rationale", ""), escalate=(count >= 2))
        else:
            self._revision_counts[task_id] = count + 1
            self._publish_revision(artifact, audit.get("revision_notes", ""))

    def _publish_verdict(self, artifact: ArtifactMessage, verdict: str, rationale: str, escalate: bool = False):
        stream = "artifacts.approved" if verdict == "APPROVE" else "artifacts.rejected"
        self.redis.xadd(stream, {
            "artifact_id": artifact.artifact_id,
            "task_id": artifact.task_id,
            "agent_id": artifact.agent_id,
            "verdict": verdict,
            "rationale": rationale,
            "escalate": str(escalate),
        })
        logger.info(f"Verdict for {artifact.task_id}: {verdict}")

    def _publish_revision(self, artifact: ArtifactMessage, notes: str):
        self.redis.xadd("tasks.pending", {
            "task_id": artifact.task_id,
            "role_required": "coder",  # re-assign to original role
            "priority": "high",
            "spec": json.dumps({"description": f"REVISION: {notes}", "constraints": [], "acceptance_criteria": []}),
            "context_refs": json.dumps([]),
        })
        logger.info(f"Revision requested for {artifact.task_id}: {notes[:80]}")

    def on_task(self, task: TaskMessage) -> None:
        pass  # Reflector doesn't process tasks from tasks.pending
