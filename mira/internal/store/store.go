package store

import (
	"context"
	"errors"
	"fmt"

	"mira/internal/core"
)

type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("note not found: %s", e.ID) }

func IsNotFound(err error) bool {
	var e ErrNotFound
	return errors.As(err, &e)
}

// Store est le repository des notes. L'implémentation de production
// (internal/store/postgres) persiste en base PostgreSQL ; l'enrichissement
// automatique s'appuie sur SaveEnrichment / MarkEnrichmentFailed pour écrire
// ses résultats de façon asynchrone, indépendamment de la requête HTTP qui a
// créé ou modifié la note.
type Store interface {
	Create(ctx context.Context, input core.CreateNoteInput) (*core.Note, error)
	GetByID(ctx context.Context, id string) (*core.Note, error)
	List(ctx context.Context, limit, offset int) ([]*core.Note, int, error)
	Patch(ctx context.Context, id string, input core.PatchNoteInput) (*core.Note, error)
	Delete(ctx context.Context, id string) error

	// Search effectue une recherche hybride texte intégral + similarité
	// vectorielle. queryEmbedding peut être nil si l'appelant ne dispose pas
	// (encore) d'un embedding pour la requête, auquel cas seul le volet
	// full-text est utilisé.
	Search(ctx context.Context, query string, queryEmbedding []float32, limit int) ([]*core.Note, error)

	SaveEnrichment(ctx context.Context, noteID string, result core.EnrichmentResult) error
	MarkEnrichmentFailed(ctx context.Context, noteID string) error
}
