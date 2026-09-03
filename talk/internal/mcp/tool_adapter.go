package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/pixime-net/talk/internal/domain"
)

// mcpToolAdapter adapts an MCP remote tool into the domain.Tool interface.
type mcpToolAdapter struct {
	manager    *Manager
	serverID   string
	serverName string
	tool       mcp.Tool
}

// Name namespaces the remote tool with its server name so that identically
// named tools from different MCP servers stay distinguishable for the LLM.
func (a *mcpToolAdapter) Name() string {
	return a.serverName + ToolNameSeparator + a.tool.Name
}

func (a *mcpToolAdapter) Description() string {
	return a.tool.Description
}

func (a *mcpToolAdapter) InputSchema() (map[string]any, error) {
	if a.tool.InputSchema == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}, nil
	}
	// InputSchema is typed as `any` in the MCP SDK. After JSON decoding from
	// ListTools it is typically already a map[string]any — try a direct assertion
	// to avoid a marshal/unmarshal round-trip.
	if m, ok := a.tool.InputSchema.(map[string]any); ok {
		return m, nil
	}
	// Fallback for non-map types (e.g. a typed struct passed in tests).
	b, err := json.Marshal(a.tool.InputSchema)
	if err != nil {
		return nil, fmt.Errorf("marshalling input schema: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshalling input schema: %w", err)
	}
	return m, nil
}

func (a *mcpToolAdapter) OutputSchema() (map[string]any, error) {
	// MCP tools do not expose an output schema; return a generic one.
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
	}, nil
}

func (a *mcpToolAdapter) Execute(ctx context.Context, input map[string]any) (domain.ToolOutput, error) {
	session, err := a.manager.EnsureConnected(ctx, a.serverID)
	if err != nil {
		return domain.ToolOutput{}, fmt.Errorf("MCP tool execution is unavailable for server %q: %w", a.serverName, err)
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      a.tool.Name,
		Arguments: input,
	})
	if err != nil {
		if !isReconnectableCallError(err) {
			return domain.ToolOutput{}, fmt.Errorf("calling tool %q on server %q: %w", a.tool.Name, a.serverName, err)
		}

		session, reconnectErr := a.manager.Reconnect(ctx, a.serverID)
		if reconnectErr != nil {
			return domain.ToolOutput{}, fmt.Errorf("MCP tool execution is unavailable for server %q: %w", a.serverName, reconnectErr)
		}

		result, err = session.CallTool(ctx, &mcp.CallToolParams{
			Name:      a.tool.Name,
			Arguments: input,
		})
		if err != nil {
			return domain.ToolOutput{}, fmt.Errorf("%w: calling tool %q on server %q after reconnect: %v", ErrSessionUnavailable, a.tool.Name, a.serverName, err)
		}
	}
	if result.IsError {
		return domain.ToolOutput{}, fmt.Errorf("tool %q returned error: %s", a.tool.Name, extractTextContent(result.Content))
	}

	modelText := textForAudience(result.Content, roleAssistant)
	out := domain.ToolOutput{Model: decodeToolText(modelText)}
	if clientText := textForAudience(result.Content, roleUser); clientText != modelText {
		out.Client = decodeToolText(clientText)
	}
	return out, nil
}

// MCP roles used to address tool result content blocks. The SDK declares Role
// as a bare string type without exported constants.
const (
	roleAssistant mcp.Role = "assistant"
	roleUser      mcp.Role = "user"
)

// decodeToolText turns a tool text payload into a map, falling back to a
// single "content" entry when the payload is not a JSON object.
func decodeToolText(text string) map[string]any {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err == nil {
		return parsed
	}
	return map[string]any{"content": text}
}

// textForAudience concatenates the text blocks addressed to the given role.
// Blocks without an audience annotation are addressed to every audience.
func textForAudience(content []mcp.Content, role mcp.Role) string {
	var text string
	for _, c := range content {
		tc, ok := c.(*mcp.TextContent)
		if !ok || !addressedTo(tc.Annotations, role) {
			continue
		}
		if text != "" {
			text += "\n"
		}
		text += tc.Text
	}
	return text
}

func addressedTo(annotations *mcp.Annotations, role mcp.Role) bool {
	if annotations == nil || len(annotations.Audience) == 0 {
		return true
	}
	for _, r := range annotations.Audience {
		if r == role {
			return true
		}
	}
	return false
}

// extractTextContent concatenates every text block of an MCP tool result.
func extractTextContent(content []mcp.Content) string {
	var text string
	for _, c := range content {
		if tc, ok := c.(*mcp.TextContent); ok {
			if text != "" {
				text += "\n"
			}
			text += tc.Text
		}
	}
	return text
}

var _ domain.Tool = (*mcpToolAdapter)(nil)
