// Package postgres implémente store.Store au-dessus de PostgreSQL via pgx.
package postgres

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// invalidTextRepresentation est le SQLSTATE renvoyé par Postgres quand une
// chaîne fournie en paramètre n'est pas un UUID valide (ex: ID CLI erroné
// dans l'URL). On le traite comme "introuvable" plutôt que comme une erreur
// serveur.
const invalidTextRepresentation = "22P02"

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation
}

// dedupTags nettoie et déduplique une liste de tags en préservant l'ordre.
func dedupTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
