package nativecore

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewProviderFactory(t *testing.T) {
	tests := []struct {
		provider string
		wantName string
		wantErr  bool
	}{
		{"llama", "llama", false},
		{"anthropic", "anthropic", false},
		{"openai", "openai", false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			cfg := ProviderConfig{
				Provider: tt.provider,
				Model:    "test-model",
				Endpoint: "http://localhost:8080",
				APIKey:   "test-key",
				Timeout:  10 * time.Second,
			}

			p, err := NewProvider(cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for provider %q, got nil", tt.provider)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for provider %q: %v", tt.provider, err)
			}
			if p.Name() != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, p.Name())
			}
		})
	}
}

func TestCompletionRequestSerialization(t *testing.T) {
	req := CompletionRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
		Stream:      false,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded CompletionRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Model != req.Model {
		t.Errorf("model mismatch: got %q, want %q", decoded.Model, req.Model)
	}
	if len(decoded.Messages) != len(req.Messages) {
		t.Fatalf("messages count mismatch: got %d, want %d", len(decoded.Messages), len(req.Messages))
	}
	if decoded.Messages[0].Role != "system" {
		t.Errorf("first message role: got %q, want %q", decoded.Messages[0].Role, "system")
	}
	if decoded.Messages[1].Content != "Hello!" {
		t.Errorf("second message content: got %q, want %q", decoded.Messages[1].Content, "Hello!")
	}
	if decoded.MaxTokens != 100 {
		t.Errorf("max_tokens: got %d, want %d", decoded.MaxTokens, 100)
	}
	if decoded.Temperature != 0.7 {
		t.Errorf("temperature: got %f, want %f", decoded.Temperature, 0.7)
	}
	if decoded.Stream != false {
		t.Errorf("stream: got %v, want %v", decoded.Stream, false)
	}
}
