package dispatcher

import (
	"testing"
)

func TestDAGGetReady(t *testing.T) {
	dag := NewDAG("test-chain")
	dag.AddNode(&DAGNode{TaskID: "A", State: NodePending, Role: "coder"})
	dag.AddNode(&DAGNode{TaskID: "B", State: NodePending, Role: "coder"})
	dag.AddNode(&DAGNode{TaskID: "C", State: NodePending, Role: "coder"})
	dag.AddEdge("A", "B") // A -> B
	dag.AddEdge("B", "C") // B -> C

	// Only A should be ready (no dependencies)
	ready := dag.GetReady()
	if len(ready) != 1 {
		t.Fatalf("expected 1 ready node, got %d", len(ready))
	}
	if ready[0].TaskID != "A" {
		t.Fatalf("expected A to be ready, got %s", ready[0].TaskID)
	}

	// Mark A approved, B should become ready
	if err := dag.SetState("A", NodeApproved); err != nil {
		t.Fatal(err)
	}
	ready = dag.GetReady()
	if len(ready) != 1 {
		t.Fatalf("expected 1 ready node after A approved, got %d", len(ready))
	}
	if ready[0].TaskID != "B" {
		t.Fatalf("expected B to be ready, got %s", ready[0].TaskID)
	}

	// C should not be ready yet
	for _, n := range ready {
		if n.TaskID == "C" {
			t.Fatal("C should not be ready before B is approved")
		}
	}
}

func TestDAGDiamondParallelism(t *testing.T) {
	dag := NewDAG("test-diamond")
	dag.AddNode(&DAGNode{TaskID: "A", State: NodePending, Role: "planner"})
	dag.AddNode(&DAGNode{TaskID: "B", State: NodePending, Role: "coder"})
	dag.AddNode(&DAGNode{TaskID: "C", State: NodePending, Role: "coder"})
	dag.AddNode(&DAGNode{TaskID: "D", State: NodePending, Role: "reviewer"})
	dag.AddEdge("A", "B")
	dag.AddEdge("A", "C")
	dag.AddEdge("B", "D")
	dag.AddEdge("C", "D")

	// Only A ready initially
	ready := dag.GetReady()
	if len(ready) != 1 || ready[0].TaskID != "A" {
		t.Fatalf("expected only A ready, got %v", nodeIDs(ready))
	}

	// Mark A approved: B and C both ready
	dag.SetState("A", NodeApproved)
	ready = dag.GetReady()
	ids := nodeIDSet(ready)
	if len(ready) != 2 {
		t.Fatalf("expected 2 ready nodes, got %d: %v", len(ready), nodeIDs(ready))
	}
	if !ids["B"] || !ids["C"] {
		t.Fatalf("expected B and C ready, got %v", nodeIDs(ready))
	}

	// Mark B approved: D not ready (C still pending)
	dag.SetState("B", NodeApproved)
	ready = dag.GetReady()
	ids = nodeIDSet(ready)
	if ids["D"] {
		t.Fatal("D should not be ready until both B and C are approved")
	}

	// Mark C approved: D ready
	dag.SetState("C", NodeApproved)
	ready = dag.GetReady()
	if len(ready) != 1 || ready[0].TaskID != "D" {
		t.Fatalf("expected D ready, got %v", nodeIDs(ready))
	}
}

func TestDAGCycleDetection(t *testing.T) {
	dag := NewDAG("test-cycle")
	dag.AddNode(&DAGNode{TaskID: "A", State: NodePending, Role: "coder"})
	dag.AddNode(&DAGNode{TaskID: "B", State: NodePending, Role: "coder"})
	dag.AddEdge("A", "B")
	dag.AddEdge("B", "A") // creates cycle

	if !dag.DetectCycle() {
		t.Fatal("expected cycle to be detected")
	}

	// TopologicalSort should also return error
	_, err := dag.TopologicalSort()
	if err == nil {
		t.Fatal("expected error from TopologicalSort on cyclic graph")
	}
}

func TestDAGNoCycle(t *testing.T) {
	dag := NewDAG("test-no-cycle")
	dag.AddNode(&DAGNode{TaskID: "A", State: NodePending})
	dag.AddNode(&DAGNode{TaskID: "B", State: NodePending})
	dag.AddNode(&DAGNode{TaskID: "C", State: NodePending})
	dag.AddEdge("A", "B")
	dag.AddEdge("B", "C")

	if dag.DetectCycle() {
		t.Fatal("expected no cycle")
	}

	sorted, err := dag.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// A must come before B, B before C
	indexA, indexB, indexC := -1, -1, -1
	for i, id := range sorted {
		switch id {
		case "A":
			indexA = i
		case "B":
			indexB = i
		case "C":
			indexC = i
		}
	}
	if indexA >= indexB || indexB >= indexC {
		t.Fatalf("invalid topological order: %v", sorted)
	}
}

func TestDAGIsComplete(t *testing.T) {
	dag := NewDAG("test-complete")
	dag.AddNode(&DAGNode{TaskID: "A", State: NodeApproved})
	dag.AddNode(&DAGNode{TaskID: "B", State: NodeApproved})
	dag.AddNode(&DAGNode{TaskID: "C", State: NodeApproved})

	if !dag.IsComplete() {
		t.Fatal("expected DAG to be complete when all nodes are approved")
	}

	// Change one back to pending
	dag.SetState("B", NodePending)
	if dag.IsComplete() {
		t.Fatal("expected DAG to not be complete with pending node")
	}
}

func TestDAGGetBlocked(t *testing.T) {
	dag := NewDAG("test-blocked")
	dag.AddNode(&DAGNode{TaskID: "A", State: NodeApproved})
	dag.AddNode(&DAGNode{TaskID: "B", State: NodeBlocked})
	dag.AddNode(&DAGNode{TaskID: "C", State: NodePending})

	blocked := dag.GetBlocked()
	if len(blocked) != 1 || blocked[0].TaskID != "B" {
		t.Fatalf("expected B blocked, got %v", nodeIDs(blocked))
	}
}

func TestDAGSummary(t *testing.T) {
	dag := NewDAG("test-summary")
	dag.AddNode(&DAGNode{TaskID: "A", State: NodePending, Role: "coder"})
	dag.AddNode(&DAGNode{TaskID: "B", State: NodeRunning, Role: "reviewer", AssignedTo: "agent-1"})

	s := dag.Summary()
	if s == "" {
		t.Fatal("expected non-empty summary")
	}
}

// helpers

func nodeIDs(nodes []*DAGNode) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.TaskID
	}
	return ids
}

func nodeIDSet(nodes []*DAGNode) map[string]bool {
	m := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		m[n.TaskID] = true
	}
	return m
}
