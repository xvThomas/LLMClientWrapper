package anthropic

import (
	"encoding/json"
	"llmclientwrapper/src/internal"

	"github.com/anthropics/anthropic-sdk-go"
)

// toSDKMessages converts domain messages to Anthropic SDK message params.
func toSDKMessages(messages []internal.Message) []anthropic.MessageParam {
	params := make([]anthropic.MessageParam, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case internal.RoleUser:
			params = append(params, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		case internal.RoleAssistant:
			params = append(params, toAssistantParam(msg))
		case internal.RoleTool:
			params = append(params, toToolResultParam(msg))
		}
	}
	return params
}

func toAssistantParam(msg internal.Message) anthropic.MessageParam {
	blocks := make([]anthropic.ContentBlockParamUnion, 0)
	if msg.Content != "" {
		blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
	}
	for _, tc := range msg.ToolCalls {
		// NewToolUseBlock signature: (id, input, name)
		blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, tc.Input, tc.Name))
	}
	return anthropic.NewAssistantMessage(blocks...)
}

func toToolResultParam(msg internal.Message) anthropic.MessageParam {
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.ToolResults))
	for _, tr := range msg.ToolResults {
		blocks = append(blocks, anthropic.NewToolResultBlock(tr.ToolCallID, tr.Content, false))
	}
	return anthropic.NewUserMessage(blocks...)
}

// toSDKTools converts domain tools to Anthropic SDK tool definitions.
func toSDKTools(tools []internal.Tool) []anthropic.ToolUnionParam {
	sdkTools := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		params := t.Parameters()
		props, _ := params["properties"]
		var required []string
		if r, ok := params["required"]; ok {
			if sl, ok := r.([]string); ok {
				required = sl
			}
		}
		sdkTools = append(sdkTools, anthropic.ToolUnionParamOfTool(
			anthropic.ToolInputSchemaParam{
				Properties: props,
				Required:   required,
			},
			t.Name(),
		))
		// set description via direct struct field — ToolUnionParamOfTool returns a
		// ToolUnionParam whose OfTool pointer we can mutate.
		sdkTools[len(sdkTools)-1].OfTool.Description = anthropic.String(t.Description())
	}
	return sdkTools
}

// fromSDKResponse converts an Anthropic SDK response to a domain Message.
func fromSDKResponse(resp *anthropic.Message) *internal.Message {
	msg := &internal.Message{Role: internal.RoleAssistant}
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			msg.Content += block.Text
		case "tool_use":
			var input map[string]any
			_ = json.Unmarshal(block.Input, &input)
			msg.ToolCalls = append(msg.ToolCalls, internal.ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: input,
			})
		}
	}
	return msg
}

// toSystemPrompt wraps the system prompt string with ephemeral cache control.
func toSystemPrompt(text string) anthropic.TextBlockParam {
	return anthropic.TextBlockParam{
		Text:         text,
		CacheControl: anthropic.NewCacheControlEphemeralParam(),
	}
}
