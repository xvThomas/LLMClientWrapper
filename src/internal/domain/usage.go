package domain

// Usage holds token consumption for a single LLM API call.
type Usage struct {
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64 // tokens served from prompt cache
	CacheWriteTokens int64 // tokens written to prompt cache (Anthropic only)
}

// Add returns the sum of two Usage values.
func (u Usage) Add(other Usage) Usage {
	return Usage{
		InputTokens:      u.InputTokens + other.InputTokens,
		OutputTokens:     u.OutputTokens + other.OutputTokens,
		CacheReadTokens:  u.CacheReadTokens + other.CacheReadTokens,
		CacheWriteTokens: u.CacheWriteTokens + other.CacheWriteTokens,
	}
}

// CallKind classifies the type of LLM call within a conversation turn.
type CallKind string

const (
	// CallKindInitial is the first call of a turn (user → assistant).
	CallKindInitial CallKind = "initial"
	// CallKindToolResult is a subsequent call after tool execution.
	CallKindToolResult CallKind = "tool_result"
)

// APICallEvent is emitted after each individual Complete() invocation.
type APICallEvent struct {
	Model string
	Kind  CallKind
	Usage Usage
}

// TurnEvent is emitted once at the end of a full Chat() turn (all calls aggregated).
type TurnEvent struct {
	Model      string
	TotalUsage Usage
	CallCount  int
}

// UsageReporter receives usage telemetry events.
type UsageReporter interface {
	OnAPICall(event APICallEvent)
	OnConversationTurn(event TurnEvent)
}

// NoOpUsageReporter silently discards all events.
type NoOpUsageReporter struct{}

func (NoOpUsageReporter) OnAPICall(APICallEvent)       {}
func (NoOpUsageReporter) OnConversationTurn(TurnEvent) {}
