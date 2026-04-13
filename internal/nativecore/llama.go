package nativecore

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// LlamaProvider implements InferenceProvider for a local llama.cpp server.
type LlamaProvider struct {
	cfg    ProviderConfig
	client *http.Client
}

// NewLlamaProvider creates a new LlamaProvider with the given configuration.
func NewLlamaProvider(cfg ProviderConfig) *LlamaProvider {
	return &LlamaProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (p *LlamaProvider) Name() string { return "llama" }

// buildPrompt converts a slice of Messages into a single prompt string
// suitable for the llama.cpp /completion endpoint.
func buildPrompt(messages []Message) string {
	var sb strings.Builder
	for _, m := range messages {
		switch m.Role {
		case "system":
			sb.WriteString("### System:\n")
			sb.WriteString(m.Content)
			sb.WriteString("\n\n")
		case "user":
			sb.WriteString("### User:\n")
			sb.WriteString(m.Content)
			sb.WriteString("\n\n")
		case "assistant":
			sb.WriteString("### Assistant:\n")
			sb.WriteString(m.Content)
			sb.WriteString("\n\n")
		}
	}
	sb.WriteString("### Assistant:\n")
	return sb.String()
}

// llamaCompletionRequest is the request payload for llama.cpp /completion.
type llamaCompletionRequest struct {
	Prompt      string  `json:"prompt"`
	NPredict    int     `json:"n_predict,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	Stream      bool    `json:"stream"`
}

// llamaCompletionResponse is the response from llama.cpp /completion.
type llamaCompletionResponse struct {
	Content        string `json:"content"`
	Stop           bool   `json:"stop"`
	TokensEvaluated int   `json:"tokens_evaluated"`
	TokensPredicted int   `json:"tokens_predicted"`
}

func (p *LlamaProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	body := llamaCompletionRequest{
		Prompt:      buildPrompt(req.Messages),
		NPredict:    req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.Endpoint+"/completion", bytes.NewReader(data))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return CompletionResponse{}, fmt.Errorf("llama server returned %d: %s", resp.StatusCode, string(respBody))
	}

	var llamaResp llamaCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&llamaResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("decode response: %w", err)
	}

	finishReason := "length"
	if llamaResp.Stop {
		finishReason = "stop"
	}

	return CompletionResponse{
		Content:      llamaResp.Content,
		Model:        p.cfg.Model,
		TokensUsed:   llamaResp.TokensEvaluated + llamaResp.TokensPredicted,
		FinishReason: finishReason,
	}, nil
}

func (p *LlamaProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	body := llamaCompletionRequest{
		Prompt:      buildPrompt(req.Messages),
		NPredict:    req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.Endpoint+"/completion", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("llama server returned %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			line = strings.TrimPrefix(line, "data: ")
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			var chunk llamaCompletionResponse
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				continue
			}

			finishReason := ""
			if chunk.Stop {
				finishReason = "stop"
			}

			select {
			case ch <- StreamChunk{
				Content:      chunk.Content,
				Done:         chunk.Stop,
				FinishReason: finishReason,
			}:
			case <-ctx.Done():
				return
			}

			if chunk.Stop {
				return
			}
		}
	}()

	return ch, nil
}

// llamaEmbeddingRequest is the request payload for llama.cpp /embedding.
type llamaEmbeddingRequest struct {
	Content string `json:"content"`
}

// llamaEmbeddingResponse is the response from llama.cpp /embedding.
type llamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

func (p *LlamaProvider) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	results := make([][]float64, 0, len(texts))

	for _, text := range texts {
		body := llamaEmbeddingRequest{Content: text}
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.Endpoint+"/embedding", bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := p.client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("do request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("llama server returned %d: %s", resp.StatusCode, string(respBody))
		}

		var embResp llamaEmbeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode response: %w", err)
		}
		resp.Body.Close()

		results = append(results, embResp.Embedding)
	}

	return results, nil
}
