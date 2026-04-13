package dispatcher

import (
	"fmt"
	"strings"
)

// RuntimeCycleCheck performs cycle detection on the current DAG state and
// returns an error containing the cycle path if one is found.
func (d *DAG) RuntimeCycleCheck() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	const (
		white = 0 // unvisited
		gray  = 1 // in current DFS path
		black = 2 // fully processed
	)
	color := make(map[string]int, len(d.Nodes))
	parent := make(map[string]string, len(d.Nodes))

	var visit func(id string) (string, bool)
	visit = func(id string) (string, bool) {
		color[id] = gray
		node, ok := d.Nodes[id]
		if !ok {
			color[id] = black
			return "", false
		}
		for _, dep := range node.DependsOn {
			if color[dep] == gray {
				// Found a cycle — reconstruct path
				cycle := []string{dep, id}
				cur := id
				for cur != dep {
					cur = parent[cur]
					if cur == "" {
						break
					}
					cycle = append(cycle, cur)
				}
				// Reverse for readable order
				for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
					cycle[i], cycle[j] = cycle[j], cycle[i]
				}
				return strings.Join(cycle, " -> "), true
			}
			if color[dep] == white {
				parent[dep] = id
				if path, found := visit(dep); found {
					return path, true
				}
			}
		}
		color[id] = black
		return "", false
	}

	for id := range d.Nodes {
		if color[id] == white {
			if path, found := visit(id); found {
				return fmt.Errorf("cycle detected: %s", path)
			}
		}
	}
	return nil
}

// FindDeadlocks returns task IDs that are stuck: tasks that are not in
// Approved state, whose dependencies are all in a terminal state
// (Approved, Failed, or Blocked), but the task itself is not progressing
// (still Pending after all deps are done, or in Blocked state).
func (d *DAG) FindDeadlocks() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	isTerminal := func(s NodeState) bool {
		return s == NodeApproved || s == NodeFailed || s == NodeBlocked
	}

	var deadlocked []string
	for id, node := range d.Nodes {
		if node.State == NodeApproved {
			continue
		}

		// Only consider nodes that might be stuck
		if node.State != NodePending && node.State != NodeBlocked {
			continue
		}

		// Check if all dependencies are in terminal states
		allDepsTerminal := true
		anyDepFailed := false
		for _, dep := range node.DependsOn {
			depNode, ok := d.Nodes[dep]
			if !ok {
				allDepsTerminal = false
				break
			}
			if !isTerminal(depNode.State) {
				allDepsTerminal = false
				break
			}
			if depNode.State == NodeFailed || depNode.State == NodeBlocked {
				anyDepFailed = true
			}
		}

		if !allDepsTerminal {
			continue
		}

		// A Pending node with all deps approved is ready (not deadlocked)
		// unless it has been stuck without being claimed
		if node.State == NodeBlocked {
			deadlocked = append(deadlocked, id)
		} else if node.State == NodePending && anyDepFailed {
			// Pending but a dependency failed/blocked, so it can never proceed
			deadlocked = append(deadlocked, id)
		}
	}

	return deadlocked
}
