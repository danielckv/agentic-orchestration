package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ShortTermMemory provides Redis-backed short-term state storage for agents,
// tasks, DAG adjacency, and session variables.
type ShortTermMemory struct {
	client *redis.Client
}

// NewShortTermMemory creates a new ShortTermMemory connected to the given Redis instance.
func NewShortTermMemory(addr, password string, db int) (*ShortTermMemory, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &ShortTermMemory{client: client}, nil
}

// SetAgentStatus stores the status fields for an agent using HSET.
func (m *ShortTermMemory) SetAgentStatus(ctx context.Context, agentID string, status map[string]string) error {
	key := fmt.Sprintf("agent:%s:status", agentID)
	fields := make([]interface{}, 0, len(status)*2)
	for k, v := range status {
		fields = append(fields, k, v)
	}
	return m.client.HSet(ctx, key, fields...).Err()
}

// GetAgentStatus retrieves all status fields for an agent using HGETALL.
func (m *ShortTermMemory) GetAgentStatus(ctx context.Context, agentID string) (map[string]string, error) {
	key := fmt.Sprintf("agent:%s:status", agentID)
	return m.client.HGetAll(ctx, key).Result()
}

// SetTaskState stores the state fields for a task using HSET.
func (m *ShortTermMemory) SetTaskState(ctx context.Context, taskID string, state map[string]string) error {
	key := fmt.Sprintf("task:%s:state", taskID)
	fields := make([]interface{}, 0, len(state)*2)
	for k, v := range state {
		fields = append(fields, k, v)
	}
	return m.client.HSet(ctx, key, fields...).Err()
}

// GetTaskState retrieves all state fields for a task using HGETALL.
func (m *ShortTermMemory) GetTaskState(ctx context.Context, taskID string) (map[string]string, error) {
	key := fmt.Sprintf("task:%s:state", taskID)
	return m.client.HGetAll(ctx, key).Result()
}

// SetDAGAdjacency stores the adjacency map for a DAG using HSET.
func (m *ShortTermMemory) SetDAGAdjacency(ctx context.Context, dagID string, adjacency map[string]string) error {
	key := fmt.Sprintf("dag:%s:adjacency", dagID)
	fields := make([]interface{}, 0, len(adjacency)*2)
	for k, v := range adjacency {
		fields = append(fields, k, v)
	}
	return m.client.HSet(ctx, key, fields...).Err()
}

// GetDAGAdjacency retrieves the adjacency map for a DAG using HGETALL.
func (m *ShortTermMemory) GetDAGAdjacency(ctx context.Context, dagID string) (map[string]string, error) {
	key := fmt.Sprintf("dag:%s:adjacency", dagID)
	return m.client.HGetAll(ctx, key).Result()
}

// SetSessionVar stores a single session variable using HSET.
func (m *ShortTermMemory) SetSessionVar(ctx context.Context, sessionID, key, value string) error {
	hkey := fmt.Sprintf("session:%s:vars", sessionID)
	return m.client.HSet(ctx, hkey, key, value).Err()
}

// GetSessionVar retrieves a single session variable using HGET.
func (m *ShortTermMemory) GetSessionVar(ctx context.Context, sessionID, key string) (string, error) {
	hkey := fmt.Sprintf("session:%s:vars", sessionID)
	val, err := m.client.HGet(ctx, hkey, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Close closes the Redis connection.
func (m *ShortTermMemory) Close() error {
	return m.client.Close()
}
