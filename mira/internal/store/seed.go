package store

import (
	"context"

	"mira/internal/core"
)

var seedNotes = []core.CreateNoteInput{
	{
		Title:   "Bienvenue sur Mira",
		Content: "Ceci est une note de démonstration créée au démarrage de l'API.",
		Tags:    []string{"demo"},
	},
	{
		Title:   "Aide-mémoire Go",
		Content: "go run ./cmd/api pour lancer le serveur, go test ./... pour les tests.",
		Tags:    []string{"go", "dev"},
	},
	{
		Title:   "Idée de fonctionnalité",
		Content: "Ajouter un export CSV des notes.",
		Tags:    []string{"idée"},
	},
}

// Seed insère un jeu de notes de démonstration dans le store.
// Destiné à l'environnement de développement uniquement. Contrairement à un
// POST /api/v1/notes, cet appel passe directement par le repository — il ne
// déclenche donc pas l'enrichissement automatique de lui-même ; onCreated
// (typiquement dispatcher.Enqueue) permet à l'appelant de le faire quand même.
func Seed(ctx context.Context, s Store, onCreated func(noteID string)) error {
	for _, input := range seedNotes {
		n, err := s.Create(ctx, input)
		if err != nil {
			return err
		}
		if onCreated != nil {
			onCreated(n.ID)
		}
	}
	return nil
}
