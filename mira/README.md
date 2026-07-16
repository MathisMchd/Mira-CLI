# Mira

Application de prise de notes. Disponible en **CLI** et en **API REST**, adossée à **PostgreSQL** (+ **pgvector**) et enrichie automatiquement en arrière-plan par deux modèles **Ollama** locaux : un modèle génératif pour tags/résumé/score, un modèle d'embedding pour la recherche vectorielle.

> La CLI ne stocke rien localement : elle passe systématiquement par l'API HTTP, seul moyen de déclencher l'enrichissement automatique de chaque note créée ou modifiée. **L'API doit donc être démarrée avant d'utiliser la CLI.**

---

## Démarrage rapide

```sh
cp .env.example .env
docker compose up --build
```

Ceci démarre :

| Service       | Rôle                                                                        |
|---------------|-------------------------------------------------------------------------------|
| `db`          | PostgreSQL 16 + extension pgvector, port `5432`                              |
| `ollama`      | Serveur Ollama (embedding + génération), port `11434`                        |
| `ollama-pull` | Tire les deux modèles une fois, puis s'arrête (l'API attend qu'il finisse)    |
| `api`         | API REST mira, port `8080`, migrations appliquées au démarrage               |

Le premier démarrage télécharge l'image Ollama et les deux modèles (`nomic-embed-text` ~300 Mo + `qwen2.5:1.5b-instruct` ~1 Go) : peut prendre plusieurs minutes. Les démarrages suivants réutilisent le volume `ollama_data` et sont quasi instantanés.

Une fois `docker compose up` prêt :

```sh
go build -o mira.exe ./cmd/mira
.\mira.exe add "titre" "contenu"
.\mira.exe list
.\mira.exe search <query>
.\mira.exe help
```

### Interface web

Une interface web minimale  est servie directement par l'API sur **http://localhost:8080/app/** : création de note, liste paginée, recherche, suppression, avec suivi en direct du statut d'enrichissement.

---

## CLI

La CLI est un client HTTP de l'API (`internal/apiclient`).

Base URL configurable via `MIRA_API_URL` (défaut `http://localhost:8080`).

Si l'API n'est pas joignable, la CLI affiche une erreur explicite plutôt qu'une erreur réseau brute (`impossible de joindre l'API sur ... — vérifie qu'elle est démarrée`).

---

## API REST

### Configuration

Variables lues depuis `.env` (voir `.env.example`) ou l'environnement (qui a priorité) :

| Variable                | Défaut               | Rôle                                              |
|--------------------------|----------------------|----------------------------------------------------|
| `ADDR`                   | `:8080`               | Adresse d'écoute HTTP                              |
| `DATABASE_URL`           | *(requis)*             | DSN PostgreSQL (fixé par docker-compose pour `api`) |
| `SEED`                   | `true`                | Insère 3 notes de démonstration au démarrage        |
| `OLLAMA_URL`              | `http://localhost:11434` | Serveur Ollama (embedding + génération)          |
| `OLLAMA_EMBED_MODEL`      | `nomic-embed-text`    | Modèle d'embedding Ollama                          |
| `OLLAMA_GEN_MODEL`        | `qwen2.5:1.5b-instruct` | Modèle génératif Ollama (tags/résumé/score)      |
| `ENRICHMENT_WORKERS`      | `4`                    | Taille du pool de workers d'enrichissement          |
| `ENRICHMENT_QUEUE_SIZE`   | `256`                  | Capacité du channel de jobs (au-delà : job abandonné) |
| `ENRICHMENT_TIMEOUT`      | `30s`                  | Timeout par job (génération LLM sur CPU = plus lente) |

### Routes

