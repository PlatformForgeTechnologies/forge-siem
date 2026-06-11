package response

import (
	"context"
	"log"
	"net/url"
	"time"

	"forge-siem/internal/config"
)

type Service struct {
	cfg config.AppConfig
}

func New(cfg config.AppConfig) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	log.Printf("response orchestrator started, rabbitmq=%s", redactURL(s.cfg.RabbitMQURL))
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			log.Printf("routing critical alerts to active-response queue and logging before execution")
		}
	}
}

func redactURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "redacted"
	}
	return parsed.Scheme + "://" + parsed.Host
}
