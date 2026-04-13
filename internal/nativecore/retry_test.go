package nativecore

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
)

// mockProvider is a test double for InferenceProvider.
type mockProvider struct {
	name      string
	completeFunc func(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	return m.completeFunc(ctx, req)
}

func (m *mockProvider) StreamComplete(_ context.Context, _ CompletionRequest) (<-chan StreamChunk, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProvider) Embed(_ context.Context, _ []string) ([][]float64, error) {
	return nil, errors.New("not implemented")
}

func TestRetryProviderFallback(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	primaryCalls := 0
	primary := &mockProvider{
		name: "failing-primary",
		completeFunc: func(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
			primaryCalls++
			return CompletionResponse{}, errors.New("primary failure")
		},
	}

	fallback := &mockProvider{
		name: "working-fallback",
		completeFunc: func(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
			return CompletionResponse{Content: "fallback response", Model: "fallback"}, nil
		},
	}

	retryProvider := NewRetryProvider(primary, fallback, 2, 1*time.Millisecond, logger)

	resp, err := retryProvider.Complete(context.Background(), CompletionRequest{
		Model:    "test",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("expected no error with fallback, got: %v", err)
	}

	// Primary should have been called 3 times (initial + 2 retries)
	if primaryCalls != 3 {
		t.Errorf("expected primary to be called 3 times, got %d", primaryCalls)
	}

	if resp.Content != "fallback response" {
		t.Errorf("expected fallback response, got %q", resp.Content)
	}

	if retryProvider.Name() != "failing-primary" {
		t.Errorf("expected name to be primary's name, got %q", retryProvider.Name())
	}
}
