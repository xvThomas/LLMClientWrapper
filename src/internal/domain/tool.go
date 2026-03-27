package domain

import (
	"context"
	"encoding/json"
	"fmt"
)

// TypedTool is the generic interface for tool implementors.
// TInput is the typed input struct; TOutput is the typed output struct.
type TypedTool[TInput any, TOutput any] interface {
	Name() string
	Description() string
	Call(ctx context.Context, input TInput) (TOutput, error)
}

// Tool is the type-erased interface used internally by the conversation
// engine and LLM converters. It works with raw maps and string output.
type Tool interface {
	Name() string
	Description() string
	// Parameters returns a JSON-Schema-compatible description of the input.
	Parameters() map[string]any
	// Execute runs the tool with the given input and returns a string result.
	Execute(ctx context.Context, input map[string]any) (string, error)
}

// toolAdapter bridges a TypedTool into the type-erased Tool interface.
type toolAdapter[TInput any, TOutput any] struct {
	tool   TypedTool[TInput, TOutput]
	schema func() map[string]any
}

// Adapt wraps a TypedTool[TInput, TOutput] into a Tool.
// schema is a function returning the JSON Schema for TInput (e.g. t.Parameters).
func Adapt[TInput any, TOutput any](tool TypedTool[TInput, TOutput], schema func() map[string]any) Tool {
	return &toolAdapter[TInput, TOutput]{tool: tool, schema: schema}
}

func (a *toolAdapter[TInput, TOutput]) Name() string               { return a.tool.Name() }
func (a *toolAdapter[TInput, TOutput]) Description() string        { return a.tool.Description() }
func (a *toolAdapter[TInput, TOutput]) Parameters() map[string]any { return a.schema() }

func (a *toolAdapter[TInput, TOutput]) Execute(ctx context.Context, input map[string]any) (string, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("marshalling tool input: %w", err)
	}
	var typed TInput
	if err := json.Unmarshal(raw, &typed); err != nil {
		return "", fmt.Errorf("unmarshalling tool input into %T: %w", typed, err)
	}

	output, err := a.tool.Call(ctx, typed)
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("marshalling tool output: %w", err)
	}
	return string(out), nil
}
