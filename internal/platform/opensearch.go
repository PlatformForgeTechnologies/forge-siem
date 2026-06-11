package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"forge-siem/internal/config"
)

type OpenSearch struct {
	baseURL  string
	client   *http.Client
	username string
	password string
}

func NewOpenSearch(cfg config.AppConfig) *OpenSearch {
	return &OpenSearch{
		baseURL:  strings.TrimSuffix(cfg.OpenSearchURL, "/"),
		username: cfg.OpenSearchUsername,
		password: cfg.OpenSearchPassword,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (o *OpenSearch) IndexDocument(ctx context.Context, index string, doc any) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal document: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/"+index+"/_doc", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if o.username != "" || o.password != "" {
		req.SetBasicAuth(o.username, o.password)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("index request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("index request failed with status %s", resp.Status)
	}
	return nil
}
