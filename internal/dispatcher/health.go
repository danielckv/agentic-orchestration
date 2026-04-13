package dispatcher

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/danielckv/agentic-orchestration/internal/eventbus"
)

// HealthMonitor periodically checks registered agents for liveness and
// automatically respawns stale agents while returning their unclaimed tasks
// back to the pending pool.
type HealthMonitor struct {
	registry *Registry
	tmux     *TmuxManager
	bus      eventbus.EventBus
	logger   *slog.Logger
	interval time.Duration
}

// NewHealthMonitor creates a HealthMonitor that checks agent health at the
// specified interval.
func NewHealthMonitor(registry *Registry, tmux *TmuxManager, bus eventbus.EventBus, logger *slog.Logger, interval time.Duration) *HealthMonitor {
	return &HealthMonitor{
		registry: registry,
		tmux:     tmux,
		bus:      bus,
		logger:   logger,
		interval: interval,
	}
}

// Start begins the periodic health-check loop. It runs until the context is
// cancelled. For each stale agent it logs a warning, attempts to respawn the
// tmux session, and publishes unclaimed tasks back to the pending stream.
func (h *HealthMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.checkAll(ctx)
		}
	}
}

func (h *HealthMonitor) checkAll(ctx context.Context) {
	agents := h.registry.ListAll()
	for _, agent := range agents {
		if !h.CheckAgent(agent) {
			h.logger.Warn("stale agent detected",
				"agent_id", agent.AgentID,
				"role", agent.Role,
				"last_heartbeat", agent.LastHeartbeat,
			)

			// Attempt to respawn the tmux session
			if agent.Session != "" {
				_ = h.tmux.KillSession(agent.Session)
				if err := h.tmux.CreateSession(agent.Session, ""); err != nil {
					h.logger.Error("failed to respawn agent session",
						"agent_id", agent.AgentID,
						"session", agent.Session,
						"error", err,
					)
				} else {
					h.logger.Info("respawned agent session",
						"agent_id", agent.AgentID,
						"session", agent.Session,
					)
				}
			}

			// Return unclaimed tasks to the pending pool
			if agent.CurrentLoad > 0 {
				msg := TaskMessage{
					Role:       agent.Role,
					AssignedTo: "",
				}
				data, _ := json.Marshal(msg)
				_, err := h.bus.Publish(ctx, "tasks.pending", map[string]string{"payload": string(data)})
				if err != nil {
					h.logger.Error("failed to return tasks to pool",
						"agent_id", agent.AgentID,
						"error", err,
					)
				}
			}

			// Update heartbeat so we don't spam on every tick
			_ = h.registry.Heartbeat(agent.AgentID)
		}
	}
}

// CheckAgent returns true if the agent has sent a heartbeat within the
// registry's stale threshold.
func (h *HealthMonitor) CheckAgent(agent *AgentInfo) (healthy bool) {
	return time.Since(agent.LastHeartbeat) <= h.registry.staleThreshold
}
