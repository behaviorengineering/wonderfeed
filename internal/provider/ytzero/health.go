package ytzero

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

// HealthClient calls the provider health endpoint over HTTP.
type HealthClient struct {
	HTTP    *http.Client
	BaseURL string
}

// NewHealthClient builds a health client with a short timeout.
func NewHealthClient(baseURL string) *HealthClient {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &HealthClient{
		HTTP: &http.Client{
			Timeout: 5 * time.Second,
		},
		BaseURL: baseURL,
	}
}

// HealthResult is the parsed /api/health payload plus transport metadata.
type HealthResult struct {
	OK         bool
	StatusCode int
	Body       map[string]any
	Raw        string
}

// CheckGET performs GET /api/health.
func (c *HealthClient) CheckGET(ctx context.Context) (HealthResult, error) {
	const op = "ytzero.HealthClient.CheckGET"
	if c == nil {
		return HealthResult{}, apperr.New(apperr.CodeInvalid, op, "health client is nil")
	}
	if c.HTTP == nil {
		return HealthResult{}, apperr.New(apperr.CodeInvalid, op, "http client is nil")
	}
	url := c.BaseURL + HealthPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return HealthResult{}, apperr.Wrap(err, apperr.CodeInvalid, op, "build request").With("url", url)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return HealthResult{}, apperr.Wrap(err, apperr.CodeUnavailable, op, "request health").With("url", url)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return HealthResult{}, apperr.Wrap(err, apperr.CodeFailed, op, "read body").With("url", url)
	}
	result := HealthResult{
		OK:         resp.StatusCode == http.StatusOK,
		StatusCode: resp.StatusCode,
		Raw:        string(raw),
		Body:       map[string]any{},
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &result.Body); err != nil {
			// Health may return non-JSON; keep raw text.
			result.Body = nil
		}
	}
	if !result.OK {
		return result, apperr.New(apperr.CodeUnavailable, op, fmt.Sprintf("health status %d", resp.StatusCode)).
			With("url", url)
	}
	return result, nil
}
