package memory

import (
	"context"
	"testing"
)

func TestInMemoryLTMStoreAndQuery(t *testing.T) {
	ltm := NewInMemoryLTM()
	ctx := context.Background()

	// Store 3 entries with distinct content
	entries := []struct {
		id, content, source string
		metadata            map[string]string
	}{
		{
			id:      "doc-1",
			content: "Go is a statically typed compiled programming language designed at Google.",
			source:  "docs",
			metadata: map[string]string{"type": "language"},
		},
		{
			id:      "doc-2",
			content: "Redis is an in-memory data structure store used as a database and cache.",
			source:  "docs",
			metadata: map[string]string{"type": "database"},
		},
		{
			id:      "doc-3",
			content: "Kubernetes orchestrates containerized applications across a cluster of machines.",
			source:  "docs",
			metadata: map[string]string{"type": "infrastructure"},
		},
	}

	for _, e := range entries {
		if err := ltm.Store(ctx, e.id, e.content, e.source, e.metadata); err != nil {
			t.Fatalf("Store(%s): %v", e.id, err)
		}
	}

	// Query for "compiled programming language" should match doc-1 best
	results, err := ltm.Query(ctx, MemoryQuery{
		Text: "compiled programming language",
		TopK: 3,
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	if results[0].ChunkID != "doc-1" {
		t.Errorf("top result ChunkID = %q, want %q", results[0].ChunkID, "doc-1")
	}

	if results[0].Score <= 0 {
		t.Errorf("top result score = %f, want > 0", results[0].Score)
	}

	// Query with metadata filter
	results, err = ltm.Query(ctx, MemoryQuery{
		Text:    "data store",
		TopK:    10,
		Filters: map[string]string{"type": "database"},
	})
	if err != nil {
		t.Fatalf("Query with filter: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 filtered result, got %d", len(results))
	}
	if results[0].ChunkID != "doc-2" {
		t.Errorf("filtered result ChunkID = %q, want %q", results[0].ChunkID, "doc-2")
	}

	// Test Delete
	if err := ltm.Delete(ctx, "doc-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	results, err = ltm.Query(ctx, MemoryQuery{
		Text: "compiled programming language",
		TopK: 3,
	})
	if err != nil {
		t.Fatalf("Query after delete: %v", err)
	}

	for _, r := range results {
		if r.ChunkID == "doc-1" {
			t.Error("doc-1 should have been deleted but still appears in results")
		}
	}
}
