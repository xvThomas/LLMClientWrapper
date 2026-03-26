# LLM Client Wrapper

A Go CLI that routes questions to Anthropic or OpenAI-compatible models (GPT, Mistral, Devstral …) through a single unified interface. Supports tool calls (OpenWeatherMap example), Anthropic prompt caching, and in-memory conversation history.

---

## Prerequisites

- Go 1.23+
- `make`
- API keys for the providers you want to use (see [Environment variables](#environment-variables))

---

## Quickstart

```bash
# 1. Clone and enter the project
git clone <repo-url>
cd LlmClientWrapper

# 2. Copy and fill in your API keys
cp .env.example .env
$EDITOR .env

# 3. Build
make build

# 4. Start an interactive session
make run MODEL=sonnet-4.6
```

Or without `make`:

```bash
go run ./src/cmd --model sonnet-4.6
```

Use a custom system prompt file:

```bash
go run ./src/cmd --model mistral-small --system-file ./my_prompt.md
```

Type `exit` or `quit` (or press `Ctrl+C`) to end the session.

---

## Available models

| Alias           | Provider  | Notes                 |
| --------------- | --------- | --------------------- |
| `haiku-4.5`     | Anthropic | Fast and cheap        |
| `sonnet-4.6`    | Anthropic | Balanced              |
| `gpt-5.4`       | OpenAI    |                       |
| `mistral-small` | Mistral   | OpenAI-compatible API |

---

## Environment variables

Copy `.env.example` to `.env` and fill in the relevant keys:

| Variable                 | Required for              |
| ------------------------ | ------------------------- |
| `ANTHROPIC_API_KEY`      | `haiku-4.5`, `sonnet-4.6` |
| `OPENAI_API_KEY`         | `gpt-5.4`                 |
| `MISTRAL_API_KEY`        | `mistral-small`           |
| `OPENWEATHERMAP_API_KEY` | weather tool calls        |

---

## System prompt

By default the CLI loads `system_prompt.md` from the same directory as the executable. Edit that file to customise the assistant's persona without recompiling.

Override at runtime with `--system-file`:

```bash
--system-file /path/to/prompt.md
```

---

## Make targets

| Target               | Description                                                     |
| -------------------- | --------------------------------------------------------------- |
| `make build`         | Compile to `bin/llmclientwrapper`                               |
| `make run`           | Interactive session — override with `MODEL=` and `SYSTEM_FILE=` |
| `make test`          | Run all unit tests                                              |
| `make cover`         | Generate `coverage.html` (opens-ready HTML report)              |
| `make cover-summary` | Print per-package coverage percentages                          |
| `make vet`           | Run `go vet`                                                    |
| `make clean`         | Remove `bin/`, `coverage.out`, `coverage.html`                  |

---

## Project structure

```txt
.
├── src/
│   ├── cmd/                        # CLI entry point (interactive REPL)
│   └── internal/
│       ├── domain/                 # Interfaces & types (Model, Message, Tool …)
│       └── infrastructure/
│           ├── llm/                # LLM provider implementations
│           │   ├── anthropic/
│           │   ├── openai/
│           │   └── router/         # Builds LlmClient from config + model alias
│           ├── config/             # .env loader
│           ├── memory/             # In-memory conversation store
│           ├── prompt/             # File & static prompt providers
│           └── weather/            # OpenWeatherMap tool
├── system_prompt.md                # Default system prompt
├── .env.example
└── Makefile
```
