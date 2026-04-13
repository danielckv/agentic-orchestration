package dispatcher

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// NodeState represents the lifecycle state of a DAG node.
type NodeState string

const (
	NodePending  NodeState = "pending"
	NodeClaimed  NodeState = "claimed"
	NodeRunning  NodeState = "running"
	NodeReview   NodeState = "review"
	NodeApproved NodeState = "approved"
	NodeBlocked  NodeState = "blocked"
	NodeFailed   NodeState = "failed"
)

// DAGNode represents a single task node in the execution graph.
type DAGNode struct {
	TaskID      string
	State       NodeState
	Role        string
	Description string
	DependsOn   []string // task IDs this depends on
	AssignedTo  string   // agent ID
}

// DAG is a directed acyclic graph of task nodes with thread-safe operations.
type DAG struct {
	ID    string
	Nodes map[string]*DAGNode // keyed by task ID
	mu    sync.RWMutex
}

// NewDAG creates a new DAG with the given identifier.
func NewDAG(id string) *DAG {
	return &DAG{
		ID:    id,
		Nodes: make(map[string]*DAGNode),
	}
}

// AddNode adds a node to the DAG.
func (d *DAG) AddNode(node *DAGNode) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Nodes[node.TaskID] = node
}

// AddEdge adds a dependency edge: from must complete before to starts.
func (d *DAG) AddEdge(from, to string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if node, ok := d.Nodes[to]; ok {
		node.DependsOn = append(node.DependsOn, from)
	}
}

// GetNode returns the node for the given task ID.
func (d *DAG) GetNode(taskID string) (*DAGNode, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	node, ok := d.Nodes[taskID]
	return node, ok
}

// SetState updates the state of a node.
func (d *DAG) SetState(taskID string, state NodeState) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	node, ok := d.Nodes[taskID]
	if !ok {
		return fmt.Errorf("node %q not found", taskID)
	}
	node.State = state
	return nil
}

// GetReady returns nodes whose ALL dependencies are Approved and whose own state is Pending.
func (d *DAG) GetReady() []*DAGNode {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var ready []*DAGNode
	for _, node := range d.Nodes {
		if node.State != NodePending {
			continue
		}
		allDepsApproved := true
		for _, dep := range node.DependsOn {
			if depNode, ok := d.Nodes[dep]; ok {
				if depNode.State != NodeApproved {
					allDepsApproved = false
					break
				}
			} else {
				allDepsApproved = false
				break
			}
		}
		if allDepsApproved {
			ready = append(ready, node)
		}
	}
	return ready
}

// IsComplete returns true if all nodes are in the Approved state.
func (d *DAG) IsComplete() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, node := range d.Nodes {
		if node.State != NodeApproved {
			return false
		}
	}
	return len(d.Nodes) > 0
}

// DetectCycle returns true if the DAG contains a cycle (using DFS).
func (d *DAG) DetectCycle() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	const (
		white = 0 // unvisited
		gray  = 1 // in current path
		black = 2 // fully processed
	)
	color := make(map[string]int, len(d.Nodes))

	var visit func(id string) bool
	visit = func(id string) bool {
		color[id] = gray
		node, ok := d.Nodes[id]
		if !ok {
			color[id] = black
			return false
		}
		for _, dep := range node.DependsOn {
			switch color[dep] {
			case gray:
				return true
			case white:
				if visit(dep) {
					return true
				}
			}
		}
		color[id] = black
		return false
	}

	for id := range d.Nodes {
		if color[id] == white {
			if visit(id) {
				return true
			}
		}
	}
	return false
}

// TopologicalSort returns task IDs in topological order or an error if a cycle exists.
func (d *DAG) TopologicalSort() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.Nodes) == 0 {
		return nil, nil
	}

	// Kahn's algorithm
	inDegree := make(map[string]int, len(d.Nodes))
	// Build adjacency: for each dep->node edge
	adj := make(map[string][]string, len(d.Nodes))
	for id := range d.Nodes {
		inDegree[id] = 0
	}
	for id, node := range d.Nodes {
		for _, dep := range node.DependsOn {
			adj[dep] = append(adj[dep], id)
			inDegree[id]++
		}
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		sorted = append(sorted, curr)
		for _, next := range adj[curr] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(sorted) != len(d.Nodes) {
		return nil, errors.New("cycle detected in DAG")
	}
	return sorted, nil
}

// GetBlocked returns all nodes in the Blocked state.
func (d *DAG) GetBlocked() []*DAGNode {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var blocked []*DAGNode
	for _, node := range d.Nodes {
		if node.State == NodeBlocked {
			blocked = append(blocked, node)
		}
	}
	return blocked
}

// Summary returns a text visualization of the DAG with each node's state and dependencies.
func (d *DAG) Summary() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var b strings.Builder
	fmt.Fprintf(&b, "DAG: %s (%d nodes)\n", d.ID, len(d.Nodes))
	for id, node := range d.Nodes {
		deps := "none"
		if len(node.DependsOn) > 0 {
			deps = strings.Join(node.DependsOn, ", ")
		}
		assigned := "unassigned"
		if node.AssignedTo != "" {
			assigned = node.AssignedTo
		}
		fmt.Fprintf(&b, "  [%s] %s (role=%s, assigned=%s, deps=[%s])\n",
			node.State, id, node.Role, assigned, deps)
	}
	return b.String()
}
