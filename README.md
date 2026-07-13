# Mira Project

Dépôt de travail pour la formation Go (ESGI). Il regroupe le projet fil rouge **Mira** ainsi que les exercices réalisés en parallèle.

## Structure

```
mira-project/
├── mira/            # Projet principal (fil rouge) — module Go "mira"
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

## go-warmup/ — exercices de prise en main

Petits exercices d'échauffement Go : types de base, tri de tags, lecture/écriture JSON, interface de stockage de notes en mémoire.

```sh
cd go-warmup
go run ./hello <votre prénom>
go run ./ex1
go run ./ex2
go run ./ex3
go run ./ex4-5
go test ./ex4-5
```

## tp-goroutines/ — TP concurrence

Exercices sur les goroutines : exécution séquentielle vs concurrente, `sync.WaitGroup`, `channel`, pattern worker pool, `sync.Mutex` pour protéger un accès concurrent.

```sh
cd tp-goroutines
go run ./ex1
go run ./ex2
go run ./ex3
go run ./ex4
go run ./ex5
go run ./ex6
```
