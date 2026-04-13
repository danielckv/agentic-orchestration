import logging
from shared.base_agent import BaseAgent
from shared.config import AgentConfig
from shared.inference import InferenceClient, InferenceConfig
from shared.schemas import TaskMessage
from .strategy_selector import select_strategy
from .cot import decompose_cot
from .tot import decompose_tot

logger = logging.getLogger(__name__)

class ReasoningAgent(BaseAgent):
    def __init__(self, config: AgentConfig, inference_config: InferenceConfig):
        super().__init__(config)
        self.inference = InferenceClient(inference_config)

    def on_task(self, task: TaskMessage) -> None:
        goal = task.spec.description
        strategy = select_strategy(goal)
        logger.info(f"Decomposing goal with {strategy}: {goal[:80]}...")

        if strategy == "cot":
            sub_tasks = decompose_cot(self.inference, goal, task.spec.constraints)
        else:
            sub_tasks = decompose_tot(self.inference, goal, task.spec.constraints)

        for st in sub_tasks:
            st.parent_task_id = task.task_id
            self.redis.xadd("tasks.pending", st.model_dump(mode="json"))
            logger.info(f"Published sub-task {st.task_id}: {st.spec.description[:60]}")

        self.submit_artifact(
            task_id=task.task_id,
            content=f"Decomposed into {len(sub_tasks)} sub-tasks using {strategy}",
            confidence=0.9,
        )
