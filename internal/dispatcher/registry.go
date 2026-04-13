package dispatcher

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type AgentInfo struct {
	AgentID            string    `json:"agent_id"`
	Role               string    `json:"role"`
	Capabilities       []string  `json:"capabilities"`
	Model              string    `json:"model"`
	MaxConcurrentTasks int       `json:"max_concurrent_tasks"`
	CurrentLoad        int       `json:"current_load"`
	Session            string    `json:"session"`
	PID                int       `json:"pid"`
	LastHeartbeat      time.Time `json:"last_heartbeat"`
}

type Registry struct {
	mu             sync.RWMutex
	agents         map[string]*AgentInfo
	staleThreshold time.Duration
}

func NewRegistry(staleThreshold time.Duration) *Registry {
	return &Registry{
		agents:         make(map[string]*AgentInfo),
		staleThreshold: staleThreshold,
	}
}

func (r *Registry) Register(agent *AgentInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	agent.LastHeartbeat = time.Now()
	r.agents[agent.AgentID] = agent
}

func (r *Registry) Deregister(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, agentID)
}

func (r *Registry) Heartbeat(agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	agent, ok := r.agents[agentID]
	if !ok {
		return ErrAgentNotFound
	}
	agent.LastHeartbeat = time.Now()
	return nil
}

func (r *Registry) Get(agentID string) (*AgentInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[agentID]
	return agent, ok
}

func (r *Registry) List() []*AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cutoff := time.Now().Add(-r.staleThreshold)
	var result []*AgentInfo
	for _, agent := range r.agents {
		if agent.LastHeartbeat.After(cutoff) {
			result = append(result, agent)
		}
	}
	return result
}

func (r *Registry) ListByRole(role string) []*AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cutoff := time.Now().Add(-r.staleThreshold)
	var result []*AgentInfo
	for _, agent := range r.agents {
		if agent.Role == role && agent.LastHeartbeat.After(cutoff) {
			result = append(result, agent)
		}
	}
	return result
}

func (r *Registry) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/registry/agents", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			r.handleList(w, req)
		case http.MethodPost:
			r.handleRegister(w, req)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/registry/agents/", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/registry/agents/")
		parts := strings.Split(path, "/")

		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "missing agent id", http.StatusBadRequest)
			return
		}

		agentID := parts[0]

		if len(parts) == 2 && parts[1] == "heartbeat" && req.Method == http.MethodPut {
			r.handleHeartbeat(w, agentID)
			return
		}

		if len(parts) == 1 {
			switch req.Method {
			case http.MethodGet:
				r.handleGet(w, agentID)
			case http.MethodDelete:
				r.handleDeregister(w, agentID)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		http.NotFound(w, req)
	})

	return mux
}

func (r *Registry) handleList(w http.ResponseWriter, _ *http.Request) {
	agents := r.List()
	writeJSON(w, http.StatusOK, agents)
}

func (r *Registry) handleGet(w http.ResponseWriter, agentID string) {
	agent, ok := r.Get(agentID)
	if !ok {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (r *Registry) handleRegister(w http.ResponseWriter, req *http.Request) {
	var agent AgentInfo
	if err := json.NewDecoder(req.Body).Decode(&agent); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	r.Register(&agent)
	writeJSON(w, http.StatusCreated, &agent)
}

func (r *Registry) handleHeartbeat(w http.ResponseWriter, agentID string) {
	if err := r.Heartbeat(agentID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *Registry) handleDeregister(w http.ResponseWriter, agentID string) {
	r.Deregister(agentID)
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

var ErrAgentNotFound = &registryError{"agent not found"}

type registryError struct {
	msg string
}

func (e *registryError) Error() string { return e.msg }