| Méthode | Chemin                   | Description                                   |
|---------|--------------------------|------------------------------------------------|
| POST    | `/api/v1/notes`          | Créer une note (déclenche l'enrichissement)     |
| GET     | `/api/v1/notes`          | Lister (paginé, plus récentes en premier)       |
| GET     | `/api/v1/notes/{id}`     | Récupérer par ID                                |
| PATCH   | `/api/v1/notes/{id}`     | Mise à jour partielle (déclenche l'enrichissement)|
| DELETE  | `/api/v1/notes/{id}`     | Supprimer                                       |
| GET     | `/api/v1/search?q=...&limit=...` | Recherche hybride (texte intégral + vecteur), `limit` optionnel (défaut 20, max 100) |
| GET     | `/docs/openapi.yaml`     | Schéma OpenAPI 3.1                              |
| GET     | `/docs/`                 | Swagger UI (interface de test)                  |

### Pagination

`GET /api/v1/notes?limit=10&offset=0`

- `limit` : nombre d'éléments par page (1–100, défaut 10)
- `offset` : index de départ (défaut 0)

### Enveloppe de réponse

**Succès**
```json
{
  "data": { ... },
  "meta": {
    "request_id": "abc123",
    "timestamp": "2026-07-13T10:00:00Z"
  }
}
```

**Liste paginée**
```json
{
  "data": [ ... ],
  "meta": {
    "request_id": "abc123",
    "timestamp": "2026-07-13T10:00:00Z",
    "total": 42,
    "limit": 10,
    "offset": 0
  }
}
```

**Erreur**
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "note introuvable"
  },
  "meta": {
    "request_id": "abc123",
    "timestamp": "2026-07-13T10:00:00Z"
  }
}
```

### Codes d'erreur

| Code              | HTTP | Situation                        |
|-------------------|------|----------------------------------|
| `INVALID_JSON`    | 400  | Corps JSON malformé              |
| `MISSING_PARAM`   | 400  | Paramètre obligatoire absent     |
| `VALIDATION_ERROR`| 422  | Champ invalide ou manquant       |
| `NOT_FOUND`       | 404  | ID inexistant                    |
| `INTERNAL_ERROR`  | 500  | Erreur serveur inattendue        |
| `TIMEOUT`         | 503  | Requête trop longue (> 10s)      |

---

## Serveur MCP

`cmd/mira-mcp` expose la mémoire mira à un agent IA (Claude Code, Claude Desktop, tout
hôte compatible [Model Context Protocol](https://modelcontextprotocol.io)) via un serveur
**MCP en transport stdio** (JSON-RPC 2.0), construit avec
[`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).

Comme la CLI, il passe systématiquement par l'API HTTP (`internal/apiclient`), jamais par
le store en direct : c'est ce qui garantit que chaque note créée par un agent déclenche
bien l'enrichissement automatique. **L'API doit donc être démarrée avant d'utiliser le
serveur MCP.**

### Tools exposés

