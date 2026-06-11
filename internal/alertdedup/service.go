package alertdedup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"forge-siem/internal/config"
	"forge-siem/internal/platform"
	"forge-siem/internal/types"
)

const (
	groupName = "alert-dedup"
	dedupTTL  = 60 * time.Second
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

	if err := redisClient.EnsureConsumerGroup(ctx, s.cfg.StreamAlerts, groupName); err != nil {
		return fmt.Errorf("ensure alert dedup consumer group: %w", err)
	}

	consumer := "alert-dedup-" + platform.ConsumerSuffix()
	log.Printf("alert dedup started, input=%s output=%s", s.cfg.StreamAlerts, s.cfg.StreamDeduped)

	return platform.ConsumeGroup(ctx, redisClient.Client(), s.cfg.StreamAlerts, groupName, consumer, func(ctx context.Context, msg redis.XMessage) error {
		alert, err := decodeAlert(msg)
		if err != nil {
			log.Printf("alert dedup payload error: %v", err)
			return nil
		}

		dedupKey := buildDedupKey(alert)
		accepted, err := redisClient.Client().SetNX(ctx, dedupKey, "1", dedupTTL).Result()
		if err != nil {
			return fmt.Errorf("set dedup key: %w", err)
		}
		if !accepted {
			return nil
		}

		values, err := platform.MarshalMap(alert)
		if err != nil {
			return err
		}
		if err := redisClient.Client().XAdd(ctx, &redis.XAddArgs{
			Stream: s.cfg.StreamDeduped,
			Values: values,
		}).Err(); err != nil {
			return fmt.Errorf("xadd deduped alert: %w", err)
		}
		return nil
	})
}

func decodeAlert(msg redis.XMessage) (types.Alert, error) {
	raw, ok := msg.Values["payload"].(string)
	if !ok {
		return types.Alert{}, fmt.Errorf("missing payload field")
	}
	var alert types.Alert
	if err := json.Unmarshal([]byte(raw), &alert); err != nil {
		return types.Alert{}, err
	}
	return alert, nil
}

func buildDedupKey(alert types.Alert) string {
	if key, ok := alert.Attributes["dedup_key"].(string); ok && key != "" {
		return "siem:dedup:" + alert.RuleID + ":" + alert.AgentID + ":" + key
	}
	hash := sha256.Sum256([]byte(alert.RuleID + "|" + alert.AgentID + "|" + alert.Title))
	return "siem:dedup:" + alert.RuleID + ":" + alert.AgentID + ":" + hex.EncodeToString(hash[:])
}
