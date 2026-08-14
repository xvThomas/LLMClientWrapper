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

func buildForecastEntry(dt int64, main owmMainResponse, weather []owmWeatherResponse, clouds owmCloudsResponse, wind owmWindResponse, visibility int, pop float64, precipitation, snow *float64) ForecastEntry {
	entry := ForecastEntry{
		DateTime:      time.Unix(dt, 0).UTC().Format(time.RFC3339),
		Temp:          main.Temp,
		FeelsLike:     main.FeelsLike,
		TempMin:       main.TempMin,
		TempMax:       main.TempMax,
		Pressure:      main.Pressure,
		Humidity:      main.Humidity,
		SeaLevel:      main.SeaLevel,
		GrndLevel:     main.GrndLevel,
		Cloudiness:    clouds.All,
		WindSpeed:     wind.Speed,
		WindDeg:       wind.Deg,
		WindGust:      wind.Gust,
		Visibility:    visibility,
		Pop:           pop,
		Precipitation: precipitation,
		Snow:          snow,
	}
	entry.Weather = make([]WeatherCondition, 0, len(weather))
	for _, w := range weather {
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
