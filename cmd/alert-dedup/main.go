package main

import (
	"context"
	"os/signal"
	"syscall"

	"forge-siem/internal/alertdedup"
	"forge-siem/internal/config"
	"forge-siem/internal/platform"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load("alert-dedup")
	platform.StartMetricsServer(ctx, cfg.MetricsAddress, cfg.ServiceName)
	platform.StartStreamLagCollector(ctx, cfg, cfg.StreamAlerts, cfg.StreamDeduped)
	platform.Run(ctx, "alert-dedup", alertdedup.New(cfg))
}
