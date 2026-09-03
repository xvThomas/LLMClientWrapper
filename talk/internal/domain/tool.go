package domain

import "context"

// ToolOutput is the result of a tool execution.
type ToolOutput struct {
	// Model is the payload sent to the LLM.
	Model map[string]any
	// Client is the full payload for non-model consumers such as the map UI.
	// Nil when the tool exposes a single payload for every audience.
	Client map[string]any
}

// Tool is the type-erased interface used internally by the conversation
// engine and LLM converters. It works with raw maps and string output.
type Tool interface {
	Name() string
	Description() string
	InputSchema() (map[string]any, error)  // JSON Schema for the input parameters
	OutputSchema() (map[string]any, error) // JSON Schema for the output
	// Execute runs the tool with the given input.
	Execute(ctx context.Context, input map[string]any) (ToolOutput, error)
}
