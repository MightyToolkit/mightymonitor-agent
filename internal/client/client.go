package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MightyToolkit/mightymonitor-agent/internal/config"
	"github.com/MightyToolkit/mightymonitor-agent/internal/metrics"
)

type Client struct {
	httpClient             *http.Client
	serverURL              string
	hostToken              string
	hostID                 string
	allowInsecureLocalhost bool
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http %d: %s", e.StatusCode, e.Body)
}

func IsTransientSendError(err error) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return isRetryableStatus(httpErr.StatusCode)
	}
	return isRetryableError(err)
}

func NewClient(cfg *config.Config) *Client {
	return NewClientWithOptions(cfg, false)
}

func NewClientWithOptions(cfg *config.Config, allowInsecureLocalhost bool) *Client {
	return &Client{
		httpClient:             &http.Client{Timeout: 30 * time.Second},
		serverURL:              strings.TrimRight(cfg.ServerURL, "/"),
		hostToken:              cfg.HostToken,
		hostID:                 cfg.HostID,
		allowInsecureLocalhost: allowInsecureLocalhost,
	}
}

func (c *Client) SendPayload(ctx context.Context, payload *metrics.Payload) (*IngestResponse, error) {
	if err := c.validateServerURL(); err != nil {
		return nil, err
	}
	response := &IngestResponse{}
	if err := c.postJSON(ctx, "/v1/ingest", payload, true, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) SendBatch(ctx context.Context, payloads []*metrics.Payload) (*BatchResponse, error) {
	if err := c.validateServerURL(); err != nil {
		return nil, err
	}
	request := map[string]any{"snapshots": payloads}
	response := &BatchResponse{}
	if err := c.postJSON(ctx, "/v1/ingest/batch", request, true, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) Enroll(ctx context.Context, enrollToken string, hostname string) (*EnrollResponse, error) {
	if err := c.validateServerURL(); err != nil {
		return nil, err
	}
	request := map[string]string{
		"token":    enrollToken,
		"hostname": hostname,
	}
	response := &EnrollResponse{}
	if err := c.postJSON(ctx, "/v1/enroll", request, false, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) postJSON(ctx context.Context, path string, requestBody any, withAuth bool, out any) error {
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	fullURL := c.serverURL + path
	makeReq := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(encoded))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if withAuth {
			req.Header.Set("Authorization", "Bearer "+c.hostToken)
		}
		return req, nil
	}

	req, err := makeReq()
	if err != nil {
		return err
	}

	resp, err := c.doWithRetry(ctx, req, 3)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(bodyBytes))}
	}

	if out == nil || len(bodyBytes) == 0 {
		return nil
	}
	if err := json.Unmarshal(bodyBytes, out); err != nil {
		return err
	}
	return nil
}

func (c *Client) validateServerURL() error {
	parsed, err := url.Parse(c.serverURL)
	if err != nil {
		return err
	}

	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if c.allowInsecureLocalhost && isLocalhost(parsed.Hostname()) {
			return nil
		}
		return errors.New("Error: server URL must use HTTPS")
	default:
		return errors.New("Error: server URL must use HTTPS")
	}
}

func isLocalhost(host string) bool {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func (c *Client) doWithRetry(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error) {
	var (
		bodyBytes []byte
		err       error
	)
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		_ = req.Body.Close()
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		attemptReq := req.Clone(ctx)
		if bodyBytes != nil {
			attemptReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := c.httpClient.Do(attemptReq)
		if err == nil {
			if !isRetryableStatus(resp.StatusCode) {
				return resp, nil
			}
			if attempt == maxRetries {
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
				_ = resp.Body.Close()
				return nil, &HTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
			}
			_ = resp.Body.Close()
		} else if !isRetryableError(err) {
			return nil, err
		} else if attempt == maxRetries {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoffWithJitter(attempt)):
		}
	}

	return nil, errors.New("retry loop exhausted")
}

func isRetryableStatus(status int) bool {
	switch status {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

func isRetryableError(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return false
}

func backoffWithJitter(attempt int) time.Duration {
	base := time.Second << attempt
	jitter := 0.75 + rand.Float64()*0.5
	return time.Duration(float64(base) * jitter)
}
