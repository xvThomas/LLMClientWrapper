package tools

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pixime-net/talk-libs/mcpserver"
	"golang.org/x/time/rate"
)

// ForecastToolInput is the typed input for Forecast5Days3HoursWeatherTool.
type ForecastToolInput struct {
	Lat   float64 `json:"lat" description:"Latitude of the location"`
	Lon   float64 `json:"lon" description:"Longitude of the location"`
	Count int     `json:"count,omitempty" description:"Optional number of 3-hour timestamps to return (1-40). If omitted, returns all 40 timestamps (5 days)."`
}

// ForecastEntry represents a single 3-hour forecast data point.
type ForecastEntry struct {
	DateTime      string             `json:"dt" description:"Forecast time in ISO 8601 format (e.g. 2026-03-30T15:00:00Z)"`
	Temp          float64            `json:"temp" description:"Temperature in Celsius"`
	FeelsLike     float64            `json:"feels_like" description:"Perceived temperature in Celsius"`
	TempMin       float64            `json:"temp_min" description:"Minimum temperature in Celsius"`
	TempMax       float64            `json:"temp_max" description:"Maximum temperature in Celsius"`
	Pressure      int                `json:"pressure" description:"Atmospheric pressure in hPa"`
	Humidity      int                `json:"humidity" description:"Humidity percentage"`
	SeaLevel      int                `json:"sea_level,omitempty" description:"Sea level atmospheric pressure in hPa"`
	GrndLevel     int                `json:"grnd_level,omitempty" description:"Ground level atmospheric pressure in hPa"`
	Weather       []WeatherCondition `json:"weather" description:"Weather conditions"`
	Cloudiness    int                `json:"cloudiness" description:"Cloudiness percentage"`
	WindSpeed     float64            `json:"wind_speed" description:"Wind speed in meter/sec"`
	WindDeg       int                `json:"wind_deg" description:"Wind direction in degrees"`
	WindGust      float64            `json:"wind_gust,omitempty" description:"Wind gust in meter/sec"`
	Visibility    int                `json:"visibility" description:"Visibility in meters"`
	Pop           float64            `json:"pop" description:"Probability of precipitation (0 to 1)"`
	Precipitation *float64           `json:"precipitation,omitempty" description:"Rain volume for the last 3 hours in mm"`
	Snow          *float64           `json:"snow,omitempty" description:"Snow volume for the last 3 hours in mm"`
}

// ForecastCity contains city information from the forecast response.
type ForecastCity struct {
	Name     string      `json:"name" description:"City name"`
	Coord    Coordinates `json:"coord" description:"Geographic coordinates of the location"`
	Country  string      `json:"country" description:"Country code"`
	Timezone int         `json:"timezone" description:"Shift in seconds from UTC"`
	Sunrise  int64       `json:"sunrise" description:"Sunrise time, unix, UTC"`
	Sunset   int64       `json:"sunset" description:"Sunset time, unix, UTC"`
}

// ForecastToolOutput is the typed output for Forecast5Days3HoursWeatherTool.
type ForecastToolOutput struct {
	City      ForecastCity    `json:"city" description:"City information"`
	Count     int             `json:"cnt" description:"Number of forecast entries"`
	Forecasts []ForecastEntry `json:"forecasts" description:"3-hour forecast entries (up to 40 timestamps for 5 days)"`
}

// Forecast5Days3HoursWeatherTool implements mcpserver.MCPTool for fetching 5-day/3-hour forecast via OpenWeatherMap.
type Forecast5Days3HoursWeatherTool struct {
	client *httpClient
}

// NewForecast5Days3HoursWeatherTool creates a Forecast5Days3HoursWeatherTool with the given API key.
func NewForecast5Days3HoursWeatherTool(apiKey string, limiter *rate.Limiter) mcpserver.MCPTool[ForecastToolInput, ForecastToolOutput] {
	return &Forecast5Days3HoursWeatherTool{client: newHTTPClient(defaultBaseURL, apiKey, nil, limiter)}
}

