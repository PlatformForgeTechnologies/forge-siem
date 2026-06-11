package types

import "time"

type Envelope struct {
	Type      string         `json:"type"`
	AgentID   string         `json:"agent_id"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

const (
	EnvelopeTypeHeartbeat = "heartbeat"
	EnvelopeTypeLog       = "log"
)
