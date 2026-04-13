package nativecore

import (
	"context"
	"fmt"
	"time"
)

// CompletionRequest represents a request to generate a completion.
type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// Message represents a single message in a conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionResponse represents the result of a completion request.
type CompletionResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	TokensUsed   int    `json:"tokens_used"`
	FinishReason string `json:"finish_reason"`
}

// StreamChunk represents a single chunk in a streaming completion response.
type StreamChunk struct {
	Content      string `json:"content"`
	Done         bool   `json:"done"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// InferenceProvider defines the interface for all inference backends.
type InferenceProvider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
	Embed(ctx context.Context, texts []string) ([][]float64, error)
	Name() string
}

// ProviderConfig holds connection details for any inference provider.
type ProviderConfig struct {
	Provider  string        `yaml:"provider"`
	Model     string        `yaml:"model"`
	Endpoint  string        `yaml:"endpoint"`
	APIKey    string        `yaml:"-"`
	APIKeyEnv string        `yaml:"api_key_env"`
	Timeout   time.Duration `yaml:"timeout"`
}

// NewProvider creates an InferenceProvider based on the given configuration.
// Supported providers: "llama", "anthropic", "openai".
func NewProvider(cfg ProviderConfig) (InferenceProvider, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	switch cfg.Provider {
	case "llama":
		return NewLlamaProvider(cfg), nil
	case "anthropic":
		return NewAnthropicProvider(cfg), nil
	case "openai":
		return NewOpenAIProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %q", cfg.Provider)
	}
}
