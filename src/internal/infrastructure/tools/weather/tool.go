package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"llmclientwrapper/src/internal/domain"
	"net/http"
	"net/url"
)

const defaultBaseURL = "https://api.openweathermap.org/data/2.5"

// OpenWeatherMapToolInput is the typed input for OpenWeatherMapTool.
type OpenWeatherMapToolInput struct {
	City string `json:"city"`
}

// OpenWeatherMapToolOutput is the typed output for OpenWeatherMapTool.
type OpenWeatherMapToolOutput struct {
	Name         string   `json:"name"`
	Temp         float64  `json:"temp"`
	Descriptions []string `json:"descriptions"`
}

// OpenWeatherMapTool implements domain.TypedTool for fetching current weather via OpenWeatherMap.
type OpenWeatherMapTool struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewOpenWeatherMapTool creates an OpenWeatherMapTool with the given API key.
func NewOpenWeatherMapTool(apiKey string) *OpenWeatherMapTool {
	return &OpenWeatherMapTool{apiKey: apiKey, baseURL: defaultBaseURL, http: &http.Client{}}
}

var _ domain.TypedTool[OpenWeatherMapToolInput, OpenWeatherMapToolOutput] = (*OpenWeatherMapTool)(nil)

// newOpenWeatherMapToolWithBaseURL creates an OpenWeatherMapTool with a custom base URL (for testing).
func newOpenWeatherMapToolWithBaseURL(apiKey, baseURL string, client *http.Client) *OpenWeatherMapTool {
	return &OpenWeatherMapTool{apiKey: apiKey, baseURL: baseURL, http: client}
}

// Name returns the tool name as expected by the model.
func (t *OpenWeatherMapTool) Name() string { return "get_current_weather" }

// Description describes what the tool does.
func (t *OpenWeatherMapTool) Description() string {
	return "Get the current weather for a given city. Returns temperature in Celsius and weather descriptions."
}

// Parameters returns the JSON Schema for the tool input.
func (t *OpenWeatherMapTool) Parameters() map[string]any {
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

// Call calls the OpenWeatherMap API and returns a typed output struct.
func (t *OpenWeatherMapTool) Call(ctx context.Context, input OpenWeatherMapToolInput) (OpenWeatherMapToolOutput, error) {
	if input.City == "" {
		return OpenWeatherMapToolOutput{}, fmt.Errorf("parameter 'city' must be a non-empty string")
	}

	data, err := t.fetchWeather(ctx, input.City)
	if err != nil {
		return OpenWeatherMapToolOutput{}, err
	}

	descs := make([]string, 0, len(data.Weather))
	for _, w := range data.Weather {
		descs = append(descs, w.Description)
	}
	return OpenWeatherMapToolOutput{
		Name:         data.Name,
		Temp:         data.Main.Temp,
		Descriptions: descs,
	}, nil
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

func (t *OpenWeatherMapTool) fetchWeather(ctx context.Context, city string) (*weatherResponse, error) {
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
