package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPTool is the generic interface for MCP tool implementors.
// TInput is the typed input struct; TOutput is the typed output struct.
type MCPTool[TInput any, TOutput any] interface {
	Name() string
	Description() string
	Call(ctx context.Context, input TInput) (TOutput, error)
}

// ModelViewer is an optional interface for tools whose output carries data the
// model has no use for, such as map geometry. When implemented, the result is
// split into two audience-annotated content blocks: the model receives the
// reduced view, the client receives the full output.
type ModelViewer[TOutput any] interface {
	// ModelView returns a reduced copy of out holding only what the model needs.
	ModelView(out TOutput) TOutput
}

// ToolRegistrar registers a tool on an mcp.Server.
// Use RegisterTool to create one from an MCPTool.
type ToolRegistrar struct {
	Name     string
	Register func(s *mcp.Server)
}

// RegisterTool returns a ToolRegistrar that adds the given MCPTool to an mcp.Server.
func RegisterTool[TInput, TOutput any](tool MCPTool[TInput, TOutput]) ToolRegistrar {
	return ToolRegistrar{
		Name: tool.Name(),
		Register: func(s *mcp.Server) {
			mcp.AddTool(s, &mcp.Tool{
				Name:        tool.Name(),
				Description: tool.Description(),
			}, func(ctx context.Context, _ *mcp.CallToolRequest, args TInput) (*mcp.CallToolResult, TOutput, error) {
				out, err := tool.Call(ctx, args)
				if err != nil {
					return nil, out, err
				}
				viewer, ok := tool.(ModelViewer[TOutput])
				if !ok {
					return nil, out, nil
				}
				// The typed value drives StructuredContent, so it must be the
				// reduced view: the full output travels in the client block only.
				modelOut := viewer.ModelView(out)
				res, err := audienceResult(modelOut, out)
				if err != nil {
					return nil, out, err
				}
				return res, modelOut, nil
			})
		},
	}
}
