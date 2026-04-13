from shared.schemas import TaskMessage, TaskSpec, ArtifactMessage, ArtifactMetadata


def test_task_message_roundtrip():
    task = TaskMessage(
        task_id="test-123",
        role_required="coder",
        spec=TaskSpec(description="Write hello world", constraints=["Python 3.11+"]),
    )
    data = task.model_dump(mode="json")
    restored = TaskMessage.model_validate(data)
    assert restored.task_id == "test-123"
    assert restored.spec.description == "Write hello world"
    assert restored.role_required == "coder"


def test_artifact_message():
    artifact = ArtifactMessage(
        artifact_id="art-1",
        task_id="task-1",
        agent_id="coder-01",
        content="print('hello')",
        metadata=ArtifactMetadata(confidence=0.95),
    )
    assert artifact.metadata.confidence == 0.95
    data = artifact.model_dump(mode="json")
    assert data["artifact_id"] == "art-1"
