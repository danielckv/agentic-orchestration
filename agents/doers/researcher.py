import logging
from shared.config import AgentConfig
from shared.inference import InferenceClient, InferenceConfig
from shared.schemas import TaskMessage
from .base_doer import BaseDoer

logger = logging.getLogger(__name__)

RESEARCH_PROMPT = """You are a research agent. Synthesize information on the following topic.

Task: {description}
Constraints: {constraints}

Provide a structured research summary in markdown with:
1. Key findings
2. Supporting evidence
3. Recommendations
4. References (if applicable)"""

class ResearcherDoer(BaseDoer):
    def __init__(self, config: AgentConfig, inference_config: InferenceConfig):
        super().__init__(config)
        self.inference = InferenceClient(inference_config)

    def execute(self, task: TaskMessage) -> None:
        prompt = RESEARCH_PROMPT.format(
            description=task.spec.description,
            constraints=", ".join(task.spec.constraints),
        )
        response = self.inference.complete([
            {"role": "system", "content": "You are a research synthesis agent."},
            {"role": "user", "content": prompt},
        ])
        self.submit_artifact(task_id=task.task_id, content=response, confidence=0.85)
