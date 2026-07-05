# talk-backend

Ce projet est piloté par la méthodologie **bmad**. Les skills, agents et artéfacts bmad se trouvent dans `../talk-bmad/`.

## Agents bmad (via `../talk-bmad/`)

| Agent | Fichier | Rôle |
|-------|---------|------|
| Architect | `../talk-bmad/.github/agents/bmad-agent-architect.agent.md` | Design technique |
| Developer | `../talk-bmad/.github/agents/bmad-agent-dev.agent.md` | Implémentation |
| PM | `../talk-bmad/.github/agents/bmad-agent-pm.agent.md` | Gestion d'epics |
| Analyst | `../talk-bmad/.github/agents/bmad-agent-analyst.agent.md` | Analyse |
| Tech Writer | `../talk-bmad/.github/agents/bmad-agent-tech-writer.agent.md` | Documentation |
| UX Designer | `../talk-bmad/.github/agents/bmad-agent-ux-designer.agent.md` | Design UX |

Les agents bmad référencent leurs skills via `{project-root}/.agents/skills/` (soit `../talk-bmad/.agents/skills/` depuis ce projet).

## Skills Go locaux (`.agents/skills/`)

| Skill | Description |
|-------|-------------|
| `golang-how-to` | Orchestrateur — charge automatiquement les autres skills Go selon le contexte |
| `golang-code-style` | Conventions de style Go |
| `golang-database` | Accès base de données (pgx, sqlx, transactions, pool) |
| `golang-performance` | Patterns d'optimisation Go |

Ces skills sont chargés automatiquement par opencode quand le contexte le nécessite (via `golang-how-to`).
