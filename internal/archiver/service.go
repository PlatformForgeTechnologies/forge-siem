package archiver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"forge-siem/internal/config"
	"forge-siem/internal/platform"
)

const (
	groupName = "raw-archiver"
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

	if err := redisClient.EnsureConsumerGroup(ctx, s.cfg.StreamRaw, groupName); err != nil {
		return fmt.Errorf("ensure raw archiver consumer group: %w", err)
	}

	indexer := platform.NewOpenSearch(s.cfg)
	consumer := "raw-archiver-" + platform.ConsumerSuffix()
	log.Printf("raw archiver started, stream=%s", s.cfg.StreamRaw)

	return platform.ConsumeGroup(ctx, redisClient.Client(), s.cfg.StreamRaw, groupName, consumer, func(ctx context.Context, msg redis.XMessage) error {
		payload, err := payloadFromMessage(msg)
		if err != nil {
			log.Printf("raw archiver payload error: %v", err)
			return nil
		}
		index := "siem-raw-" + time.Now().UTC().Format("2006.01.02")
		if err := indexer.IndexDocument(ctx, index, payload); err != nil {
			log.Printf("raw archiver index error: %v", err)
			return err
		}
		return nil
	})
}

func payloadFromMessage(msg redis.XMessage) (map[string]any, error) {
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
