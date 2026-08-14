package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pixime-net/mcp-owm/internal/ratelimit"
)

const httpClientTimeout = 10 * time.Second

// httpClient centralizes HTTP transport, rate-limiting, and API key injection for OWM tools.
type httpClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
	limiter *ratelimit.Limiter
}

func newHTTPClient(baseURL, apiKey string, client *http.Client, limiter *ratelimit.Limiter) *httpClient {
	if client == nil {
		client = &http.Client{Timeout: httpClientTimeout}
	}
	if limiter == nil {
		limiter = ratelimit.Noop()
	}
	return &httpClient{baseURL: baseURL, apiKey: apiKey, client: client, limiter: limiter}
}

func (c *httpClient) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}

	if query == nil {
		query = url.Values{}
	}
	query.Set("appid", c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+query.Encode(), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}
