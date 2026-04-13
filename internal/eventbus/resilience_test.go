package eventbus

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

func TestResilientEventBusBuffering(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Connect to a non-existent Redis instance so all publishes fail
	bus := NewResilientEventBus("localhost:59999", "", 0, 100, logger)

	ctx := context.Background()

	// Publish should buffer the message instead of returning data
	_, err := bus.Publish(ctx, "test.stream", map[string]string{"key": "value"})
	if err == nil {
		t.Fatal("expected error from publish with no redis, got nil")
	}

	// Verify the message was buffered
	if bus.BufferLen() != 1 {
		t.Errorf("expected 1 buffered message, got %d", bus.BufferLen())
	}

	// Publish a second message
	_, _ = bus.Publish(ctx, "test.stream", map[string]string{"key": "value2"})
	if bus.BufferLen() != 2 {
		t.Errorf("expected 2 buffered messages, got %d", bus.BufferLen())
	}
}
