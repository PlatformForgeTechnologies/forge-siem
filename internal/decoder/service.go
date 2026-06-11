package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"forge-siem/internal/config"
	"forge-siem/internal/platform"
	"forge-siem/internal/types"
)

const (
	groupName = "decoder-workers"
)

type Service struct {
	cfg config.AppConfig
}

func New(cfg config.AppConfig) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) Run(ctx context.Context) error {
	redisClient := platform.NewRedis(s.cfg)
	defer redisClient.Close()
	indexer := platform.NewOpenSearch(s.cfg)

	if err := redisClient.EnsureConsumerGroup(ctx, s.cfg.StreamRaw, groupName); err != nil {
		return fmt.Errorf("ensure decoder consumer group: %w", err)
	}

	consumer := "decoder-" + uuid.NewString()
	log.Printf("decoder worker started, source stream=%s target stream=%s", s.cfg.StreamRaw, s.cfg.StreamDecoded)
	return platform.ConsumeGroup(ctx, redisClient.Client(), s.cfg.StreamRaw, groupName, consumer, func(ctx context.Context, msg redis.XMessage) error {
		raw, err := s.rawPayload(msg)
		if err != nil {
			log.Printf("decode payload error: %v", err)
			return nil
		}

		if raw["type"] == types.EnvelopeTypeHeartbeat {
			return nil
		}

		agentID, _ := raw["agent_id"].(string)
		payload, _ := raw["payload"].(map[string]any)
		line, _ := payload["line"].(string)
		event := Decode(agentID, line)
		if path, ok := payload["path"].(string); ok {
			event.Fields["path"] = path
		}
		values, err := platform.MarshalMap(event)
		if err != nil {
			return err
		}
		if err := redisClient.Client().XAdd(ctx, &redis.XAddArgs{
			Stream: s.cfg.StreamDecoded,
			Values: values,
		}).Err(); err != nil {
			return fmt.Errorf("xadd decoded event: %w", err)
		}
		if err := indexer.IndexDocument(ctx, "siem-events-"+time.Now().UTC().Format("2006.01.02"), event); err != nil {
			return fmt.Errorf("index decoded event: %w", err)
		}
		return nil
	})
}

func Decode(agentID, payload string) types.Event {
	event := types.Event{
		ID:        uuid.NewString(),
		Timestamp: time.Now().UTC(),
		AgentID:   agentID,
		Source:    "syslog",
		Category:  "system",
		Severity:  "info",
		Raw:       payload,
		Fields:    map[string]any{},
		Tags:      []string{},
	}

	if strings.HasPrefix(strings.TrimSpace(payload), "{") {
		event.Source = "json"
		_ = json.Unmarshal([]byte(payload), &event.Fields)
		if category, ok := event.Fields["category"].(string); ok {
			event.Category = category
		}
		if severity, ok := event.Fields["severity"].(string); ok {
			event.Severity = severity
		}
	}
	if strings.Contains(payload, "sshd") {
		event.Source = "syslog"
		event.Category = "authentication"
	}
	return event
}

func (s *Service) rawPayload(msg redis.XMessage) (map[string]any, error) {
	raw, ok := msg.Values["payload"].(string)
	if !ok {
		return nil, fmt.Errorf("missing payload field")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	return payload, nil
}
