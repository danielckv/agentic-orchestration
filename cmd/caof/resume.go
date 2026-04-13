package main

import (
	"context"
	"fmt"
	"time"

	"github.com/danielckv/agentic-orchestration/internal/eventbus"
	"github.com/spf13/cobra"
)

var (
	resumeTaskID   string
	resumeGuidance string
)

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a paused or failed task",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Step 1: Connect to Redis.
		bus, err := eventbus.NewRedisEventBus(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			return fmt.Errorf("connect to redis: %w", err)
		}
		defer bus.Close()

		// Step 2: Publish resume message.
		msg := eventbus.TaskMessage{
			TaskID:       resumeTaskID,
			RoleRequired: "planner",
			Priority:     2,
			Spec: eventbus.TaskSpec{
				Description: fmt.Sprintf("Resume task %s", resumeTaskID),
			},
			CreatedAt:  time.Now(),
			TTLSeconds: 3600,
		}

		if resumeGuidance != "" {
			msg.Spec.Constraints = []string{resumeGuidance}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		streamID, err := bus.Publish(ctx, cfg.Streams.TasksPending, msg)
		if err != nil {
			return fmt.Errorf("publish resume: %w", err)
		}

		fmt.Printf("Task resumed:\n")
		fmt.Printf("  Task ID:    %s\n", resumeTaskID)
		fmt.Printf("  Stream ID:  %s\n", streamID)
		if resumeGuidance != "" {
			fmt.Printf("  Guidance:   %s\n", resumeGuidance)
		}

		return nil
	},
}

func init() {
	resumeCmd.Flags().StringVar(&resumeTaskID, "task", "", "task ID to resume")
	resumeCmd.Flags().StringVar(&resumeGuidance, "guidance", "", "optional guidance for the resumed task")
	_ = resumeCmd.MarkFlagRequired("task")
}
