package weather

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenWeatherMapTool_Metadata(t *testing.T) {
	tool := NewOpenWeatherMapTool("key")
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

func TestOpenWeatherMapTool_Call_Success(t *testing.T) {
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

	tool := newOpenWeatherMapToolWithBaseURL("testkey", srv.URL, srv.Client())
	result, err := tool.Call(context.Background(), OpenWeatherMapToolInput{City: "Paris"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Paris" {
		t.Errorf("expected Name %q, got %q", "Paris", result.Name)
	}
	if result.Temp != 18.5 {
		t.Errorf("expected Temp 18.5, got %f", result.Temp)
	}
	if len(result.Descriptions) != 1 || result.Descriptions[0] != "clear sky" {
		t.Errorf("unexpected descriptions: %v", result.Descriptions)
	}
}

func TestOpenWeatherMapTool_Call_EmptyCity(t *testing.T) {
	tool := NewOpenWeatherMapTool("key")
	_, err := tool.Call(context.Background(), OpenWeatherMapToolInput{City: ""})
	if err == nil {
		t.Error("expected error for empty city")
	}
}

func TestOpenWeatherMapTool_Call_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	tool := newOpenWeatherMapToolWithBaseURL("badkey", srv.URL, srv.Client())
	_, err := tool.Call(context.Background(), OpenWeatherMapToolInput{City: "Paris"})
	if err == nil {
		t.Error("expected error for non-200 API response")
	}
}
