import json
import logging
import uuid
from shared.inference import InferenceClient
from shared.schemas import TaskMessage, TaskSpec, DAGPosition

logger = logging.getLogger(__name__)

TOT_PROMPT = """You are a task decomposition agent using Tree-of-Thought reasoning.
The goal may have multiple viable approaches. Generate a DAG of sub-tasks that may include parallel branches.

Goal: {goal}
Constraints: {constraints}

Return a JSON array of sub-tasks. Each must have:
- "task_id": unique string
- "description": what needs to be done
- "role_required": one of "researcher", "coder", "reviewer", "planner"
- "depends_on": array of task_ids this depends on (can be multiple for merge points)
- "acceptance_criteria": array of strings
- "priority": "high", "medium", or "low"

Tasks with the same depends_on can run in parallel. Return ONLY valid JSON."""

def decompose_tot(client: InferenceClient, goal: str, constraints: list[str] = None) -> list[TaskMessage]:
    """Decompose a goal into a potentially branching DAG using ToT."""
    prompt = TOT_PROMPT.format(goal=goal, constraints=", ".join(constraints or []))
    response = client.complete([
        {"role": "system", "content": "You are a task decomposition agent specializing in parallel task planning."},
        {"role": "user", "content": prompt},
    ])

    try:
        tasks_data = json.loads(response)
    except json.JSONDecodeError:
        if "```" in response:
            json_str = response.split("```")[1]
            if json_str.startswith("json"):
                json_str = json_str[4:]
            tasks_data = json.loads(json_str.strip())
        else:
            raise

    # Build depth map from dependencies
    id_to_deps = {td["task_id"]: td.get("depends_on", []) for td in tasks_data}
    depths: dict[str, int] = {}
    def get_depth(tid):
        if tid in depths:
            return depths[tid]
        deps = id_to_deps.get(tid, [])
        if not deps:
            depths[tid] = 0
            return 0
        d = max(get_depth(d) for d in deps) + 1
        depths[tid] = d
        return d
    for tid in id_to_deps:
        get_depth(tid)

    tasks = []
    for td in tasks_data:
        tid = td.get("task_id", f"task-{uuid.uuid4().hex[:8]}")
        task = TaskMessage(
            task_id=tid,
            role_required=td["role_required"],
            priority=td.get("priority", "medium"),
            dag_position=DAGPosition(depth=depths.get(tid, 0), index=0),
            spec=TaskSpec(
                description=td["description"],
                acceptance_criteria=td.get("acceptance_criteria", []),
            ),
            context_refs=td.get("depends_on", []),  # store deps in context_refs
        )
        tasks.append(task)
    return tasks
