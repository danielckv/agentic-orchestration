package dispatcher

import (
	"sync"
	"time"

	"github.com/danielckv/agentic-orchestration/internal/memory"
)

// AdaptiveScheduler wraps the base Scheduler with historical task-duration
// awareness to make smarter agent assignment decisions.
type AdaptiveScheduler struct {
	inner   *Scheduler
	ltm     memory.LongTermMemory
	history map[string][]time.Duration // role -> past task durations
	mu      sync.RWMutex
}

// NewAdaptiveScheduler creates an AdaptiveScheduler backed by the given
// Scheduler and long-term memory store.
func NewAdaptiveScheduler(inner *Scheduler, ltm memory.LongTermMemory) *AdaptiveScheduler {
	return &AdaptiveScheduler{
		inner:   inner,
		ltm:     ltm,
		history: make(map[string][]time.Duration),
	}
}

// EstimateTaskDuration returns the average historical duration for tasks
// assigned to the given role. Returns 5 minutes if no history is available.
func (a *AdaptiveScheduler) EstimateTaskDuration(role string) time.Duration {
	a.mu.RLock()
	defer a.mu.RUnlock()

	durations, ok := a.history[role]
	if !ok || len(durations) == 0 {
		return 5 * time.Minute
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

// RecordCompletion records the duration of a completed task for the given role.
func (a *AdaptiveScheduler) RecordCompletion(role string, duration time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history[role] = append(a.history[role], duration)
}

// SelectBestAgent picks the agent with the lowest product of current load
// and estimated task duration for the given role. Returns nil if agents is empty.
func (a *AdaptiveScheduler) SelectBestAgent(role string, agents []*AgentInfo) *AgentInfo {
	if len(agents) == 0 {
		return nil
	}

	estimate := a.EstimateTaskDuration(role)

	var best *AgentInfo
	bestScore := int64(-1)

	for _, agent := range agents {
		score := int64(agent.CurrentLoad) * int64(estimate)
		if best == nil || score < bestScore {
			best = agent
			bestScore = score
		}
	}

	return best
}
