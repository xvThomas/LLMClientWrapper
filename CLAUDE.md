# talk-backend

Go monorepo backend for the talk project, following the **bmad** methodology.

> **Single source of truth**: rules and agent references live in the files below.
> This file imports them so Claude Code sees the same context as GitHub Copilot and OpenCode —
> without duplicating content.

## Agents & Local Skills Catalog

@AGENTS.md

## Coding Rules

These rule files are shared with GitHub Copilot (same source). GitHub Copilot applies them
via `applyTo:` patterns; Claude Code applies them according to the file-type context indicated
in each file's frontmatter.

@.github/instructions/go.instructions.md

@.github/instructions/project-layout.instructions.md

@.github/instructions/mcp-server.instructions.md

@.github/instructions/cli-style.instructions.md

@.github/instructions/environment-variables.instructions.md

## Required Go Skills

Always load the following skills from `.agents/skills/` when starting a Go-related task.
`golang-how-to` is the orchestrator: it identifies which additional community skills
(`samber/cc-skills-golang@*`) to load based on the task context.

- `.agents/skills/golang-how-to/SKILL.md`
- `.agents/skills/golang-code-style/SKILL.md`
- `.agents/skills/golang-database/SKILL.md`
- `.agents/skills/golang-performance/SKILL.md`
