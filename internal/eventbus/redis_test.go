package eventbus

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestPublishSubscribeRoundTrip(t *testing.T) {
	bus, err := NewRedisEventBus("localhost:6379", "", 0)
	if err != nil {
		t.Skip("redis not available:", err)
	}
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := "test.tasks.pending." + time.Now().Format("20060102150405.000000000")
	group := "test-group"
	consumer := "test-consumer"

	sent := TaskMessage{
		TaskID:       "task-001",
		RoleRequired: "coder",
		Priority:     1,
		Spec: TaskSpec{
			Description:        "implement feature X",
			Constraints:        []string{"no external deps"},
			AcceptanceCriteria: []string{"tests pass"},
		},
		CreatedAt:  time.Now().UTC().Truncate(time.Millisecond),
		TTLSeconds: 300,
	}

	_, err = bus.Publish(ctx, stream, sent)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	received := make(chan TaskMessage, 1)
	errCh := make(chan error, 1)

	go func() {
		errCh <- bus.Subscribe(ctx, stream, group, consumer, func(id string, data map[string]string) error {
			var msg TaskMessage
			if err := json.Unmarshal([]byte(data["payload"]), &msg); err != nil {
				return err
			}
			received <- msg
			return nil
		})
	}()

	select {
	case msg := <-received:
		if msg.TaskID != sent.TaskID {
			t.Errorf("task_id = %q, want %q", msg.TaskID, sent.TaskID)
		}
		if msg.RoleRequired != sent.RoleRequired {
			t.Errorf("role_required = %q, want %q", msg.RoleRequired, sent.RoleRequired)
		}
		if msg.Spec.Description != sent.Spec.Description {
			t.Errorf("description = %q, want %q", msg.Spec.Description, sent.Spec.Description)
		}
		if msg.Priority != sent.Priority {
			t.Errorf("priority = %d, want %d", msg.Priority, sent.Priority)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	cancel()

	// Clean up test stream
	cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cleanCancel()
	bus.client.Del(cleanCtx, stream)
}
