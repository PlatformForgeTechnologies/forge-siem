package main

import (
	"context"
	"os/signal"
	"syscall"

	"forge-siem/internal/config"
	"forge-siem/internal/decoder"
	"forge-siem/internal/platform"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load("decoder")
	platform.StartMetricsServer(ctx, cfg.MetricsAddress, cfg.ServiceName)
	platform.StartStreamLagCollector(ctx, cfg, cfg.StreamRaw, cfg.StreamDecoded)
	platform.Run(ctx, "decoder", decoder.New(cfg))
}
