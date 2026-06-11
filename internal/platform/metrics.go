package platform

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"forge-siem/internal/config"
)

var (
	serviceInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "siem_service_info",
			Help: "Static service information.",
		},
		[]string{"service"},
	)
	streamLagGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "siem_redis_stream_lag",
			Help: "Approximate Redis stream backlog for a named stream.",
		},
		[]string{"stream"},
	)
	streamRawLagGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "siem_redis_stream_raw_lag",
			Help: "Approximate backlog for the raw events stream.",
		},
	)
	streamDecodedLagGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "siem_redis_stream_decoded_lag",
			Help: "Approximate backlog for the decoded events stream.",
		},
	)
)

func init() {
	prometheus.MustRegister(serviceInfo, streamLagGauge, streamRawLagGauge, streamDecodedLagGauge)
}

func SetServiceInfo(service string) {
	serviceInfo.WithLabelValues(service).Set(1)
}

func StartStreamLagCollector(ctx context.Context, cfg config.AppConfig, streams ...string) {
	if len(streams) == 0 {
		return
	}

	go func() {
		redisClient := NewRedis(cfg)
		defer redisClient.Close()

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		scrape := func() {
			for _, stream := range streams {
				length, err := redisClient.Client().XLen(ctx, stream).Result()
				if err != nil {
					log.Printf("metrics xlen failed for %s: %v", stream, err)
					continue
				}
				setStreamLag(stream, float64(length))
			}
		}

		scrape()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				scrape()
			}
		}
	}()
}

func setStreamLag(stream string, value float64) {
	streamLagGauge.WithLabelValues(stream).Set(value)
	switch stream {
	case "siem:raw-events":
		streamRawLagGauge.Set(value)
	case "siem:decoded-events":
		streamDecodedLagGauge.Set(value)
	default:
	}
}

func MustMetricPort(addr string) string {
	if len(addr) > 0 && addr[0] == ':' {
		return addr[1:]
	}
	panic(fmt.Sprintf("metrics address must be in :port form, got %q", addr))
}
