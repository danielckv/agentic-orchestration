package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type EventBus interface {
	Publish(ctx context.Context, stream string, msg any) (string, error)
	Subscribe(ctx context.Context, stream, group, consumer string, handler func(id string, data map[string]string) error) error
	Close() error
}

type RedisEventBus struct {
	client *redis.Client
}

func NewRedisEventBus(addr, password string, db int) (*RedisEventBus, error) {
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

	return &RedisEventBus{client: client}, nil
}

func (b *RedisEventBus) Publish(ctx context.Context, stream string, msg any) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal message: %w", err)
	}

	id, err := b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{"payload": string(data)},
	}).Result()
	if err != nil {
		return "", fmt.Errorf("xadd: %w", err)
	}

	return id, nil
}

func (b *RedisEventBus) Subscribe(ctx context.Context, stream, group, consumer string, handler func(id string, data map[string]string) error) error {
	err := b.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("create consumer group: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		streams, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  []string{stream, ">"},
			Count:    10,
			Block:    time.Second,
		}).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("xreadgroup: %w", err)
		}

		for _, s := range streams {
			for _, msg := range s.Messages {
				data := make(map[string]string, len(msg.Values))
				for k, v := range msg.Values {
					data[k] = fmt.Sprintf("%v", v)
				}

				if err := handler(msg.ID, data); err != nil {
					return fmt.Errorf("handler error for %s: %w", msg.ID, err)
				}

				if err := b.client.XAck(ctx, stream, group, msg.ID).Err(); err != nil {
					return fmt.Errorf("xack: %w", err)
				}
			}
		}
	}
}

func (b *RedisEventBus) Close() error {
	return b.client.Close()
}
