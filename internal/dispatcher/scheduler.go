package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/danielckv/agentic-orchestration/internal/config"
	"github.com/danielckv/agentic-orchestration/internal/eventbus"
	"github.com/danielckv/agentic-orchestration/internal/memory"
)

// TaskMessage is the payload published to the tasks.pending stream.
type TaskMessage struct {
	TaskID      string `json:"task_id"`
	Role        string `json:"role"`
	Description string `json:"description"`
	AssignedTo  string `json:"assigned_to"`
	DAGID       string `json:"dag_id"`
}

// ArtifactEvent is the payload received from artifacts.approved / artifacts.rejected streams.
type ArtifactEvent struct {
	TaskID   string `json:"task_id"`
	AgentID  string `json:"agent_id"`
	Escalate bool   `json:"escalate,omitempty"`
}

// Scheduler orchestrates task execution through a DAG, assigning work to agents
// and reacting to approval/rejection events.
type Scheduler struct {
	dag      *DAG
	registry *Registry
	bus      eventbus.EventBus
	memory   *memory.ShortTermMemory
	cfg      *config.Config
	done     chan struct{}
	once     sync.Once
}

// NewScheduler creates a Scheduler wired to the given DAG, registry, event bus, memory, and config.
func NewScheduler(dag *DAG, registry *Registry, bus eventbus.EventBus, mem *memory.ShortTermMemory, cfg *config.Config) *Scheduler {
	return &Scheduler{
		dag:      dag,
		registry: registry,
		bus:      bus,
		memory:   mem,
		cfg:      cfg,
		done:     make(chan struct{}),
	}
}

// Start begins the scheduling loop. It dispatches ready tasks and listens for
// approval/rejection events until the DAG is complete or the context is cancelled.
func (s *Scheduler) Start(ctx context.Context) error {
	// Initial dispatch of ready nodes
	if err := s.dispatchReady(ctx); err != nil {
		return fmt.Errorf("initial dispatch: %w", err)
	}

	if s.dag.IsComplete() {
		s.signalDone()
		return nil
	}

	// Listen for approved artifacts
	go func() {
		stream := s.cfg.Streams.ArtifactsApproved
		_ = s.bus.Subscribe(ctx, stream, "scheduler", "scheduler-approved", func(id string, data map[string]string) error {
			payload, ok := data["payload"]
			if !ok {
				return nil
			}
			var evt ArtifactEvent
			if err := json.Unmarshal([]byte(payload), &evt); err != nil {
				log.Printf("scheduler: unmarshal approved event: %v", err)
				return nil
			}
			if err := s.dag.SetState(evt.TaskID, NodeApproved); err != nil {
				log.Printf("scheduler: set state approved for %s: %v", evt.TaskID, err)
				return nil
			}
			if s.dag.IsComplete() {
				s.signalDone()
				return nil
			}
			if err := s.dispatchReady(ctx); err != nil {
				log.Printf("scheduler: dispatch after approval: %v", err)
			}
			return nil
		})
	}()

	// Listen for rejected artifacts
	go func() {
		stream := s.cfg.Streams.ArtifactsRejected
		_ = s.bus.Subscribe(ctx, stream, "scheduler", "scheduler-rejected", func(id string, data map[string]string) error {
			payload, ok := data["payload"]
			if !ok {
				return nil
			}
			var evt ArtifactEvent
			if err := json.Unmarshal([]byte(payload), &evt); err != nil {
				log.Printf("scheduler: unmarshal rejected event: %v", err)
				return nil
			}
			if evt.Escalate {
				if err := s.dag.SetState(evt.TaskID, NodeBlocked); err != nil {
					log.Printf("scheduler: set state blocked for %s: %v", evt.TaskID, err)
				}
			}
			return nil
		})
	}()

	return nil
}

// Wait blocks until the DAG is complete or the context is cancelled.
func (s *Scheduler) Wait() {
	<-s.done
}

// AssignTask finds the best available agent for the node's role and publishes
// the task assignment to the tasks.pending stream.
func (s *Scheduler) AssignTask(node *DAGNode) error {
	agents := s.registry.ListByRole(node.Role)
	if len(agents) == 0 {
		return fmt.Errorf("no available agent for role %q", node.Role)
	}

	// Pick agent with lowest current load
	best := agents[0]
	for _, a := range agents[1:] {
		if a.CurrentLoad < best.CurrentLoad {
			best = a
		}
	}

	node.AssignedTo = best.AgentID

	msg := TaskMessage{
		TaskID:      node.TaskID,
		Role:        node.Role,
		Description: node.Description,
		AssignedTo:  best.AgentID,
		DAGID:       s.dag.ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := s.bus.Publish(ctx, s.cfg.Streams.TasksPending, msg); err != nil {
		return fmt.Errorf("publish task %s: %w", node.TaskID, err)
	}

	return s.dag.SetState(node.TaskID, NodeClaimed)
}

// dispatchReady assigns all currently ready nodes in the DAG.
func (s *Scheduler) dispatchReady(ctx context.Context) error {
	ready := s.dag.GetReady()
	for _, node := range ready {
		if err := s.AssignTask(node); err != nil {
			log.Printf("scheduler: assign task %s: %v", node.TaskID, err)
			// Continue dispatching other tasks even if one fails
		}
	}
	return nil
}

func (s *Scheduler) signalDone() {
	s.once.Do(func() {
		close(s.done)
	})
}
