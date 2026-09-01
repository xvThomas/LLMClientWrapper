package agui

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/google/uuid"
	"github.com/pixime-net/talk/internal/domain"
)

var _ domain.MessageEventHandler = (*AGUIEmitter)(nil)

// AGUIEmitter emits all AG-UI content events (text messages and tool calls)
// to an SSE stream. It implements domain.MessageEventHandler.
type AGUIEmitter struct {
	sse *SSEWriter
	log *slog.Logger
}

// NewAGUIEmitter creates an AGUIEmitter that writes content events to the given SSE writer.
func NewAGUIEmitter(sse *SSEWriter, log *slog.Logger) *AGUIEmitter {
	if log == nil {
		log = slog.Default()
	}
	return &AGUIEmitter{sse: sse, log: log}
}

// HandleMessageEvent emits REASONING_* events for thinking content (any assistant message),
// then TEXT_MESSAGE_START → TEXT_MESSAGE_CONTENT → TEXT_MESSAGE_END for final assistant messages
// (no tool calls, non-empty content).
func (e *AGUIEmitter) HandleMessageEvent(ctx context.Context, event domain.MessageEvent) error {
	if event.Role != domain.RoleAssistant {
		return nil
	}
	if event.Thinking != "" {
		e.emitReasoning(ctx, event.Thinking)
	}
	if len(event.ToolCalls) == 0 && event.Content != "" {
		e.emitTextMessage(ctx, event.Content)
	}
	return nil
}

// HandleTurnEvent is a no-op for the SSE emitter.
func (e *AGUIEmitter) HandleTurnEvent(_ context.Context, _ domain.TurnEvent) error {
	return nil
}

// HandleToolCallStart emits TOOL_CALL_START and TOOL_CALL_ARGS events before tool execution.
func (e *AGUIEmitter) HandleToolCallStart(ctx context.Context, event domain.ToolCallEvent) error {
	_ = e.writeEvent(ctx, events.NewToolCallStartEvent(event.ToolCall.ID, event.ToolCall.Name))
	argsJSON, _ := json.Marshal(event.ToolCall.Input)
	_ = e.writeEvent(ctx, events.NewToolCallArgsEvent(event.ToolCall.ID, string(argsJSON)))
	return nil
}

// HandleToolCallEnd emits TOOL_CALL_END then TOOL_CALL_RESULT with the tool output.
func (e *AGUIEmitter) HandleToolCallEnd(ctx context.Context, event domain.ToolCallEndEvent) error {
	_ = e.writeEvent(ctx, events.NewToolCallEndEvent(event.ToolCall.ID))
	_ = e.writeEvent(ctx, events.NewToolCallResultEvent(uuid.New().String(), event.ToolCall.ID, event.Result.Content))
	return nil
}

// emitReasoning emits the REASONING_* event sequence for a thinking block.
func (e *AGUIEmitter) emitReasoning(ctx context.Context, thinking string) {
	id := uuid.New().String()
	_ = e.writeEvent(ctx, events.NewReasoningStartEvent(id))
	_ = e.writeEvent(ctx, events.NewReasoningMessageStartEvent(id, "reasoning"))
	_ = e.writeEvent(ctx, events.NewReasoningMessageContentEvent(id, thinking))
	_ = e.writeEvent(ctx, events.NewReasoningMessageEndEvent(id))
	_ = e.writeEvent(ctx, events.NewReasoningEndEvent(id))
}

// emitTextMessage emits the TEXT_MESSAGE_* event sequence for a final assistant message.
func (e *AGUIEmitter) emitTextMessage(ctx context.Context, content string) {
	id := uuid.New().String()
	_ = e.writeEvent(ctx, events.NewTextMessageStartEvent(id, events.WithRole("assistant")))
	_ = e.writeEvent(ctx, events.NewTextMessageContentEvent(id, content))
	_ = e.writeEvent(ctx, events.NewTextMessageEndEvent(id))
}

// writeEvent checks for context cancellation, writes the event, and handles errors
// according to the best-effort policy: log unexpected errors, return nil always.
func (e *AGUIEmitter) writeEvent(ctx context.Context, event events.Event) error {
	if ctx.Err() != nil {
		e.log.Debug("skipping SSE write, context canceled")
		return ctx.Err()
	}
	if err := e.sse.WriteEvent(ctx, event); err != nil {
		if ctx.Err() != nil {
			e.log.Debug("SSE write failed due to client disconnect")
		} else {
			e.log.Warn("unexpected SSE write error", slog.String("error", err.Error()))
		}
		return err
	}
	return nil
}
