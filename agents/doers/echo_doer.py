import logging
from shared.base_agent import BaseAgent
from shared.config import AgentConfig
from shared.schemas import TaskMessage

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(name)s] %(message)s")
logger = logging.getLogger(__name__)


class EchoDoer(BaseAgent):
    def on_task(self, task: TaskMessage) -> None:
        logger.info(f"Echo: {task.spec.description}")
        self.submit_artifact(
            task_id=task.task_id,
            content=f"Echo: {task.spec.description}",
            confidence=1.0,
        )


if __name__ == "__main__":
    config = AgentConfig.from_env()
    if not config.role:
        config.role = "coder"
    agent = EchoDoer(config)
    agent.start()
