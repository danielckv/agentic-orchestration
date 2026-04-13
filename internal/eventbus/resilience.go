package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// bufferedMessage holds a message that could not be published to Redis and is
// awaiting replay once the connection is restored.
type bufferedMessage struct {
	stream string
	msg    any
	time   time.Time
}

// ResilientEventBus wraps a RedisEventBus with automatic buffering during
// outages and reconnection with exponential backoff.
type ResilientEventBus struct {
	inner     *RedisEventBus
	buffer    []bufferedMessage
	mu        sync.Mutex
	maxBuffer int
	logger    *slog.Logger
	addr      string
	pass      string
	db        int
}

// NewResilientEventBus creates a resilient wrapper around a Redis event bus.
// If the initial connection fails the bus starts in disconnected mode and
// buffers messages until reconnection succeeds.
func NewResilientEventBus(addr, password string, db int, maxBuffer int, logger *slog.Logger) *ResilientEventBus {
	r := &ResilientEventBus{
		maxBuffer: maxBuffer,
		logger:    logger,
		addr:      addr,
		pass:      password,
		db:        db,
	}

	inner, err := NewRedisEventBus(addr, password, db)
	if err != nil {
		r.logger.Warn("initial redis connection failed, starting in buffered mode", "error", err)
	} else {
		r.inner = inner
	}

	return r
}

// Publish attempts to send a message through the inner RedisEventBus. On
// failure the message is buffered (up to maxBuffer) and reconnection is
// attempted with exponential backoff.
func (r *ResilientEventBus) Publish(ctx context.Context, stream string, msg any) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.inner != nil {
		id, err := r.inner.Publish(ctx, stream, msg)
		if err == nil {
			return id, nil
		}
		r.logger.Warn("publish failed, buffering message", "stream", stream, "error", err)
	}

	// Buffer the message
	if len(r.buffer) < r.maxBuffer {
		r.buffer = append(r.buffer, bufferedMessage{
			stream: stream,
			msg:    msg,
			time:   time.Now(),
		})
	} else {
		r.logger.Error("buffer full, dropping message", "stream", stream)
	}

	// Attempt reconnection in background
	go r.tryReconnect()

	return "", fmt.Errorf("message buffered: redis unavailable")
}

// FlushBuffer replays all buffered messages to Redis after a successful
// reconnection. Returns the first error encountered, if any.
func (r *ResilientEventBus) FlushBuffer(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.inner == nil {
		return fmt.Errorf("not connected to redis")
	}

	var remaining []bufferedMessage
	for _, bm := range r.buffer {
		if _, err := r.inner.Publish(ctx, bm.stream, bm.msg); err != nil {
			remaining = append(remaining, bm)
			r.logger.Warn("flush failed for buffered message", "stream", bm.stream, "error", err)
		}
	}
	r.buffer = remaining

	if len(remaining) > 0 {
		return fmt.Errorf("%d messages could not be flushed", len(remaining))
	}
	return nil
}

// Subscribe delegates to the inner RedisEventBus with reconnection logic
// on failure.
func (r *ResilientEventBus) Subscribe(ctx context.Context, stream, group, consumer string, handler func(id string, data map[string]string) error) error {
	r.mu.Lock()
	inner := r.inner
	r.mu.Unlock()

	if inner == nil {
		if err := r.reconnect(); err != nil {
			return fmt.Errorf("subscribe failed: redis unavailable: %w", err)
		}
		r.mu.Lock()
		inner = r.inner
		r.mu.Unlock()
	}

	return inner.Subscribe(ctx, stream, group, consumer, handler)
}

// Close shuts down the inner Redis connection.
func (r *ResilientEventBus) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.inner != nil {
		return r.inner.Close()
	}
	return nil
}

// reconnect creates a new RedisEventBus and replaces the inner connection.
func (r *ResilientEventBus) reconnect() error {
	inner, err := NewRedisEventBus(r.addr, r.pass, r.db)
	if err != nil {
		return err
	}
	r.mu.Lock()
	if r.inner != nil {
		_ = r.inner.Close()
	}
	r.inner = inner
	r.mu.Unlock()
	r.logger.Info("reconnected to redis", "addr", r.addr)
	return nil
}

// tryReconnect attempts reconnection with exponential backoff (up to 3 attempts).
func (r *ResilientEventBus) tryReconnect() {
	delay := 100 * time.Millisecond
	for attempt := 0; attempt < 3; attempt++ {
		time.Sleep(delay)
		if err := r.reconnect(); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = r.FlushBuffer(ctx)
			cancel()
			return
		}
		delay *= 2
	}
}

// BufferLen returns the current number of buffered messages (for testing).
func (r *ResilientEventBus) BufferLen() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.buffer)
}
