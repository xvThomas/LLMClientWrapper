package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/pixime-net/mcp-owm/internal/ratelimit"
	"github.com/pixime-net/talk-libs/mcpserver"
)

// ReverseGeocodingToolInput is the typed input for ReverseGeocodingTool.
type ReverseGeocodingToolInput struct {
	Lat   float64 `json:"lat" description:"Latitude of the location"`
	Lon   float64 `json:"lon" description:"Longitude of the location"`
	Limit int     `json:"limit,omitempty" description:"Optional maximum number of results (1-5). Defaults to 5."`
}

// ReverseGeocodingToolOutput is the typed output for ReverseGeocodingTool.
type ReverseGeocodingToolOutput struct {
	Locations []GeocodingLocation `json:"locations" description:"List of location names for the given coordinates"`
}

// ReverseGeocodingTool implements mcpserver.MCPTool for reverse geocoding via OpenWeatherMap.
type ReverseGeocodingTool struct {
	client *httpClient
}

// NewReverseGeocodingTool creates a ReverseGeocodingTool with the given API key.
func NewReverseGeocodingTool(apiKey string, limiter *ratelimit.Limiter) mcpserver.MCPTool[ReverseGeocodingToolInput, ReverseGeocodingToolOutput] {
	return &ReverseGeocodingTool{client: newHTTPClient(defaultGeoBaseURL, apiKey, nil, limiter)}
}

var _ mcpserver.MCPTool[ReverseGeocodingToolInput, ReverseGeocodingToolOutput] = (*ReverseGeocodingTool)(nil)

// newReverseGeocodingToolWithBaseURL creates a ReverseGeocodingTool with a custom base URL (for testing).
func newReverseGeocodingToolWithBaseURL(apiKey, baseURL string, httpCl *http.Client) *ReverseGeocodingTool {
	return &ReverseGeocodingTool{client: newHTTPClient(baseURL, apiKey, httpCl, ratelimit.Noop())}
}

// Name returns the tool name as expected by the model.
func (t *ReverseGeocodingTool) Name() string { return "reverse_geocode" }

// Description describes what the tool does.
func (t *ReverseGeocodingTool) Description() string {
	return "Convert geographic coordinates (latitude, longitude) into location names. Returns up to 5 nearby location names for the given coordinates."
}

// Call calls the OpenWeatherMap Reverse Geocoding API and returns matching locations.
func (t *ReverseGeocodingTool) Call(ctx context.Context, input ReverseGeocodingToolInput) (ReverseGeocodingToolOutput, error) {
	if input.Lat == 0 && input.Lon == 0 {
		return ReverseGeocodingToolOutput{}, fmt.Errorf("parameters 'lat' and 'lon' must not both be zero")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 5
	}

	q := url.Values{
		"lat":   {strconv.FormatFloat(input.Lat, 'f', -1, 64)},
		"lon":   {strconv.FormatFloat(input.Lon, 'f', -1, 64)},
		"limit": {strconv.Itoa(limit)},
	}
	var results []geocodingResponse
	if err := t.client.getJSON(ctx, "/reverse", q, &results); err != nil {
		return ReverseGeocodingToolOutput{}, err
	}

	out := ReverseGeocodingToolOutput{
		Locations: make([]GeocodingLocation, 0, len(results)),
	}
	for _, r := range results {
		out.Locations = append(out.Locations, GeocodingLocation(r))
	}

	return out, nil
}
