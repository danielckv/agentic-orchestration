package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"text/tabwriter"
	"os"

	"github.com/danielckv/agentic-orchestration/internal/dispatcher"
	"github.com/spf13/cobra"
)

var (
	statusDAG     bool
	statusVerbose bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show agent status from the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Step 1: Query registry HTTP API.
		registryURL := fmt.Sprintf("http://localhost:%d/registry/agents", cfg.Registry.Port)
		resp, err := http.Get(registryURL)
		if err != nil {
			return fmt.Errorf("query registry: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}

		var agents []*dispatcher.AgentInfo
		if err := json.Unmarshal(body, &agents); err != nil {
			return fmt.Errorf("decode agents: %w", err)
		}

		// Step 2: Print table.
		if len(agents) == 0 {
			fmt.Println("No agents registered.")
		} else {
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tROLE\tSESSION\tLAST HEARTBEAT")
			fmt.Fprintln(w, "--\t----\t-------\t--------------")
			for _, a := range agents {
				hb := a.LastHeartbeat.Format("2006-01-02 15:04:05")
				if statusVerbose {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\tmodel=%s load=%d/%d\n",
						a.AgentID, a.Role, a.Session, hb, a.Model, a.CurrentLoad, a.MaxConcurrentTasks)
				} else {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.AgentID, a.Role, a.Session, hb)
				}
			}
			w.Flush()
		}

		// Step 3: DAG flag.
		if statusDAG {
			fmt.Println("\nDAG visualization not yet implemented")
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusDAG, "dag", false, "show DAG visualization")
	statusCmd.Flags().BoolVar(&statusVerbose, "verbose", false, "show verbose agent details")
}
