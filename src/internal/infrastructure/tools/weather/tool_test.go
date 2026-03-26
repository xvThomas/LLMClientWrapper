package weather

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTool_Metadata(t *testing.T) {
	tool := NewTool("key")
	if tool.Name() != "get_current_weather" {
		t.Errorf("unexpected tool name: %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
	params := tool.Parameters()
	if params["type"] != "object" {
		t.Error("parameters should be of type object")
	}
}

func TestTool_Execute_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := weatherResponse{Name: "Paris"}
		resp.Main.Temp = 18.5
		resp.Weather = []struct {
			Description string `json:"description"`
		}{{Description: "clear sky"}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	tool := newToolWithBaseURL("testkey", srv.URL, srv.Client())
	result, err := tool.Execute(context.Background(), map[string]any{"city": "Paris"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Paris") || !strings.Contains(result, "18.5") {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestTool_Execute_MissingCity(t *testing.T) {
	tool := NewTool("key")
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing city parameter")
	}
}

func TestTool_Execute_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	tool := newToolWithBaseURL("badkey", srv.URL, srv.Client())
	_, err := tool.Execute(context.Background(), map[string]any{"city": "Paris"})
	if err == nil {
		t.Error("expected error for non-200 API response")
	}
}
