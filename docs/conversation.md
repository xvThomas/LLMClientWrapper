# Gestion des conversations avec Langfuse

L'objectif est d'utiliser langfuse pour retrouver le contenu des conversations passés. On considère que les conversations sont correctement stockées à l'aide de langfuseUsageProvider.

Il s'agit de gérer maintenant:

- un user_id
- un session_id

On considère qu'une conversation est créée au lancement du CLI (un session_id est créee sous la forme d'un UUID). Dans un premier temps le userID est "anonymous" (sa réelle utilité interviendra plus tard, lorsque le code évéouera vers un service web, sécurisé par keycloak).

Toutes les questions/réponses (et leur traces) sont stockées en utilisant le même session_id jusqu'à l'arrêt du CLI.

Cependant, on prévoit deux nouvelles commandes du prompt CLI:
/memory : permettant de lister les questions/réponses et leur traces pour la session en cours (du plus récent au plus ancien)
/session : présentant toutes les session_id associé à l'utilisateur courant (du plus récent au plus ancien). Dans la même session du cli, on se donnée la possibilité de changer de conversation (session_id) à l'aide du mécanisme déjà utilisé dans le cadre du choix du modèle (main.go).

Enfin on s'assure que l'implémentaion actuelle permet de transmettre le user_id et le session_id à langfuse via LangfuseUsageReporter.

Avant de procéder à l'implémentation, on s'assurera que langfuse permet de gerer ces mécanismes.

Le code principal de cette évolution sera créé et rangé dans le répertoire /internal/infrastructure/memory/langfuse. L'interface domain.MessageStore est probablement à modifier et enrichir. Essayons tout de même de conserver l'implementation memory.Store actuelle tout en l'adaptant, histoire d'avoir un fallback, même si elle devient inutilisée.
