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

const (
	anthropicDefaultEndpoint = "https://api.anthropic.com"
	anthropicAPIVersion      = "2023-06-01"
)

// AnthropicProvider implements InferenceProvider for the Anthropic Messages API.
type AnthropicProvider struct {
	cfg    ProviderConfig
	client *http.Client
}

// NewAnthropicProvider creates a new AnthropicProvider with the given configuration.
func NewAnthropicProvider(cfg ProviderConfig) *AnthropicProvider {
	if cfg.Endpoint == "" {
		cfg.Endpoint = anthropicDefaultEndpoint
	}
	return &AnthropicProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

// anthropicMessage is the Anthropic API message format.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicRequest is the Anthropic Messages API request payload.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream,omitempty"`
}

// anthropicResponse is the Anthropic Messages API response.
type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// splitSystemAndMessages separates the system message from user/assistant messages
// as required by the Anthropic API format.
func splitSystemAndMessages(msgs []Message) (string, []anthropicMessage) {
	var system string
	var messages []anthropicMessage

	for _, m := range msgs {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		messages = append(messages, anthropicMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	return system, messages
}

func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	system, messages := splitSystemAndMessages(req.Messages)

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}

	body := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  messages,
		Stream:    false,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.Endpoint+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)
	httpReq.Header.Set("x-api-key", p.cfg.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return CompletionResponse{}, fmt.Errorf("anthropic returned %d: %s", resp.StatusCode, string(respBody))
	}

	var anthResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("decode response: %w", err)
	}

	var content string
	for _, block := range anthResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return CompletionResponse{
		Content:      content,
		Model:        anthResp.Model,
		TokensUsed:   anthResp.Usage.InputTokens + anthResp.Usage.OutputTokens,
		FinishReason: anthResp.StopReason,
	}, nil
}

// anthropicSSEEvent represents a server-sent event from the Anthropic streaming API.
type anthropicSSEEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta struct {
		Type       string `json:"type"`
		Text       string `json:"text"`
		StopReason string `json:"stop_reason"`
	} `json:"delta,omitempty"`
}

func (p *AnthropicProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	system, messages := splitSystemAndMessages(req.Messages)

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}

	body := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  messages,
		Stream:    true,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.Endpoint+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)
	httpReq.Header.Set("x-api-key", p.cfg.APIKey)

	// Use a client without timeout for streaming
	streamClient := &http.Client{}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("anthropic returned %d: %s", resp.StatusCode, string(respBody))
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
				continue
			}

			var event anthropicSSEEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta.Type == "text_delta" {
					select {
					case ch <- StreamChunk{
						Content: event.Delta.Text,
						Done:    false,
					}:
					case <-ctx.Done():
						return
					}
				}
			case "message_delta":
				select {
				case ch <- StreamChunk{
					Done:         true,
					FinishReason: event.Delta.StopReason,
				}:
				case <-ctx.Done():
					return
				}
				return
			case "message_stop":
				select {
				case ch <- StreamChunk{
					Done:         true,
					FinishReason: "stop",
				}:
				case <-ctx.Done():
					return
				}
				return
			}
		}
	}()

	return ch, nil
}

func (p *AnthropicProvider) Embed(_ context.Context, _ []string) ([][]float64, error) {
	return nil, fmt.Errorf("anthropic embedding not supported")
}
