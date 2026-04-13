def select_strategy(goal: str, context: str = "") -> str:
    """Select CoT or ToT based on goal complexity heuristics."""
    # Simple heuristic: if goal has branching keywords or is long, use ToT
    branching_keywords = ["compare", "evaluate", "multiple", "alternatives", "options", "trade-off", "versus", "vs"]
    goal_lower = goal.lower()
    if any(kw in goal_lower for kw in branching_keywords) or len(goal.split()) > 50:
        return "tot"
    return "cot"
