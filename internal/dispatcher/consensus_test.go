package dispatcher

import (
	"testing"
)

func TestTallyMajority(t *testing.T) {
	mgr := &ConsensusMgr{}

	votes := []Vote{
		{AgentID: "agent-1", OptionSelected: "A", Confidence: 0.8},
		{AgentID: "agent-2", OptionSelected: "A", Confidence: 0.7},
		{AgentID: "agent-3", OptionSelected: "B", Confidence: 0.9},
	}

	winner := mgr.Tally(votes)
	if winner != "A" {
		t.Fatalf("expected A to win by majority, got %s", winner)
	}
}

func TestTallyTieBreaker(t *testing.T) {
	mgr := &ConsensusMgr{}

	votes := []Vote{
		{AgentID: "agent-1", OptionSelected: "A", Confidence: 0.9},
		{AgentID: "agent-2", OptionSelected: "B", Confidence: 0.7},
	}

	winner := mgr.Tally(votes)
	if winner != "A" {
		t.Fatalf("expected A to win on confidence tie-break, got %s", winner)
	}
}

func TestTallyEmpty(t *testing.T) {
	mgr := &ConsensusMgr{}
	winner := mgr.Tally(nil)
	if winner != "" {
		t.Fatalf("expected empty string for no votes, got %s", winner)
	}
}

func TestTallyTieBreakerReversed(t *testing.T) {
	mgr := &ConsensusMgr{}

	// B has higher confidence this time
	votes := []Vote{
		{AgentID: "agent-1", OptionSelected: "A", Confidence: 0.5},
		{AgentID: "agent-2", OptionSelected: "B", Confidence: 0.95},
	}

	winner := mgr.Tally(votes)
	if winner != "B" {
		t.Fatalf("expected B to win on confidence tie-break, got %s", winner)
	}
}
