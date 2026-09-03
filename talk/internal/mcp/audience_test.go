package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// audienceServer exposes a tool returning one content block per audience.
func audienceServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "v0.0.1"}, nil)
	server.AddTool(&mcp.Tool{
		Name:        "route",
		Description: "returns audience-split route",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}, func(_ context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{
			&mcp.TextContent{
				Text:        `{"distance":1200}`,
				Annotations: &mcp.Annotations{Audience: []mcp.Role{roleAssistant}},
			},
			&mcp.TextContent{
				Text:        `{"distance":1200,"geometry":{"coordinates":[[1,2],[3,4]]}}`,
				Annotations: &mcp.Annotations{Audience: []mcp.Role{roleUser}},
			},
		}}, nil
	})
	return connectInMemorySession(t, server)
}

func TestToolAdapter_Execute_SplitsByAudience(t *testing.T) {
	session := audienceServer(t)
	defer func() { _ = session.Close() }()

	adapter := &mcpToolAdapter{
		manager:    NewManager(&stubRegistry{}),
		serverID:   "srv-1",
		serverName: "ign-nav",
		tool:       mcpTool("route", "route", nil),
	}
	adapter.manager.sessions["srv-1"] = session

	got, err := adapter.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, has := got.Model["geometry"]; has {
		t.Error("model payload must not carry the geometry")
	}
	if got.Model["distance"] != float64(1200) {
		t.Errorf("model distance = %v, want 1200", got.Model["distance"])
	}
	if got.Client == nil {
		t.Fatal("client payload must be populated when audiences differ")
	}
	if _, has := got.Client["geometry"]; !has {
		t.Error("client payload must carry the geometry")
	}
}

func TestToolAdapter_Execute_UnannotatedContentHasNoClientPayload(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "v0.0.1"}, nil)
	server.AddTool(&mcp.Tool{
		Name:        "plain",
		Description: "returns a single payload",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}, func(_ context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{
			&mcp.TextContent{Text: `{"temp":21}`},
		}}, nil
	})

	session := connectInMemorySession(t, server)
	defer func() { _ = session.Close() }()

	adapter := &mcpToolAdapter{
		manager:    NewManager(&stubRegistry{}),
		serverID:   "srv-1",
		serverName: "owm",
		tool:       mcpTool("plain", "plain", nil),
	}
	adapter.manager.sessions["srv-1"] = session

	got, err := adapter.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Model["temp"] != float64(21) {
		t.Errorf("model temp = %v, want 21", got.Model["temp"])
	}
	if got.Client != nil {
		t.Error("client payload must stay nil when every audience sees the same content")
	}
}

func TestTextForAudience_UnannotatedBlockReachesEveryAudience(t *testing.T) {
	content := []mcp.Content{
		&mcp.TextContent{Text: "shared"},
		&mcp.TextContent{Text: "model only", Annotations: &mcp.Annotations{Audience: []mcp.Role{roleAssistant}}},
		&mcp.TextContent{Text: "client only", Annotations: &mcp.Annotations{Audience: []mcp.Role{roleUser}}},
	}

	model := textForAudience(content, roleAssistant)
	if !strings.Contains(model, "shared") || !strings.Contains(model, "model only") {
		t.Errorf("assistant text = %q, want shared + model only", model)
	}
	if strings.Contains(model, "client only") {
		t.Errorf("assistant text must not include client-only blocks, got %q", model)
	}

	client := textForAudience(content, roleUser)
	if !strings.Contains(client, "shared") || !strings.Contains(client, "client only") {
		t.Errorf("user text = %q, want shared + client only", client)
	}
	if strings.Contains(client, "model only") {
		t.Errorf("user text must not include model-only blocks, got %q", client)
	}
}

func TestDecodeToolText_NonJSONFallsBackToContentKey(t *testing.T) {
	got := decodeToolText("plain text")
	if got["content"] != "plain text" {
		t.Errorf("decodeToolText() = %v, want content key", got)
	}

	raw, _ := json.Marshal(map[string]any{"a": 1})
	got = decodeToolText(string(raw))
	if _, has := got["content"]; has {
		t.Errorf("JSON payloads must not be wrapped, got %v", got)
	}
}
