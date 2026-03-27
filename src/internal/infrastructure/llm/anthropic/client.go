package anthropic

import (
	"context"
	"fmt"
	"llmclientwrapper/src/internal/domain"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Client implements domain.LlmClient using the Anthropic API.
type Client struct {
	sdk     *anthropic.Client
	modelID string
}

var _ domain.LlmClient = (*Client)(nil) // ensure Client implements domain.LlmClient

// NewClient creates an Anthropic Client.
func NewClient(apiKey, modelID string) *Client {
	sdk := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Client{sdk: &sdk, modelID: modelID}
}

// Complete sends the conversation to Anthropic and returns the assistant response with token usage.
func (c *Client) Complete(ctx context.Context, systemPrompt string, messages []domain.Message, tools []domain.Tool) (*domain.Message, domain.Usage, error) {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.modelID),
		MaxTokens: 4096,
		Messages:  toSDKMessages(messages),
	}

	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{toSystemPrompt(systemPrompt)}
	}

	if len(tools) > 0 {
		params.Tools = toSDKTools(tools)
	}

	resp, err := c.sdk.Messages.New(ctx, params)
	if err != nil {
		return nil, domain.Usage{}, fmt.Errorf("anthropic completion: %w", err)
	}

	msg, usage := fromSDKResponse(resp)
	return msg, usage, nil
}
