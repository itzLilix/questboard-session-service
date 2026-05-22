package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/itzLilix/questboard-shared/dtos"
)

const (
	profileBatchCap     = 100
	profileHTTPTimeout  = 3 * time.Second
	internalTokenHeader = "X-Internal-Token"
)

type HTTPProfileClient struct {
	baseURL string
	client  *http.Client
	internalToken string
}

func NewHTTPProfileClient(baseURL, token string) *HTTPProfileClient {
	return &HTTPProfileClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: profileHTTPTimeout},
		internalToken: token,
	}
}

func (c *HTTPProfileClient) GetBriefs(ctx context.Context, ids []string) (map[string]dtos.UserBrief, error) {
	if len(ids) == 0 {
		return map[string]dtos.UserBrief{}, nil
	}

	seen := make(map[string]struct{}, len(ids))
	deduped := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		deduped = append(deduped, id)
	}
	if len(deduped) == 0 {
		return map[string]dtos.UserBrief{}, nil
	}
	if len(deduped) > profileBatchCap {
		return nil, fmt.Errorf("profile batch too large: %d (max %d)", len(deduped), profileBatchCap)
	}

	q := url.Values{}
	q.Set("ids", strings.Join(deduped, ","))
	endpoint := c.baseURL + "/internal/briefs?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build profile request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call profile service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profile service returned %d", resp.StatusCode)
	}

	var briefs []dtos.UserBrief
	if err := json.NewDecoder(resp.Body).Decode(&briefs); err != nil {
		return nil, fmt.Errorf("decode profile response: %w", err)
	}

	out := make(map[string]dtos.UserBrief, len(briefs))
	for _, b := range briefs {
		if b.ID == "" {
			continue
		}
		out[b.ID] = b
	}
	return out, nil
}

func (c *HTTPProfileClient) UpdateStats(ctx context.Context, stat map[string]int, statName dtos.UserStatName) error {
	endpoint := c.baseURL + "/internal/stats"

	type request struct {
		Stats map[string]int `json:"stats"`
		StatName string `json:"statName"`
	}
	content := &request{Stats: stat, StatName: string(statName)}
	jsonData, err := json.Marshal(content)
	if err != nil {
        return fmt.Errorf("encode stats body: %w", err)
    }
	
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("stats request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalTokenHeader, c.internalToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("call profile service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("profile service returned %d", resp.StatusCode)
	}
	return nil
}