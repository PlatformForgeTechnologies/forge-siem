package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type AppConfig struct {
	ServiceName        string
	DeploymentMode     string
	AllowInsecure      bool
	LogLevel           string
	ListenAddress      string
	MetricsAddress     string
	RedisAddress       string
	RedisUsername      string
	RedisPassword      string
	RedisDB            int
	PostgresDSN        string
	OpenSearchURL      string
	OpenSearchUsername string
	OpenSearchPassword string
	LokiEnabled        bool
	LokiPushURL        string
	LokiTenantID       string
	LokiUsername       string
	LokiPassword       string
	RabbitMQURL        string
	StreamRaw          string
	StreamDecoded      string
	StreamAlerts       string
	StreamDeduped      string
	StreamResponseAck  string
	TLSCertFile        string
	TLSKeyFile         string
	TLSCAFile          string
	ResponseHMACKey    string
	APIAuthToken       string
	HeartbeatTTL       time.Duration
}

func Load(service string) AppConfig {
	cfg := AppConfig{
		ServiceName:        service,
		DeploymentMode:     getenv("DEPLOYMENT_MODE", "dev"),
		AllowInsecure:      getenvBool("ALLOW_INSECURE_BACKENDS", false),
		LogLevel:           getenv("LOG_LEVEL", "info"),
		ListenAddress:      getenv("LISTEN_ADDRESS", ":8080"),
		MetricsAddress:     getenv("METRICS_ADDRESS", ":9090"),
		RedisAddress:       getenv("REDIS_ADDRESS", "redis:6379"),
		RedisUsername:      os.Getenv("REDIS_USERNAME"),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		RedisDB:            getenvInt("REDIS_DB", 2),
		PostgresDSN:        getenv("POSTGRES_DSN", "postgres://siem:siem@postgres:5432/siem?sslmode=disable"),
		OpenSearchURL:      getenv("OPENSEARCH_URL", "http://opensearch:9200"),
		OpenSearchUsername: os.Getenv("OPENSEARCH_USERNAME"),
		OpenSearchPassword: os.Getenv("OPENSEARCH_PASSWORD"),
		LokiEnabled:        getenvBool("LOKI_ENABLED", false),
		LokiPushURL:        os.Getenv("LOKI_PUSH_URL"),
		LokiTenantID:       os.Getenv("LOKI_TENANT_ID"),
		LokiUsername:       os.Getenv("LOKI_USERNAME"),
		LokiPassword:       os.Getenv("LOKI_PASSWORD"),
		RabbitMQURL:        getenv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
		StreamRaw:          getenv("STREAM_RAW", "siem:raw-events"),
		StreamDecoded:      getenv("STREAM_DECODED", "siem:decoded-events"),
		StreamAlerts:       getenv("STREAM_ALERTS", "siem:alerts"),
		StreamDeduped:      getenv("STREAM_DEDUPED", "siem:deduped-alerts"),
		StreamResponseAck:  getenv("STREAM_RESPONSE_ACKS", "siem:response-acks"),
		TLSCertFile:        os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:         os.Getenv("TLS_KEY_FILE"),
		TLSCAFile:          os.Getenv("TLS_CA_FILE"),
		ResponseHMACKey:    getenv("RESPONSE_HMAC_KEY", "change-me"),
		APIAuthToken:       os.Getenv("API_AUTH_TOKEN"),
		HeartbeatTTL:       getenvDuration("HEARTBEAT_TTL", 90*time.Second),
	}
	cfg.validate()
	return cfg
}

func (c AppConfig) validate() {
	if c.AllowInsecure || strings.EqualFold(c.DeploymentMode, "dev") {
		return
	}
	if c.ServiceName == "api" && c.APIAuthToken == "" {
		panic("API_AUTH_TOKEN is required outside dev mode")
	}
	if strings.HasPrefix(c.OpenSearchURL, "http://") {
		panic("OPENSEARCH_URL must use https outside dev mode")
	}
	if strings.Contains(c.PostgresDSN, "sslmode=disable") {
		panic("POSTGRES_DSN must not disable SSL outside dev mode")
	}
	if strings.HasPrefix(c.RabbitMQURL, "amqp://") {
		panic("RABBITMQ_URL must use amqps outside dev mode")
	}
	if c.LokiEnabled && strings.HasPrefix(c.LokiPushURL, "http://") {
		panic("LOKI_PUSH_URL must use https outside dev mode when Loki is enabled")
	}
	if c.ResponseHMACKey == "" || c.ResponseHMACKey == "change-me" {
		panic("RESPONSE_HMAC_KEY must be set outside dev mode")
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		panic(fmt.Errorf("invalid integer for %s: %w", key, err))
	}
	return value
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		panic(fmt.Errorf("invalid duration for %s: %w", key, err))
	}
	return value
}

func getenvBool(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		panic(fmt.Errorf("invalid boolean for %s", key))
	}
}
