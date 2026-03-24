# Plan: LLM Client Wrapper (Go)

## TL;DR
Build a CLI Go app wrapping Anthropic & OpenAI behind a unified interface. Features: Anthropic prompt caching, OpenAI-compatible multi-provider routing (Mistral/Devstral), tool calls with OpenWeatherMap example, multi-turn conversation loop (max 5 tool calls), in-memory message store. Clean architecture: domain interfaces in `src/internal/`, infrastructure in `src/internal/infrastructure/`, CLI entry in `src/cmd/`.

---

## Phases

### Phase 1 — Project Bootstrap
1. `go.mod` at project root — module `llmclientwrapper`, Go 1.23+
2. `.env.example` — keys: ANTHROPIC_API_KEY, OPENAI_API_KEY, MISTRAL_API_KEY, OPENWEATHERMAP_API_KEY
3. `.gitignore` — ignore `.env`, binaries
4. `README.md` — project overview, prerequisites, quickstart (`make build`, `make run`), available models table, environment variables reference, `make cover` usage

### Phase 2 — Domain Layer (`src/internal/`)
4. `message.go` — `Role` type (user/assistant/tool), `Message`, `ToolCall`, `ToolResult`
5. `model.go` — `Model` string type, `Provider` enum, `ModelDescriptor{Provider, APIModelID}`, registry map (haiku-4.5, sonnet-4.6, gpt-5.4, devstral)
6. `client.go` — `LlmClient` interface: `Complete(ctx, systemPrompt string, messages []Message, tools []Tool) (*Message, error)`
7. `tool.go` — `Tool` interface: `Name()`, `Description()`, `Parameters() map[string]any`, `Execute(ctx, input map[string]any) (string, error)`
8. `store.go` — `MessageStore` interface: `Add(Message)`, `All() []Message`, `Clear()`
9. `prompt_provider.go` — `PromptProvider` interface: `SystemPrompt(ctx context.Context) (string, error)`
10. `router.go` — `Router` struct: `Register(model, client)`, `Get(model) (LlmClient, error)`
11. `conversation.go` — `ConversationManager{client, store, promptProvider, tools, maxToolCalls=5}` + `Chat(ctx, userInput) (string, error)` — fetches system prompt via `PromptProvider`, tool loop

### Phase 3 — Infrastructure Layer (`src/internal/infrastructure/`)
12. `config/loader.go` — load `.env` (godotenv), typed `Config` struct
13. `memory/store.go` — `InMemoryStore` implements `MessageStore` (slice-backed)
14. `prompt/file_provider.go` — `FilePromptProvider{path string}` implements `PromptProvider`; reads Markdown file from path resolved relative to `os.Executable()`
15. `prompt/static_provider.go` — `StaticPromptProvider{text string}` implements `PromptProvider`; returns a fixed inline string — used when `--system` flag is passed
16. `anthropic/converter.go` — domain↔Anthropic SDK types; apply `cache_control:{type:ephemeral}` to system prompt
17. `anthropic/client.go` — `AnthropicClient` implementing `LlmClient` (anthropic-sdk-go)
18. `openai/converter.go` — domain↔OpenAI SDK types
19. `openai/client.go` — `OpenAIClient{baseURL, apiKey, modelID}` implementing `LlmClient`; reused for GPT + Mistral/Devstral via base URL override
20. `weather/tool.go` — `WeatherTool` implementing `Tool`; calls OpenWeatherMap current weather API

### Phase 4 — CLI (`src/cmd/`)
21. `main.go` — built with **`cobra`**: root command `ask` with flags `--model` (required), `--question` (required), `--system` (inline, optional), `--system-file` (default `system_prompt.md`); cobra handles `--help`, flag validation and shell-completion out of the box; if `--system` is set use `StaticPromptProvider`, else use `FilePromptProvider`; build `Router`; wire `ConversationManager`; print response
22. `system_prompt.md` — default system prompt Markdown file at project root

