package internal

import "fmt"

// Provider identifies the LLM provider backend.
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
	ProviderMistral   Provider = "mistral"
)

// Model is a friendly alias for a model (e.g. "sonnet-4.6").
type Model string

// ModelDescriptor maps a friendly Model alias to provider-specific details.
type ModelDescriptor struct {
	Provider   Provider
	APIModelID string
}

// registry holds all supported model aliases.
var registry = map[Model]ModelDescriptor{
	"haiku-4.5":  {Provider: ProviderAnthropic, APIModelID: "claude-haiku-4-5"},
	"sonnet-4.6": {Provider: ProviderAnthropic, APIModelID: "claude-sonnet-4-5"},
	"gpt-5.4":    {Provider: ProviderOpenAI, APIModelID: "gpt-4o"},
	"devstral":   {Provider: ProviderMistral, APIModelID: "devstral-small-2505"},
}

// Lookup returns the ModelDescriptor for a given alias.
func Lookup(m Model) (ModelDescriptor, error) {
	d, ok := registry[m]
	if !ok {
		return ModelDescriptor{}, fmt.Errorf("unknown model %q", m)
	}
	return d, nil
}

// SupportedModels returns all registered model aliases.
func SupportedModels() []Model {
	models := make([]Model, 0, len(registry))
	for m := range registry {
		models = append(models, m)
	}
	return models
}
