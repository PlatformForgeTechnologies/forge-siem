package main

import (
	"context"
	"os/signal"
	"syscall"

	"forge-siem/internal/api"
	"forge-siem/internal/config"
	"forge-siem/internal/platform"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load("api")
	platform.StartMetricsServer(ctx, cfg.MetricsAddress, cfg.ServiceName)
	platform.Run(ctx, "api", api.New(cfg))
}
