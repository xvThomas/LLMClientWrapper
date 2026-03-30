# Langfuse Integration

Langfuse is integrated via the `domain.UsageReporter` interface. There is no official Go SDK for Langfuse; the implementation sends traces directly over HTTP using the OpenTelemetry OTLP format.

## Architecture

```
domain.UsageReporter          (interface, in internal/domain/usage.go)
  ├── ConsoleUsageReporter    (internal/infrastructure/usage/console.go)
  └── LangfuseUsageReporter   (internal/infrastructure/usage/langfuse.go)
```

Reporters are **additive**: the `ConversationManager` holds a slice of `UsageReporter` and fires all of them in parallel via `sync.WaitGroup` after each API call and conversation turn.

## Data sent to Langfuse

Each recorded event is converted to an OTLP span and posted to the `/v1/traces` endpoint:

| Field             | Example                                                        |
| ----------------- | -------------------------------------------------------------- |
| Prompt / question | `"What is the temperature in Orléans?"`                        |
| Model response    | `"In Orléans, it is 5.5°C..."`                                 |
| Tool calls        | `{name: "get_current_weather", params: {city: "Orléans"}}`     |
| Tool input/output | Input: `{city: "Orléans"}`, Output: `"5.5°C, sunny"`           |
| Token usage       | `{prompt: 637, completion: 58, cache_read: 0, cache_write: 0}` |
| Latency           | `850ms` per API call                                           |
| Model & provider  | `claude-sonnet-4-5 (Anthropic)`                                |
| Service name      | `talks`                                                        |

Cost is calculated automatically by Langfuse from the token counts and model name — no additional handling is needed in the codebase.

Custom metadata fields (`user_id`, `session_id`, `tags`) are present in the OTLP span structure but left empty for now.

## Transport

`LangfuseUsageReporter` buffers events in a channel (capacity 1000) and processes them in a background goroutine. Authentication uses HTTP Basic Auth: `base64(LANGFUSE_PUBLIC_KEY:LANGFUSE_SECRET_KEY)`.

## Environment variables

| Variable                 | Required        | Default                      | Description                                           |
| ------------------------ | --------------- | ---------------------------- | ----------------------------------------------------- |
| `LANGFUSE_PUBLIC_KEY`    | yes (to enable) | —                            | `pk-lf-…`                                             |
| `LANGFUSE_SECRET_KEY`    | yes (to enable) | —                            | `sk-lf-…`                                             |
| `LANGFUSE_BASE_URL`      | no              | `https://cloud.langfuse.com` | Use `https://us.cloud.langfuse.com` for the US region |
| `CONSOLE_USAGE_REPORTER` | no              | `true`                       | Set to `false` to disable terminal token output       |

`LangfuseUsageReporter` is only instantiated when both `LANGFUSE_PUBLIC_KEY` and `LANGFUSE_SECRET_KEY` are present. The two reporters are independent and can be active simultaneously.
