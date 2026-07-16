# Mira Project

Dépôt de travail pour la formation Go (ESGI). Il regroupe le projet fil rouge **Mira** ainsi que les exercices réalisés en parallèle.


[Vidéo de présentation du projet](https://youtu.be/1-CJCLDuaTk)
A savoir que j'ai oublié de présenté la recherche vectorielle :/)
Readme du projet spéicifique mira avec l'API, le CLI et le MCP [mira/README.md](mira/README.md)

## Mira — présentation

**Mira, c'est un carnet de notes qui comprend ce que vous écrivez.**

### Le problème

Un outil de prise de notes classique se contente de stocker du texte. Pour retrouver une information plus tard, il faut se souvenir des mots exacts utilisés, et rien n'aide à organiser ou hiérarchiser ce qu'on a noté.

### Ce que fait Mira

On écrit une note, comme dans n'importe quel bloc-notes — un titre, un contenu. Mira s'occupe automatiquement du reste en arrière-plan :

- **Il résume** la note en une phrase
- **Il l'étiquette** avec des mots-clés pertinents, sans effort de classement de la part de l'utilisateur
- **Il évalue la richesse** du contenu (un score de qualité)
- **Il comprend le sens** de la note, pas seulement les mots qu'elle contient

Résultat : la recherche fonctionne même quand on ne se souvient pas des mots exacts. Par exemple, une note « RDV chez le dentiste jeudi 14h, plombage à refaire » peut être retrouvée en cherchant simplement « douleur aux dents » — parce que Mira comprend que les deux idées sont liées, même si aucun mot n'est commun.

Tout cela tourne en local, sans dépendre d'un service payant tiers ni envoyer les notes à un cloud externe : l'intelligence artificielle est auto-hébergée.


## Structure

```
mira-project/
├── mira/            # Projet principal (fil rouge) — module Go "mira" contenant l'api, le mcp et cli
├── go-warmup/       # Exercices de prise en main de Go — module Go "go-warmup"
└── tp-goroutines/   # TP sur les goroutines, channels et sync — module Go "tp-goroutines"
```

Chaque dossier est un module Go indépendant (son propre `go.mod`) : les commandes `go build` / `go run` / `go test` doivent être lancées depuis le dossier du module concerné.

## mira/ — projet principal

Application de prise de notes, disponible en **CLI** et en **API REST** (routing, middlewares, store en mémoire, recherche, doc OpenAPI/Swagger). C'est le fil rouge de la formation : voir [mira/README.md](mira/README.md) pour l'installation, l'usage et la documentation des routes.

```sh
cd mira
go build -o mira.exe .
go run ./cmd/api
go test ./...
```

# Autre
## go-warmup/ — exercices de prise en main

Petits exercices d'échauffement Go : types de base, tri de tags, lecture/écriture JSON, interface de stockage de notes en mémoire.


## tp-goroutines/ — TP concurrence

Exercices sur les goroutines : exécution séquentielle vs concurrente, `sync.WaitGroup`, `channel`, pattern worker pool, `sync.Mutex` pour protéger un accès concurrent.
