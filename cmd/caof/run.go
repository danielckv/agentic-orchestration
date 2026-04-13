package main

import (
	"context"
	"fmt"
	"time"

	"github.com/danielckv/agentic-orchestration/internal/eventbus"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var runGoal string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Submit a goal as a new task",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Step 1: Connect to Redis event bus.
		bus, err := eventbus.NewRedisEventBus(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			return fmt.Errorf("connect to redis: %w", err)
		}
		defer bus.Close()

		// Step 2: Create TaskMessage.
		taskID := uuid.New().String()
		task := eventbus.TaskMessage{
			TaskID:       taskID,
			RoleRequired: "planner",
			Priority:     1,
			Spec: eventbus.TaskSpec{
				Description: runGoal,
			},
			CreatedAt:  time.Now(),
			TTLSeconds: 3600,
		}

		// Step 3: Publish to tasks.pending.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		streamID, err := bus.Publish(ctx, cfg.Streams.TasksPending, task)
		if err != nil {
			return fmt.Errorf("publish task: %w", err)
		}

		fmt.Printf("Task submitted successfully:\n")
		fmt.Printf("  Task ID:    %s\n", taskID)
		fmt.Printf("  Stream ID:  %s\n", streamID)
		fmt.Printf("  Stream:     %s\n", cfg.Streams.TasksPending)
		fmt.Printf("  Goal:       %s\n", runGoal)

		return nil
	},
}

func init() {
	runCmd.Flags().StringVar(&runGoal, "goal", "", "the goal to accomplish")
	_ = runCmd.MarkFlagRequired("goal")
}
