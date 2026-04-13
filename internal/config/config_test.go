package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	yaml := []byte(`
redis:
  address: "redis.example.com:6380"
  password: "secret"
  db: 2
workspace:
  path: "/tmp/ws"
  worktree_dir: ".wt"
heartbeat:
  interval_seconds: 10
  stale_threshold_seconds: 30
reflection:
  max_revisions: 5
inference:
  provider: "anthropic"
  model: "claude-3"
  endpoint: "https://api.anthropic.com"
  timeout_seconds: 60
registry:
  port: 8080
streams:
  tasks_pending: "t.pending"
  tasks_claimed: "t.claimed"
  artifacts_review: "a.review"
  artifacts_approved: "a.approved"
  artifacts_rejected: "a.rejected"
  agents_heartbeat: "a.heartbeat"
  consensus_votes: "c.votes"
`)

	cfg, err := LoadConfig(yaml)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Redis.Address != "redis.example.com:6380" {
		t.Errorf("redis address = %q, want %q", cfg.Redis.Address, "redis.example.com:6380")
	}
	if cfg.Redis.Password != "secret" {
		t.Errorf("redis password = %q, want %q", cfg.Redis.Password, "secret")
	}
	if cfg.Redis.DB != 2 {
		t.Errorf("redis db = %d, want 2", cfg.Redis.DB)
	}
	if cfg.Heartbeat.IntervalSeconds != 10 {
		t.Errorf("heartbeat interval = %d, want 10", cfg.Heartbeat.IntervalSeconds)
	}
	if cfg.Reflection.MaxRevisions != 5 {
		t.Errorf("max_revisions = %d, want 5", cfg.Reflection.MaxRevisions)
	}
	if cfg.Inference.Provider != "anthropic" {
		t.Errorf("provider = %q, want %q", cfg.Inference.Provider, "anthropic")
	}
	if cfg.Registry.Port != 8080 {
		t.Errorf("port = %d, want 8080", cfg.Registry.Port)
	}
	if cfg.Streams.TasksPending != "t.pending" {
		t.Errorf("tasks_pending = %q, want %q", cfg.Streams.TasksPending, "t.pending")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	_, err := LoadConfig([]byte("{{invalid"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Redis.Address == "" {
		t.Error("default redis address is empty")
	}
	if cfg.Heartbeat.IntervalSeconds == 0 {
		t.Error("default heartbeat interval is zero")
	}
	if cfg.Heartbeat.StaleThresholdSeconds == 0 {
		t.Error("default stale threshold is zero")
	}
	if cfg.Reflection.MaxRevisions == 0 {
		t.Error("default max_revisions is zero")
	}
	if cfg.Inference.Provider == "" {
		t.Error("default inference provider is empty")
	}
	if cfg.Inference.TimeoutSeconds == 0 {
		t.Error("default inference timeout is zero")
	}
	if cfg.Registry.Port == 0 {
		t.Error("default registry port is zero")
	}
	if cfg.Streams.TasksPending == "" {
		t.Error("default tasks_pending stream is empty")
	}
}

func TestValidRole(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"researcher", true},
		{"coder", true},
		{"reviewer", true},
		{"planner", true},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := ValidRole(tt.input); got != tt.want {
			t.Errorf("ValidRole(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
