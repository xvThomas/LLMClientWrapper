package tools

import (
	"llmclientwrapper/src/internal/domain"
	"llmclientwrapper/src/internal/infrastructure/config"
	"llmclientwrapper/src/internal/infrastructure/tools/weather"
)

// Tools aggregates all available domain.Tool implementations.
type Tools struct {
	cfg *config.Config
}

// New creates a Tools aggregator backed by the given configuration.
func New(cfg *config.Config) *Tools {
	return &Tools{cfg: cfg}
}

// All returns the list of all registered tools as type-erased Tool handlers.
func (t *Tools) All() []domain.Tool {
	w := weather.NewOpenWeatherMapTool(t.cfg.OpenWeatherMapAPIKey)
	return []domain.Tool{
		domain.Adapt(w, w.Parameters),
	}
}
