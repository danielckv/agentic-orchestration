import json
import logging
import uuid
from shared.inference import InferenceClient
from shared.schemas import TaskMessage, TaskSpec, DAGPosition

logger = logging.getLogger(__name__)

DECOMPOSITION_PROMPT = """You are a task decomposition agent using Chain-of-Thought reasoning.
Break this goal into a linear sequence of sub-tasks.

Goal: {goal}
Constraints: {constraints}

Return a JSON array of sub-tasks. Each must have:
- "task_id": unique string
- "description": what needs to be done
- "role_required": one of "researcher", "coder", "reviewer", "planner"
- "depends_on": array of task_ids (empty for first task, previous task_id for sequential)
- "acceptance_criteria": array of strings
- "priority": "high", "medium", or "low"

Return ONLY valid JSON, no markdown."""

def decompose_cot(client: InferenceClient, goal: str, constraints: list[str] = None) -> list[TaskMessage]:
    """Decompose a goal into a linear chain of sub-tasks using CoT."""
    prompt = DECOMPOSITION_PROMPT.format(
        goal=goal,
        constraints=", ".join(constraints or []),
    )
    response = client.complete([
        {"role": "system", "content": "You are a task decomposition agent."},
        {"role": "user", "content": prompt},
    ])

    try:
        tasks_data = json.loads(response)
    except json.JSONDecodeError:
        # Try to extract JSON from markdown code block
        if "```" in response:
            json_str = response.split("```")[1]
            if json_str.startswith("json"):
                json_str = json_str[4:]
            tasks_data = json.loads(json_str.strip())
        else:
            raise

    tasks = []
    for i, td in enumerate(tasks_data):
        task = TaskMessage(
            task_id=td.get("task_id", f"task-{uuid.uuid4().hex[:8]}"),
            role_required=td["role_required"],
            priority=td.get("priority", "medium"),
            dag_position=DAGPosition(depth=i, index=0),
            spec=TaskSpec(
                description=td["description"],
                acceptance_criteria=td.get("acceptance_criteria", []),
            ),
        )
        tasks.append(task)
    return tasks
