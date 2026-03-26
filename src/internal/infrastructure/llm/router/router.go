package router

import (
	"fmt"

	"llmclientwrapper/src/internal/domain"
	"llmclientwrapper/src/internal/infrastructure/llm/anthropic"
	"llmclientwrapper/src/internal/infrastructure/config"
	openaiinfra "llmclientwrapper/src/internal/infrastructure/llm/openai"
)

// Router builds LlmClient instances for model aliases from configuration.
type Router struct {
	cfg *config.Config
}

// New creates a Router backed by the given configuration.
func New(cfg *config.Config) *Router {
	return &Router{cfg: cfg}
}

// Get returns an LlmClient for the given model alias, building it from configuration.
func (r *Router) Get(model domain.Model) (domain.LlmClient, error) {
	d, err := domain.Lookup(model)
	if err != nil {
		return nil, err
	}

	switch d.Provider {
	case domain.ProviderAnthropic:
		key, err := r.cfg.RequireAnthropicKey()
		if err != nil {
			return nil, err
		}
		return anthropic.NewClient(key, d.APIModelID), nil

	case domain.ProviderOpenAI:
		key, err := r.cfg.RequireOpenAIKey()
		if err != nil {
			return nil, err
		}
		return openaiinfra.NewClient(key, d.APIModelID, ""), nil

	case domain.ProviderMistral:
		key, err := r.cfg.RequireMistralKey()
		if err != nil {
			return nil, err
		}
		return openaiinfra.NewClient(key, d.APIModelID, "https://api.mistral.ai/v1"), nil

	default:
		return nil, fmt.Errorf("unsupported provider %q", d.Provider)
	}
}
