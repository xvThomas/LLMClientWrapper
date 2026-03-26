# Plan: LLM Client Wrapper (Go)

## TL;DR

Interactive CLI Go app wrapping Anthropic, OpenAI and Mistral behind a unified interface.
Features: interactive REPL with slash commands, ANSI colour output, persistent input history,
Anthropic prompt caching, multi-provider routing, tool calls (OpenWeatherMap example),
multi-turn conversation loop (max 5 tool calls), in-memory message store.
Clean architecture: domain interfaces in `src/internal/domain/`, infrastructure in
`src/internal/infrastructure/`, CLI entry in `src/cmd/`.

---

## Current Architecture

### Domain Layer (`src/internal/domain/`)

- `message.go` — `Role` type (user/assistant/tool), `Message`, `ToolCall`, `ToolResult`
- `model.go` — `Model` string type, `Provider` enum (anthropic/openai/mistral),
  `ModelDescriptor{Provider, APIModelID}`, registry: `haiku-4.5`, `sonnet-4.6`, `gpt-5.4`, `mistral-small`
- `client.go` — `LlmClient` interface: `Complete(ctx, systemPrompt, messages, tools) (*Message, error)`
- `tool.go` — `Tool` interface: `Name()`, `Description()`, `Parameters()`, `Execute()`
- `store.go` — `MessageStore` interface: `Add`, `All`, `Clear`
- `prompt_provider.go` — `PromptProvider` interface: `SystemPrompt(ctx) (string, error)`
- `conversation.go` — `ConversationManager` + `Chat()` + `SetClient()` (for runtime model switch)

### Infrastructure Layer (`src/internal/infrastructure/`)

- `config/loader.go` — `.env` loading via godotenv, `Config` struct, `Require*Key()` helpers
- `memory/store.go` — `InMemoryStore` implements `MessageStore`
- `prompt/file_provider.go` — `FilePromptProvider{path}` reads Markdown file
- `prompt/static_provider.go` — `StaticPromptProvider{text}` returns fixed string (tests only)
- `llm/anthropic/{converter,client}.go` — domain↔Anthropic SDK; ephemeral cache_control on system prompt
- `llm/openai/{converter,client}.go` — domain↔OpenAI SDK; reused for Mistral via base URL override
- `llm/router/router.go` — `Router{cfg}` with `New(cfg)` and `Get(model) (LlmClient, error)`;
  builds the right client from provider + API key; absorbs the former `buildClient` that was in main
- `tools/tools.go` — `Tools{cfg}` aggregator: `New(cfg)` + `All() []domain.Tool`
- `tools/weather/tool.go` — `WeatherTool` calls OpenWeatherMap current weather API

### CLI (`src/cmd/`)

- `main.go` — cobra root command; flags: `--model` (required), `--system-file` (default: `system_prompt.md`
  next to executable); interactive REPL loop using `LineReader`; slash-command dispatcher; ANSI colour helpers
- `history.go` — `History` type: loads/saves `~/.llmclientwrapper_history`, cursor navigation, dedup
- `reader.go` — `LineReader` type: raw-mode terminal (golang.org/x/term), ↑/↓ history navigation,
  `\033[2K` line redraw, UTF-8 multi-byte support, TTY fallback for piped input

### REPL Slash Commands

| Command   | Effect                                      |
| --------- | ------------------------------------------- |
| `/model`  | Interactive model switch (updates client)   |
| `/prompt` | Prints raw system prompt text               |
| `/tools`  | Lists registered tools (name + description) |
| `/q`      | Exits the session                           |

### ANSI Colour Conventions (see `.github/instructions/cli-style.instructions.md`)

Helpers defined in `main.go`: `cyan()`, `green()`, `yellow()`, `red()`, `faint()`, `emphasize()`

- Errors → `red()`; warnings → `yellow()`; prompts/labels → `cyan()` or `green()`; hints → `faint()`
- LLM answer text is **never** coloured

---

## Key Decisions

- **Language:** All identifiers, comments, test names, commit messages in **English**
- `go.mod` at project root; source under `src/`
- `LlmClient.Complete` receives `systemPrompt` separately (Anthropic API requirement)
- Anthropic cache: ephemeral `cache_control` on system prompt only
- Mistral shares `OpenAIClient` with `https://api.mistral.ai/v1` base URL override
- Model registry isolates friendly aliases from API-specific IDs
- `PromptProvider` is a domain interface — `ConversationManager` has no file/env knowledge
- `Router` lives in `infrastructure/llm/router/` and owns the `buildClient` logic
- `Tools` aggregator in `infrastructure/tools/tools.go` — no key validation at startup;
  errors surface at call time and are displayed in red in the conversation
- Input history stored in `~/.llmclientwrapper_history` (stable across `go run` and compiled binary)
- `--system` inline flag removed; only `--system-file` is supported
- CLI uses `cobra` for flag parsing; interactive loop uses raw-mode `LineReader`, not `bufio.Scanner`
- No external colour library — 7 ANSI constants + 6 helpers inline in `main.go`

---

## Dependencies

```
github.com/anthropics/anthropic-sdk-go
github.com/openai/openai-go
github.com/joho/godotenv
github.com/spf13/cobra
golang.org/x/term
```

---

## File Map

```
src/
  cmd/
    main.go           # CLI entrypoint, REPL, slash commands, colour helpers
    history.go        # History type — persistence + cursor navigation
    reader.go         # LineReader type — raw-mode terminal input
  internal/
    domain/
      message.go
      model.go
      client.go
      tool.go
      store.go
      prompt_provider.go
      conversation.go
      conversation_test.go
    infrastructure/
      config/loader.go
      memory/store.go, store_test.go
      prompt/file_provider.go, file_provider_test.go
      prompt/static_provider.go, static_provider_test.go
      llm/
        anthropic/converter.go, client.go
        openai/converter.go, client.go
        router/router.go, router_test.go
      tools/
        tools.go
        weather/tool.go, tool_test.go
.github/
  instructions/cli-style.instructions.md   # applyTo: src/cmd/**
  prompts/plan-llmClientWrapper.prompt.md  # this file
```

---

## Makefile Targets

| Target          | Command                                                          |
| --------------- | ---------------------------------------------------------------- |
| `build`         | `go build -o bin/llmclientwrapper ./src/cmd`                     |
| `run`           | `go run ./src/cmd --model $(MODEL) --system-file $(SYSTEM_FILE)` |
| `test`          | `go test ./...`                                                  |
| `cover`         | coverage HTML report                                             |
| `cover-summary` | per-package coverage in terminal                                 |
| `vet`           | `go vet ./...`                                                   |
| `clean`         | remove `bin/`, coverage files                                    |
| `all`           | `vet test build` (default)                                       |

---

## Verification

1. `make vet` — no issues
2. `make test` — all unit tests pass
3. `make build && ./bin/llmclientwrapper --model sonnet-4.6` — interactive session starts
4. ↑/↓ arrows navigate history across sessions (`~/.llmclientwrapper_history`)

5. `make cover` — `coverage.html` generated and opens in browser
6. `make build` — produces `bin/llmclientwrapper`
7. Manual: `make run MODEL=sonnet-4.6 QUESTION="Quelle est la température à Paris?"` → WeatherTool called, answer printed
8. Manual: `go run src/cmd/main.go --model devstral --system "Tu es un assistant." --question "Bonjour"` → `StaticPromptProvider` used
9. Manual: `go run src/cmd/main.go --model devstral --question "Hello"` → routes to Mistral, `FilePromptProvider` used
10. Check each function body ≤ 50 lines, cognitive complexity ≤ 15
