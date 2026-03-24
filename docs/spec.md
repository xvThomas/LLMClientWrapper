# Spécifications

## Objectif

Créer une couche d'abstraction des API client Anthropic et OpenAI se résumant à un seul client et un routeur permettant d'utiliser l'API Athropic ou l'API OpenAI. Dans le premier cas, cette abstraction devra être capable de gérer le cache anthropic afin de minimiser les couts d'utilisation des modèles Anthropic. Dans le second cas, pas de gestion de cache, mais on s'assurera que le client sera capable d'adresser d'autres modèles frontières compatible avec l'API Anthropic (Mistral, llama, etc).

La couche d'abstration devra être également capable de gérer des appels à des `tools`. Un exemple sera donné à l'aide de openweathermaps.org. L'application devra être capable de gérer un conversation multi-tour avec un maximum de 5 appels à des tools. Le code gérant la conversation devra être le plus agnostique possible, sans connaissance des réels client API mis en jeu mais utilisant la couche d'abstraction. La persistence des messages User, AI est egalement géré avec un couche d'abstration (interface). L'implémentation de la persitence des messages est pour l'instant la plus simple possible: en mémoire.

Le produit du projet sera un executable en ligne de commande avec des paramètres permettant de choisir le modèle (Haiku 4.5, Sonnet 4.6, GPT 5.4, Devstral, etc.), de saisir une question et d'obtenir la réponse en un seul coup (pas de réponse au fil de l'eau token par token) afin de répondre des questions du genre "quelel est la température dans la capitale de la France."

## Réalisation

### Variables d'environnement

Le clés API (Anthropic, OpenAI, Mistral, etc.)  sont stockées en tant que variables d'environnement dans un fichier `.env`.

### Méthodologie

Le projet sera dévéloppé selon les principe du clean code (avec interface et types métiers et leur implémentation).

### Organisation du code source

Le code est dévéloppé en GoLang.

Le projet utilisera un répertoire /src pour les sources

un répertoire /src/internal est utiliser pour stockers les types, fonctions, interfaces métiers

Les implémentations dépéndantes des librairies externes (anthopic, openAI, logger, etc.) sont stockées dans le répertoire /src/internal/infrastructure.

Le code de l'application prinipale (mode commande) sera placé dans le répertoire /src/cmd.

### Style

Le code generé doit être simple.

Prioriser le nombre de types (classes) par rapport au nombre de lignes de code par classe: peu de code par type mais beaucoup de type.

Le corps des fonction ne doivent pas contenir plus de 50 lignes.

La complexité cognitive ne doit pas exceder 15.

