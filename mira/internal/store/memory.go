package store

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"mira/internal/core"
)

// MemoryStore est une implémentation en mémoire de Store. Elle n'est plus
// utilisée en production (voir internal/store/postgres) — elle sert de
// double de test léger pour les handlers HTTP, sans dépendance à une base
// PostgreSQL réelle.
type MemoryStore struct {
	mu    sync.RWMutex
	notes map[string]*core.Note
	order []string
}

func NewMemory() *MemoryStore {
	return &MemoryStore{notes: make(map[string]*core.Note)}
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (s *MemoryStore) Create(ctx context.Context, input core.CreateNoteInput) (*core.Note, error) {
	tags := input.Tags
	if tags == nil {
		tags = []string{}
	}
	now := time.Now().UTC()
	n := &core.Note{
		ID:               newID(),
		Title:            strings.TrimSpace(input.Title),
		Content:          strings.TrimSpace(input.Content),
		Tags:             tags,
		EnrichmentStatus: core.EnrichmentPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.mu.Lock()
	s.notes[n.ID] = n
	s.order = append(s.order, n.ID)
	s.mu.Unlock()
	cp := *n
	return &cp, nil
}

func (s *MemoryStore) GetByID(ctx context.Context, id string) (*core.Note, error) {
	s.mu.RLock()
	n, ok := s.notes[id]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound{ID: id}
	}
	cp := *n
	return &cp, nil
}

// List retourne les notes les plus récentes en premier, comme le
// repository Postgres (ORDER BY created_at DESC).
func (s *MemoryStore) List(ctx context.Context, limit, offset int) ([]*core.Note, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.order)
	if offset >= total {
		return []*core.Note{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	out := make([]*core.Note, 0, end-offset)
	for j := offset; j < end; j++ {
		cp := *s.notes[s.order[total-1-j]]
		out = append(out, &cp)
	}
	return out, total, nil
}

func (s *MemoryStore) Patch(ctx context.Context, id string, input core.PatchNoteInput) (*core.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, ok := s.notes[id]
	if !ok {
		return nil, ErrNotFound{ID: id}
	}
	if input.Title != nil {
		n.Title = strings.TrimSpace(*input.Title)
	}
	if input.Content != nil {
		n.Content = strings.TrimSpace(*input.Content)
	}
	if input.Tags != nil {
		n.Tags = input.Tags
	}
	n.UpdatedAt = time.Now().UTC()
	cp := *n
	return &cp, nil
}

func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.notes[id]; !ok {
		return ErrNotFound{ID: id}
	}
	delete(s.notes, id)
	for i, oid := range s.order {
		if oid == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return nil
}

// Search fait une recherche naïve par sous-chaîne sur titre+contenu.
// queryEmbedding est ignoré : ce fake ne simule pas la similarité vectorielle.
func (s *MemoryStore) Search(ctx context.Context, query string, queryEmbedding []float32, limit int) ([]*core.Note, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*core.Note, 0)
	for _, id := range s.order {
		n := s.notes[id]
		if strings.Contains(strings.ToLower(n.Title), q) ||
			strings.Contains(strings.ToLower(n.Content), q) {
			cp := *n
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (s *MemoryStore) SaveEnrichment(ctx context.Context, noteID string, result core.EnrichmentResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, ok := s.notes[noteID]
	if !ok {
		return ErrNotFound{ID: noteID}
	}
	n.Tags = result.Tags
	n.Summary = result.Summary
	n.Score = result.Score
	n.EnrichmentStatus = core.EnrichmentDone
	n.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *MemoryStore) MarkEnrichmentFailed(ctx context.Context, noteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, ok := s.notes[noteID]
	if !ok {
		return ErrNotFound{ID: noteID}
	}
	n.EnrichmentStatus = core.EnrichmentFailed
	n.UpdatedAt = time.Now().UTC()
	return nil
}
