package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"forge-siem/internal/config"
)

type lokiClient struct {
	pushURL  string
	tenantID string
	username string
	password string
	labels   map[string]string
	client   *http.Client
}

type lokiPushRequest struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"`
}

func newLokiClient(cfg config.AgentFileConfig, agentID string) (*lokiClient, error) {
	lokiCfg := cfg.LogCollection.Outputs.Loki
	if !lokiCfg.Enabled {
		return nil, nil
	}
	if strings.TrimSpace(lokiCfg.PushURL) == "" {
		return nil, fmt.Errorf("loki output enabled but push_url is empty")
	}

	labels := map[string]string{}
	for key, value := range lokiCfg.Labels {
		labels[key] = value
	}
	labels["agent_id"] = sanitizeLabelValue(agentID)
	labels["hostname"] = sanitizeLabelValue(hostname())

	return &lokiClient{
		pushURL:  lokiCfg.PushURL,
		tenantID: lokiCfg.TenantID,
		username: lokiCfg.Username,
		password: lokiCfg.Password,
		labels:   labels,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (c *lokiClient) push(ctx context.Context, path, line string, timestamp time.Time) error {
	labels := map[string]string{}
	for key, value := range c.labels {
		labels[key] = value
	}
	labels["source_file"] = sanitizeLabelValue(filepathBase(path))

	body, err := json.Marshal(lokiPushRequest{
		Streams: []lokiStream{
			{
				Stream: labels,
				Values: [][2]string{
					{fmt.Sprintf("%d", timestamp.UTC().UnixNano()), line},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("marshal loki request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.pushURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create loki request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.tenantID != "" {
		req.Header.Set("X-Scope-OrgID", c.tenantID)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("push to loki: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("loki push failed with status %s", resp.Status)
	}
	return nil
}

func sanitizeLabelValue(value string) string {
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		".", "_",
		":", "_",
		"-", "_",
	)
	sanitized := replacer.Replace(strings.ToLower(value))
	sanitized = strings.Trim(sanitized, "_")
	if sanitized == "" {
		return "unknown"
	}
	return sanitized
}

func filepathBase(path string) string {
	base := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 && idx < len(path)-1 {
		base = path[idx+1:]
	}
	if base == "" {
		return os.Getenv("HOSTNAME")
	}
	return base
}