### Phase 5 — Tests
23. `router_test.go` — Register/Get happy path, unknown model error
24. `conversation_test.go` — mock `LlmClient`, `MessageStore`, `PromptProvider`: no-tool response, single tool call resolved, 5-call cap
25. `memory/store_test.go` — Add/All/Clear
26. `prompt/file_provider_test.go` — file found, file missing
27. `prompt/static_provider_test.go` — returns expected string
28. `weather/tool_test.go` — `httptest.NewServer` mock of OpenWeatherMap

### Phase 6 — Makefile
29. `Makefile` at project root with targets:
    - `build` — `go build -o bin/llmclientwrapper ./src/cmd`
    - `run` — `go run ./src/cmd --model $(MODEL) --question $(QUESTION)`
    - `test` — `go test ./...`
    - `cover` — `go test -coverprofile=coverage.out ./...` then `go tool cover -html=coverage.out -o coverage.html`
    - `cover-summary` — `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`
    - `vet` — `go vet ./...`
    - `clean` — remove `bin/`, `coverage.out`, `coverage.html`
    - `all` — `vet test build` (default)

### Phase 7 — Git
30. `git init` at project root + initial commit with all source files; add `bin/`, `coverage.out`, `coverage.html` to `.gitignore`

### Phase 8 — Dependencies
31. `go.mod` + `go.sum`: `github.com/anthropics/anthropic-sdk-go`, `github.com/openai/openai-go`, `github.com/joho/godotenv`, `github.com/spf13/cobra`

---

## Key Decisions
- **Language:** All identifiers (types, functions, variables, constants), comments, test function names, commit messages, and doc strings are written in **English** — no exceptions
- `go.mod` at project root; source trees under `src/`
- `LlmClient.Complete` receives `systemPrompt string` separately (Anthropic treats system apart from messages)
- Anthropic cache: ephemeral `cache_control` on system prompt
- Mistral/Devstral share `OpenAIClient` with different base URL + API key
- Model registry isolates friendly names from API-specific IDs
- `PromptProvider` is a domain interface — `ConversationManager` has no knowledge of files, env vars, or static strings
- **CLI framework:** `cobra` (`github.com/spf13/cobra`) — auto-generated `--help`, required-flag enforcement, shell completion; single `ask` command keeps the interface simple
- If `--system` given → `StaticPromptProvider`; otherwise → `FilePromptProvider` (default path: `system_prompt.md` next to executable)

---

## Relevant files to create
- `go.mod`, `.env.example`, `.gitignore`, `system_prompt.md`, `Makefile`, `README.md`
- `src/internal/{message,model,client,tool,store,prompt_provider,router,conversation}.go`
- `src/internal/{router_test,conversation_test}.go`
- `src/internal/infrastructure/config/loader.go`
- `src/internal/infrastructure/memory/store.go`, `store_test.go`
- `src/internal/infrastructure/prompt/file_provider.go`, `file_provider_test.go`
- `src/internal/infrastructure/prompt/static_provider.go`, `static_provider_test.go`
- `src/internal/infrastructure/anthropic/{converter,client}.go`
- `src/internal/infrastructure/openai/{converter,client}.go`
- `src/internal/infrastructure/weather/tool.go`, `tool_test.go`
- `src/cmd/main.go`

---

## Verification
1. `git log --oneline` — initial commit present
2. `make vet` — no issues
3. `make test` — all unit tests pass
4. `make cover-summary` — per-package coverage percentages printed
5. `make cover` — `coverage.html` generated and opens in browser
6. `make build` — produces `bin/llmclientwrapper`
7. Manual: `make run MODEL=sonnet-4.6 QUESTION="Quelle est la température à Paris?"` → WeatherTool called, answer printed
8. Manual: `go run src/cmd/main.go --model devstral --system "Tu es un assistant." --question "Bonjour"` → `StaticPromptProvider` used
9. Manual: `go run src/cmd/main.go --model devstral --question "Hello"` → routes to Mistral, `FilePromptProvider` used
10. Check each function body ≤ 50 lines, cognitive complexity ≤ 15
