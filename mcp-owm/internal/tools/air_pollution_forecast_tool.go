package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/time/rate"
	"github.com/pixime-net/talk-libs/mcpserver"
)

// AirPollutionForecastToolInput is the typed input for AirPollutionForecastTool.
type AirPollutionForecastToolInput struct {
	Lat float64 `json:"lat" description:"Latitude of the location"`
	Lon float64 `json:"lon" description:"Longitude of the location"`
}

// AirPollutionForecastItem represents a single forecast data point.
type AirPollutionForecastItem struct {
	DateTime   string               `json:"dt" description:"Forecast time in ISO 8601 format"`
	AQI        int                  `json:"aqi" description:"Air Quality Index: 1=Good, 2=Fair, 3=Moderate, 4=Poor, 5=Very Poor"`
	Components AirQualityComponents `json:"components" description:"Concentrations of polluting gases in μg/m3"`
}

// AirPollutionForecastToolOutput is the typed output for AirPollutionForecastTool.
type AirPollutionForecastToolOutput struct {
	Items []AirPollutionForecastItem `json:"items" description:"Hourly air pollution forecast for the next 4 days"`
}

// AirPollutionForecastTool implements mcpserver.MCPTool for fetching air pollution forecast data via OpenWeatherMap.
type AirPollutionForecastTool struct {
	client *httpClient
}

// NewAirPollutionForecastTool creates an AirPollutionForecastTool with the given API key.
func NewAirPollutionForecastTool(apiKey string, limiter *rate.Limiter) mcpserver.MCPTool[AirPollutionForecastToolInput, AirPollutionForecastToolOutput] {
	return &AirPollutionForecastTool{client: newHTTPClient(defaultBaseURL, apiKey, nil, limiter)}
}

var _ mcpserver.MCPTool[AirPollutionForecastToolInput, AirPollutionForecastToolOutput] = (*AirPollutionForecastTool)(nil)

// newAirPollutionForecastToolWithBaseURL creates an AirPollutionForecastTool with a custom base URL (for testing).
func newAirPollutionForecastToolWithBaseURL(apiKey, baseURL string, httpCl *http.Client) *AirPollutionForecastTool {
	return &AirPollutionForecastTool{client: newHTTPClient(baseURL, apiKey, httpCl, nil)}
}

// Name returns the tool name as expected by the model.
func (t *AirPollutionForecastTool) Name() string { return "get_air_pollution_forecast" }

// Description describes what the tool does.
func (t *AirPollutionForecastTool) Description() string {
	return "Get air pollution forecast for the next 4 days with hourly granularity for a given location specified by latitude and longitude. Use the geocode_city tool first to convert a city name to coordinates. Returns a list of hourly forecasts with Air Quality Index (1=Good to 5=Very Poor) and concentrations of polluting gases: CO, NO, NO2, O3, SO2, PM2.5, PM10, and NH3."
}

// Call calls the OpenWeatherMap Air Pollution Forecast API and returns a typed output struct.
func (t *AirPollutionForecastTool) Call(ctx context.Context, input AirPollutionForecastToolInput) (AirPollutionForecastToolOutput, error) {
	if input.Lat == 0 && input.Lon == 0 {
		return AirPollutionForecastToolOutput{}, fmt.Errorf("parameters 'lat' and 'lon' must not both be zero")
	}

	response, err := t.fetchForecast(ctx, input.Lat, input.Lon)
	if err != nil {
		return AirPollutionForecastToolOutput{}, err
	}

	if len(response.List) == 0 {
		return AirPollutionForecastToolOutput{}, fmt.Errorf("air pollution forecast API returned empty data")
	}

	items := make([]AirPollutionForecastItem, len(response.List))
	for i, entry := range response.List {
		items[i] = AirPollutionForecastItem{
			DateTime: time.Unix(entry.Dt, 0).UTC().Format(time.RFC3339),
			AQI:      entry.Main.AQI,
			Components: AirQualityComponents{
				CO:   entry.Components.CO,
				NO:   entry.Components.NO,
				NO2:  entry.Components.NO2,
				O3:   entry.Components.O3,
				SO2:  entry.Components.SO2,
				PM25: entry.Components.PM25,
				PM10: entry.Components.PM10,
				NH3:  entry.Components.NH3,
			},
		}
	}

	return AirPollutionForecastToolOutput{Items: items}, nil
}

func (t *AirPollutionForecastTool) fetchForecast(ctx context.Context, lat, lon float64) (*airPollutionResponse, error) {
	q := url.Values{
		"lat": {strconv.FormatFloat(lat, 'f', -1, 64)},
		"lon": {strconv.FormatFloat(lon, 'f', -1, 64)},
	}
	var data airPollutionResponse
	if err := t.client.getJSON(ctx, "/air_pollution/forecast", q, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
