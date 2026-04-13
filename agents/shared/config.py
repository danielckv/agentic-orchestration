import os
from dataclasses import dataclass, field
import yaml

@dataclass
class AgentConfig:
    agent_id: str = ""
    role: str = ""
    model: str = ""
    redis_url: str = "redis://localhost:6379/0"
    registry_url: str = "http://localhost:9400"
    heartbeat_interval: int = 30  # seconds

    @classmethod
    def from_env(cls) -> "AgentConfig":
        """Load config from environment variables."""
        return cls(
            agent_id=os.getenv("CAOF_AGENT_ID", ""),
            role=os.getenv("CAOF_ROLE", ""),
            model=os.getenv("CAOF_MODEL", ""),
            redis_url=os.getenv("CAOF_REDIS_URL", "redis://localhost:6379/0"),
            registry_url=os.getenv("CAOF_REGISTRY_URL", "http://localhost:9400"),
            heartbeat_interval=int(os.getenv("CAOF_HEARTBEAT_INTERVAL", "30")),
        )

    @classmethod
    def from_yaml(cls, path: str) -> "AgentConfig":
        """Load config from YAML file, with env var overrides."""
        with open(path) as f:
            data = yaml.safe_load(f)
        cfg = cls()
        # map from defaults.yaml structure
        if "redis" in data:
            addr = data["redis"].get("address", "localhost:6379")
            cfg.redis_url = f"redis://{addr}/0"
        if "registry" in data:
            port = data["registry"].get("port", 9400)
            cfg.registry_url = f"http://localhost:{port}"
        if "heartbeat" in data:
            cfg.heartbeat_interval = data["heartbeat"].get("interval_seconds", 30)
        # env overrides
        if os.getenv("CAOF_AGENT_ID"): cfg.agent_id = os.getenv("CAOF_AGENT_ID")
        if os.getenv("CAOF_ROLE"): cfg.role = os.getenv("CAOF_ROLE")
        if os.getenv("CAOF_MODEL"): cfg.model = os.getenv("CAOF_MODEL")
        return cfg
