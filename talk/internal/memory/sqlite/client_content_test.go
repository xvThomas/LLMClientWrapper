package sqlite

import (
	"context"
	"testing"

	"github.com/pixime-net/talk/internal/domain"
)

func TestMessageRepository_PersistsClientContentSeparately(t *testing.T) {
	s, _, _ := newTestStore(t)
	ctx := context.Background()

	// The session is only materialized on the first user message.
	mustAddMessage(t, s, domain.Message{Role: domain.RoleUser, Content: "route to Lyon"}, scope)

	msg := domain.Message{
		Role:      domain.RoleTool,
		TurnID:    "turn-1",
		ToolCalls: []domain.ToolCall{{ID: "call-1", Name: "ign-nav__route", Input: map[string]any{}}},
		ToolResults: []domain.ToolResult{{
			ToolCallID:    "call-1",
			Content:       `{"distance":1200}`,
			ClientContent: `{"distance":1200,"geometry":{"coordinates":[[1,2]]}}`,
		}},
	}
	mustAddMessage(t, s, msg, scope)

	loaded, err := s.AllMessages(ctx, scope.SessionID())
	if err != nil {
		t.Fatalf("AllMessages: %v", err)
	}
	if len(loaded) != 2 || len(loaded[1].ToolResults) != 1 {
		t.Fatalf("expected a user message and a tool message, got %+v", loaded)
	}

	got := loaded[1].ToolResults[0]
	if got.Content != `{"distance":1200}` {
		t.Errorf("Content = %q, want the model payload", got.Content)
	}
	if got.ClientContent != `{"distance":1200,"geometry":{"coordinates":[[1,2]]}}` {
		t.Errorf("ClientContent = %q, want the full payload", got.ClientContent)
	}
	if got.ClientPayload() != got.ClientContent {
		t.Errorf("ClientPayload() = %q, want ClientContent", got.ClientPayload())
	}
}

func TestMessageRepository_ClientPayloadFallsBackToContent(t *testing.T) {
	s, _, _ := newTestStore(t)
	ctx := context.Background()

	mustAddMessage(t, s, domain.Message{Role: domain.RoleUser, Content: "weather"}, scope)

	msg := domain.Message{
		Role:      domain.RoleTool,
		TurnID:    "turn-1",
		ToolCalls: []domain.ToolCall{{ID: "call-1", Name: "owm__get_current_weather", Input: map[string]any{}}},
		ToolResults: []domain.ToolResult{{
			ToolCallID: "call-1",
			Content:    `{"temp":21}`,
		}},
	}
	mustAddMessage(t, s, msg, scope)

	loaded, err := s.AllMessages(ctx, scope.SessionID())
	if err != nil {
		t.Fatalf("AllMessages: %v", err)
	}

	got := loaded[1].ToolResults[0]
	if got.ClientContent != "" {
		t.Errorf("ClientContent = %q, want empty", got.ClientContent)
	}
	if got.ClientPayload() != `{"temp":21}` {
		t.Errorf("ClientPayload() = %q, want the model payload", got.ClientPayload())
	}
}
