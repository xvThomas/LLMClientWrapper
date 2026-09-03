package domain

// Role represents the author of a message in a conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall describes a tool invocation requested by the model.
type ToolCall struct {
	ID    string
	Name  string
	Input map[string]any
}

// ToolResult holds the output of an executed tool call.
type ToolResult struct {
	ToolCallID string
	// Content is the payload sent to the model.
	Content string
	// ClientContent is the full payload for non-model consumers such as the map UI.
	// Empty when it would be identical to Content.
	ClientContent string
}

// ClientPayload returns the content intended for non-model consumers.
func (r ToolResult) ClientPayload() string {
	if r.ClientContent != "" {
		return r.ClientContent
	}
	return r.Content
}

// Message is a single entry in a conversation.
type Message struct {
	Role        Role
	Content     string
	Thinking    string // summarized reasoning/thinking from the model (ephemeral, not persisted)
	ToolCalls   []ToolCall
	ToolResults []ToolResult
	TurnID      string // unique turn identifier used for reconciliation across sources
}
