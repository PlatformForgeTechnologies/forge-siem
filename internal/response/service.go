package response

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"

	"forge-siem/internal/config"
	"forge-siem/internal/platform"
	"forge-siem/internal/types"
)

type Service struct {
	cfg config.AppConfig
}

func New(cfg config.AppConfig) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) Run(ctx context.Context) error {
	log.Printf("response orchestrator started, rabbitmq=%s", redactURL(s.cfg.RabbitMQURL))
	conn, err := amqp.Dial(s.cfg.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("connect rabbitmq: %w", err)
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("create rabbitmq channel: %w", err)
	}
	defer channel.Close()

	if err := setupBroker(channel); err != nil {
		return err
	}

	redisClient := platform.NewRedis(s.cfg)
	defer redisClient.Close()

	if err := redisClient.EnsureConsumerGroup(ctx, s.cfg.StreamDeduped, "response-router"); err != nil {
		return fmt.Errorf("ensure response router consumer group: %w", err)
	}

	consumer := "response-router-" + platform.ConsumerSuffix()
	return platform.ConsumeGroup(ctx, redisClient.Client(), s.cfg.StreamDeduped, "response-router", consumer, func(ctx context.Context, msg redis.XMessage) error {
		alert, err := decodeAlert(msg)
		if err != nil {
			log.Printf("response router payload error: %v", err)
			return nil
		}
		if err := publishAlert(channel, alert); err != nil {
			return fmt.Errorf("publish alert to rabbitmq: %w", err)
		}
		return nil
	})
}

func redactURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "redacted"
	}
	return parsed.Scheme + "://" + parsed.Host
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
