# Plan d'implémentation - Intégration Langfuse

## 📊 Analyse initiale

### État des SDKs Langfuse

- ✅ **Python & JavaScript/TypeScript** : SDKs officiels maintenus
- ❌ **Go** : Pas de SDK officiel
- ✅ **Solution** : Implémentation HTTP custom via API publique Langfuse

### Architecture actuelle

- **UsageReporter** : Interface unique dans `domain/usage.go`
- **ConsoleUsageReporter** : Implémentation dans `/cmd/usage_reporter.go`
- **ConversationManager** : Accepte un seul `UsageReporter`

### Endpoint API cible

- **URL** : `https://cloud.langfuse.com/api/public/otel/v1/traces`
- **Protocole** : OpenTelemetry OTLP over HTTP/JSON
- **Authentification** : Basic Auth (Public Key : Secret Key)
- **Headers** : `Authorization=Basic ${base64(pk:sk)}`, `x-langfuse-ingestion-version=4`

## 🗺️ Plan en 5 phases

### Phase 1 : ✅ Recherche & Architecture API Langfuse [TERMINÉE]

**Résultats** :

- API OpenTelemetry endpoint identifié
- Authentification Basic Auth avec Public/Secret Key
- Format OpenTelemetry OTLP avec mapping détaillé des attributs
- Stratégie de mapping : `UsageReporter` events → OpenTelemetry spans → Langfuse observations

**Mapping OpenTelemetry** :

- `APICallEvent` → OpenTelemetry Span avec attributs `gen_ai.*`
- `TurnEvent` → OpenTelemetry Trace aggregé
- Tool calls → Spans imbriqués avec attributs tool-specific
- Usage tokens → `gen_ai.usage.*` pour calcul automatique des coûts

### Phase 2 : Refactoring système UsageReporter

#### Phase 2A : Structure & Déplacement

**Tâches** :

- Créer package `src/internal/infrastructure/usage/`
- Déplacer `ConsoleUsageReporter` de `/cmd/usage_reporter.go` vers `infrastructure/usage/console.go`
- Ajouter fonction `faint()` helper dans le nouveau fichier
- Supprimer ancien fichier `/cmd/usage_reporter.go`

**Livrables** :

- `infrastructure/usage/console.go` avec `ConsoleUsageReporter`
- Structure propre séparant domain et infrastructure

#### Phase 2B : Multi-reporters

**Tâches** :

- Modifier `domain/conversation.go` : `UsageReporter` → `[]UsageReporter`
- Adapter `NewConversationManager` pour accepter slice
- Implémenter appels parallèles vers tous les reporters
- Gestion d'erreur isolée (une failure n'impacte pas les autres)

**Signature cible** :

```go
func NewConversationManager(
    client LlmClient,
    modelID string,
    store MessageStore,
    pp PromptProvider,
    tools []Tool,
    reporters []UsageReporter,  // ← changement ici
    maxConcurrentTools int
) *ConversationManager
```

**Livrables** :

- Interface multi-reporters opérationnelle
- Tests mis à jour pour le nouveau système
- Appels parallèles avec gestion d'erreur robuste

### Phase 3 : Implémentation LangfuseUsageReporter

#### Phase 3A : Client HTTP OpenTelemetry

**Tâches** :

- Créer `infrastructure/usage/langfuse.go`
- Implémenter client HTTP with Basic Auth
- Structure OpenTelemetry Trace/Span selon spec OTLP
- Gestion retry logic + circuit breaker pour résilience

**Architecture client** :

```go
type LangfuseUsageReporter struct {
    httpClient *http.Client
    baseURL    string
    authHeader string    // base64(publicKey:secretKey)
    buffer     chan TraceData
}
```

#### Phase 3B : Mapping des événements

**Tâches** :

- Mapper `APICallEvent` → OpenTelemetry Span avec attributs :
  - `gen_ai.request.model` / `gen_ai.response.model`
  - `gen_ai.usage.input_tokens` / `gen_ai.usage.output_tokens`
  - `gen_ai.prompt` / `gen_ai.completion`
  - `gen_ai.operation.name` (completion, tool_call)
- Mapper `TurnEvent` → OpenTelemetry Trace complet
- Gérer tool calls comme spans imbriqués
- Calculer timing/latency automatiquement

**Données envoyées** :

- Question/Prompt : "temps à Orléans ?"
- Réponse du modèle : "À Orléans, il fait 5,5°C..."
- Tool calls : `{name: "get_current_weather", params: {city: "Orléans"}}`
- Tool input/output : Input: `{city: "Orléans"}`, Output: `"5.5°C, sunny"`
- Token usage : `{prompt: 637, completion: 58, cache_read: 0, cache_write: 0}`
- Latency : 850ms pour le call API
- Model & provider : claude-sonnet-4-5 (Anthropic)
- Métadonnées custom : `{user_id, session_id, tags, custom_fields}` (vides initialement)

#### Phase 3C : Gestion asynchrone

**Tâches** :

- Buffer events pour envoi en batch
- Worker goroutine pour traitement asynchrone
- Retry logic avec exponential backoff
- Graceful shutdown avec flush des events en attente

**Livrables** :

- `LangfuseUsageReporter` fonctionnel
- Mapping complet des données (tokens, latency, model, tool calls)
- Tests unitaires pour le client Langfuse
- Performance impact minimal (envois asynchrones)

