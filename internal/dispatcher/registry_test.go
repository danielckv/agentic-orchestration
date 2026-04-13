package dispatcher

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRegistryRegisterAndList(t *testing.T) {
	reg := NewRegistry(10 * time.Second)

	reg.Register(&AgentInfo{AgentID: "agent-1", Role: "dev"})
	reg.Register(&AgentInfo{AgentID: "agent-2", Role: "pm"})

	agents := reg.List()
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}

	found := map[string]bool{}
	for _, a := range agents {
		found[a.AgentID] = true
	}
	if !found["agent-1"] || !found["agent-2"] {
		t.Fatalf("missing expected agents: %v", found)
	}
}

func TestRegistryStaleDetection(t *testing.T) {
	reg := NewRegistry(1 * time.Millisecond)

	reg.Register(&AgentInfo{AgentID: "agent-1", Role: "dev"})
	time.Sleep(5 * time.Millisecond)

	agents := reg.List()
	if len(agents) != 0 {
		t.Fatalf("expected 0 agents (stale), got %d", len(agents))
	}
}

func TestRegistryHTTPHandler(t *testing.T) {
	reg := NewRegistry(10 * time.Second)
	srv := httptest.NewServer(reg.Handler())
	defer srv.Close()

	// POST register
	body, _ := json.Marshal(AgentInfo{AgentID: "http-agent", Role: "dev", Capabilities: []string{"code"}})
	resp, err := http.Post(srv.URL+"/registry/agents", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// GET list
	resp, err = http.Get(srv.URL + "/registry/agents")
	if err != nil {
		t.Fatalf("GET list failed: %v", err)
	}
	defer resp.Body.Close()

	var agents []*AgentInfo
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].AgentID != "http-agent" {
		t.Fatalf("expected agent_id=http-agent, got %s", agents[0].AgentID)
	}

	// GET single
	resp2, err := http.Get(srv.URL + "/registry/agents/http-agent")
	if err != nil {
		t.Fatalf("GET single failed: %v", err)
	}
	defer resp2.Body.Close()

	var agent AgentInfo
	if err := json.NewDecoder(resp2.Body).Decode(&agent); err != nil {
		t.Fatalf("decode single failed: %v", err)
	}
	if agent.AgentID != "http-agent" {
		t.Fatalf("expected http-agent, got %s", agent.AgentID)
	}
}
