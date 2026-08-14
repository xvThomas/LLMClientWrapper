package tools

import (
	"net/url"
	"strconv"
	"time"
)

func buildForecastCity(city owmCityResponse) ForecastCity {
	return ForecastCity{
		Name:     city.Name,
		Coord:    Coordinates{Lon: city.Coord.Lon, Lat: city.Coord.Lat},
		Country:  city.Country,
		Timezone: city.Timezone,
		Sunrise:  city.Sunrise,
		Sunset:   city.Sunset,
	}
}

type forecastEntryInput struct {
	Dt            int64
	Main          owmMainResponse
	Weather       []owmWeatherResponse
	Clouds        owmCloudsResponse
	Wind          owmWindResponse
	Visibility    int
	Pop           float64
	Precipitation *float64
	Snow          *float64
}

func buildForecastEntry(in forecastEntryInput) ForecastEntry {
	entry := ForecastEntry{
		DateTime:      time.Unix(in.Dt, 0).UTC().Format(time.RFC3339),
		Temp:          in.Main.Temp,
		FeelsLike:     in.Main.FeelsLike,
		TempMin:       in.Main.TempMin,
		TempMax:       in.Main.TempMax,
		Pressure:      in.Main.Pressure,
		Humidity:      in.Main.Humidity,
		SeaLevel:      in.Main.SeaLevel,
		GrndLevel:     in.Main.GrndLevel,
		Cloudiness:    in.Clouds.All,
		WindSpeed:     in.Wind.Speed,
		WindDeg:       in.Wind.Deg,
		WindGust:      in.Wind.Gust,
		Visibility:    in.Visibility,
		Pop:           in.Pop,
		Precipitation: in.Precipitation,
		Snow:          in.Snow,
	}
	entry.Weather = make([]WeatherCondition, 0, len(in.Weather))
	for _, w := range in.Weather {
		entry.Weather = append(entry.Weather, WeatherCondition{
			Main:        w.Main,
			Description: w.Description,
		})
	}
	return entry
}

func buildForecastQueryParams(lat, lon float64, count int) url.Values {
	q := url.Values{
		"lat":   {strconv.FormatFloat(lat, 'f', -1, 64)},
		"lon":   {strconv.FormatFloat(lon, 'f', -1, 64)},
		"units": {"metric"},
	}
	if count > 0 {
		q.Set("cnt", strconv.Itoa(count))
	}
	return q
}
