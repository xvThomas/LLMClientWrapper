package mcpserver

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP roles used to address tool result content blocks. The SDK declares Role
// as a bare string type without exported constants.
const (
	RoleAssistant mcp.Role = "assistant"
	RoleUser      mcp.Role = "user"
)

// audienceResult builds a tool result holding one content block per audience.
func audienceResult(modelOut, clientOut any) (*mcp.CallToolResult, error) {
	modelBlock, err := jsonBlock(modelOut, RoleAssistant)
	if err != nil {
		return nil, err
	}
	clientBlock, err := jsonBlock(clientOut, RoleUser)
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{modelBlock, clientBlock}}, nil
}

// jsonBlock marshals v into a text content block addressed to the given role.
func jsonBlock(v any, role mcp.Role) (*mcp.TextContent, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshalling %s tool content: %w", role, err)
	}
	return &mcp.TextContent{
		Text:        string(raw),
		Annotations: &mcp.Annotations{Audience: []mcp.Role{role}},
	}, nil
}
