package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type splitIn struct {
	Msg string `json:"msg"`
}

type splitOut struct {
	Summary  string     `json:"summary"`
	Geometry *[]float64 `json:"geometry,omitempty"`
}

type splitTool struct{}

func (splitTool) Name() string        { return "split" }
func (splitTool) Description() string { return "returns audience-split content" }
func (splitTool) Call(_ context.Context, in splitIn) (splitOut, error) {
	return splitOut{Summary: in.Msg, Geometry: &[]float64{1, 2, 3}}, nil
}
func (splitTool) ModelView(out splitOut) splitOut {
	out.Geometry = nil
	return out
}

type plainTool struct{}

func (plainTool) Name() string        { return "plain" }
func (plainTool) Description() string { return "returns a single payload" }
func (plainTool) Call(_ context.Context, in splitIn) (splitOut, error) {
	return splitOut{Summary: in.Msg}, nil
}

func callTool(t *testing.T, registrar ToolRegistrar, name string) *mcp.CallToolResult {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	registrar.Register(server)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	go func() { _ = server.Run(ctx, serverTransport) }()

	session, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "v0"}, nil).
		Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: map[string]any{"msg": "hello"},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	return res
}

func TestRegisterTool_ModelViewerSplitsAudiences(t *testing.T) {
	res := callTool(t, RegisterTool[splitIn, splitOut](splitTool{}), "split")

	if len(res.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(res.Content))
	}

	model, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("block 0 is not text content")
	}
	if model.Annotations == nil || len(model.Annotations.Audience) != 1 ||
		model.Annotations.Audience[0] != RoleAssistant {
		t.Fatalf("block 0 audience = %v, want [assistant]", model.Annotations)
	}
	var modelOut map[string]any
	if err := json.Unmarshal([]byte(model.Text), &modelOut); err != nil {
		t.Fatalf("model block is not JSON: %v", err)
	}
	if _, has := modelOut["geometry"]; has {
		t.Error("model block must not carry the geometry")
	}

	client, ok := res.Content[1].(*mcp.TextContent)
	if !ok {
		t.Fatalf("block 1 is not text content")
	}
	if client.Annotations == nil || len(client.Annotations.Audience) != 1 ||
		client.Annotations.Audience[0] != RoleUser {
		t.Fatalf("block 1 audience = %v, want [user]", client.Annotations)
	}
	var clientOut map[string]any
	if err := json.Unmarshal([]byte(client.Text), &clientOut); err != nil {
		t.Fatalf("client block is not JSON: %v", err)
	}
	if _, has := clientOut["geometry"]; !has {
		t.Error("client block must carry the geometry")
	}
}

func TestRegisterTool_WithoutModelViewerKeepsSingleBlock(t *testing.T) {
	res := callTool(t, RegisterTool[splitIn, splitOut](plainTool{}), "plain")

	if len(res.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(res.Content))
	}
	block, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("block is not text content")
	}
	if block.Annotations != nil {
		t.Errorf("unannotated tools must not carry audience annotations, got %v", block.Annotations)
	}
}
