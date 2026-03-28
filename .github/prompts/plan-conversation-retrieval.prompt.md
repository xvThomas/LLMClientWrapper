# Plan d'implémentation — Gestion des conversations (session_id / user_id)

## Contexte

Utiliser Langfuse pour retrouver le contenu des conversations passées. Gérer un `session_id` (UUID, généré au démarrage du CLI) et un `user_id` (`"anonymous"` pour l'instant). Deux nouvelles commandes CLI : `/memory` et `/session`.

## Décisions

- Session switch **recharge les messages** depuis Langfuse dans le buffer (contexte LLM restauré)
- `langfuse.Store` remplace `memory.Store` si Langfuse est configuré ; sinon fallback `memory.Store`
- `/memory` : Q&A paginé + trace breakdown (llm_initial, llm_tool_result) via REST Langfuse
- `user_id` = `"anonymous"` codé en dur pour l'instant

---

## Phase 1 — Couche Domain

**Étape 1** — `src/internal/domain/store.go`
- Enrichir `MessageStore` : ajouter `SessionID() string` et `UserID() string`
- Créer la nouvelle interface `SessionBrowser` :
  - `SetSession(ctx context.Context, sessionID string) error`
  - `ListSessions(ctx context.Context, userID string) ([]SessionSummary, error)`
  - `LoadSession(ctx context.Context, sessionID string) ([]HistoryTurn, error)`
- Ajouter les types `SessionSummary{ID, CreatedAt, TurnCount}` et `HistoryTurn{Question, Answer, At, Model, CallCount}`

**Étape 2** — `src/internal/domain/usage.go`
- Ajouter `SessionID string` et `UserID string` sur `TurnEvent` et `APICallEvent`
- Ajouter `GenerateSessionID() string` (UUID v4 via `crypto/rand`)

---

## Phase 2 — ConversationManager

**Étape 3** — `src/internal/domain/conversation.go`
- Lire `sessionID` / `userID` depuis le store dans `NewConversationManager`
- Dans `Chat()`, injecter `SessionID` / `UserID` dans les `TurnEvent` et `APICallEvent`

---

## Phase 3 — Adapter memory.Store (fallback)

**Étape 4** — `src/internal/infrastructure/memory/store.go`
- Ajouter `sessionID string` et `userID string` comme champs
- Mettre à jour `NewStore(sessionID, userID string) *Store`
- Implémenter `SessionID()`, `UserID()`
- Stubs : `SetSession` (change ID + clear), `ListSessions` / `LoadSession` (retournent nil, nil)
- `Store` implémente `domain.MessageStore` + `domain.SessionBrowser`

---

## Phase 4 — Nouveau langfuse.Store

**Étape 5** — `src/internal/infrastructure/memory/langfuse/store.go` *(fichier créé)*

Struct `Store{sessionID, userID, httpClient, baseURL, authHeader, buffer []domain.Message, mu sync.Mutex}`

- `NewStore(sessionID, userID string, cfg Config) *Store` avec `Config{PublicKey, SecretKey, BaseURL}`
- `Add` / `All` / `Clear` : délèguent au buffer in-memory
- `SessionID()`, `UserID()` : retournent les champs
- `SetSession(ctx, sessionID)` : clear buffer → `loadMessagesForSession` → repeupler le buffer
- `ListSessions(ctx, userID)` : `GET /api/public/sessions?userId={userId}` Basic Auth
- `LoadSession(ctx, sessionID)` : `GET /api/public/traces?sessionId={sid}&orderBy=timestamp&order=ASC` → `[]HistoryTurn`
- `loadMessagesForSession` : reconstruit `[]domain.Message{Role, Content}` depuis les traces pour recharger le contexte LLM
- `fetchObservations(ctx, traceID)` : `GET /api/public/observations?traceId={id}` pour le détail dans `/memory`

---

## Phase 5 — OTLP session/user

**Étape 6** — `src/internal/infrastructure/usage/otlp.go`
- Dans `conversationTurnToOTLP` : ajouter `langfuse.trace.session_id` et `langfuse.trace.user_id` depuis `event.SessionID` / `event.UserID`

---

## Phase 6 — CLI

**Étape 7** — `src/cmd/main.go`
- Générer `sessionID := domain.GenerateSessionID()` et `userID := "anonymous"` dans `run()`
- Si Langfuse configuré : `store = langfuse.NewStore(sessionID, userID, langfuseCfg)` ; sinon `store = memory.NewStore(sessionID, userID)`
- Mettre à jour le banner (ajouter `/memory`, `/session`)
- `cmdMemory(ctx, store)` : type-assert → `domain.SessionBrowser`, affichage Q&A paginé + détail trace
- `cmdSession(ctx, store, manager, lr)` : liste sessions (style `/model`), appel `SetSession` sur la sélection
- Mettre à jour `handleSlashCommand`

---

## Fichiers modifiés

| Fichier | Action |
|---|---|
| `src/internal/domain/store.go` | Interface enrichie + types SessionSummary, HistoryTurn |
| `src/internal/domain/usage.go` | Champs SessionID/UserID + GenerateSessionID |
| `src/internal/domain/conversation.go` | Propagation dans les events |
| `src/internal/infrastructure/memory/store.go` | Adapter au nouveau contrat |
| `src/internal/infrastructure/usage/otlp.go` | Attributs OTLP session/user |
| `src/cmd/main.go` | UUID, store switch, /memory, /session |

## Fichiers créés

| Fichier | Action |
|---|---|
| `src/internal/infrastructure/memory/langfuse/store.go` | Nouveau store Langfuse |

---

## Vérification

1. `go build ./...` sans erreur
2. `go test ./...` — tests existants passent (constructeur `NewStore` mis à jour dans les tests)
3. Lancer le CLI, poser 2 questions → vérifier dans l'UI Langfuse que `sessionId` et `userId` sont visibles sur les traces
4. `/memory` → Q&A de la session en cours s'affiche avec timestamps et détail llm_initial / llm_tool_result
5. `/session` → liste les sessions passées de "anonymous" ; sélectionner une ancienne session → messages rechargés dans `All()` (vérifiable en posant une question qui référence le contexte)
6. Compilation sans Langfuse configuré → `memory.Store` fallback fonctionne sans régression