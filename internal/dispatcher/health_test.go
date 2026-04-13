package dispatcher

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestHealthCheckStaleAgent(t *testing.T) {
	registry := NewRegistry(30 * time.Second)

	// Register an agent with a very old heartbeat
	staleAgent := &AgentInfo{
		AgentID:       "agent-stale",
		Role:          "developer",
		Session:       "sess-stale",
		LastHeartbeat: time.Now().Add(-5 * time.Minute), // well past threshold
	}
	registry.mu.Lock()
	registry.agents[staleAgent.AgentID] = staleAgent
	registry.mu.Unlock()

	// Register a healthy agent
	healthyAgent := &AgentInfo{
		AgentID:       "agent-healthy",
		Role:          "reviewer",
		Session:       "sess-healthy",
		LastHeartbeat: time.Now(),
	}
	registry.Register(healthyAgent)

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	monitor := NewHealthMonitor(registry, NewTmuxManager(), nil, logger, 10*time.Second)

	// Stale agent should be unhealthy
	if monitor.CheckAgent(staleAgent) {
		t.Error("expected stale agent to be unhealthy")
	}

	// Healthy agent should be healthy
	if !monitor.CheckAgent(healthyAgent) {
		t.Error("expected healthy agent to be healthy")
	}
}
