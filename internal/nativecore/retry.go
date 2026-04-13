package nativecore

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// RetryProvider wraps an InferenceProvider with retry logic and optional
// fallback. Retries use exponential backoff. If all retries are exhausted
// and a fallback provider is configured, the request is forwarded there.
type RetryProvider struct {
	primary    InferenceProvider
	fallback   InferenceProvider // can be nil
	maxRetries int
	baseDelay  time.Duration
	logger     *slog.Logger
}

// NewRetryProvider creates a RetryProvider wrapping the given primary provider.
// fallback may be nil. maxRetries is the number of retry attempts after the
// initial call fails. baseDelay is the initial backoff delay (doubled each retry).
func NewRetryProvider(primary, fallback InferenceProvider, maxRetries int, baseDelay time.Duration, logger *slog.Logger) *RetryProvider {
	return &RetryProvider{
		primary:    primary,
		fallback:   fallback,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
		logger:     logger,
	}
}

// Name returns the name of the primary provider.
func (r *RetryProvider) Name() string {
	return r.primary.Name()
}

// Complete implements InferenceProvider with retry and fallback logic.
func (r *RetryProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	var lastErr error
	delay := r.baseDelay

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			r.logger.Warn("retrying Complete",
				"attempt", attempt,
				"delay", delay,
				"provider", r.primary.Name(),
				"error", lastErr,
			)
			select {
			case <-ctx.Done():
				return CompletionResponse{}, ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
		}

		resp, err := r.primary.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}

	// Fallback
	if r.fallback != nil {
		r.logger.Warn("falling back to secondary provider",
			"primary", r.primary.Name(),
			"fallback", r.fallback.Name(),
			"error", lastErr,
		)
		return r.fallback.Complete(ctx, req)
	}

	return CompletionResponse{}, fmt.Errorf("all retries exhausted for %s: %w", r.primary.Name(), lastErr)
}

// StreamComplete implements InferenceProvider with retry and fallback logic.
func (r *RetryProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	var lastErr error
	delay := r.baseDelay

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			r.logger.Warn("retrying StreamComplete",
				"attempt", attempt,
				"provider", r.primary.Name(),
				"error", lastErr,
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
		}

		ch, err := r.primary.StreamComplete(ctx, req)
		if err == nil {
			return ch, nil
		}
		lastErr = err
	}

	if r.fallback != nil {
		r.logger.Warn("falling back to secondary provider for StreamComplete",
			"primary", r.primary.Name(),
			"fallback", r.fallback.Name(),
		)
		return r.fallback.StreamComplete(ctx, req)
	}

	return nil, fmt.Errorf("all retries exhausted for StreamComplete on %s: %w", r.primary.Name(), lastErr)
}

// Embed implements InferenceProvider with retry and fallback logic.
func (r *RetryProvider) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	var lastErr error
	delay := r.baseDelay

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			r.logger.Warn("retrying Embed",
				"attempt", attempt,
				"provider", r.primary.Name(),
				"error", lastErr,
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
		}

		result, err := r.primary.Embed(ctx, texts)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}

	if r.fallback != nil {
		return r.fallback.Embed(ctx, texts)
	}

	return nil, fmt.Errorf("all retries exhausted for Embed on %s: %w", r.primary.Name(), lastErr)
}
