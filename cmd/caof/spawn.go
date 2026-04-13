package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/danielckv/agentic-orchestration/internal/config"
	"github.com/danielckv/agentic-orchestration/internal/dispatcher"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	spawnRole    string
	spawnModel   string
	spawnSession string
)

var spawnCmd = &cobra.Command{
	Use:   "spawn",
	Short: "Spawn a new agent in a tmux session",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Step 1: Validate role.
		if !config.ValidRole(spawnRole) {
			return fmt.Errorf("invalid role %q; valid roles: researcher, coder, reviewer, planner", spawnRole)
		}

		// Step 2: Construct the Python command.
		pyCmd := "python3 -m agents.doers.echo_doer"

		// Step 3: Create tmux session.
		tmux := dispatcher.NewTmuxManager()
		sessionName := spawnSession
		if sessionName == "" {
			sessionName = fmt.Sprintf("caof-%s-%s", spawnRole, uuid.New().String()[:8])
		}

		if err := tmux.CreateSession(sessionName, pyCmd); err != nil {
			return fmt.Errorf("create tmux session %s: %w", sessionName, err)
		}

		// Step 4: Register agent with registry via HTTP POST.
		agentID := uuid.New().String()
		agent := dispatcher.AgentInfo{
			AgentID:            agentID,
			Role:               spawnRole,
			Capabilities:       config.RoleCapabilities[config.Role(spawnRole)],
			Model:              spawnModel,
			MaxConcurrentTasks: 1,
			CurrentLoad:        0,
			Session:            sessionName,
		}

		body, err := json.Marshal(agent)
		if err != nil {
			return fmt.Errorf("marshal agent info: %w", err)
		}

		registryURL := fmt.Sprintf("http://localhost:%d/registry/agents", cfg.Registry.Port)
		resp, err := http.Post(registryURL, "application/json", bytes.NewReader(body))
		if err != nil {
			fmt.Printf("Warning: could not register agent with registry: %v\n", err)
		} else {
			resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				fmt.Printf("Warning: registry returned status %d\n", resp.StatusCode)
			}
		}

		// Confirmation.
		fmt.Printf("Agent spawned successfully:\n")
		fmt.Printf("  Agent ID:  %s\n", agentID)
		fmt.Printf("  Role:      %s\n", spawnRole)
		fmt.Printf("  Model:     %s\n", spawnModel)
		fmt.Printf("  Session:   %s\n", sessionName)
		fmt.Printf("  Command:   %s\n", pyCmd)

		return nil
	},
}

func init() {
	spawnCmd.Flags().StringVar(&spawnRole, "role", "", "agent role (researcher, coder, reviewer, planner)")
	spawnCmd.Flags().StringVar(&spawnModel, "model", "gpt-4o", "inference model to use")
	spawnCmd.Flags().StringVar(&spawnSession, "session", "", "tmux session name (auto-generated if empty)")
	_ = spawnCmd.MarkFlagRequired("role")
}
