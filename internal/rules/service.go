package rules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"forge-siem/internal/config"
	"forge-siem/internal/platform"
	"forge-siem/internal/types"
)

const groupName = "rules-engine"

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
	redisClient := platform.NewRedis(e.cfg)
	defer redisClient.Close()

	if err := redisClient.EnsureConsumerGroup(ctx, e.cfg.StreamDecoded, groupName); err != nil {
		return fmt.Errorf("ensure rules engine consumer group: %w", err)
	}

	consumer := "rules-engine-" + uuid.NewString()
	log.Printf("rules engine started, input=%s output=%s", e.cfg.StreamDecoded, e.cfg.StreamAlerts)
	return platform.ConsumeGroup(ctx, redisClient.Client(), e.cfg.StreamDecoded, groupName, consumer, func(ctx context.Context, msg redis.XMessage) error {
		event, err := decodeEventMessage(msg)
		if err != nil {
			log.Printf("rules payload error: %v", err)
			return nil
		}
		for _, alert := range e.Evaluate(event) {
			values, err := platform.MarshalMap(alert)
			if err != nil {
				return err
			}
			if err := redisClient.Client().XAdd(ctx, &redis.XAddArgs{
				Stream: e.cfg.StreamAlerts,
				Values: values,
			}).Err(); err != nil {
				return fmt.Errorf("xadd alert: %w", err)
			}
			log.Printf("alert emitted: %s severity=%s", alert.Title, alert.Severity)
		}
		return nil
	})
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

func decodeEventMessage(msg redis.XMessage) (types.Event, error) {
	raw, ok := msg.Values["payload"].(string)
	if !ok {
		return types.Event{}, fmt.Errorf("missing payload field")
	}
	var event types.Event
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		return types.Event{}, err
	}
	return event, nil
}
