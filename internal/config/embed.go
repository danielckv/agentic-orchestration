package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type RedisConfig struct {
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type WorkspaceConfig struct {
	Path        string `yaml:"path"`
	WorktreeDir string `yaml:"worktree_dir"`
}

type HeartbeatConfig struct {
	IntervalSeconds        int `yaml:"interval_seconds"`
	StaleThresholdSeconds  int `yaml:"stale_threshold_seconds"`
}

type ReflectionConfig struct {
	MaxRevisions int `yaml:"max_revisions"`
}

type InferenceConfig struct {
	Provider       string `yaml:"provider"`
	Model          string `yaml:"model"`
	Endpoint       string `yaml:"endpoint"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type RegistryConfig struct {
	Port int `yaml:"port"`
}

type StreamsConfig struct {
	TasksPending      string `yaml:"tasks_pending"`
	TasksClaimed      string `yaml:"tasks_claimed"`
	ArtifactsReview   string `yaml:"artifacts_review"`
	ArtifactsApproved string `yaml:"artifacts_approved"`
	ArtifactsRejected string `yaml:"artifacts_rejected"`
	AgentsHeartbeat   string `yaml:"agents_heartbeat"`
	ConsensusVotes    string `yaml:"consensus_votes"`
}

type Config struct {
	Redis      RedisConfig      `yaml:"redis"`
	Workspace  WorkspaceConfig  `yaml:"workspace"`
	Heartbeat  HeartbeatConfig  `yaml:"heartbeat"`
	Reflection ReflectionConfig `yaml:"reflection"`
	Inference  InferenceConfig  `yaml:"inference"`
	Registry   RegistryConfig   `yaml:"registry"`
	Streams    StreamsConfig    `yaml:"streams"`
}

func LoadConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Redis: RedisConfig{
			Address:  "localhost:6379",
			Password: "",
			DB:       0,
		},
		Workspace: WorkspaceConfig{
			Path:        "~/caof-workspace",
			WorktreeDir: ".worktrees",
		},
		Heartbeat: HeartbeatConfig{
			IntervalSeconds:       30,
			StaleThresholdSeconds: 90,
		},
		Reflection: ReflectionConfig{
			MaxRevisions: 3,
		},
		Inference: InferenceConfig{
			Provider:       "openai",
			Model:          "gpt-4o",
			TimeoutSeconds: 120,
		},
		Registry: RegistryConfig{
			Port: 9400,
		},
		Streams: StreamsConfig{
			TasksPending:      "tasks.pending",
			TasksClaimed:      "tasks.claimed",
			ArtifactsReview:   "artifacts.review",
			ArtifactsApproved: "artifacts.approved",
			ArtifactsRejected: "artifacts.rejected",
			AgentsHeartbeat:   "agents.heartbeat",
			ConsensusVotes:    "consensus.votes",
		},
	}
}
