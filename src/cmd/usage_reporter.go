package main

import (
	"fmt"

	"llmclientwrapper/src/internal/domain"
)

// ConsoleUsageReporter implements domain.UsageReporter by printing token usage
// to the terminal using the ANSI helpers defined in main.go.
// It is the default reporter for the CLI session. Replace it with a remote
// reporter (e.g. Langfuse) when observability integration is needed.
type ConsoleUsageReporter struct{}

// OnAPICall is called after every API call and prints the token usage for that call.
//
// Parameters:
// - e: The APICallEvent containing details about the API call and its token usage.
func (ConsoleUsageReporter) OnAPICall(e domain.APICallEvent) {
	fmt.Printf(
		faint("  ↳ [token] model=%-14s kind=%-12s in=%5d out=%5d cache_read=%5d cache_write=%5d\n"),
		e.Model, string(e.Kind),
		e.Usage.InputTokens, e.Usage.OutputTokens,
		e.Usage.CacheReadTokens, e.Usage.CacheWriteTokens,
	)
}

// OnConversationTurn is called after every conversation turn and prints the token usage for that turn.
//
// Parameters:
// - e: The TurnEvent containing details about the conversation turn and its token usage.
func (ConsoleUsageReporter) OnConversationTurn(e domain.TurnEvent) {
	fmt.Printf(
		faint("  ↳ [turn]  model=%-14s calls=%d  total_in=%5d total_out=%5d cache_read=%5d cache_write=%5d\n"),
		e.Model, e.CallCount,
		e.TotalUsage.InputTokens, e.TotalUsage.OutputTokens,
		e.TotalUsage.CacheReadTokens, e.TotalUsage.CacheWriteTokens,
	)
}
