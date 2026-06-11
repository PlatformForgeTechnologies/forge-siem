package main

import (
	"context"
	"os/signal"
	"syscall"

	"forge-siem/internal/archiver"
	"forge-siem/internal/config"
	"forge-siem/internal/platform"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load("raw-archiver")
	platform.StartMetricsServer(ctx, cfg.MetricsAddress, cfg.ServiceName)
	platform.StartStreamLagCollector(ctx, cfg, cfg.StreamRaw)
	platform.Run(ctx, "raw-archiver", archiver.New(cfg))
}
