from pydantic import BaseModel, Field
from datetime import datetime
from typing import Optional


class DAGPosition(BaseModel):
    depth: int = 0
    index: int = 0


class TaskSpec(BaseModel):
    description: str
    constraints: list[str] = Field(default_factory=list)
    acceptance_criteria: list[str] = Field(default_factory=list)


class TaskMessage(BaseModel):
    task_id: str
    parent_task_id: Optional[str] = None
    dag_position: DAGPosition = Field(default_factory=DAGPosition)
    role_required: str
    priority: str = "medium"
    spec: TaskSpec
    context_refs: list[str] = Field(default_factory=list)
    created_at: datetime = Field(default_factory=datetime.utcnow)
    ttl_seconds: int = 3600


class ArtifactMetadata(BaseModel):
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    confidence: float = 0.0
    source_refs: list[str] = Field(default_factory=list)


class ArtifactMessage(BaseModel):
    artifact_id: str
    task_id: str
    agent_id: str
    content: str
    metadata: ArtifactMetadata = Field(default_factory=ArtifactMetadata)
    verdict: str = ""


class HeartbeatMessage(BaseModel):
    agent_id: str
    role: str
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    current_task_id: Optional[str] = None


class AgentRegistration(BaseModel):
    agent_id: str
    role: str
    capabilities: list[str] = Field(default_factory=list)
    model: str = ""
    max_concurrent_tasks: int = 1
    current_load: int = 0
    session: str = ""
    pid: int = 0


class VoteMessage(BaseModel):
    decision_id: str
    voter_agent_id: str
    option_selected: str
    confidence: float = 0.0
    rationale: str = ""
    references: list[str] = Field(default_factory=list)


class StageResult(BaseModel):
    stage: str
    passed: bool
    details: str = ""
    warnings: list[str] = Field(default_factory=list)
