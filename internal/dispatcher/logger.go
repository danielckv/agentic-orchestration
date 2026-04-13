package dispatcher

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// NewLogger creates a structured JSON logger with the given level and output.
// It adds the default attribute "service"="caof" to every log entry.
// If output is nil, os.Stdout is used.
func NewLogger(level string, output io.Writer) *slog.Logger {
	if output == nil {
		output = os.Stdout
	}

	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: lvl,
	})

	return slog.New(handler).With("service", "caof")
}

// TaskLogger returns a logger enriched with dag_id and task_id attributes
// for correlating log entries to specific DAG task executions.
func TaskLogger(base *slog.Logger, dagID, taskID string) *slog.Logger {
	return base.With("dag_id", dagID, "task_id", taskID)
}

// AgentLogger returns a logger enriched with agent_id and role attributes
// for correlating log entries to specific agent activity.
func AgentLogger(base *slog.Logger, agentID, role string) *slog.Logger {
	return base.With("agent_id", agentID, "role", role)
}
