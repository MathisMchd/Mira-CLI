package store

import "mira/internal/core"

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
// Destiné à l'environnement de développement uniquement.
func Seed(s Store) error {
	for _, input := range seedNotes {
		if _, err := s.Create(input); err != nil {
			return err
		}
	}
	return nil
}
