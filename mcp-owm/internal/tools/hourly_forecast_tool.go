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

const defaultProBaseURL = "https://pro.openweathermap.org/data/2.5"

// HourlyForecastToolInput is the typed input for HourlyForecastTool.
type HourlyForecastToolInput struct {
	Lat   float64 `json:"lat" description:"Latitude of the location"`
	Lon   float64 `json:"lon" description:"Longitude of the location"`
	Count int     `json:"count,omitempty" description:"Optional number of hourly timestamps to return (1-96). If omitted, returns all 96 timestamps (4 days)."`
}

// HourlyForecastToolOutput is the typed output for HourlyForecastTool.
type HourlyForecastToolOutput struct {
	City      ForecastCity    `json:"city" description:"City information"`
	Count     int             `json:"cnt" description:"Number of forecast entries"`
	Forecasts []ForecastEntry `json:"forecasts" description:"Hourly forecast entries (up to 96 timestamps for 4 days)"`
}

// HourlyForecastTool implements mcpserver.MCPTool for fetching 4-day hourly forecast via OpenWeatherMap Pro.
type HourlyForecastTool struct {
	client *httpClient
}

// NewHourlyForecastTool creates a HourlyForecastTool with the given API key.
func NewHourlyForecastTool(apiKey string, limiter *rate.Limiter) mcpserver.MCPTool[HourlyForecastToolInput, HourlyForecastToolOutput] {
	return &HourlyForecastTool{client: newHTTPClient(defaultProBaseURL, apiKey, nil, limiter)}
}

var _ mcpserver.MCPTool[HourlyForecastToolInput, HourlyForecastToolOutput] = (*HourlyForecastTool)(nil)

// newHourlyForecastToolWithBaseURL creates a HourlyForecastTool with a custom base URL (for testing).
func newHourlyForecastToolWithBaseURL(apiKey, baseURL string, httpCl *http.Client) *HourlyForecastTool {
	return &HourlyForecastTool{client: newHTTPClient(baseURL, apiKey, httpCl, nil)}
}

// Name returns the tool name as expected by the model.
func (t *HourlyForecastTool) Name() string { return "get_hourly_forecast" }

// Description describes what the tool does.
func (t *HourlyForecastTool) Description() string {
	return "Get the hourly weather forecast for the next 4 days (96 hours) for a given location specified by latitude and longitude. Use the geocode_city tool first to convert a city name to coordinates. Returns up to 96 data points including temperature, humidity, wind, precipitation probability, and weather conditions. Use the optional 'count' parameter to limit the number of hourly timestamps returned."
}

// Call calls the OpenWeatherMap Pro hourly forecast API and returns a typed output struct.
func (t *HourlyForecastTool) Call(ctx context.Context, input HourlyForecastToolInput) (HourlyForecastToolOutput, error) {
	if input.Lat == 0 && input.Lon == 0 {
		return HourlyForecastToolOutput{}, fmt.Errorf("parameters 'lat' and 'lon' must not both be zero")
	}

	response, err := t.fetchHourlyForecast(ctx, input.Lat, input.Lon, input.Count)
	if err != nil {
		return HourlyForecastToolOutput{}, err
	}

	out := HourlyForecastToolOutput{
		City: ForecastCity{
			Name:     response.City.Name,
			Coord:    Coordinates{Lon: response.City.Coord.Lon, Lat: response.City.Coord.Lat},
			Country:  response.City.Country,
			Timezone: response.City.Timezone,
			Sunrise:  response.City.Sunrise,
			Sunset:   response.City.Sunset,
		},
		Count:     response.Cnt,
		Forecasts: make([]ForecastEntry, 0, len(response.List)),
	}

	for _, item := range response.List {
		entry := ForecastEntry{
			DateTime:   time.Unix(item.Dt, 0).UTC().Format(time.RFC3339),
			Temp:       item.Main.Temp,
			FeelsLike:  item.Main.FeelsLike,
			TempMin:    item.Main.TempMin,
			TempMax:    item.Main.TempMax,
			Pressure:   item.Main.Pressure,
			Humidity:   item.Main.Humidity,
			SeaLevel:   item.Main.SeaLevel,
			GrndLevel:  item.Main.GrndLevel,
			Cloudiness: item.Clouds.All,
			WindSpeed:  item.Wind.Speed,
			WindDeg:    item.Wind.Deg,
			WindGust:   item.Wind.Gust,
			Visibility: item.Visibility,
			Pop:        item.Pop,
		}

		entry.Weather = make([]WeatherCondition, 0, len(item.Weather))
		for _, w := range item.Weather {
			entry.Weather = append(entry.Weather, WeatherCondition{
				Main:        w.Main,
				Description: w.Description,
			})
		}

		if item.Rain != nil {
			entry.Precipitation = &item.Rain.OneH
		}
		if item.Snow != nil {
			entry.Snow = &item.Snow.OneH
		}

		out.Forecasts = append(out.Forecasts, entry)
	}

	return out, nil
}

type hourlyForecastPrecipitationResponse struct {
	OneH float64 `json:"1h"`
}

type hourlyRespListItem struct {
	Dt         int64                                `json:"dt"`
	Main       owmMainResponse                      `json:"main"`
	Weather    []owmWeatherResponse                 `json:"weather"`
	Clouds     owmCloudsResponse                    `json:"clouds"`
	Wind       owmWindResponse                      `json:"wind"`
	Rain       *hourlyForecastPrecipitationResponse `json:"rain"`
	Snow       *hourlyForecastPrecipitationResponse `json:"snow"`
	Visibility int                                  `json:"visibility"`
	Pop        float64                              `json:"pop"`
	Sys        owmSysResponse                       `json:"sys"`
	DtTxt      string                               `json:"dt_txt"`
}

type hourlyForecastResponse struct {
	Cod  string               `json:"cod"`
	Cnt  int                  `json:"cnt"`
	List []hourlyRespListItem `json:"list"`
	City owmCityResponse      `json:"city"`
}

func (t *HourlyForecastTool) fetchHourlyForecast(ctx context.Context, lat, lon float64, count int) (*hourlyForecastResponse, error) {
	q := url.Values{
		"lat":   {strconv.FormatFloat(lat, 'f', -1, 64)},
		"lon":   {strconv.FormatFloat(lon, 'f', -1, 64)},
		"units": {"metric"},
	}
	if count > 0 {
		q.Set("cnt", strconv.Itoa(count))
	}
	var data hourlyForecastResponse
	if err := t.client.getJSON(ctx, "/forecast/hourly", q, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
