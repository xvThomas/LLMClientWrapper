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

# 4. Ask a question
make run MODEL=sonnet-4.6 QUESTION="What is the temperature in the capital of France?"
```

Or without `make`:

```bash
go run ./src/cmd --model sonnet-4.6 --question "What is the temperature in the capital of France?"
```

Use a custom inline system prompt:

```bash
go run ./src/cmd --model gpt-5.4 --system "You are a concise assistant." --question "Hello"
```

Use a custom system prompt file:

```bash
go run ./src/cmd --model devstral --system-file ./my_prompt.md --question "Hello"
```

---

## Available models

| Alias        | Provider  | Notes                          |
|--------------|-----------|--------------------------------|
| `haiku-4.5`  | Anthropic | Fast and cheap                 |
| `sonnet-4.6` | Anthropic | Balanced                       |
| `gpt-5.4`    | OpenAI    |                                |
| `devstral`   | Mistral   | OpenAI-compatible API          |

---

## Environment variables

Copy `.env.example` to `.env` and fill in the relevant keys:

| Variable                  | Required for              |
|---------------------------|---------------------------|
| `ANTHROPIC_API_KEY`       | `haiku-4.5`, `sonnet-4.6` |
| `OPENAI_API_KEY`          | `gpt-5.4`                 |
| `MISTRAL_API_KEY`         | `devstral`                |
| `OPENWEATHERMAP_API_KEY`  | weather tool calls        |

---

## System prompt

By default the CLI loads `system_prompt.md` from the same directory as the executable. Edit that file to customise the assistant's persona without recompiling.

Override at runtime:

```bash
# inline
--system "You are a helpful assistant."

# file
--system-file /path/to/prompt.md
```

---

## Make targets

| Target          | Description                                       |
|-----------------|---------------------------------------------------|
| `make build`    | Compile to `bin/llmclientwrapper`                 |
| `make run`      | `go run` with `MODEL=` and `QUESTION=` vars       |
| `make test`     | Run all unit tests                                |
| `make cover`    | Generate `coverage.html` (opens-ready HTML report)|
| `make cover-summary` | Print per-package coverage percentages      |
| `make vet`      | Run `go vet`                                      |
| `make clean`    | Remove `bin/`, `coverage.out`, `coverage.html`    |

---

## Project structure

```
.
├── src/
│   ├── internal/               # Domain interfaces & types
│   │   ├── infrastructure/     # External-library implementations
│   │   │   ├── anthropic/
│   │   │   ├── openai/
│   │   │   ├── weather/
│   │   │   ├── prompt/
│   │   │   ├── memory/
│   │   │   └── config/
│   └── cmd/                    # CLI entry point
├── system_prompt.md            # Default system prompt
├── .env.example
└── Makefile
```
