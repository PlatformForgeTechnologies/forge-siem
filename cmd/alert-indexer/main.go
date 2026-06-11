package main

import (
	"context"
	"os/signal"
	"syscall"

	"forge-siem/internal/alertindexer"
	"forge-siem/internal/config"
	"forge-siem/internal/platform"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load("alert-indexer")
	platform.StartMetricsServer(ctx, cfg.MetricsAddress, cfg.ServiceName)
	platform.StartStreamLagCollector(ctx, cfg, cfg.StreamAlerts)
	platform.Run(ctx, "alert-indexer", alertindexer.New(cfg))
}
