from shared.schemas import ArtifactMessage, StageResult

def validate_schema(artifact: ArtifactMessage, expected_role: str = "") -> StageResult:
    """Stage 1: Validate artifact structure."""
    warnings = []
    if not artifact.content:
        return StageResult(stage="schema", passed=False, details="Empty artifact content")
    if not artifact.task_id:
        return StageResult(stage="schema", passed=False, details="Missing task_id")
    if not artifact.agent_id:
        warnings.append("Missing agent_id")
    if artifact.metadata.confidence <= 0:
        warnings.append("Zero or negative confidence score")
    return StageResult(stage="schema", passed=True, details="Schema valid", warnings=warnings)
