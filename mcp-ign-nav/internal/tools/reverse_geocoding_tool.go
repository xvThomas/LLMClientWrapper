package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pixime-net/talk-libs/mcpserver"
	"golang.org/x/time/rate"
)

const (
	defaultBaseURL    = "https://data.geopf.fr/geocodage"
	maxCoordinates    = 10
	defaultIndex      = "address"
	defaultLimit      = 1
	httpClientTimeout = 10 * time.Second
)

// Coordinate represents a WGS84 geographic coordinate pair.
type Coordinate struct {
	Lon float64 `json:"lon" description:"Longitude (WGS84)"`
	Lat float64 `json:"lat" description:"Latitude (WGS84)"`
}

// ReverseGeocodingToolInput is the typed input for the reverse geocoding tool.
type ReverseGeocodingToolInput struct {
	Coordinates []Coordinate `json:"coordinates" description:"List of WGS84 coordinate pairs to reverse geocode (min 1, max 10)"`
	Index       string       `json:"index,omitempty" description:"Index to search: address, poi, or parcel. Defaults to address."`
	Limit       int          `json:"limit,omitempty" description:"Maximum number of results per coordinate (1-50). Defaults to 1."`
}

// Feature represents a single geocoded result from the IGN API.
type Feature struct {
	Label       string  `json:"label" description:"Full formatted address"`
	City        string  `json:"city" description:"City name"`
	Postcode    string  `json:"postcode" description:"Postal code"`
	Street      string  `json:"street,omitempty" description:"Street name"`
	HouseNumber string  `json:"housenumber,omitempty" description:"House number"`
	Type        string  `json:"type" description:"Result type: housenumber, street, locality, or municipality"`
	Score       float64 `json:"score" description:"Relevance score"`
	Distance    float64 `json:"distance" description:"Distance in meters from the queried point"`
}

// ReverseGeocodingResult holds results for a single coordinate query.
type ReverseGeocodingResult struct {
	Lon      float64   `json:"lon" description:"Queried longitude"`
	Lat      float64   `json:"lat" description:"Queried latitude"`
	Features []Feature `json:"features" description:"Matching addresses found near the coordinate"`
}

// ReverseGeocodingToolOutput is the typed output for the reverse geocoding tool.
type ReverseGeocodingToolOutput struct {
	Results []ReverseGeocodingResult `json:"results" description:"Reverse geocoding results for each input coordinate"`
}

// ReverseGeocodingTool implements mcpserver.MCPTool for reverse geocoding via the IGN Géoplateforme API.
type ReverseGeocodingTool struct {
	client *httpClient
}

var _ mcpserver.MCPTool[ReverseGeocodingToolInput, ReverseGeocodingToolOutput] = (*ReverseGeocodingTool)(nil)

// NewReverseGeocodingTool creates a ReverseGeocodingTool using the IGN Géoplateforme API.
func NewReverseGeocodingTool(limiter *rate.Limiter) *ReverseGeocodingTool {
	return &ReverseGeocodingTool{client: newHTTPClient(defaultBaseURL, &http.Client{Timeout: httpClientTimeout}, limiter)}
}

// newReverseGeocodingToolWithBaseURL creates a ReverseGeocodingTool with a custom base URL (for testing).
func newReverseGeocodingToolWithBaseURL(baseURL string, client *http.Client) *ReverseGeocodingTool {
	return &ReverseGeocodingTool{client: newHTTPClient(baseURL, client, rate.NewLimiter(rate.Inf, 0))}
}

// Name returns the tool name.
func (t *ReverseGeocodingTool) Name() string { return "reverse_geocode" }

// Description describes what the tool does.
func (t *ReverseGeocodingTool) Description() string {
	return "Convert WGS84 geographic coordinates (longitude, latitude) into French addresses using the IGN Géoplateforme reverse geocoding API. Accepts 1 to 10 coordinate pairs."
}

// Call performs reverse geocoding for each input coordinate by calling the IGN API sequentially.
func (t *ReverseGeocodingTool) Call(ctx context.Context, input ReverseGeocodingToolInput) (ReverseGeocodingToolOutput, error) {
	if len(input.Coordinates) == 0 {
		return ReverseGeocodingToolOutput{}, fmt.Errorf("at least one coordinate is required")
	}
	if len(input.Coordinates) > maxCoordinates {
		return ReverseGeocodingToolOutput{}, fmt.Errorf("too many coordinates: got %d, maximum is %d", len(input.Coordinates), maxCoordinates)
	}

	index := input.Index
	if index == "" {
		index = defaultIndex
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	results := make([]ReverseGeocodingResult, 0, len(input.Coordinates))

	for _, coord := range input.Coordinates {
		features, err := t.reverseGeocode(ctx, coord, index, limit)
		if err != nil {
			return ReverseGeocodingToolOutput{}, fmt.Errorf("reverse geocoding (%f, %f): %w", coord.Lon, coord.Lat, err)
		}
		results = append(results, ReverseGeocodingResult{
			Lon:      coord.Lon,
			Lat:      coord.Lat,
			Features: features,
		})
	}

	return ReverseGeocodingToolOutput{Results: results}, nil
}

func (t *ReverseGeocodingTool) reverseGeocode(ctx context.Context, coord Coordinate, index string, limit int) ([]Feature, error) {
	params := url.Values{}
	params.Set("lon", strconv.FormatFloat(coord.Lon, 'f', -1, 64))
	params.Set("lat", strconv.FormatFloat(coord.Lat, 'f', -1, 64))
	params.Set("index", index)
	params.Set("limit", strconv.Itoa(limit))

	var fc featureCollection
	if err := t.client.getJSON(ctx, "/reverse", params, &fc); err != nil {
		return nil, err
	}

	features := make([]Feature, 0, len(fc.Features))
	for _, f := range fc.Features {
		features = append(features, Feature{
			Label:       f.Properties.Label,
			City:        f.Properties.City,
			Postcode:    f.Properties.Postcode,
			Street:      f.Properties.Street,
			HouseNumber: f.Properties.HouseNumber,
			Type:        f.Properties.Type,
			Score:       f.Properties.Score,
			Distance:    f.Properties.Distance,
		})
	}

	return features, nil
}

// featureCollection represents the GeoJSON FeatureCollection returned by the IGN API.
type featureCollection struct {
	Type     string       `json:"type"`
	Features []ignFeature `json:"features"`
}

// ignFeature represents a single GeoJSON Feature from the IGN reverse geocoding response.
type ignFeature struct {
	Type       string        `json:"type"`
	Properties ignProperties `json:"properties"`
}

// ignProperties holds the properties of a reverse-geocoded feature.
type ignProperties struct {
	Label       string  `json:"label"`
	City        string  `json:"city"`
	Postcode    string  `json:"postcode"`
	Street      string  `json:"street"`
	HouseNumber string  `json:"housenumber"`
	Type        string  `json:"type"`
	Score       float64 `json:"_score"`
	Distance    float64 `json:"distance"`
}
