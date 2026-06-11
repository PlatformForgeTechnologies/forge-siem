package rules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"forge-siem/internal/config"
	"forge-siem/internal/types"
)

type Rule struct {
	ID        string
	Title     string
	Source    string
	Category  string
	Severity  string
	MitreTags []string
	Contains  []string
	GroupBy   []string
}

type Engine struct {
	cfg   config.AppConfig
	rules []Rule
}

func New(cfg config.AppConfig) *Engine {
	return &Engine{
		cfg: cfg,
		rules: []Rule{
			{
				ID:        "ssh-bruteforce",
				Title:     "SSH brute force",
				Source:    "syslog",
				Category:  "authentication",
				Severity:  "high",
				MitreTags: []string{"T1110"},
				Contains:  []string{"Failed password"},
				GroupBy:   []string{"src_ip"},
			},
			{
				ID:        "fim-shadow-modified",
				Title:     "/etc/shadow modified",
				Source:    "fim",
				Category:  "file",
				Severity:  "critical",
				MitreTags: []string{"T1098"},
				Contains:  []string{"/etc/shadow"},
				GroupBy:   []string{"path"},
			},
		},
	}
}

func (e *Engine) Run(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Printf("rules engine started, input=%s output=%s", e.cfg.StreamDecoded, e.cfg.StreamAlerts)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			event := types.Event{
				ID:        uuid.NewString(),
				Timestamp: time.Now().UTC(),
				AgentID:   "demo-agent",
				Source:    "syslog",
				Category:  "authentication",
				Severity:  "medium",
				Raw:       "sshd Failed password for invalid user root from 10.0.0.25",
				Fields:    map[string]any{"src_ip": "10.0.0.25"},
			}
			for _, alert := range e.Evaluate(event) {
				log.Printf("alert emitted: %s severity=%s", alert.Title, alert.Severity)
			}
		}
	}
}

func (e *Engine) Evaluate(event types.Event) []types.Alert {
	var alerts []types.Alert
	for _, rule := range e.rules {
		if rule.Source != "" && rule.Source != event.Source {
			continue
		}
		if rule.Category != "" && rule.Category != event.Category {
			continue
		}
		matched := true
		for _, needle := range rule.Contains {
			if !strings.Contains(event.Raw, needle) {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		alerts = append(alerts, types.Alert{
			ID:        uuid.NewString(),
			RuleID:    rule.ID,
			AgentID:   event.AgentID,
			Severity:  rule.Severity,
			Title:     rule.Title,
			MitreTags: rule.MitreTags,
			EventID:   event.ID,
			Status:    "open",
			CreatedAt: time.Now().UTC(),
			Attributes: map[string]any{
				"dedup_key": DedupKey(rule.ID, event.AgentID, rule.GroupBy, event.Fields),
			},
		})
	}
	return alerts
}

func DedupKey(ruleID, agentID string, groupBy []string, fields map[string]any) string {
	hash := sha256.New()
	hash.Write([]byte(ruleID))
	hash.Write([]byte(agentID))
	for _, key := range groupBy {
		if value, ok := fields[key]; ok {
			hash.Write([]byte(key))
			hash.Write([]byte("|"))
			hash.Write([]byte(toString(value)))
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}
