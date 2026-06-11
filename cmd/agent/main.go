package main

import (
	"context"
	"os/signal"
	"syscall"

	"forge-siem/internal/agent"
	"forge-siem/internal/config"
	"forge-siem/internal/platform"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load("agent")
	platform.StartMetricsServer(ctx, cfg.MetricsAddress, cfg.ServiceName)
	platform.Run(ctx, "agent", agent.New(cfg))
}
