package main

import (
	"context"
	"os/signal"
	"syscall"

	"forge-siem/internal/config"
	"forge-siem/internal/platform"
	"forge-siem/internal/vuln"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load("vuln-service")
	platform.StartMetricsServer(ctx, cfg.MetricsAddress, cfg.ServiceName)
	platform.Run(ctx, "vuln-service", vuln.New(cfg))
}
