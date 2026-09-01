package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	prompt "github.com/elk-language/go-prompt"
)

func TestCommandSuggestionsTopLevel(t *testing.T) {
	suggestions := commandSuggestions("/m")
	if len(suggestions) == 0 {
		t.Fatal("expected top-level suggestions for /m")
	}

	foundModel := false
	foundMCP := false
	for _, suggestion := range suggestions {
		if suggestion.Text == "/model" {
			foundModel = true
		}
		if suggestion.Text == "/mcp" {
			foundMCP = true
		}
	}

	if !foundModel || !foundMCP {
		t.Fatalf("expected /model and /mcp in suggestions, got %+v", suggestions)
	}
}

func TestCommandSuggestionsSubcommands(t *testing.T) {
	suggestions := commandSuggestions("/mcp r")
	if len(suggestions) == 0 {
		t.Fatal("expected /mcp subcommand suggestions")
	}

	foundRemove := false
	foundRefresh := false
	for _, suggestion := range suggestions {
		if suggestion.Text == "remove" {
			foundRemove = true
		}
		if suggestion.Text == "refresh" {
			foundRefresh = true
		}
	}

	if !foundRemove || !foundRefresh {
		t.Fatalf("expected remove and refresh suggestions, got %+v", suggestions)
	}
}

func TestPersistHistorySkipsConsecutiveDuplicates(t *testing.T) {
	dir := t.TempDir()
	historyPath := filepath.Join(dir, "history")

	reader := &GoPromptReader{historyPath: historyPath}
	if _, err := reader.persistHistory("/help"); err != nil {
		t.Fatalf("persist first entry: %v", err)
	}
	if _, err := reader.persistHistory("/help"); err != nil {
		t.Fatalf("persist duplicate entry: %v", err)
	}
	if _, err := reader.persistHistory("hello"); err != nil {
		t.Fatalf("persist second unique entry: %v", err)
	}

	entries, err := loadHistoryEntries(historyPath)
	if err != nil {
		t.Fatalf("load history entries: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries after dedupe, got %d: %+v", len(entries), entries)
	}
	if entries[0] != "/help" || entries[1] != "hello" {
		t.Fatalf("unexpected history entries: %+v", entries)
	}
}

func TestStripANSI(t *testing.T) {
	got := stripANSI("\x1b[31mHello\x1b[0m world")
	if got != "Hello world" {
		t.Fatalf("stripANSI() = %q, want %q", got, "Hello world")
	}
}

func TestGoPromptReaderComplete(t *testing.T) {
	gr := &GoPromptReader{}
	suggestions, _, _ := gr.complete(*prompt.NewDocument())
	if suggestions != nil {
		t.Fatalf("expected nil suggestions for empty doc, got: %+v", suggestions)
	}
}

func TestNewGoPromptReader(t *testing.T) {
	dir := t.TempDir()
	historyPath := filepath.Join(dir, "history")
	if err := os.WriteFile(historyPath, []byte("/help\nhello\n"), 0o600); err != nil {
		t.Fatalf("write history: %v", err)
	}

	gr, err := NewGoPromptReader(historyPath)
	if err != nil {
		t.Fatalf("NewGoPromptReader() error = %v", err)
	}
	if len(gr.historyEntries) != 2 {
		t.Fatalf("historyEntries len = %d, want 2", len(gr.historyEntries))
	}
	if gr.lastPersisted != "hello" {
		t.Fatalf("lastPersisted = %q, want %q", gr.lastPersisted, "hello")
	}
}

func TestLoadHistoryEntries_ErrorOnDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := loadHistoryEntries(dir)
	if err == nil {
		t.Fatal("expected error when loading history from directory")
	}
}

func TestLoadHistoryEntries_FileNotFound(t *testing.T) {
	entries, err := loadHistoryEntries(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("expected nil error for nonexistent file, got %v", err)
	}
	if entries != nil {
		t.Fatalf("expected nil entries, got %v", entries)
	}
}

