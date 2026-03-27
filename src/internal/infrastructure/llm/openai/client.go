package openai

import (
	"context"
	"fmt"
	"llmclientwrapper/src/internal/domain"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Client implements domain.LlmClient using an OpenAI-compatible API.
// It works with OpenAI, Mistral, and any other provider that exposes the
// OpenAI chat completions API.
type Client struct {
	sdk     *openai.Client
	modelID string
}

// NewClient creates an OpenAI-compatible Client.
// Pass a custom baseURL to target Mistral / Devstral / local endpoints.
func NewClient(apiKey, modelID, baseURL string) *Client {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	sdk := openai.NewClient(opts...)
	return &Client{sdk: &sdk, modelID: modelID}
}

var _ domain.LlmClient = (*Client)(nil) // ensure Client implements domain.LlmClient

// Complete sends the conversation to the OpenAI-compatible API and returns the response with token usage.
func (c *Client) Complete(ctx context.Context, systemPrompt string, messages []domain.Message, tools []domain.Tool) (*domain.Message, domain.Usage, error) {
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(c.modelID),
		Messages: toSDKMessages(systemPrompt, messages),
	}

	if len(tools) > 0 {
		params.Tools = toSDKTools(tools)
	}

	resp, err := c.sdk.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, domain.Usage{}, fmt.Errorf("openai completion: %w", err)
	}

	msg, usage := fromSDKResponse(resp)
	return msg, usage, nil
}
