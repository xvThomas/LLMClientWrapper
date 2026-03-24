package anthropic

import (
	"context"
	"fmt"
	"llmclientwrapper/src/internal"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Client implements internal.LlmClient using the Anthropic API.
type Client struct {
	sdk     *anthropic.Client
	modelID string
}

// NewClient creates an Anthropic Client.
func NewClient(apiKey, modelID string) *Client {
	sdk := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Client{sdk: &sdk, modelID: modelID}
}

// Complete sends the conversation to Anthropic and returns the assistant response.
func (c *Client) Complete(ctx context.Context, systemPrompt string, messages []internal.Message, tools []internal.Tool) (*internal.Message, error) {
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
		return nil, fmt.Errorf("anthropic completion: %w", err)
	}

	return fromSDKResponse(resp), nil
}