func TestLoadHistoryEntries_SkipsBlankLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	if err := os.WriteFile(path, []byte("first\n\n  \nsecond\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := loadHistoryEntries(path)
	if err != nil {
		t.Fatalf("loadHistoryEntries: %v", err)
	}
	if len(entries) != 2 || entries[0] != "first" || entries[1] != "second" {
		t.Fatalf("expected [first second], got %v", entries)
	}
}

func TestNewGoPromptReader_ErrorOnBadPath(t *testing.T) {
	// passing a directory triggers a scanner error inside loadHistoryEntries
	_, err := NewGoPromptReader(t.TempDir())
	if err == nil {
		t.Fatal("expected error when history path is a directory")
	}
}

func TestPersistHistory_EmptyStringSkipped(t *testing.T) {
	gr := &GoPromptReader{historyPath: filepath.Join(t.TempDir(), "history")}
	added, err := gr.persistHistory("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added {
		t.Fatal("expected added=false for empty string")
	}
}

func TestPersistHistory_ReturnsTrueOnFirstAdd(t *testing.T) {
	gr := &GoPromptReader{historyPath: filepath.Join(t.TempDir(), "history")}
	added, err := gr.persistHistory("first")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Fatal("expected added=true for first unique entry")
	}
}

func TestCommandSuggestions_NonCommandInput(t *testing.T) {
	for _, input := range []string{"", "hello", "some text"} {
		if got := commandSuggestions(input); got != nil {
			t.Errorf("commandSuggestions(%q) = %v, want nil", input, got)
		}
	}
}

func TestCommandSuggestions_Session(t *testing.T) {
	suggestions := commandSuggestions("/session l")
	if len(suggestions) == 0 {
		t.Fatal("expected /session subcommand suggestions for 'l'")
	}
	found := false
	for _, s := range suggestions {
		if s.Text == "list" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'list' in suggestions, got %+v", suggestions)
	}
}

func TestCommandSuggestions_UnknownSubcommand(t *testing.T) {
	if got := commandSuggestions("/foo bar"); got != nil {
		t.Errorf("commandSuggestions('/foo bar') = %v, want nil", got)
	}
	// 3+ parts — falls through to bottom nil
	if got := commandSuggestions("/mcp add extra"); got != nil {
		t.Errorf("commandSuggestions('/mcp add extra') = %v, want nil", got)
	}
}

func TestReadLine_NonTTY_UpdatesHistory(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
	}()
	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.WriteString("hello world\n")
	}()

	gr := &GoPromptReader{
		historyPath: filepath.Join(t.TempDir(), "history"),
		nonTTY:      true,
	}
	line, err := gr.ReadLine("> ")
	if err != nil {
		t.Fatalf("ReadLine: %v", err)
	}
	if line != "hello world" {
		t.Fatalf("ReadLine() = %q, want %q", line, "hello world")
	}
	if len(gr.historyEntries) != 1 || gr.historyEntries[0] != "hello world" {
		t.Fatalf("historyEntries = %v, want [hello world]", gr.historyEntries)
	}
}

func TestReadLineRaw(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
	}()
	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.WriteString("secret-value\n")
	}()

	gr := &GoPromptReader{historyPath: filepath.Join(t.TempDir(), "h")}
	line, err := gr.ReadLineRaw("> ")
	if err != nil {
		t.Fatalf("ReadLineRaw: %v", err)
	}
	if line != "secret-value" {
		t.Fatalf("ReadLineRaw() = %q, want %q", line, "secret-value")
	}
}

func TestReadLineFallback_EOF(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	_ = w.Close() // immediate EOF, no content
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
	}()

	_, err = readLineFallback("> ")
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestReadLineFallback_EOFWithContent(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	_, _ = w.WriteString("partial") // no newline — triggers EOF path with content
	_ = w.Close()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
	}()

	line, err := readLineFallback("> ")
	if err != nil {
		t.Fatalf("expected nil error for content at EOF, got %v", err)
	}
	if line != "partial" {
		t.Fatalf("readLineFallback() = %q, want %q", line, "partial")
	}
}
