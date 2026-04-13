package dispatcher

import (
	"testing"
	"time"
)

func TestMetricsRecordAndJSON(t *testing.T) {
	m := NewMetrics()

	// Record tasks
	m.RecordTask("pending")
	m.RecordTask("pending")
	m.RecordTask("pending")
	m.RecordTask("approved")
	m.RecordTask("approved")
	m.RecordTask("failed")

	// Record agents
	m.RecordAgent("developer", 1)
	m.RecordAgent("developer", 1)
	m.RecordAgent("reviewer", 1)

	// Record latency
	m.RecordInferenceLatency(100 * time.Millisecond)
	m.RecordInferenceLatency(200 * time.Millisecond)

	// Counters
	m.ArtifactsApproved.Add(7)
	m.ArtifactsRejected.Add(2)
	m.DAGsCompleted.Add(3)
	m.DAGsInProgress.Add(1)

	j := m.JSON()

	// Verify tasks
	tasks, ok := j["tasks_total"].(map[string]int64)
	if !ok {
		t.Fatal("tasks_total is not map[string]int64")
	}
	if tasks["pending"] != 3 {
		t.Errorf("expected pending=3, got %d", tasks["pending"])
	}
	if tasks["approved"] != 2 {
		t.Errorf("expected approved=2, got %d", tasks["approved"])
	}
	if tasks["failed"] != 1 {
		t.Errorf("expected failed=1, got %d", tasks["failed"])
	}

	// Verify agents
	agents, ok := j["agent_count"].(map[string]int64)
	if !ok {
		t.Fatal("agent_count is not map[string]int64")
	}
	if agents["developer"] != 2 {
		t.Errorf("expected developer=2, got %d", agents["developer"])
	}
	if agents["reviewer"] != 1 {
		t.Errorf("expected reviewer=1, got %d", agents["reviewer"])
	}

	// Verify DAGs
	if j["dags_completed"] != int64(3) {
		t.Errorf("expected dags_completed=3, got %v", j["dags_completed"])
	}
	if j["dags_in_progress"] != int64(1) {
		t.Errorf("expected dags_in_progress=1, got %v", j["dags_in_progress"])
	}

	// Verify artifacts
	if j["artifacts_approved"] != int64(7) {
		t.Errorf("expected artifacts_approved=7, got %v", j["artifacts_approved"])
	}
	if j["artifacts_rejected"] != int64(2) {
		t.Errorf("expected artifacts_rejected=2, got %v", j["artifacts_rejected"])
	}

	// Verify latency
	latency, ok := j["inference_latency"].(map[string]any)
	if !ok {
		t.Fatal("inference_latency is not map[string]any")
	}
	if latency["count"] != int64(2) {
		t.Errorf("expected latency count=2, got %v", latency["count"])
	}
	if latency["max_ms"] != float64(200) {
		t.Errorf("expected latency max_ms=200, got %v", latency["max_ms"])
	}
}
