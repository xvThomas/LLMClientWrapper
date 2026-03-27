package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration loaded from environment variables.
type Config struct {
	AnthropicAPIKey      string
	OpenAIAPIKey         string
	MistralAPIKey        string
	OpenWeatherMapAPIKey string
	ToolsMaxConcurrent   int // Max concurrent tool executions (default: 4)
}

// Load reads the .env file (if present) then reads environment variables.
func Load(envFile string) (*Config, error) {
	// godotenv.Load does not override already-set env vars.
	_ = godotenv.Load(envFile)

	cfg := &Config{
		AnthropicAPIKey:      os.Getenv("ANTHROPIC_API_KEY"),
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		MistralAPIKey:        os.Getenv("MISTRAL_API_KEY"),
		OpenWeatherMapAPIKey: os.Getenv("OPENWEATHERMAP_API_KEY"),
		ToolsMaxConcurrent:   parseToolsMaxConcurrent(os.Getenv("TOOLS_MAX_CONCURRENT")),
	}

	return cfg, nil
}

// RequireAnthropicKey returns the Anthropic API key or an error if missing.
func (c *Config) RequireAnthropicKey() (string, error) {
	return requireKey(c.AnthropicAPIKey, "ANTHROPIC_API_KEY")
}

// RequireOpenAIKey returns the OpenAI API key or an error if missing.
func (c *Config) RequireOpenAIKey() (string, error) {
	return requireKey(c.OpenAIAPIKey, "OPENAI_API_KEY")
}

// RequireMistralKey returns the Mistral API key or an error if missing.
func (c *Config) RequireMistralKey() (string, error) {
	return requireKey(c.MistralAPIKey, "MISTRAL_API_KEY")
}

// RequireOpenWeatherMapKey returns the OpenWeatherMap API key or an error if missing.
func (c *Config) RequireOpenWeatherMapKey() (string, error) {
	return requireKey(c.OpenWeatherMapAPIKey, "OPENWEATHERMAP_API_KEY")
}

func requireKey(value, name string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("missing required environment variable %q", name)
	}
	return value, nil
}

// parseToolsMaxConcurrent parses TOOLS_MAX_CONCURRENT with fallback to 4.
func parseToolsMaxConcurrent(value string) int {
	if value == "" {
		return 4 // Default value
	}
	if n, err := strconv.Atoi(value); err == nil && n > 0 {
		return n
	}
	return 4 // Fallback on invalid input
}
