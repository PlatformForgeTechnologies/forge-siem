package main

import (
	"context"
	"os/signal"
	"syscall"

	"forge-siem/internal/config"
	"forge-siem/internal/platform"
	"forge-siem/internal/rules"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load("rules-engine")
	platform.StartMetricsServer(ctx, cfg.MetricsAddress, cfg.ServiceName)
	platform.StartStreamLagCollector(ctx, cfg, cfg.StreamDecoded, cfg.StreamAlerts)
	platform.Run(ctx, "rules-engine", rules.New(cfg))
}
