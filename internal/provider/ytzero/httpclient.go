package ytzero

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
)

// HTTPDoer performs a single HTTP round trip.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// ResilientClient wraps outbound HTTP with failsafe-go retry and circuit breaker.
type ResilientClient struct {
	BaseURL       string
	SessionCookie string
	HTTP          HTTPDoer
	breaker       circuitbreaker.CircuitBreaker[[]byte]
	retry         retrypolicy.RetryPolicy[[]byte]
}

// NewResilientClient builds a client. Panics on nil HTTP doer.
func NewResilientClient(baseURL string, httpClient HTTPDoer, sessionCookie string) *ResilientClient {
	if httpClient == nil {
		panic("ytzero.NewResilientClient: http client is nil")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	retry := retrypolicy.NewBuilder[[]byte]().
		HandleIf(func(_ []byte, err error) bool { return isTransient(err) }).
		WithBackoff(100*time.Millisecond, time.Second).
		WithJitterFactor(0.2).
		WithMaxRetries(2).
		Build()
	breaker := circuitbreaker.NewBuilder[[]byte]().
		HandleIf(func(_ []byte, err error) bool { return isTransient(err) }).
		WithFailureThreshold(5).
		WithDelay(30 * time.Second).
		Build()
	return &ResilientClient{
		BaseURL:       baseURL,
		SessionCookie: sessionCookie,
		HTTP:          httpClient,
		breaker:       breaker,
		retry:         retry,
	}
}

type permanentHTTPError struct {
	status int
	body   string
}

func (e *permanentHTTPError) Error() string {
	return fmt.Sprintf("http %d: %s", e.status, e.body)
}

type transientHTTPError struct {
	status int
	body   string
}

func (e *transientHTTPError) Error() string {
	return fmt.Sprintf("http %d: %s", e.status, e.body)
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	var t *transientHTTPError
	if apperrAsTransient(err, &t) {
		return true
	}
	// Transport / context deadline often wrap as plain errors.
	msg := err.Error()
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "temporary") ||
		strings.Contains(msg, "reset by peer")
}

func apperrAsTransient(err error, target **transientHTTPError) bool {
	for err != nil {
		if t, ok := err.(*transientHTTPError); ok {
			*target = t
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

// DoJSON performs method path with optional JSON body and returns response bytes.
func (c *ResilientClient) DoJSON(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	const op = "ytzero.ResilientClient.DoJSON"
	if c == nil {
		return nil, 0, apperr.New(apperr.CodeInvalid, op, "client is nil")
	}
	if ctx == nil {
		return nil, 0, apperr.New(apperr.CodeInvalid, op, "context is required")
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, 0, apperr.New(apperr.CodeInvalid, op, "outbound: missing deadline")
	}
	url := c.BaseURL + path
	raw, err := failsafe.With(c.breaker, c.retry).
		WithContext(ctx).
		Get(func() ([]byte, error) {
			return c.doOnce(ctx, method, url, body)
		})
	if err != nil {
		var perm *permanentHTTPError
		if asPermanent(err, &perm) {
			return nil, perm.status, apperr.Wrap(err, apperr.CodeFailed, op, "provider rejected request").
				With("url", url).With("status", fmt.Sprintf("%d", perm.status))
		}
		return nil, 0, apperr.Wrap(err, apperr.CodeUnavailable, op, "provider request failed").With("url", url)
	}
	return raw, http.StatusOK, nil
}

func asPermanent(err error, target **permanentHTTPError) bool {
	for err != nil {
		if p, ok := err.(*permanentHTTPError); ok {
			*target = p
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func (c *ResilientClient) doOnce(ctx context.Context, method, url string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.SessionCookie != "" {
		req.Header.Set("Cookie", c.SessionCookie)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return raw, nil
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, &transientHTTPError{status: resp.StatusCode, body: string(raw)}
	}
	return nil, &permanentHTTPError{status: resp.StatusCode, body: string(raw)}
}
