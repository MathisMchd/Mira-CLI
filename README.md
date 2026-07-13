# Mira

Application de prise de notes. Disponible en **CLI** et en **API REST**.

---

## CLI

Les notes sont persistées dans `~/.mira/notes.jsonl`.

```sh
go build -o mira.exe .
.\mira.exe add "titre" "contenu"
.\mira.exe list
.\mira.exe search <query>
.\mira.exe help
```

---

## API REST

Stockage en mémoire (remis à zéro au redémarrage).

### Démarrage

```sh
go run ./cmd/api
# ou avec une adresse personnalisée
ADDR=:9000 go run ./cmd/api
```

### Routes

| Méthode | Chemin                   | Description                     |
|---------|--------------------------|---------------------------------|
| POST    | `/api/v1/notes`          | Créer une note                  |
| GET     | `/api/v1/notes`          | Lister (paginé)                 |
| GET     | `/api/v1/notes/{id}`     | Récupérer par ID                |
| PATCH   | `/api/v1/notes/{id}`     | Mise à jour partielle           |
| DELETE  | `/api/v1/notes/{id}`     | Supprimer                       |
| GET     | `/api/v1/search?q=...`   | Recherche texte (titre+contenu) |
| GET     | `/docs/openapi.yaml`     | Schéma OpenAPI 3.1              |

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

## Structure

```
mira/
├── cmd/api/            # point d'entrée du serveur HTTP
├── internal/
│   ├── core/           # modèle métier (Note, inputs, validation)
│   ├── store/          # interface Store + implémentation mémoire
│   └── http/
│       ├── handlers/   # handlers HTTP + tests
│       ├── middleware/ # requestID, logging slog, recovery, timeout
│       ├── response/   # enveloppe JSON stable
│       └── router.go   # montage des routes
├── docs/
│   └── openapi.yaml    # schéma OpenAPI 3.1
└── internal/
    ├── notes/          # CLI : store JSONL
    └── search/         # CLI : recherche texte
```
