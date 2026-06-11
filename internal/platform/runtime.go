package platform

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Runnable interface {
	Run(context.Context) error
}

func Run(ctx context.Context, name string, r Runnable) {
	if err := r.Run(ctx); err != nil {
		log.Fatalf("%s failed: %v", name, err)
	}
}

func StartMetricsServer(ctx context.Context, addr string, service string) {
	mux := http.NewServeMux()
	SetServiceInfo(service)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"service": service,
			"status":  "ok",
		})
	})
	mux.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics server failed: %v", err)
		}
	}()
}
