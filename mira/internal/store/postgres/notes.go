package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"mira/internal/core"
	"mira/internal/store"
)

// noteColumns qualifie les colonnes avec l'alias "n" — utilisé dans les
// SELECT avec jointures (GetByID, List, Search).
const noteColumns = `n.id::text, n.title, n.content, n.summary, n.score, n.enrichment_status, n.created_at, n.updated_at`

// noteReturningColumns est la même liste sans qualification, pour les
// clauses RETURNING des INSERT/UPDATE sur la seule table notes : il n'y a
// pas d'alias "n" déclaré dans ce contexte (pas de FROM/jointure).
const noteReturningColumns = `id::text, title, content, summary, score, enrichment_status, created_at, updated_at`

func scanNote(row pgx.Row, n *core.Note) error {
	return row.Scan(&n.ID, &n.Title, &n.Content, &n.Summary, &n.Score, &n.EnrichmentStatus, &n.CreatedAt, &n.UpdatedAt, &n.Tags)
}

func (r *Repository) Create(ctx context.Context, input core.CreateNoteInput) (*core.Note, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	title := strings.TrimSpace(input.Title)
	content := strings.TrimSpace(input.Content)

	var n core.Note
	err = tx.QueryRow(ctx, `
		INSERT INTO notes (title, content)
		VALUES ($1, $2)
		RETURNING `+noteReturningColumns, title, content,
	).Scan(&n.ID, &n.Title, &n.Content, &n.Summary, &n.Score, &n.EnrichmentStatus, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert note: %w", err)
	}

	tags := dedupTags(input.Tags)
	if err := replaceTags(ctx, tx, n.ID, tags); err != nil {
		return nil, err
	}
	n.Tags = tags

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &n, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*core.Note, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+noteColumns+`,
		       coalesce(array_agg(nt.tag ORDER BY nt.tag) FILTER (WHERE nt.tag IS NOT NULL), '{}')
		FROM notes n
		LEFT JOIN note_tags nt ON nt.note_id = n.id
		WHERE n.id = $1::uuid
		GROUP BY n.id
	`, id)

	var n core.Note
	if err := scanNote(row, &n); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return nil, store.ErrNotFound{ID: id}
		}
		return nil, fmt.Errorf("get note: %w", err)
	}
	return &n, nil
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]*core.Note, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM notes`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notes: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT `+noteColumns+`,
		       coalesce(array_agg(nt.tag ORDER BY nt.tag) FILTER (WHERE nt.tag IS NOT NULL), '{}')
		FROM notes n
		LEFT JOIN note_tags nt ON nt.note_id = n.id
		GROUP BY n.id
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	out := make([]*core.Note, 0)
	for rows.Next() {
		var n core.Note
		if err := scanNote(rows, &n); err != nil {
			return nil, 0, fmt.Errorf("scan note: %w", err)
		}
		out = append(out, &n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list notes: %w", err)
	}
	return out, total, nil
}

func (r *Repository) Patch(ctx context.Context, id string, input core.PatchNoteInput) (*core.Note, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	args := []any{id}
	setParts := []string{"updated_at = now()"}
	if input.Title != nil {
		args = append(args, strings.TrimSpace(*input.Title))
		setParts = append(setParts, fmt.Sprintf("title = $%d", len(args)))
	}
	if input.Content != nil {
		args = append(args, strings.TrimSpace(*input.Content))
		setParts = append(setParts, fmt.Sprintf("content = $%d", len(args)))
	}

	query := fmt.Sprintf(`UPDATE notes SET %s WHERE id = $1::uuid RETURNING %s`, strings.Join(setParts, ", "), noteReturningColumns)

	var n core.Note
	err = tx.QueryRow(ctx, query, args...).Scan(&n.ID, &n.Title, &n.Content, &n.Summary, &n.Score, &n.EnrichmentStatus, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return nil, store.ErrNotFound{ID: id}
		}
		return nil, fmt.Errorf("update note: %w", err)
	}

	if input.Tags != nil {
		tags := dedupTags(input.Tags)
		if err := replaceTags(ctx, tx, id, tags); err != nil {
			return nil, err
		}
		n.Tags = tags
	} else {
		tags, err := fetchTags(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		n.Tags = tags
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &n, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM notes WHERE id = $1::uuid`, id)
	if err != nil {
		if isInvalidUUID(err) {
			return store.ErrNotFound{ID: id}
		}
		return fmt.Errorf("delete note: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound{ID: id}
	}
	return nil
}

// replaceTags remplace intégralement les tags d'une note par la liste fournie.
func replaceTags(ctx context.Context, tx pgx.Tx, noteID string, tags []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM note_tags WHERE note_id = $1::uuid`, noteID); err != nil {
		return fmt.Errorf("clear tags: %w", err)
	}
	if len(tags) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, t := range tags {
		batch.Queue(`INSERT INTO note_tags (note_id, tag) VALUES ($1::uuid, $2) ON CONFLICT DO NOTHING`, noteID, t)
	}
	br := tx.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()
	for range tags {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert tags: %w", err)
		}
	}
	return nil
}

func fetchTags(ctx context.Context, tx pgx.Tx, noteID string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT tag FROM note_tags WHERE note_id = $1::uuid ORDER BY tag`, noteID)
	if err != nil {
		return nil, fmt.Errorf("fetch tags: %w", err)
	}
	defer rows.Close()

	tags := make([]string, 0)
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
