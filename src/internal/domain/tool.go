package domain

import "context"

// Tool represents a callable function that the model can invoke.
type Tool interface {
	Name() string
	Description() string
	// Parameters returns a JSON-Schema-compatible description of the input.
	Parameters() map[string]any
	// Execute runs the tool with the given input and returns a string result.
	Execute(ctx context.Context, input map[string]any) (string, error)
}
