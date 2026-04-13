import logging
from shared.base_agent import BaseAgent
from shared.schemas import TaskMessage

logger = logging.getLogger(__name__)

class BaseDoer(BaseAgent):
    """Base class for all Doer agents. Filters tasks by role."""

    def on_task(self, task: TaskMessage) -> None:
        if task.role_required != self.config.role:
            return
        self.execute(task)

    def execute(self, task: TaskMessage) -> None:
        raise NotImplementedError("Subclasses must implement execute()")