| Tool | Paramètres | Rôle |
|---|---|---|
| `search_notes` | `query` (requis), `limit` (défaut 10, max 50) | recherche hybride plein texte + vectorielle |
| `get_note` | `id` (requis) | note complète (contenu, tags, résumé, statut d'enrichissement) |
| `add_note` | `title`, `content` (requis), `tags` (optionnel) | crée une note (enrichissement asynchrone) |
| `list_recent_notes` | `limit` (défaut 10, max 100) | dernières notes créées |

`search_notes` et `list_recent_notes` retournent des résumés légers (extrait, tags,
statut) plutôt que le contenu complet ; utiliser `get_note` avec l'`id` retourné pour
récupérer le contenu intégral d'une note.

### Build

```sh
go build -o mira-mcp.exe ./cmd/mira-mcp
```

(ou `go run ./cmd/mira-mcp` en développement).

### Configuration

Deux façons complémentaires de configurer le serveur, indépendantes du client MCP utilisé
pour le lancer :

1. **Fichier `cmd/mira-mcp/config.json`** (copier `config.example.json`, ignoré par git —
   peut contenir une clé) :
   ```json
   {
     "api_url": "http://localhost:8080",
     "api_key": ""
   }
   ```
2. **Variables d'environnement** — ont toujours priorité sur le fichier :

   | Variable | Défaut | Rôle |
   |---|---|---|
   | `MIRA_API_URL` | `http://localhost:8080` | Base URL de l'API mira |
   | `MIRA_API_KEY` | *(vide)* | Envoyée en en-tête `Authorization: Bearer ...` sur chaque appel API (l'API mira ne vérifie pas encore d'authentification) |
   | `MIRA_MCP_CONFIG` | `config.json` | Chemin du fichier de configuration JSON |

Le fichier `config.json` rend le serveur utilisable tel quel par n'importe quel hôte MCP
(pas seulement Claude Code) : il suffit de lancer le même binaire avec le même fichier à
côté, sans rien connaître du format d'enregistrement propre à un client particulier.

### Enregistrement dans Claude Code

**Option A — `.mcp.json` au niveau du projet** (fourni à la racine du repo, détecté
automatiquement) :

```json
{
  "mcpServers": {
    "mira": {
      "type": "stdio",
      "command": "go",
      "args": ["run", "./cmd/mira-mcp"],
      "env": { "MIRA_API_URL": "http://localhost:8080" }
    }
  }
}
```

**Option B — CLI `claude mcp add`** :

```sh
claude mcp add mira -- go run ./cmd/mira-mcp
```

### Enregistrement dans Claude Desktop

Claude Desktop n'hérite pas du répertoire de travail du projet : utiliser un binaire
compilé avec un chemin absolu, dans `claude_desktop_config.json` :

```json
{
  "mcpServers": {
    "mira": {
      "command": "/chemin/absolu/vers/mira-mcp.exe",
      "args": [],
      "env": { "MIRA_API_URL": "http://localhost:8080" }
    }
  }
}
```

### Exemples de prompts

- « Retrouve ma note sur les channels Go et ajoute une note résumant ce qu'on vient de faire. »
- « Quelles sont mes 5 dernières notes ? »
- « Cherche mes notes qui parlent de pgvector. »
- « Montre-moi le contenu complet de la note `<id>`. »

---

## Enrichissement automatique

Chaque `POST` ou `PATCH` sur une note :

1. écrit la note en base PostgreSQL de façon **synchrone** — l'API répond immédiatement, sans attendre l'enrichissement ;
2. publie un job (`note_id`) dans un **channel interne** ;
3. un **pool de workers borné** (`ENRICHMENT_WORKERS`) consomme les jobs et calcule, pour chaque note, via deux modèles Ollama locaux : tags additionnels, résumé court et score (modèle génératif `OLLAMA_GEN_MODEL`, réponse JSON structurée), et embedding vectoriel (modèle d'embedding `OLLAMA_EMBED_MODEL`) ;
4. chaque job a un **timeout** (`ENRICHMENT_TIMEOUT`) appliqué via `context.WithTimeout` ;
5. le résultat est écrit en base (`enrichment_status` passe à `done`), ou la note est marquée `failed` en cas d'erreur/timeout.

Si le channel de jobs est plein (charge trop importante), le job est abandonné et loggé — la note reste `pending` (pas de retry dans le scope de ce projet).

`enrichment_status` vaut `pending`, `done` ou `failed` — visible sur chaque note retournée par l'API.

---

## Recherche hybride

`GET /api/v1/search?q=...` combine :

- une recherche **plein texte** PostgreSQL (`tsvector` généré + index GIN, config `french`) ;
- une **similarité vectorielle** (`pgvector`, index HNSW, distance cosinus) entre l'embedding de la requête et les embeddings des notes déjà enrichies.

Une note apparaît si elle matche le texte, ou si elle est sémantiquement proche de la requête. Si le service d'embeddings (Ollama) est indisponible, la recherche se replie automatiquement sur le plein texte seul.

Le paramètre optionnel `limit` (1–100, défaut 20) borne le nombre de résultats, ex. `GET /api/v1/search?q=go&limit=5`.

---

### Exemples curl

**Créer une note**
```sh
curl -s -X POST http://localhost:8080/api/v1/notes \
  -H "Content-Type: application/json" \
  -d '{"title":"Go","content":"Un langage compilé et typé statiquement","tags":["go","dev"]}' | jq
```

**Lister les notes**
```sh
curl -s "http://localhost:8080/api/v1/notes?limit=5&offset=0" | jq
```

**Récupérer une note**
```sh
curl -s http://localhost:8080/api/v1/notes/<id> | jq
```

**Mettre à jour partiellement**
```sh
curl -s -X PATCH http://localhost:8080/api/v1/notes/<id> \
  -H "Content-Type: application/json" \
  -d '{"title":"Nouveau titre"}' | jq
```

**Supprimer**
```sh
curl -s -X DELETE http://localhost:8080/api/v1/notes/<id>
# → 204 No Content
```

**Rechercher**
```sh
curl -s "http://localhost:8080/api/v1/search?q=compilé" | jq
```

---

## Tests

```sh
go test ./...
```

Tests unitaires uniquement (handlers HTTP avec un store en mémoire, CLI avec une fausse API `httptest`) : aucune dépendance à Postgres/Ollama pour `go test ./...`.

## Structure

```
mira/
├── cmd/
│   ├── api/                 # point d'entrée du serveur HTTP
│   ├── mira/                 # point d'entrée de la CLI (client HTTP de l'API)
│   └── mira-mcp/             # serveur MCP (stdio) : expose les notes à un agent IA
├── internal/
│   ├── config/              # chargeur .env minimal
│   ├── core/                # modèle métier (Note, inputs, validation, EnrichmentResult)
│   ├── db/                  # pool pgx + migrations SQL embarquées (golang-migrate)
│   ├── store/                # interface Store + fake mémoire (tests) + seed
│   │   └── postgres/         # implémentation pgx : notes, recherche hybride, enrichissement
│   ├── enrichment/           # Enricher heuristique, OllamaEmbedder, pool de workers
│   ├── apiclient/             # client HTTP utilisé par la CLI
│   └── http/
│       ├── handlers/          # handlers HTTP + tests
│       ├── middleware/         # requestID, logging slog, recovery, timeout
│       ├── response/            # enveloppe JSON stable
│       └── router.go            # montage des routes
├── docs/
│   ├── openapi.yaml           # schéma OpenAPI 3.1
│   └── index.html              # Swagger UI (servi sur /docs/)
├── web/
│   ├── index.html               # interface web statique (servie sur /app/)
│   ├── app.js                    # logique (fetch API, rendu, polling du statut d'enrichissement)
│   └── styles.css                # styles
├── Dockerfile                  # build multi-stage de l'API
├── docker-compose.yml           # db (pgvector) + ollama + api
├── .env.example                  # variables disponibles
└── .mcp.json                      # enregistrement du serveur MCP pour Claude Code
```
