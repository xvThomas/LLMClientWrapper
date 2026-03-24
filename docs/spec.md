# Specifications

## Goal

Build an abstraction layer over the Anthropic and OpenAI client APIs, exposing a single unified client and a router that transparently delegates to either backend. For Anthropic, the abstraction must handle prompt caching to minimise token costs. For OpenAI-compatible backends, there is no caching, but the client must be able to target third-party models that expose an OpenAI-compatible API (Mistral, Llama, etc.).

The abstraction layer must also support `tool` calls. A working example is provided using the OpenWeatherMap API. The application must handle multi-turn conversations with a maximum of 5 tool-call iterations per turn. The conversation logic must remain fully provider-agnostic: it operates exclusively through the abstraction layer and has no knowledge of the underlying API clients. Message persistence (user and assistant messages) is also handled through an interface. The current implementation stores messages in memory.

The deliverable is a command-line executable. It accepts flags to select a model (e.g. Haiku 4.5, Sonnet 4.6, GPT-4o, Devstral) and runs an interactive prompt loop, returning complete responses — not streamed token by token — to answer questions such as *"What is the temperature in the capital of France?"*

## Implementation

### Environment variables

API keys (Anthropic, OpenAI, Mistral, etc.) are stored as environment variables in a `.env` file.

### Methodology

The project follows clean code principles: domain types, interfaces, and their implementations are kept clearly separated.

### Source layout

The project is written in Go.

| Path | Contents |
|---|---|
| `/src` | All source files |
| `/src/internal` | Domain types, interfaces, and business logic |
| `/src/internal/infrastructure` | Implementations that depend on external libraries (Anthropic, OpenAI, etc.) |
| `/src/cmd` | Main application entry point (CLI) |

### Coding style

- Keep the code simple and readable.
- Favour many small types over few large ones: each type should be focused and concise.
- Function bodies must not exceed **50 lines**.
- Cognitive complexity must not exceed **15**.

