package main

import (
	"os"

	"github.com/pixime-net/talkbackend/mcp-owm/internal/config"
	"github.com/pixime-net/talkbackend/mcp-owm/internal/prompts"
	"github.com/pixime-net/talkbackend/mcp-owm/internal/ratelimit"
	"github.com/pixime-net/talkbackend/mcp-owm/internal/tools"
	"github.com/pixime-net/talkbackend/talk-libs/logger"
	"github.com/pixime-net/talkbackend/talk-libs/mcpserver"
	"github.com/pixime-net/talkbackend/talk-libs/version"
)

func main() {
	log := logger.GetLogger()

	env, err := config.LoadServerEnv(".env")
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	app := buildApp(env)
	app.Run()
}

// buildApp creates the MCP server application from the given configuration.
func buildApp(env *config.ServerEnv) *mcpserver.App {
	limiter := ratelimit.NewLimiter(env.RateLimitPerMinute)

	weatherTool := tools.NewCurrentWeatherTool(env.OpenWeatherMapAPIKey, limiter)
	geocodingTool := tools.NewGeocodingTool(env.OpenWeatherMapAPIKey, limiter)
	reverseGeocodingTool := tools.NewReverseGeocodingTool(env.OpenWeatherMapAPIKey, limiter)
	airPollutionTool := tools.NewAirPollutionTool(env.OpenWeatherMapAPIKey, limiter)
	airPollutionForecastTool := tools.NewAirPollutionForecastTool(env.OpenWeatherMapAPIKey, limiter)

	opts := []mcpserver.Option{
		mcpserver.WithTools(mcpserver.RegisterTool(weatherTool)),
		mcpserver.WithTools(mcpserver.RegisterTool(geocodingTool)),
		mcpserver.WithTools(mcpserver.RegisterTool(reverseGeocodingTool)),
		mcpserver.WithTools(mcpserver.RegisterTool(airPollutionTool)),
		mcpserver.WithTools(mcpserver.RegisterTool(airPollutionForecastTool)),
		mcpserver.WithPrompts(
			mcpserver.RegisterPrompt(prompts.CurrentWeather),
			mcpserver.RegisterPrompt(prompts.CurrentAir),
			mcpserver.RegisterPrompt(prompts.ForecastAir),
		),
		mcpserver.WithBaseEnvHTTPSecurity(env.BaseEnv),
	}

	if env.FreePlan {
		forecastTool := tools.NewForecast5Days3HoursWeatherTool(env.OpenWeatherMapAPIKey, limiter)
		opts = append(opts,
			mcpserver.WithTools(mcpserver.RegisterTool(forecastTool)),
			mcpserver.WithPrompts(mcpserver.RegisterPrompt(prompts.ForecastWeather)),
		)
	} else {
		hourlyForecastTool := tools.NewHourlyForecastTool(env.OpenWeatherMapAPIKey, limiter)
		dailyForecastTool := tools.NewDailyForecastTool(env.OpenWeatherMapAPIKey, limiter)
		opts = append(opts,
			mcpserver.WithTools(mcpserver.RegisterTool(hourlyForecastTool)),
			mcpserver.WithTools(mcpserver.RegisterTool(dailyForecastTool)),
		)
	}
	opts = append(opts, mcpserver.WithBaseEnvAuth(env.BaseEnv)...)
	return mcpserver.NewApp("owm-mcp", version.Version, opts...)
}
