package dispatcher

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/danielckv/agentic-orchestration/internal/eventbus"
)

// BlockedEvent is published when a task is escalated to human-in-the-loop.
type BlockedEvent struct {
	TaskID    string    `json:"task_id"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

// ResumeMessage is published to tasks.pending when a human resumes a blocked task.
type ResumeMessage struct {
	TaskID   string `json:"task_id"`
	Guidance string `json:"guidance"`
	Resumed  bool   `json:"resumed"`
}

// HITLManager handles human-in-the-loop escalation when tasks fail repeatedly.
type HITLManager struct {
	bus     eventbus.EventBus
	tmux    *TmuxManager
	blocked map[string]int // task_id -> rejection count
	mu      sync.Mutex
}

// NewHITLManager creates a new HITLManager.
func NewHITLManager(bus eventbus.EventBus, tmux *TmuxManager) *HITLManager {
	return &HITLManager{
		bus:     bus,
		tmux:    tmux,
		blocked: make(map[string]int),
	}
}

// TrackRejection increments the rejection count for a task. Returns true if
// the count reaches 3 or more, indicating the task should be escalated.
func (h *HITLManager) TrackRejection(taskID string) (shouldEscalate bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.blocked[taskID]++
	return h.blocked[taskID] >= 3
}

// Escalate marks a task as blocked, publishes a blocked event, and sends a
// tmux display-message notification to alert the human operator.
func (h *HITLManager) Escalate(ctx context.Context, taskID, reason string) error {
	evt := BlockedEvent{
		TaskID:    taskID,
		Reason:    reason,
		Timestamp: time.Now(),
	}

	if _, err := h.bus.Publish(ctx, "tasks.blocked", evt); err != nil {
		return fmt.Errorf("publish blocked event for %s: %w", taskID, err)
	}

	// Send tmux display-message notification (best effort)
	msg := fmt.Sprintf("HITL ESCALATION: Task %s blocked - %s", taskID, reason)
	_ = exec.Command("tmux", "display-message", msg).Run()

	return nil
}

// Resume resets the rejection count for a task and publishes a resume message
// to the tasks.pending stream with the human's guidance.
func (h *HITLManager) Resume(ctx context.Context, taskID string, guidance string) error {
	h.mu.Lock()
	delete(h.blocked, taskID)
	h.mu.Unlock()

	msg := ResumeMessage{
		TaskID:   taskID,
		Guidance: guidance,
		Resumed:  true,
	}

	if _, err := h.bus.Publish(ctx, "tasks.pending", msg); err != nil {
		return fmt.Errorf("publish resume for %s: %w", taskID, err)
	}

	return nil
}
