// Package db fournit la connexion PostgreSQL (pool pgx) et l'application des migrations.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
)

// NewPool crée un pool de connexions pgx, enregistre le type pgvector "vector"
// sur chaque connexion (nécessaire pour encoder/décoder pgvector.Vector en
// binaire natif) et vérifie la connectivité. Les migrations doivent avoir été
// appliquées au préalable (extension "vector" déjà créée).
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("configuration du pool pgx: %w", err)
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("création du pool pgx: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connexion à la base impossible: %w", err)
	}
	return pool, nil
}
