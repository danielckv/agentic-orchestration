package main

import (
	"fmt"
	"os"

	"github.com/danielckv/agentic-orchestration/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgPath string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "caof",
	Short: "Collective Agentic Orchestration Framework",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			// Fall back to defaults when config file is not found.
			cfg = config.DefaultConfig()
			return nil
		}
		loaded, err := config.LoadConfig(data)
		if err != nil {
			return fmt.Errorf("load config %s: %w", cfgPath, err)
		}
		cfg = loaded
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "config/defaults.yaml", "path to configuration file")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(spawnCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(teardownCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
