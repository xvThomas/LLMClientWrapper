package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/time/rate"
)

// errorTransport always returns an error from RoundTrip.
type errorTransport struct{}

func (errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("forced transport error")
}

func TestOWMHTTPClient_RateLimiter_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()

	limiter := rate.NewLimiter(rate.Inf, 1)
	client := newHTTPClient(srv.URL, "testkey", srv.Client(), limiter)

	var result map[string]string
	if err := client.getJSON(context.Background(), "/", nil, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOWMHTTPClient_RateLimiter_ContextCancelled(t *testing.T) {
	// Burst=0, rate near-zero: Wait will block until context is cancelled.
	limiter := rate.NewLimiter(rate.Limit(0.0001), 0)
	client := newHTTPClient("http://localhost", "key", &http.Client{}, limiter)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.getJSON(ctx, "/", nil, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestOWMHTTPClient_NilQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("appid") != "testkey" {
			t.Errorf("expected appid in query, got: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer srv.Close()

	client := newHTTPClient(srv.URL, "testkey", srv.Client(), nil)
	var result map[string]string
	if err := client.getJSON(context.Background(), "/", nil, &result); err != nil {
		t.Fatalf("unexpected error with nil query: %v", err)
	}
}

func TestOWMHTTPClient_BuildRequestError(t *testing.T) {
	client := newHTTPClient("://bad-url", "key", &http.Client{}, nil)
	err := client.getJSON(context.Background(), "", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestOWMHTTPClient_ClientDoError(t *testing.T) {
	client := newHTTPClient("http://localhost", "key", &http.Client{Transport: errorTransport{}}, nil)
	err := client.getJSON(context.Background(), "/", url.Values{}, nil)
	if err == nil {
		t.Fatal("expected error from failing transport")
	}
}

func TestOWMHTTPClient_OutNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":"ignored"}`))
	}))
	defer srv.Close()

	client := newHTTPClient(srv.URL, "key", srv.Client(), nil)
	err := client.getJSON(context.Background(), "/", nil, nil)
	if err != nil {
		t.Fatalf("expected nil error when out is nil, got: %v", err)
	}
}

func TestOWMHTTPClient_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not valid json{{"))
	}))
	defer srv.Close()

	client := newHTTPClient(srv.URL, "key", srv.Client(), nil)
	var result map[string]string
	err := client.getJSON(context.Background(), "/", nil, &result)
	if err == nil {
		t.Fatal("expected decode error for invalid JSON response")
	}
}
