import logging
import os
from shared.config import AgentConfig
from shared.inference import InferenceClient, InferenceConfig
from shared.schemas import TaskMessage
from .base_doer import BaseDoer

logger = logging.getLogger(__name__)

CODER_PROMPT = """You are a coding agent. Write the code requested below.

Task: {description}
Constraints: {constraints}
Acceptance criteria: {criteria}

Return ONLY the code, no explanations. If multiple files are needed, use this format:
### FILE: path/to/file.py
<code>
### FILE: path/to/other.py
<code>"""

class CoderDoer(BaseDoer):
    def __init__(self, config: AgentConfig, inference_config: InferenceConfig, worktree_path: str = ""):
        super().__init__(config)
        self.inference = InferenceClient(inference_config)
        self.worktree_path = worktree_path

    def execute(self, task: TaskMessage) -> None:
        prompt = CODER_PROMPT.format(
            description=task.spec.description,
            constraints=", ".join(task.spec.constraints),
            criteria=", ".join(task.spec.acceptance_criteria),
        )
        response = self.inference.complete([
            {"role": "system", "content": "You are a code generation agent."},
            {"role": "user", "content": prompt},
        ])

        # Write files if worktree is available
        if self.worktree_path and os.path.isdir(self.worktree_path):
            files = self._parse_files(response)
            for filepath, content in files.items():
                full_path = os.path.join(self.worktree_path, filepath)
                os.makedirs(os.path.dirname(full_path), exist_ok=True)
                with open(full_path, "w") as f:
                    f.write(content)
                logger.info(f"Wrote {filepath}")

        self.submit_artifact(task_id=task.task_id, content=response, confidence=0.8)

    def _parse_files(self, response: str) -> dict[str, str]:
        files = {}
        current_file = None
        current_content = []
        for line in response.split("\n"):
            if line.startswith("### FILE:"):
                if current_file:
                    files[current_file] = "\n".join(current_content)
                current_file = line.replace("### FILE:", "").strip()
                current_content = []
            elif current_file is not None:
                current_content.append(line)
        if current_file:
            files[current_file] = "\n".join(current_content)
        if not files:
            files["output.txt"] = response
        return files
