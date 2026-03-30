---
description: "Project layout conventions for the talks monorepo"
applyTo: "**/*.go,**/go.mod,Makefile"
---

# Project Layout

This project follows the [Go standard project layout](https://github.com/golang-standards/project-layout). The module name is `talks`.

## Directory structure

```
cmd/<executable>/   One subdirectory per binary. Each contains only a minimal main.go
                    that wires dependencies from internal/ and runs.
internal/           All private shared code. Never import from outside this module.
  domain/           Core business types and interfaces. No external dependencies.
  infrastructure/   Implementations that depend on external libraries.
    config/         Environment variable loading (.env parsing).
    llm/            LLM provider adapters (anthropic/, openai/) and router/.
    memory/         MessageStore implementations (inmemory/, langfuse/).
    prompt/         PromptProvider implementations (file, static).
    tools/          Tool aggregator and individual tool implementations.
    usage/          UsageReporter implementations (console, langfuse, otlp).
  version/          Build-time injected version string.
```

## Rules

- **New executables** go in `cmd/<name>/main.go`. Name the directory after the binary (e.g. `cmd/talk-cli/`).
- **Shared business logic** goes in `internal/domain/`. It must have zero external library dependencies.
- **Provider/library-specific code** goes in `internal/infrastructure/`. Keep each adapter in its own sub-package.
- **Public reusable packages** (importable by future external repos) go in `pkg/`. Do not create `pkg/` until there is a concrete external consumer.
- **No `src/` prefix.** Source files live directly under `cmd/` and `internal/`.
- **`main.go` must stay thin**: import, wire, run — no business logic.
- Keep `internal/domain/` and `internal/infrastructure/` cleanly separated; domain must never import infrastructure.
