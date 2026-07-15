package postgres

import (
	"context"
	"fmt"

	"github.com/pgvector/pgvector-go"

	"mira/internal/core"
)

// hybridVectorThreshold est le seuil de similarité cosinus (0..1) en dessous
// duquel une note n'est pas considérée comme un match sémantique si elle ne
// matche pas non plus le texte intégral. Calibré empiriquement sur
// nomic-embed-text : des textes courts sans rapport se situent déjà autour
// de 0.27-0.38 de similarité cosinus (bruit de fond du modèle), donc un
// seuil bas (ex. 0.15) laisse tout matcher. 0.5 sépare nettement ce bruit
// d'un vrai match sémantique (~0.6-0.7 observé).
const hybridVectorThreshold = 0.5

const defaultSearchLimit = 20

// Search combine recherche texte intégral (tsvector/GIN) et similarité
// vectorielle (pgvector) : une note est retournée si son contenu matche la
// requête au sens plein texte, OU si son embedding est suffisamment proche
// de queryEmbedding. Le classement final pondère les deux scores à parts
// égales. queryEmbedding peut être nil (recherche full-text seule).
func (r *Repository) Search(ctx context.Context, query string, queryEmbedding []float32, limit int) ([]*core.Note, error) {
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	var queryVec any
	if len(queryEmbedding) > 0 {
		queryVec = pgvector.NewVector(queryEmbedding)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT `+noteColumns+`,
		       coalesce(array_agg(nt.tag ORDER BY nt.tag) FILTER (WHERE nt.tag IS NOT NULL), '{}')
		FROM notes n
		LEFT JOIN note_tags nt ON nt.note_id = n.id
		LEFT JOIN note_embeddings ne ON ne.note_id = n.id
		WHERE n.search_vector @@ plainto_tsquery('french', $1)
		   OR (
		        $2::vector IS NOT NULL
		        AND ne.embedding IS NOT NULL
		        AND 1 - (ne.embedding <=> $2::vector) > $3
		      )
		GROUP BY n.id, ne.note_id
		ORDER BY (
		    0.5 * ts_rank(n.search_vector, plainto_tsquery('french', $1))
		    + 0.5 * CASE
		        WHEN $2::vector IS NULL OR ne.embedding IS NULL THEN 0
		        ELSE 1 - (ne.embedding <=> $2::vector)
		      END
		) DESC
		LIMIT $4
	`, query, queryVec, hybridVectorThreshold, limit)
	if err != nil {
		return nil, fmt.Errorf("search notes: %w", err)
	}
	defer rows.Close()

	out := make([]*core.Note, 0)
	for rows.Next() {
		var n core.Note
		if err := scanNote(rows, &n); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}
