# Integration Langfuse

Il s'agit d'intégrer LangFuse via le mécanisme de UsageReporter pour envoyer les données vers langFuse (par une implémentation LangfuseUsageReporter.go)

- Verifier qu'il existe une librairie langfuse pour go (sinon créeer une implémentataion à partir d'un client http)
- envoyer un maximum d'information dans un premier via l'API langfuse:
    Question/Prompt : "temps à Orléans ?"
    Réponse du modèle : "À Orléans, il fait 5,5°C..."
    Tool calls : {name: "get_current_weather", params: {city: "Orléans"}}
    Tool input/output : Input: {city: "Orléans"}, Output: "5.5°C, sunny"
    Token usage : {prompt: 637, completion: 58, cache_read: 0, cache_write: 0}
    Latency : 850ms pour le call API
    Model & provider : claude-sonnet-4-5 (Anthropic)

    Métadonnées custom : {user_id, session_id, tags, custom_fields}, pour l'instant elles seront vides (nous les renseigneront utltérieurement)

Cost : Calculé automatiquement à partir des tokens (est ce que cela se fait dans langfuse, ou faut il faire quelque chose au niveau de la code base ?)

- Ajouter au variables d'environnement, les vraiables supplémentaires
  - LANGFUSE_SECRET_KEY="sk-lf-..."
  - LANGFUSE_PUBLIC_KEY="pk-lf-..."
  - LANGFUSE_BASE_URL="https://cloud.langfuse.com" (valeur par défaut si nons stipulé)

- les usageReporter sont cumulatifs, c'est à dire
 - si la (nouvelle) variable d'env CONSOLE_USAGE_REPORTER est spécifiée (égale à 1 ou true, voir avec la bonne pratique pour cela), alors ConsoleUsageReporter est utilisé
 - d'autre part si les variables LANGFUSE_SECRET_KEY, LANGFUSE_PUBLIC_KEY ne sont toutes deux présentes alors LangfuseUsageReporter est également utilisé
 - il s'agit donc de gerer maintenant un tableau d'UsageReporter (éventuellement vide) et de les utilser en parrallèle si il contient plusieurs occurences.

 Enfin ConsoleUsageReporter.go devrait maintant être rangé dans /internal/infrastructure/usage à coté de LangfuseUsageReporter.go (UsageReporter est bien placé, dans le réprtoire domain)
 
 La mise à jour succinte du README.md et de plan_llmClientWrapper.prompt.md sont à faire également
