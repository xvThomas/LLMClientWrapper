package domain

import (
	"context"
	"errors"
	"testing"
)

// stubClient is a controllable LlmClient for tests.
type stubClient struct {
	responses []*Message
	callCount int
}

func (s *stubClient) Complete(_ context.Context, _ string, _ []Message, _ []Tool) (*Message, error) {
	if s.callCount >= len(s.responses) {
		return nil, errors.New("stub: no more responses")
	}
	resp := s.responses[s.callCount]
	s.callCount++
	return resp, nil
}

// stubStore is a simple in-memory MessageStore for tests.
type stubStore struct {
	messages []Message
}

func (s *stubStore) Add(msg Message)    { s.messages = append(s.messages, msg) }
func (s *stubStore) All() []Message    { return s.messages }
func (s *stubStore) Clear()            { s.messages = nil }

// stubPromptProvider returns a fixed system prompt.
type stubPromptProvider struct{ text string }

func (p *stubPromptProvider) SystemPrompt(_ context.Context) (string, error) {
	return p.text, nil
}

// stubTool is a Tool that records calls and returns a fixed result.
type stubTool struct {
	name   string
	result string
	err    error
	called int
}

func (t *stubTool) Name() string                      { return t.name }
func (t *stubTool) Description() string               { return "stub tool" }
func (t *stubTool) Parameters() map[string]any        { return map[string]any{} }
func (t *stubTool) Execute(_ context.Context, _ map[string]any) (string, error) {
	t.called++
	return t.result, t.err
}

// --- tests ---

func TestConversation_NoToolCall(t *testing.T) {
	client := &stubClient{responses: []*Message{
		{Role: RoleAssistant, Content: "Hello!"},
	}}
	mgr := NewConversationManager(client, &stubStore{}, &stubPromptProvider{"system"}, nil)

	answer, err := mgr.Chat(context.Background(), "Hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != "Hello!" {
		t.Errorf("expected 'Hello!', got %q", answer)
	}
}

func TestConversation_SingleToolCall(t *testing.T) {
	tool := &stubTool{name: "get_current_weather", result: "20°C, sunny"}
	client := &stubClient{responses: []*Message{
		{
			Role: RoleAssistant,
			ToolCalls: []ToolCall{{ID: "1", Name: "get_current_weather", Input: map[string]any{"city": "Paris"}}},
		},
		{Role: RoleAssistant, Content: "It is 20°C and sunny in Paris."},
	}}
	mgr := NewConversationManager(client, &stubStore{}, &stubPromptProvider{"system"}, []Tool{tool})

	answer, err := mgr.Chat(context.Background(), "Weather in Paris?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != "It is 20°C and sunny in Paris." {
		t.Errorf("unexpected answer: %q", answer)
	}
	if tool.called != 1 {
		t.Errorf("expected tool called once, got %d", tool.called)
	}
}

func TestConversation_MaxToolCallsExceeded(t *testing.T) {
	tool := &stubTool{name: "loop_tool", result: "looping"}
	responses := make([]*Message, maxToolCalls+1)
	for i := range responses {
		responses[i] = &Message{
			Role:      RoleAssistant,
			ToolCalls: []ToolCall{{ID: "x", Name: "loop_tool", Input: map[string]any{}}},
		}
	}
	client := &stubClient{responses: responses}
	mgr := NewConversationManager(client, &stubStore{}, &stubPromptProvider{"system"}, []Tool{tool})

	_, err := mgr.Chat(context.Background(), "loop?")
	if err == nil {
		t.Error("expected error when max tool calls exceeded")
	}
}

func TestConversation_UnknownToolReturnsError(t *testing.T) {
	client := &stubClient{responses: []*Message{
		{
			Role:      RoleAssistant,
			ToolCalls: []ToolCall{{ID: "1", Name: "nonexistent", Input: map[string]any{}}},
		},
	}}
	mgr := NewConversationManager(client, &stubStore{}, &stubPromptProvider{"system"}, nil)

	_, err := mgr.Chat(context.Background(), "call unknown tool")
	if err == nil {
		t.Error("expected error for unknown tool")
	}
}
