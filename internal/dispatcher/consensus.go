package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/danielckv/agentic-orchestration/internal/eventbus"
)

// Vote represents a single agent's vote in a consensus decision.
type Vote struct {
	AgentID        string  `json:"agent_id"`
	DecisionID     string  `json:"decision_id"`
	OptionSelected string  `json:"option_selected"`
	Confidence     float64 `json:"confidence"`
	Rationale      string  `json:"rationale"`
}

// DecisionRecord captures the full context and outcome of a consensus decision.
type DecisionRecord struct {
	DecisionID string    `json:"decision_id"`
	Context    string    `json:"context"`
	Options    []string  `json:"options"`
	Votes      []Vote    `json:"votes"`
	Winner     string    `json:"winner"`
	Timestamp  time.Time `json:"timestamp"`
}

// VoteRequest is published to initiate a vote.
type VoteRequest struct {
	DecisionID string   `json:"decision_id"`
	Context    string   `json:"context"`
	Options    []string `json:"options"`
}

// ConsensusMgr manages consensus voting through the event bus.
type ConsensusMgr struct {
	bus eventbus.EventBus
}

// NewConsensusMgr creates a new ConsensusMgr.
func NewConsensusMgr(bus eventbus.EventBus) *ConsensusMgr {
	return &ConsensusMgr{bus: bus}
}

// InitiateVote publishes a vote request to the consensus.votes stream.
func (c *ConsensusMgr) InitiateVote(ctx context.Context, decisionID, voteContext string, options []string) error {
	req := VoteRequest{
		DecisionID: decisionID,
		Context:    voteContext,
		Options:    options,
	}
	if _, err := c.bus.Publish(ctx, "consensus.votes", req); err != nil {
		return fmt.Errorf("publish vote request: %w", err)
	}
	return nil
}

// CollectVotes subscribes to consensus.votes and collects votes matching decisionID
// until expectedVoters have voted or timeout is reached.
func (c *ConsensusMgr) CollectVotes(ctx context.Context, decisionID string, expectedVoters int, timeout time.Duration) ([]Vote, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	votes := make([]Vote, 0, expectedVoters)
	voteCh := make(chan Vote, expectedVoters)
	errCh := make(chan error, 1)

	go func() {
		err := c.bus.Subscribe(ctx, "consensus.votes", "consensus-"+decisionID, "collector", func(id string, data map[string]string) error {
			payload, ok := data["payload"]
			if !ok {
				return nil
			}
			var vote Vote
			if err := json.Unmarshal([]byte(payload), &vote); err != nil {
				return nil // skip malformed votes
			}
			if vote.DecisionID == decisionID && vote.AgentID != "" {
				voteCh <- vote
			}
			return nil
		})
		if err != nil && ctx.Err() == nil {
			errCh <- err
		}
	}()

	for {
		select {
		case vote := <-voteCh:
			votes = append(votes, vote)
			if len(votes) >= expectedVoters {
				return votes, nil
			}
		case err := <-errCh:
			return votes, err
		case <-ctx.Done():
			return votes, nil // return whatever we collected
		}
	}
}

// Tally determines the winner by simple majority. On a tie, the option with
// the highest individual confidence vote wins.
func (c *ConsensusMgr) Tally(votes []Vote) string {
	if len(votes) == 0 {
		return ""
	}

	counts := make(map[string]int)
	maxConf := make(map[string]float64)

	for _, v := range votes {
		counts[v.OptionSelected]++
		if v.Confidence > maxConf[v.OptionSelected] {
			maxConf[v.OptionSelected] = v.Confidence
		}
	}

	winner := ""
	winnerCount := 0
	winnerConf := 0.0

	for option, count := range counts {
		conf := maxConf[option]
		if count > winnerCount || (count == winnerCount && conf > winnerConf) {
			winner = option
			winnerCount = count
			winnerConf = conf
		}
	}

	return winner
}
