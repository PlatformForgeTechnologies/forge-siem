package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"forge-siem/internal/config"
)

type Redis struct {
	client *redis.Client
}

func NewRedis(cfg config.AppConfig) *Redis {
	return &Redis{
		client: redis.NewClient(&redis.Options{
			Addr:         cfg.RedisAddress,
			Username:     cfg.RedisUsername,
			Password:     cfg.RedisPassword,
			DB:           cfg.RedisDB,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		}),
	}
}

func (r *Redis) Client() *redis.Client {
	return r.client
}

func (r *Redis) Close() error {
	return r.client.Close()
}

func (r *Redis) EnsureConsumerGroup(ctx context.Context, stream, group string) error {
	err := r.client.XGroupCreateMkStream(ctx, stream, group, "$").Err()
	if err == nil || err.Error() == "BUSYGROUP Consumer Group name already exists" {
		return nil
	}
	return err
}

func MarshalMap(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal value: %w", err)
	}
	return map[string]any{"payload": string(data)}, nil
}
