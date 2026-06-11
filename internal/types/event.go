package types

import "time"

type Event struct {
	ID        string         `json:"id"`
	Timestamp time.Time      `json:"timestamp"`
	AgentID   string         `json:"agent_id"`
	Source    string         `json:"source"`
	Category  string         `json:"category"`
	Severity  string         `json:"severity"`
	Raw       string         `json:"raw"`
	Fields    map[string]any `json:"fields"`
	Tags      []string       `json:"tags"`
}

type Alert struct {
	ID         string         `json:"id"`
	RuleID     string         `json:"rule_id"`
	AgentID    string         `json:"agent_id"`
	Severity   string         `json:"severity"`
	Title      string         `json:"title"`
	MitreTags  []string       `json:"mitre_tags"`
	EventID    string         `json:"event_id"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	Attributes map[string]any `json:"attributes"`
}

type AgentHeartbeat struct {
	AgentID   string    `json:"agent_id"`
	Hostname  string    `json:"hostname"`
	IP        string    `json:"ip"`
	OS        string    `json:"os"`
	Kernel    string    `json:"kernel"`
	Uptime    int64     `json:"uptime"`
	Timestamp time.Time `json:"timestamp"`
}

type ResponseCommand struct {
	AlertID   string         `json:"alert_id"`
	AgentID   string         `json:"agent_id"`
	Action    string         `json:"action"`
	Params    map[string]any `json:"params"`
	HMAC      string         `json:"hmac"`
	ExecuteBy time.Time      `json:"execute_by"`
}
