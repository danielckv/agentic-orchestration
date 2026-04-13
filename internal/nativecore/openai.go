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

const openaiDefaultEndpoint = "https://api.openai.com"

// OpenAIProvider implements InferenceProvider for the OpenAI Chat Completions API.
type OpenAIProvider struct {
	cfg    ProviderConfig
	client *http.Client
}

// NewOpenAIProvider creates a new OpenAIProvider with the given configuration.
func NewOpenAIProvider(cfg ProviderConfig) *OpenAIProvider {
	if cfg.Endpoint == "" {
		cfg.Endpoint = openaiDefaultEndpoint
	}
	return &OpenAIProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

// openaiMessage is the OpenAI chat message format.
type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiRequest is the OpenAI Chat Completions API request payload.
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

// openaiResponse is the OpenAI Chat Completions API response.
type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}

	messages := make([]openaiMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openaiMessage{Role: m.Role, Content: m.Content}
	}

	body := openaiRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.Endpoint+"/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return CompletionResponse{}, fmt.Errorf("openai returned %d: %s", resp.StatusCode, string(respBody))
	}

	var oaiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&oaiResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("decode response: %w", err)
	}

	var content, finishReason string
	if len(oaiResp.Choices) > 0 {
		content = oaiResp.Choices[0].Message.Content
		finishReason = oaiResp.Choices[0].FinishReason
	}

	return CompletionResponse{
		Content:      content,
		Model:        oaiResp.Model,
		TokensUsed:   oaiResp.Usage.TotalTokens,
		FinishReason: finishReason,
	}, nil
}

// openaiStreamChunk is a single chunk from OpenAI streaming response.
type openaiStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func (p *OpenAIProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}

	messages := make([]openaiMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openaiMessage{Role: m.Role, Content: m.Content}
	}

	body := openaiRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.Endpoint+"/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	// Use a client without timeout for streaming
	streamClient := &http.Client{}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("openai returned %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			line = strings.TrimPrefix(line, "data: ")
			line = strings.TrimSpace(line)
			if line == "" || line == "[DONE]" {
				if line == "[DONE]" {
					select {
					case ch <- StreamChunk{Done: true, FinishReason: "stop"}:
					case <-ctx.Done():
					}
					return
				}
				continue
			}

			var chunk openaiStreamChunk
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			choice := chunk.Choices[0]
			sc := StreamChunk{
				Content: choice.Delta.Content,
			}

			if choice.FinishReason != nil {
				sc.Done = true
				sc.FinishReason = *choice.FinishReason
			}

			select {
			case ch <- sc:
			case <-ctx.Done():
				return
			}

			if sc.Done {
				return
			}
		}
	}()

	return ch, nil
}

// openaiEmbeddingRequest is the OpenAI Embeddings API request payload.
type openaiEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// openaiEmbeddingResponse is the OpenAI Embeddings API response.
type openaiEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func (p *OpenAIProvider) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	body := openaiEmbeddingRequest{
		Input: texts,
		Model: "text-embedding-3-small",
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.Endpoint+"/v1/embeddings", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai returned %d: %s", resp.StatusCode, string(respBody))
	}

	var embResp openaiEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	results := make([][]float64, len(embResp.Data))
	for _, d := range embResp.Data {
		results[d.Index] = d.Embedding
	}

	return results, nil
}
