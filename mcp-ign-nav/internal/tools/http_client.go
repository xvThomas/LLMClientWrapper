package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/time/rate"
)

// httpClient centralizes the HTTP transport and rate-limiting logic used by IGN tools.
type httpClient struct {
	baseURL string
	client  *http.Client
	limiter *rate.Limiter
}

func newHTTPClient(baseURL string, client *http.Client, limiter *rate.Limiter) *httpClient {
	if client == nil {
		client = &http.Client{Timeout: httpClientTimeout}
	}
	return &httpClient{baseURL: baseURL, client: client, limiter: limiter}
}

func (c *httpClient) doRequest(ctx context.Context, method, path string, query url.Values, payload any) (*http.Response, error) {
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}
	}

	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint = endpoint + "?" + query.Encode()
	}

	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	return resp, nil
}

func (c *httpClient) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	resp, err := c.doRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return err
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

func (c *httpClient) postJSON(ctx context.Context, path string, payload any, out any) error {
	resp, err := c.doRequest(ctx, http.MethodPost, path, nil, payload)
	if err != nil {
		return err
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
