package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"llmclientwrapper/src/internal"
	"llmclientwrapper/src/internal/infrastructure/anthropic"
	"llmclientwrapper/src/internal/infrastructure/config"
	"llmclientwrapper/src/internal/infrastructure/memory"
	openaiinfra "llmclientwrapper/src/internal/infrastructure/openai"
	"llmclientwrapper/src/internal/infrastructure/prompt"
	"llmclientwrapper/src/internal/infrastructure/weather"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		modelFlag      string
		questionFlag   string
		systemFlag     string
		systemFileFlag string
	)

	cmd := &cobra.Command{
		Use:   "llmclientwrapper",
		Short: "Ask a question to an LLM model",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), modelFlag, questionFlag, systemFlag, systemFileFlag)
		},
	}

	cmd.Flags().StringVar(&modelFlag, "model", "", "Model alias to use (e.g. sonnet-4.6, devstral)")
	cmd.Flags().StringVar(&questionFlag, "question", "", "Question to ask")
	cmd.Flags().StringVar(&systemFlag, "system", "", "Inline system prompt (overrides --system-file)")
	cmd.Flags().StringVar(&systemFileFlag, "system-file", defaultSystemPromptPath(), "Path to a Markdown system prompt file")

	_ = cmd.MarkFlagRequired("model")
	_ = cmd.MarkFlagRequired("question")

	return cmd
}

func run(ctx context.Context, modelAlias, question, systemInline, systemFile string) error {
	cfg, err := config.Load(".env")
	if err != nil {
		return err
	}

	descriptor, err := internal.Lookup(internal.Model(modelAlias))
	if err != nil {
		return err
	}

	client, err := buildClient(cfg, descriptor)
	if err != nil {
		return err
	}

	pp := buildPromptProvider(systemInline, systemFile)
	weatherKey, _ := cfg.RequireOpenWeatherMapKey()
	tools := []internal.Tool{weather.NewTool(weatherKey)}

	store := memory.NewStore()
	manager := internal.NewConversationManager(client, store, pp, tools)

	answer, err := manager.Chat(ctx, question)
	if err != nil {
		return err
	}

	fmt.Println(answer)
	return nil
}

func buildClient(cfg *config.Config, d internal.ModelDescriptor) (internal.LlmClient, error) {
	switch d.Provider {
	case internal.ProviderAnthropic:
		key, err := cfg.RequireAnthropicKey()
		if err != nil {
			return nil, err
		}
		return anthropic.NewClient(key, d.APIModelID), nil

	case internal.ProviderOpenAI:
		key, err := cfg.RequireOpenAIKey()
		if err != nil {
			return nil, err
		}
		return openaiinfra.NewClient(key, d.APIModelID, ""), nil

	case internal.ProviderMistral:
		key, err := cfg.RequireMistralKey()
		if err != nil {
			return nil, err
		}
		return openaiinfra.NewClient(key, d.APIModelID, "https://api.mistral.ai/v1"), nil

	default:
		return nil, fmt.Errorf("unsupported provider %q", d.Provider)
	}
}

func buildPromptProvider(systemInline, systemFile string) internal.PromptProvider {
	if systemInline != "" {
		return prompt.NewStaticProvider(systemInline)
	}
	return prompt.NewFileProvider(systemFile)
}

func defaultSystemPromptPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "system_prompt.md"
	}
	return filepath.Join(filepath.Dir(exe), "system_prompt.md")
}
