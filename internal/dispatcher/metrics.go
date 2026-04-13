package dispatcher

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks Prometheus-compatible operational metrics for the CAOF framework
// using only the standard library (expvar-style atomics, no external dependencies).
type Metrics struct {
	TasksTotal        map[string]*atomic.Int64 // by state
	tasksMu           sync.RWMutex
	AgentCount        map[string]*atomic.Int64 // by role
	agentsMu          sync.RWMutex
	DAGsCompleted     atomic.Int64
	DAGsInProgress    atomic.Int64
	InferenceLatency  *LatencyTracker
	ArtifactsApproved atomic.Int64
	ArtifactsRejected atomic.Int64
	StartTime         time.Time
}

// LatencyTracker records timing statistics for inference calls.
type LatencyTracker struct {
	mu    sync.Mutex
	total time.Duration
	count int64
	max   time.Duration
}

// NewMetrics creates an initialised Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		TasksTotal:       make(map[string]*atomic.Int64),
		AgentCount:       make(map[string]*atomic.Int64),
		InferenceLatency: &LatencyTracker{},
		StartTime:        time.Now(),
	}
}

// RecordTask increments the counter for the given task state.
func (m *Metrics) RecordTask(state string) {
	m.tasksMu.RLock()
	counter, ok := m.TasksTotal[state]
	m.tasksMu.RUnlock()
	if ok {
		counter.Add(1)
		return
	}
	m.tasksMu.Lock()
	counter, ok = m.TasksTotal[state]
	if !ok {
		counter = &atomic.Int64{}
		m.TasksTotal[state] = counter
	}
	m.tasksMu.Unlock()
	counter.Add(1)
}

// RecordAgent adjusts the gauge for the given agent role by delta (+1 for register, -1 for deregister).
func (m *Metrics) RecordAgent(role string, delta int) {
	m.agentsMu.RLock()
	counter, ok := m.AgentCount[role]
	m.agentsMu.RUnlock()
	if ok {
		counter.Add(int64(delta))
		return
	}
	m.agentsMu.Lock()
	counter, ok = m.AgentCount[role]
	if !ok {
		counter = &atomic.Int64{}
		m.AgentCount[role] = counter
	}
	m.agentsMu.Unlock()
	counter.Add(int64(delta))
}

// RecordInferenceLatency records a single inference call duration.
func (m *Metrics) RecordInferenceLatency(d time.Duration) {
	m.InferenceLatency.mu.Lock()
	defer m.InferenceLatency.mu.Unlock()
	m.InferenceLatency.total += d
	m.InferenceLatency.count++
	if d > m.InferenceLatency.max {
		m.InferenceLatency.max = d
	}
}

