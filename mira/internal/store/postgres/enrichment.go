package postgres

import (
	"context"
	"fmt"

	"github.com/pgvector/pgvector-go"

	"mira/internal/core"
	"mira/internal/store"
)

// fallbackEmbeddingModel est utilisé si l'Enricher n'a pas renseigné
// EnrichmentResult.Model (ne devrait pas arriver avec OllamaEmbedder).
const fallbackEmbeddingModel = "unknown"

// SaveEnrichment écrit le résultat d'un job d'enrichissement : résumé,
// score, statut "done", tags (remplacés par la liste déjà fusionnée par
// l'Enricher) et embedding (upsert). Le tout dans une transaction.
func (r *Repository) SaveEnrichment(ctx context.Context, noteID string, result core.EnrichmentResult) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE notes
		SET summary = $2, score = $3, enrichment_status = 'done', updated_at = now()
		WHERE id = $1::uuid
	`, noteID, result.Summary, result.Score)
	if err != nil {
		return fmt.Errorf("update note enrichment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound{ID: noteID}
	}

	if err := replaceTags(ctx, tx, noteID, dedupTags(result.Tags)); err != nil {
		return err
	}

	if len(result.Embedding) > 0 {
		model := result.Model
		if model == "" {
			model = fallbackEmbeddingModel
		}
		vec := pgvector.NewVector(result.Embedding)
		if _, err := tx.Exec(ctx, `
			INSERT INTO note_embeddings (note_id, embedding, model)
			VALUES ($1::uuid, $2, $3)
			ON CONFLICT (note_id) DO UPDATE
			SET embedding = EXCLUDED.embedding, model = EXCLUDED.model, created_at = now()
		`, noteID, vec, model); err != nil {
			return fmt.Errorf("upsert embedding: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// MarkEnrichmentFailed marque une note comme ayant échoué son enrichissement
// (timeout ou erreur du worker). La note reste consultable avec son contenu
// original ; seul enrichment_status change.
func (r *Repository) MarkEnrichmentFailed(ctx context.Context, noteID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notes SET enrichment_status = 'failed', updated_at = now() WHERE id = $1::uuid
	`, noteID)
	if err != nil {
		if isInvalidUUID(err) {
			return store.ErrNotFound{ID: noteID}
		}
		return fmt.Errorf("mark enrichment failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound{ID: noteID}
	}
	return nil
}
