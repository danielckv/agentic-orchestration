package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/danielckv/agentic-orchestration/internal/eventbus"
	"github.com/danielckv/agentic-orchestration/internal/nativecore"
	"github.com/google/uuid"
)

// RAGPipeline processes artifacts into chunked long-term memory entries
// and listens for new approved artifacts on the event bus.
type RAGPipeline struct {
	ltm      LongTermMemory
	bus      eventbus.EventBus
	provider nativecore.InferenceProvider
}

// NewRAGPipeline creates a new RAGPipeline.
func NewRAGPipeline(ltm LongTermMemory, bus eventbus.EventBus, provider nativecore.InferenceProvider) *RAGPipeline {
	return &RAGPipeline{
		ltm:      ltm,
		bus:      bus,
		provider: provider,
	}
}

// maxChunkSize is the maximum number of characters per chunk.
const maxChunkSize = 512

// ProcessArtifact chunks the given content and stores each chunk in long-term memory.
func (r *RAGPipeline) ProcessArtifact(ctx context.Context, artifactID, content, source string) error {
	chunks := chunkContent(content, maxChunkSize)

	for i, chunk := range chunks {
		chunkID := fmt.Sprintf("%s-chunk-%d", artifactID, i)
		metadata := map[string]string{
			"artifact_id": artifactID,
			"chunk_index": fmt.Sprintf("%d", i),
			"total_chunks": fmt.Sprintf("%d", len(chunks)),
			"source":       source,
		}

		if err := r.ltm.Store(ctx, chunkID, chunk, source, metadata); err != nil {
			return fmt.Errorf("store chunk %d: %w", i, err)
		}
	}

	return nil
}

// StartListener subscribes to the artifacts.approved stream and processes
// each artifact through the RAG pipeline.
func (r *RAGPipeline) StartListener(ctx context.Context) error {
	consumerID := uuid.New().String()

	return r.bus.Subscribe(ctx, "artifacts.approved", "rag-pipeline", consumerID, func(id string, data map[string]string) error {
		payload, ok := data["payload"]
		if !ok {
			return fmt.Errorf("missing payload in message %s", id)
		}

		var artifact eventbus.ArtifactMessage
		if err := json.Unmarshal([]byte(payload), &artifact); err != nil {
			return fmt.Errorf("unmarshal artifact: %w", err)
		}

		source := artifact.AgentID
		if source == "" {
			source = "unknown"
		}

		return r.ProcessArtifact(ctx, artifact.ArtifactID, artifact.Content, source)
	})
}

// chunkContent splits content into chunks of at most maxSize characters,
// preferring to break on paragraph or sentence boundaries.
func chunkContent(content string, maxSize int) []string {
	if len(content) == 0 {
		return nil
	}

	// First, try splitting by paragraphs (double newline)
	paragraphs := strings.Split(content, "\n\n")

	var chunks []string
	var current strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// If this paragraph alone exceeds maxSize, split by sentences
		if len(para) > maxSize {
			// Flush current buffer first
			if current.Len() > 0 {
				chunks = append(chunks, current.String())
				current.Reset()
			}
			// Split long paragraph into sentence-level chunks
			chunks = append(chunks, chunkBySentences(para, maxSize)...)
			continue
		}

		// Check if adding this paragraph would exceed maxSize
		if current.Len() > 0 && current.Len()+len(para)+2 > maxSize {
			chunks = append(chunks, current.String())
			current.Reset()
		}

		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	if len(chunks) == 0 {
		chunks = append(chunks, content)
	}

	return chunks
}

// chunkBySentences splits text into chunks respecting sentence boundaries.
func chunkBySentences(text string, maxSize int) []string {
	// Simple sentence splitting on . ! ?
	var sentences []string
	current := strings.Builder{}

	for _, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			sentences = append(sentences, strings.TrimSpace(current.String()))
			current.Reset()
		}
	}
	if current.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(current.String()))
	}

	var chunks []string
	var buf strings.Builder

	for _, s := range sentences {
		if s == "" {
			continue
		}

		// If single sentence exceeds maxSize, force-split it
		if len(s) > maxSize {
			if buf.Len() > 0 {
				chunks = append(chunks, buf.String())
				buf.Reset()
			}
			for len(s) > 0 {
				end := maxSize
				if end > len(s) {
					end = len(s)
				}
				chunks = append(chunks, s[:end])
				s = s[end:]
			}
			continue
		}

		if buf.Len() > 0 && buf.Len()+len(s)+1 > maxSize {
			chunks = append(chunks, buf.String())
			buf.Reset()
		}

		if buf.Len() > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(s)
	}

	if buf.Len() > 0 {
		chunks = append(chunks, buf.String())
	}

	return chunks
}
