package domain

import (
	"context"
	"fmt"
	"sync"
)

const maxToolCalls = 5

// ConversationManager orchestrates a multi-turn conversation with optional tool calls.
type ConversationManager struct {
	client             LlmClient
	modelID            string
	store              MessageStore
	promptProvider     PromptProvider
	tools              []Tool
	reporter           UsageReporter
	maxConcurrentTools int // Maximum concurrent tool executions
}

// NewConversationManager creates a ConversationManager.
func NewConversationManager(client LlmClient, modelID string, store MessageStore, pp PromptProvider, tools []Tool, reporter UsageReporter, maxConcurrentTools int) *ConversationManager {
	return &ConversationManager{
		client:             client,
		modelID:            modelID,
		store:              store,
		promptProvider:     pp,
		tools:              tools,
		reporter:           reporter,
		maxConcurrentTools: maxConcurrentTools,
	}
}

// SetClient replaces the active LLM client and model without resetting the conversation history.
func (m *ConversationManager) SetClient(client LlmClient, modelID string) {
	m.client = client
	m.modelID = modelID
}

// Chat sends a user message and returns the final assistant text response.
// Tool calls are resolved automatically up to maxToolCalls iterations.
func (m *ConversationManager) Chat(ctx context.Context, userInput string) (string, error) {
	systemPrompt, err := m.promptProvider.SystemPrompt(ctx)
	if err != nil {
		return "", fmt.Errorf("loading system prompt: %w", err)
	}

	m.store.Add(Message{Role: RoleUser, Content: userInput})

	var totalUsage Usage
	callCount := 0
	kind := CallKindInitial

	for range maxToolCalls {
		response, usage, err := m.client.Complete(ctx, systemPrompt, m.store.All(), m.tools)
		if err != nil {
			return "", fmt.Errorf("model completion: %w", err)
		}

		m.reporter.OnAPICall(APICallEvent{Model: m.modelID, Kind: kind, Usage: usage})
		totalUsage = totalUsage.Add(usage)
		callCount++
		kind = CallKindToolResult

		m.store.Add(*response)

		if len(response.ToolCalls) == 0 {
			m.reporter.OnConversationTurn(TurnEvent{
				Model:      m.modelID,
				TotalUsage: totalUsage,
				CallCount:  callCount,
			})
			return response.Content, nil
		}

		if err := m.executeToolCalls(ctx, response.ToolCalls); err != nil {
			return "", err
		}
	}

	return "", fmt.Errorf("exceeded maximum tool call iterations (%d)", maxToolCalls)
}

func (m *ConversationManager) executeToolCalls(ctx context.Context, calls []ToolCall) error {
	if len(calls) == 1 || m.maxConcurrentTools <= 1 {
		// Execute sequentially for single calls or when concurrency is disabled
		return m.executeToolCallsSequential(ctx, calls)
	}
	// Execute in parallel with concurrency limit
	return m.executeToolCallsParallel(ctx, calls)
}

func (m *ConversationManager) executeToolCallsSequential(ctx context.Context, calls []ToolCall) error {
	results := make([]ToolResult, 0, len(calls))

	for _, call := range calls {
		result, err := m.executeTool(ctx, call)
		if err != nil {
			return err
		}
		results = append(results, result)
	}

	m.store.Add(Message{Role: RoleTool, ToolResults: results})
	return nil
}

func (m *ConversationManager) executeToolCallsParallel(ctx context.Context, calls []ToolCall) error {
	results := make([]ToolResult, len(calls))
	errors := make([]error, len(calls))

	// Limit concurrency using a semaphore pattern
	sem := make(chan struct{}, m.maxConcurrentTools)
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		go func(idx int, toolCall ToolCall) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := m.executeTool(ctx, toolCall)
			if err != nil {
				errors[idx] = err
				return
			}
			results[idx] = result
		}(i, call)
	}

	wg.Wait()

	// Check for any errors and return the first one found
	for i, err := range errors {
		if err != nil {
			return fmt.Errorf("tool %q failed: %w", calls[i].Name, err)
		}
	}

	m.store.Add(Message{Role: RoleTool, ToolResults: results})
	return nil
}

func (m *ConversationManager) executeTool(ctx context.Context, call ToolCall) (ToolResult, error) {
	for _, t := range m.tools {
		if t.Name() == call.Name {
			content, err := t.Execute(ctx, call.Input)
			if err != nil {
				return ToolResult{}, fmt.Errorf("tool %q execution: %w", call.Name, err)
			}
			return ToolResult{ToolCallID: call.ID, Content: content}, nil
		}
	}
	return ToolResult{}, fmt.Errorf("unknown tool %q", call.Name)
}