var _ mcpserver.MCPTool[ForecastToolInput, ForecastToolOutput] = (*Forecast5Days3HoursWeatherTool)(nil)

// newForecast5Days3HoursWeatherToolWithBaseURL creates a Forecast5Days3HoursWeatherTool with a custom base URL (for testing).
func newForecast5Days3HoursWeatherToolWithBaseURL(apiKey, baseURL string, httpCl *http.Client) *Forecast5Days3HoursWeatherTool {
	return &Forecast5Days3HoursWeatherTool{client: newHTTPClient(baseURL, apiKey, httpCl, nil)}
}

// Name returns the tool name as expected by the model.
func (t *Forecast5Days3HoursWeatherTool) Name() string { return "get_weather_forecast" }

// Description describes what the tool does.
func (t *Forecast5Days3HoursWeatherTool) Description() string {
	return "Get the weather forecast for the next 5 days with 3-hour intervals for a given location specified by latitude and longitude. Use the geocode_city tool first to convert a city name to coordinates. Returns up to 40 data points including temperature, humidity, wind, precipitation probability, and weather conditions. Use the optional 'count' parameter to limit the number of 3-hour timestamps returned."
}

// Call calls the OpenWeatherMap 5-day/3-hour forecast API and returns a typed output struct.
func (t *Forecast5Days3HoursWeatherTool) Call(ctx context.Context, input ForecastToolInput) (ForecastToolOutput, error) {
	if input.Lat == 0 && input.Lon == 0 {
		return ForecastToolOutput{}, fmt.Errorf("parameters 'lat' and 'lon' must not both be zero")
	}

	response, err := t.fetchForecast(ctx, input.Lat, input.Lon, input.Count)
	if err != nil {
		return ForecastToolOutput{}, err
	}

	out := ForecastToolOutput{
		City:      buildForecastCity(response.City),
		Count:     response.Cnt,
		Forecasts: make([]ForecastEntry, 0, len(response.List)),
	}

	for _, item := range response.List {
		var precipitation, snow *float64
		if item.Rain != nil {
			v := item.Rain.ThreeH
			precipitation = &v
		}
		if item.Snow != nil {
			v := item.Snow.ThreeH
			snow = &v
		}
		out.Forecasts = append(out.Forecasts, buildForecastEntry(forecastEntryInput{
			Dt: item.Dt, Main: item.Main, Weather: item.Weather, Clouds: item.Clouds,
			Wind: item.Wind, Visibility: item.Visibility, Pop: item.Pop,
			Precipitation: precipitation, Snow: snow,
		}))
	}

	return out, nil
}

type forecastPrecipitationResponse struct {
	ThreeH float64 `json:"3h"`
}

type forecastItemResponse struct {
	Dt         int64                          `json:"dt"`
	Main       owmMainResponse                `json:"main"`
	Weather    []owmWeatherResponse           `json:"weather"`
	Clouds     owmCloudsResponse              `json:"clouds"`
	Wind       owmWindResponse                `json:"wind"`
	Rain       *forecastPrecipitationResponse `json:"rain"`
	Snow       *forecastPrecipitationResponse `json:"snow"`
	Visibility int                            `json:"visibility"`
	Pop        float64                        `json:"pop"`
	Sys        owmSysResponse                 `json:"sys"`
	DtTxt      string                         `json:"dt_txt"`
}

type forecastResponse struct {
	Cod  string                 `json:"cod"`
	Cnt  int                    `json:"cnt"`
	List []forecastItemResponse `json:"list"`
	City owmCityResponse        `json:"city"`
}

func (t *Forecast5Days3HoursWeatherTool) fetchForecast(ctx context.Context, lat, lon float64, count int) (*forecastResponse, error) {
	var data forecastResponse
	if err := t.client.getJSON(ctx, "/forecast", buildForecastQueryParams(lat, lon, count), &data); err != nil {
		return nil, err
	}
	return &data, nil
}
