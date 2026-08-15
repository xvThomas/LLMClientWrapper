package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/time/rate"
)

// failingTransport always returns an error from RoundTrip.
type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("forced transport error")
}

// unmarshalable cannot be serialized to JSON.
type unmarshalable struct {
	Ch chan int
}

func TestHTTPClient_GetJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("q") != "paris" {
			t.Fatalf("unexpected query: %q", r.URL.Query().Get("q"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"label": "Paris"})
	}))
	defer srv.Close()

	client := newHTTPClient(srv.URL, srv.Client(), rate.NewLimiter(rate.Inf, 0))
	query := url.Values{"q": []string{"paris"}}

	var payload struct {
		Label string `json:"label"`
	}
	if err := client.getJSON(context.Background(), "/search", query, &payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.Label != "Paris" {
		t.Fatalf("unexpected label: %q", payload.Label)
	}
}

func TestHTTPClient_PostJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content-type: %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if payload["start"] != "A" || payload["end"] != "B" {
			t.Fatalf("unexpected body: %v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()

	client := newHTTPClient(srv.URL, srv.Client(), rate.NewLimiter(rate.Inf, 0))
	var payload map[string]string
	if err := client.postJSON(context.Background(), "/route", map[string]string{"start": "A", "end": "B"}, &payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload["ok"] != "true" {
		t.Fatalf("unexpected response: %#v", payload)
	}
}

func TestHTTPClient_HandlesNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	client := newHTTPClient(srv.URL, srv.Client(), rate.NewLimiter(rate.Inf, 0))
	var payload map[string]string
	if err := client.getJSON(context.Background(), "/error", nil, &payload); err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestNewHTTPClient_NilClientUsesDefault(t *testing.T) {
	c := newHTTPClient("http://localhost", nil, nil)
	if c.client == nil {
		t.Fatal("expected default http.Client to be created")
	}
}

func TestHTTPClient_RateLimiter_ContextCancelled(t *testing.T) {
	limiter := rate.NewLimiter(rate.Limit(0.0001), 0)
	client := newHTTPClient("http://localhost", &http.Client{}, limiter)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.doRequest(ctx, http.MethodGet, "/", nil, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context with blocked limiter")
	}
}

func TestHTTPClient_DoRequest_MarshalError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newHTTPClient(srv.URL, srv.Client(), nil)
	_, err := client.doRequest(context.Background(), http.MethodPost, "/", nil, unmarshalable{Ch: make(chan int)})
	if err == nil {
		t.Fatal("expected marshal error for non-serialisable payload")
	}
}

func TestHTTPClient_DoRequest_BuildRequestError(t *testing.T) {
	client := newHTTPClient("://bad-url", &http.Client{}, nil)
	_, err := client.doRequest(context.Background(), http.MethodGet, "", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestHTTPClient_DoRequest_ClientDoError(t *testing.T) {
	client := newHTTPClient("http://localhost", &http.Client{Transport: failingTransport{}}, nil)
	_, err := client.doRequest(context.Background(), http.MethodGet, "/", nil, nil)
	if err == nil {
		t.Fatal("expected error from failing transport")
	}
}

func TestHTTPClient_GetJSON_OutNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ignored":"value"}`))
	}))
	defer srv.Close()

	client := newHTTPClient(srv.URL, srv.Client(), nil)
	if err := client.getJSON(context.Background(), "/", nil, nil); err != nil {
		t.Fatalf("expected nil error when out is nil, got: %v", err)
	}
}

func TestHTTPClient_GetJSON_ErrorFromDoRequest(t *testing.T) {
	client := newHTTPClient("://bad-url", &http.Client{}, nil)
	if err := client.getJSON(context.Background(), "", nil, nil); err == nil {
		t.Fatal("expected error propagated from doRequest in getJSON")
	}
}

func TestHTTPClient_GetJSON_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json{{"))
	}))
	defer srv.Close()

	client := newHTTPClient(srv.URL, srv.Client(), nil)
	var result map[string]string
	if err := client.getJSON(context.Background(), "/", nil, &result); err == nil {
		t.Fatal("expected decode error for invalid JSON response")
	}
}

func TestHTTPClient_PostJSON_ErrorFromDoRequest(t *testing.T) {
	client := newHTTPClient("://bad-url", &http.Client{}, nil)
	err := client.postJSON(context.Background(), "", nil, nil)
	if err == nil {
		t.Fatal("expected error propagated from doRequest")
	}
}

func TestHTTPClient_PostJSON_OutNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ignored":"value"}`))
	}))
	defer srv.Close()

	client := newHTTPClient(srv.URL, srv.Client(), nil)
	if err := client.postJSON(context.Background(), "/", nil, nil); err != nil {
		t.Fatalf("expected nil error when out is nil, got: %v", err)
	}
}

func TestHTTPClient_PostJSON_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json{{"))
	}))
	defer srv.Close()

	client := newHTTPClient(srv.URL, srv.Client(), nil)
	var result map[string]string
	if err := client.postJSON(context.Background(), "/", nil, &result); err == nil {
		t.Fatal("expected decode error for invalid JSON response")
	}
}
