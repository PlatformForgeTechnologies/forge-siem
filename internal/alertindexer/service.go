package alertindexer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"forge-siem/internal/config"
	"forge-siem/internal/platform"
	"forge-siem/internal/types"
)

const groupName = "alert-indexer"

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
		return fmt.Errorf("ensure alert indexer consumer group: %w", err)
	}

	indexer := platform.NewOpenSearch(s.cfg)
	consumer := "alert-indexer-" + platform.ConsumerSuffix()
	log.Printf("alert indexer started, stream=%s", s.cfg.StreamAlerts)

	return platform.ConsumeGroup(ctx, redisClient.Client(), s.cfg.StreamAlerts, groupName, consumer, func(ctx context.Context, msg redis.XMessage) error {
		alert, err := decodeAlertMessage(msg)
		if err != nil {
			log.Printf("alert indexer payload error: %v", err)
			return nil
		}
		index := "siem-alerts-" + time.Now().UTC().Format("2006.01.02")
		if err := indexer.IndexDocument(ctx, index, alert); err != nil {
			log.Printf("alert indexer write failed: %v", err)
			return err
		}
		return nil
	})
}

func decodeAlertMessage(msg redis.XMessage) (types.Alert, error) {
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
