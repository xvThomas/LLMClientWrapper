package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const defaultBaseURL = "https://api.openweathermap.org/data/2.5"

// Tool implements internal.Tool for fetching current weather via OpenWeatherMap.
type Tool struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewTool creates a WeatherTool with the given API key.
func NewTool(apiKey string) *Tool {
	return &Tool{apiKey: apiKey, baseURL: defaultBaseURL, http: &http.Client{}}
}

// newToolWithBaseURL creates a WeatherTool with a custom base URL (for testing).
func newToolWithBaseURL(apiKey, baseURL string, client *http.Client) *Tool {
	return &Tool{apiKey: apiKey, baseURL: baseURL, http: client}
}

// Name returns the tool name as expected by the model.
func (t *Tool) Name() string { return "get_current_weather" }

// Description describes what the tool does.
func (t *Tool) Description() string {
	return "Get the current weather for a given city. Returns temperature in Celsius and weather description."
}

// Parameters returns the JSON Schema for the tool input.
func (t *Tool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"city": map[string]any{
				"type":        "string",
				"description": "The city name, e.g. Paris",
			},
		},
		"required": []string{"city"},
	}
}

// Execute calls the OpenWeatherMap API and returns a formatted weather string.
func (t *Tool) Execute(ctx context.Context, input map[string]any) (string, error) {
	city, err := extractCity(input)
	if err != nil {
		return "", err
	}

	data, err := t.fetchWeather(ctx, city)
	if err != nil {
		return "", err
	}

	return formatWeather(data), nil
}

func extractCity(input map[string]any) (string, error) {
	raw, ok := input["city"]
	if !ok {
		return "", fmt.Errorf("missing required parameter 'city'")
	}
	city, ok := raw.(string)
	if !ok || city == "" {
		return "", fmt.Errorf("parameter 'city' must be a non-empty string")
	}
	return city, nil
}

type weatherResponse struct {
	Name string `json:"name"`
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
}

func (t *Tool) fetchWeather(ctx context.Context, city string) (*weatherResponse, error) {
	endpoint := fmt.Sprintf("%s/weather?q=%s&appid=%s&units=metric",
		t.baseURL, url.QueryEscape(city), t.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("building weather request: %w", err)
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weather API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API returned status %d", resp.StatusCode)
	}

	var data weatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding weather response: %w", err)
	}
	return &data, nil
}

func formatWeather(data *weatherResponse) string {
	desc := ""
	if len(data.Weather) > 0 {
		desc = data.Weather[0].Description
	}
	return fmt.Sprintf("Weather in %s: %.1f°C, %s", data.Name, data.Main.Temp, desc)
}
