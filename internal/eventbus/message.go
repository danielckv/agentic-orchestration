package eventbus

import "time"

type DAGPosition struct {
	Depth int `json:"depth"`
	Index int `json:"index"`
}

type TaskSpec struct {
	Description        string   `json:"description"`
	Constraints        []string `json:"constraints"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

type TaskMessage struct {
	TaskID       string      `json:"task_id"`
	ParentTaskID string      `json:"parent_task_id,omitempty"`
	DAGPosition  DAGPosition `json:"dag_position"`
	RoleRequired string      `json:"role_required"`
	Priority     int         `json:"priority"`
	Spec         TaskSpec    `json:"spec"`
	ContextRefs  []string    `json:"context_refs,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	TTLSeconds   int         `json:"ttl_seconds"`
}

type ArtifactMetadata struct {
	Timestamp  time.Time `json:"timestamp"`
	Confidence float64   `json:"confidence"`
	SourceRefs []string  `json:"source_refs,omitempty"`
}

type ArtifactMessage struct {
	ArtifactID string           `json:"artifact_id"`
	TaskID     string           `json:"task_id"`
	AgentID    string           `json:"agent_id"`
	Content    string           `json:"content"`
	Metadata   ArtifactMetadata `json:"metadata"`
	Verdict    string           `json:"verdict,omitempty"`
}

type HeartbeatMessage struct {
	AgentID       string    `json:"agent_id"`
	Role          string    `json:"role"`
	Timestamp     time.Time `json:"timestamp"`
	CurrentTaskID string    `json:"current_task_id,omitempty"`
}

type VoteMessage struct {
	DecisionID     string  `json:"decision_id"`
	VoterAgentID   string  `json:"voter_agent_id"`
	OptionSelected string  `json:"option_selected"`
	Confidence     float64 `json:"confidence"`
	Rationale      string  `json:"rationale"`
	References     []string `json:"references,omitempty"`
}