### Phase 4 : Configuration & Variables d'environnement

#### Phase 4A : Extension Config

**Nouvelles variables** :

```go
type Config struct {
    // ... existing fields ...

    // Langfuse configuration
    LangfuseSecretKey   string // LANGFUSE_SECRET_KEY="sk-lf-..."
    LangfusePublicKey   string // LANGFUSE_PUBLIC_KEY="pk-lf-..."
    LangfuseBaseURL     string // LANGFUSE_BASE_URL="https://cloud.langfuse.com" (défaut)

    // Reporter configuration
    ConsoleUsageReporter bool   // CONSOLE_USAGE_REPORTER=true/false
}
```

#### Phase 4B : Logique conditionnelle

**Tâches** :

- Modifier `config/loader.go` pour parser nouvelles env vars
- Implémenter logique d'instanciation dans `main.go` :

```go
var reporters []domain.UsageReporter

// Console reporter si configuré
if cfg.ConsoleUsageReporter {
    reporters = append(reporters, &usage.ConsoleUsageReporter{})
}

// Langfuse reporter si clés présentes
if cfg.LangfuseSecretKey != "" && cfg.LangfusePublicKey != "" {
    langfuseClient := usage.NewLangfuseUsageReporter(cfg)
    reporters = append(reporters, langfuseClient)
}
```

#### Phase 4C : Configuration par défaut

**Tâches** :

- Mettre à jour `.env.example` avec nouvelles variables et documentation
- Valeurs par défaut : `LANGFUSE_BASE_URL="https://cloud.langfuse.com"`
- Documentation automatique du README (grâce aux instructions existantes)

**Livrables** :

- Système de configuration complet
- Instanciation automatique des reporters selon env vars
- Documentation variables mise à jour automatiquement
- Reporters cumulatifs (console + langfuse simultanément possible)

### Phase 5 : Finalisation & Documentation

#### Phase 5A : Tests d'intégration

**Tâches** :

- Tests end-to-end avec vraie API Langfuse (en option)
- Validation données dans dashboard Langfuse
- Tests de performance (impact latency du double reporting)
- Tests de résilience (network failures, API errors)

#### Phase 5B : Documentation utilisateur

**Tâches** :

- Mise à jour `README.md` avec section Langfuse
- Documenter configuration des env vars
- Exemples d'usage avec multiple reporters
- Guide de troubleshooting

#### Phase 5C : Optimisations finales

**Tâches** :

- Profiling performance impact
- Optimisation buffer size / flush frequency
- Monitoring health check du client Langfuse
- Metrics internes (events sent/failed)

**Livrables** :

- Integration complète et testée en production
- Documentation utilisateur complète
- Système prêt pour déploiement
- Performance baseline établie

## ⚡ Points techniques critiques

### Architecture multi-reporters

- **Changement breaking** : `UsageReporter` → `[]UsageReporter`
- **Appels parallèles** : goroutines avec sync.WaitGroup
- **Gestion d'erreur** : isolation des failures entre reporters
- **Performance** : impact minimal grâce à asynchronisme

### Client HTTP Langfuse

- **Authentification** : Basic Auth avec base64 encoding
- **Format requests** : OpenTelemetry OTLP JSON selon spec officielle
- **Résilience** : retry logic + circuit breaker + timeouts
- **Buffering** : events buffering pour optimiser throughput

### Mapping des données

- **APICallEvent** → OpenTelemetry Span (generation)
- **TurnEvent** → OpenTelemetry Trace complet
- **Tool calls** → OpenTelemetry Tool spans imbriqués
- **Coûts** : Calcul automatique par Langfuse via token counts
- **Timing** : Latency tracking précis via timestamps

### Configuration conditionnelle

- **Console** : Activé si `CONSOLE_USAGE_REPORTER=true`
- **Langfuse** : Activé si `LANGFUSE_SECRET_KEY` + `LANGFUSE_PUBLIC_KEY` présentes
- **Cumulatif** : Plusieurs reporters peuvent être actifs simultanément
- **Graceful degradation** : Failure d'un reporter n'impacte pas les autres

## 🎯 Critères de succès

### Fonctionnel

- ✅ Reports console maintenus (backward compatibility)
- ✅ Reports Langfuse complets (traces, spans, generations, events)
- ✅ Configuration flexible via env vars
- ✅ Performance impact < 5ms per call

### Technique

- ✅ Architecture clean maintenue (domain/infrastructure separation)
- ✅ Tests coverage > 80% sur nouvelles fonctionnalités
- ✅ Zero breaking changes pour utilisateurs existants
- ✅ Résilience network failures + API errors

### Observabilité

- ✅ Toutes données LLM disponibles dans Langfuse dashboard
- ✅ Tool calls tracés avec input/output complets
- ✅ Token usage + coûts calculés automatiquement
- ✅ Sessions et utilisateurs supportés (metadata futures)

## 🚀 Prochaines actions

1. **Phase 2A** : Créer structure `infrastructure/usage/` et déplacer `ConsoleUsageReporter`
2. **Phase 2B** : Refactor `ConversationManager` pour multi-reporters
3. **Phase 3A** : Implémenter client HTTP Langfuse avec OpenTelemetry OTLP
4. **Phase 3B** : Mapping complet des événements vers format Langfuse
5. **Phase 4** : Configuration système + env vars + documentation

**Estimation** : 3-4 jours de développement pour implémentation complète
