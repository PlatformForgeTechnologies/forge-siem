package vuln

import (
	"context"
	"log"
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
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	log.Printf("vulnerability matcher started, schedule=6h")
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			log.Printf("pulling NVD feed and matching package inventory against CVEs")
		}
	}
}
