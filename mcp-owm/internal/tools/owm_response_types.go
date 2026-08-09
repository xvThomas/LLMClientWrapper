package tools

// owmWeatherResponse is the raw OWM API weather condition item, common to all forecast responses.
type owmWeatherResponse struct {
	ID          int    `json:"id"`
	Main        string `json:"main"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// owmCoordResponse is the raw OWM API coordinate block, common to all forecast city responses.
type owmCoordResponse struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// owmCityResponse is the raw OWM API city block, common to all forecast responses.
type owmCityResponse struct {
	ID       int              `json:"id"`
	Name     string           `json:"name"`
	Coord    owmCoordResponse `json:"coord"`
	Country  string           `json:"country"`
	Timezone int              `json:"timezone"`
	Sunrise  int64            `json:"sunrise"`
	Sunset   int64            `json:"sunset"`
}

// owmMainResponse is the raw OWM API main weather block, shared by 5-day and hourly forecast responses.
type owmMainResponse struct {
	Temp      float64 `json:"temp"`
	FeelsLike float64 `json:"feels_like"`
	TempMin   float64 `json:"temp_min"`
	TempMax   float64 `json:"temp_max"`
	Pressure  int     `json:"pressure"`
	Humidity  int     `json:"humidity"`
	SeaLevel  int     `json:"sea_level"`
	GrndLevel int     `json:"grnd_level"`
}

// owmCloudsResponse is the raw OWM API clouds block, shared by 5-day and hourly forecast responses.
type owmCloudsResponse struct {
	All int `json:"all"`
}

// owmWindResponse is the raw OWM API wind block, shared by 5-day and hourly forecast responses.
type owmWindResponse struct {
	Speed float64 `json:"speed"`
	Deg   int     `json:"deg"`
	Gust  float64 `json:"gust"`
}

// owmSysResponse is the raw OWM API sys block in forecast list items, shared by 5-day and hourly.
type owmSysResponse struct {
	Pod string `json:"pod"`
}
