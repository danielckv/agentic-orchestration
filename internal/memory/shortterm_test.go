package memory

import (
	"context"
	"os"
	"testing"
)

func getRedisAddr() string {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	return addr
}

func TestShortTermMemoryRoundTrip(t *testing.T) {
	addr := getRedisAddr()
	stm, err := NewShortTermMemory(addr, "", 0)
	if err != nil {
		t.Skipf("skipping: redis not available at %s: %v", addr, err)
	}
	defer stm.Close()

	ctx := context.Background()

	// Test agent status round-trip
	agentID := "test-agent-stm-001"
	status := map[string]string{
		"state":       "active",
		"current_task": "task-42",
		"load":        "0.75",
	}

	if err := stm.SetAgentStatus(ctx, agentID, status); err != nil {
		t.Fatalf("SetAgentStatus: %v", err)
	}

	got, err := stm.GetAgentStatus(ctx, agentID)
	if err != nil {
		t.Fatalf("GetAgentStatus: %v", err)
	}

	for k, want := range status {
		if got[k] != want {
			t.Errorf("agent status[%q] = %q, want %q", k, got[k], want)
		}
	}

	// Test task state round-trip
	taskID := "test-task-stm-001"
	state := map[string]string{
		"status":   "running",
		"progress": "50",
		"agent_id": agentID,
	}

	if err := stm.SetTaskState(ctx, taskID, state); err != nil {
		t.Fatalf("SetTaskState: %v", err)
	}

	gotState, err := stm.GetTaskState(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskState: %v", err)
	}

	for k, want := range state {
		if gotState[k] != want {
			t.Errorf("task state[%q] = %q, want %q", k, gotState[k], want)
		}
	}

	// Test session var round-trip
	sessionID := "test-session-001"
	if err := stm.SetSessionVar(ctx, sessionID, "theme", "dark"); err != nil {
		t.Fatalf("SetSessionVar: %v", err)
	}

	val, err := stm.GetSessionVar(ctx, sessionID, "theme")
	if err != nil {
		t.Fatalf("GetSessionVar: %v", err)
	}
	if val != "dark" {
		t.Errorf("session var theme = %q, want %q", val, "dark")
	}
}