// Handler returns an http.Handler that renders metrics in Prometheus text exposition format.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		// Tasks total by state
		fmt.Fprintln(w, "# HELP caof_tasks_total Total tasks by state")
		fmt.Fprintln(w, "# TYPE caof_tasks_total counter")
		m.tasksMu.RLock()
		taskStates := make([]string, 0, len(m.TasksTotal))
		for s := range m.TasksTotal {
			taskStates = append(taskStates, s)
		}
		m.tasksMu.RUnlock()
		sort.Strings(taskStates)
		for _, state := range taskStates {
			m.tasksMu.RLock()
			counter := m.TasksTotal[state]
			m.tasksMu.RUnlock()
			fmt.Fprintf(w, "caof_tasks_total{state=%q} %d\n", state, counter.Load())
		}

		// Agent count by role
		fmt.Fprintln(w, "# HELP caof_agents_count Current agent count by role")
		fmt.Fprintln(w, "# TYPE caof_agents_count gauge")
		m.agentsMu.RLock()
		roles := make([]string, 0, len(m.AgentCount))
		for r := range m.AgentCount {
			roles = append(roles, r)
		}
		m.agentsMu.RUnlock()
		sort.Strings(roles)
		for _, role := range roles {
			m.agentsMu.RLock()
			counter := m.AgentCount[role]
			m.agentsMu.RUnlock()
			fmt.Fprintf(w, "caof_agents_count{role=%q} %d\n", role, counter.Load())
		}

		// DAGs
		fmt.Fprintln(w, "# HELP caof_dags_completed Total DAGs completed")
		fmt.Fprintln(w, "# TYPE caof_dags_completed counter")
		fmt.Fprintf(w, "caof_dags_completed %d\n", m.DAGsCompleted.Load())

		fmt.Fprintln(w, "# HELP caof_dags_in_progress DAGs currently in progress")
		fmt.Fprintln(w, "# TYPE caof_dags_in_progress gauge")
		fmt.Fprintf(w, "caof_dags_in_progress %d\n", m.DAGsInProgress.Load())

		// Inference latency
		m.InferenceLatency.mu.Lock()
		count := m.InferenceLatency.count
		var avgMs float64
		if count > 0 {
			avgMs = float64(m.InferenceLatency.total.Milliseconds()) / float64(count)
		}
		maxMs := float64(m.InferenceLatency.max.Milliseconds())
		m.InferenceLatency.mu.Unlock()

		fmt.Fprintln(w, "# HELP caof_inference_latency_ms Inference latency in milliseconds")
		fmt.Fprintln(w, "# TYPE caof_inference_latency_ms summary")
		fmt.Fprintf(w, "caof_inference_latency_avg_ms %f\n", avgMs)
		fmt.Fprintf(w, "caof_inference_latency_max_ms %f\n", maxMs)
		fmt.Fprintf(w, "caof_inference_latency_count %d\n", count)

		// Artifacts
		fmt.Fprintln(w, "# HELP caof_artifacts_approved Total approved artifacts")
		fmt.Fprintln(w, "# TYPE caof_artifacts_approved counter")
		fmt.Fprintf(w, "caof_artifacts_approved %d\n", m.ArtifactsApproved.Load())

		fmt.Fprintln(w, "# HELP caof_artifacts_rejected Total rejected artifacts")
		fmt.Fprintln(w, "# TYPE caof_artifacts_rejected counter")
		fmt.Fprintf(w, "caof_artifacts_rejected %d\n", m.ArtifactsRejected.Load())

		// Uptime
		fmt.Fprintln(w, "# HELP caof_uptime_seconds Seconds since process start")
		fmt.Fprintln(w, "# TYPE caof_uptime_seconds gauge")
		fmt.Fprintf(w, "caof_uptime_seconds %f\n", time.Since(m.StartTime).Seconds())
	})
}

// JSON returns all metrics as a JSON-serialisable map.
func (m *Metrics) JSON() map[string]any {
	result := make(map[string]any)

	// Tasks
	tasks := make(map[string]int64)
	m.tasksMu.RLock()
	for state, counter := range m.TasksTotal {
		tasks[state] = counter.Load()
	}
	m.tasksMu.RUnlock()
	result["tasks_total"] = tasks

	// Agents
	agents := make(map[string]int64)
	m.agentsMu.RLock()
	for role, counter := range m.AgentCount {
		agents[role] = counter.Load()
	}
	m.agentsMu.RUnlock()
	result["agent_count"] = agents

	result["dags_completed"] = m.DAGsCompleted.Load()
	result["dags_in_progress"] = m.DAGsInProgress.Load()

	m.InferenceLatency.mu.Lock()
	latency := map[string]any{
		"count": m.InferenceLatency.count,
		"max_ms": float64(m.InferenceLatency.max.Milliseconds()),
	}
	if m.InferenceLatency.count > 0 {
		latency["avg_ms"] = float64(m.InferenceLatency.total.Milliseconds()) / float64(m.InferenceLatency.count)
	} else {
		latency["avg_ms"] = float64(0)
	}
	m.InferenceLatency.mu.Unlock()
	result["inference_latency"] = latency

	result["artifacts_approved"] = m.ArtifactsApproved.Load()
	result["artifacts_rejected"] = m.ArtifactsRejected.Load()
	result["uptime_seconds"] = time.Since(m.StartTime).Seconds()

	return result
}
