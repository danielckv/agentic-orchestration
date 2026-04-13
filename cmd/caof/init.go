package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/danielckv/agentic-orchestration/internal/dispatcher"
	"github.com/danielckv/agentic-orchestration/internal/eventbus"
	"github.com/spf13/cobra"
)

var workspacePath string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap the CAOF environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws := workspacePath
		if ws == "" {
			ws = cfg.Workspace.Path
		}

		// Step 1: Validate prerequisites.
		required := []string{"go", "python3", "redis-server", "git", "tmux", "make"}
		fmt.Println("Checking prerequisites...")
		for _, bin := range required {
			if _, err := exec.LookPath(bin); err != nil {
				return fmt.Errorf("required tool %q not found in PATH", bin)
			}
			fmt.Printf("  [ok] %s\n", bin)
		}

		// Step 2: Create workspace directory.
		if err := os.MkdirAll(ws, 0o755); err != nil {
			return fmt.Errorf("create workspace %s: %w", ws, err)
		}
		fmt.Printf("Workspace directory ready: %s\n", ws)

		// Step 3: Initialize git repo if needed.
		gitDir := ws + "/.git"
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			gitInit := exec.Command("git", "init", ws)
			if out, err := gitInit.CombinedOutput(); err != nil {
				return fmt.Errorf("git init: %w: %s", err, out)
			}
			fmt.Println("Initialized git repository in workspace")
		} else {
			fmt.Println("Git repository already exists in workspace")
		}

		// Step 4: Start registry HTTP server.
		staleThreshold := time.Duration(cfg.Heartbeat.StaleThresholdSeconds) * time.Second
		registry := dispatcher.NewRegistry(staleThreshold)
		addr := fmt.Sprintf(":%d", cfg.Registry.Port)
		srv := &http.Server{
			Addr:    addr,
			Handler: registry.Handler(),
		}
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Fprintf(os.Stderr, "registry server error: %v\n", err)
			}
		}()
		fmt.Printf("Registry HTTP server started on %s\n", addr)

		// Step 5: Connect to Redis event bus.
		bus, err := eventbus.NewRedisEventBus(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			return fmt.Errorf("connect to redis: %w", err)
		}
		defer bus.Close()
		fmt.Printf("Connected to Redis at %s\n", cfg.Redis.Address)

		// Step 6: Create caof-main tmux session.
		tmux := dispatcher.NewTmuxManager()
		if !tmux.SessionExists("caof-main") {
			if err := tmux.CreateSession("caof-main", "echo 'Dispatcher running'; read"); err != nil {
				return fmt.Errorf("create tmux session caof-main: %w", err)
			}
			fmt.Println("Created tmux session: caof-main")
		} else {
			fmt.Println("Tmux session caof-main already exists")
		}

		// Summary.
		fmt.Println("\n--- CAOF Init Summary ---")
		fmt.Printf("  Workspace:  %s\n", ws)
		fmt.Printf("  Registry:   http://localhost:%d\n", cfg.Registry.Port)
		fmt.Printf("  Redis:      %s\n", cfg.Redis.Address)
		fmt.Printf("  Tmux:       caof-main\n")
		fmt.Println("CAOF environment is ready.")

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&workspacePath, "workspace", "", "workspace directory path (default from config)")
}
