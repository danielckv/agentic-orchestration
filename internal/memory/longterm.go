package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// MemoryQuery defines search parameters for querying long-term memory.
type MemoryQuery struct {
	Text     string
	Filters  map[string]string
	TopK     int
	MinScore float64
}

// MemoryResult represents a single result from a long-term memory query.
type MemoryResult struct {
	ChunkID  string
	Content  string
	Score    float64
	Source   string
	Metadata map[string]string
}

// LongTermMemory defines the interface for persistent vector-based memory storage.
type LongTermMemory interface {
	Store(ctx context.Context, id, content, source string, metadata map[string]string) error
	Query(ctx context.Context, q MemoryQuery) ([]MemoryResult, error)
	Delete(ctx context.Context, id string) error
	Close() error
}

// memoryEntry is an internal storage record for InMemoryLTM.
type memoryEntry struct {
	ID       string
	Content  string
	Source   string
	Metadata map[string]string
}

// InMemoryLTM is a simple in-memory implementation of LongTermMemory that uses
// keyword matching as a placeholder until FAISS integration is available.
type InMemoryLTM struct {
	mu      sync.RWMutex
	entries map[string]*memoryEntry
}

// NewInMemoryLTM creates a new InMemoryLTM instance.
func NewInMemoryLTM() *InMemoryLTM {
	return &InMemoryLTM{
		entries: make(map[string]*memoryEntry),
	}
}

// Store adds or updates an entry in memory.
func (m *InMemoryLTM) Store(_ context.Context, id, content, source string, metadata map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries[id] = &memoryEntry{
		ID:       id,
		Content:  content,
		Source:   source,
		Metadata: metadata,
	}
	return nil
}

// Query performs keyword-based search over stored entries.
// Relevance is computed as the fraction of query words found in the entry content.
func (m *InMemoryLTM) Query(_ context.Context, q MemoryQuery) ([]MemoryResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queryWords := strings.Fields(strings.ToLower(q.Text))
	if len(queryWords) == 0 {
		return nil, fmt.Errorf("empty query text")
	}

	topK := q.TopK
	if topK <= 0 {
		topK = 10
	}

	var results []MemoryResult

	for _, entry := range m.entries {
		// Apply metadata filters
		if !matchesFilters(entry.Metadata, q.Filters) {
			continue
		}

		contentLower := strings.ToLower(entry.Content)
		matchCount := 0
		for _, word := range queryWords {
			if strings.Contains(contentLower, word) {
				matchCount++
			}
		}

		if matchCount == 0 {
			continue
		}

		score := float64(matchCount) / float64(len(queryWords))
		if score < q.MinScore {
			continue
		}

		results = append(results, MemoryResult{
			ChunkID:  entry.ID,
			Content:  entry.Content,
			Score:    score,
			Source:   entry.Source,
			Metadata: entry.Metadata,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// Delete removes an entry from memory by ID.
func (m *InMemoryLTM) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries, id)
	return nil
}

// Close is a no-op for InMemoryLTM.
func (m *InMemoryLTM) Close() error {
	return nil
}

// matchesFilters checks if all filter key-value pairs exist in the metadata.
func matchesFilters(metadata, filters map[string]string) bool {
	for k, v := range filters {
		if metadata[k] != v {
			return false
		}
	}
	return true
}
