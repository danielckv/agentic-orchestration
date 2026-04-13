package main

import (
	"fmt"
	"strings"

	"github.com/danielckv/agentic-orchestration/internal/dispatcher"
	"github.com/spf13/cobra"
)

var teardownForce bool

var teardownCmd = &cobra.Command{
	Use:   "teardown",
	Short: "Kill all CAOF tmux sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		tmux := dispatcher.NewTmuxManager()

		// Step 1: List all sessions starting with "caof-".
		sessions, err := tmux.ListSessions()
		if err != nil {
			if !teardownForce {
				return fmt.Errorf("list tmux sessions: %w", err)
			}
			fmt.Printf("Warning: could not list sessions: %v\n", err)
			return nil
		}

		var killed int
		for _, s := range sessions {
			if !strings.HasPrefix(s, "caof-") {
				continue
			}

			// Step 2: Kill each one.
			if err := tmux.KillSession(s); err != nil {
				if teardownForce {
					fmt.Printf("Warning: could not kill session %s: %v\n", s, err)
					continue
				}
				return fmt.Errorf("kill session %s: %w", s, err)
			}
			fmt.Printf("  Killed session: %s\n", s)
			killed++
		}

		// Step 3: Summary.
		if killed == 0 {
			fmt.Println("No CAOF sessions found.")
		} else {
			fmt.Printf("\nTeardown complete: %d session(s) killed.\n", killed)
		}

		return nil
	},
}

func init() {
	teardownCmd.Flags().BoolVar(&teardownForce, "force", false, "continue on errors")
}
