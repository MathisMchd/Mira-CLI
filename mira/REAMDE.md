# Mira

Application de prise de notes en ligne de commande. Les notes sont persistées dans `~/.mira/notes.jsonl`.

## Installation

```cmd
go build -o mira.exe .
```

## Utilisation

```cmd
mira add "titre" "contenu"    # ajouter une note
mira list                     # lister les 10 dernières notes
mira search <query>           # rechercher dans les titres et contenus
mira help                     # afficher l'aide
```

### Exemples

```cmd
mira add "Go" "Un langage compilé et typé statiquement"
mira add "Python" "Un langage interprété et dynamique"

mira list
# - Go: Un langage compilé et typé statiquement
# - Python: Un langage interprété et dynamique

mira search compilé
# - Go: Un langage compilé et typé statiquement
```

## Tests

```cmd
go test ./...
```
