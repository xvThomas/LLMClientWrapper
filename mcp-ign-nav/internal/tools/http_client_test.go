package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/time/rate"
)

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
