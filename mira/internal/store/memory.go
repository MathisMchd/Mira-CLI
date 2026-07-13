package store

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"mira/internal/core"
)

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

func (s *MemoryStore) Create(input core.CreateNoteInput) (*core.Note, error) {
	tags := input.Tags
	if tags == nil {
		tags = []string{}
	}
	now := time.Now().UTC()
	n := &core.Note{
		ID:        newID(),
		Title:     strings.TrimSpace(input.Title),
		Content:   strings.TrimSpace(input.Content),
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	s.notes[n.ID] = n
	s.order = append(s.order, n.ID)
	s.mu.Unlock()
	cp := *n
	return &cp, nil
}

func (s *MemoryStore) GetByID(id string) (*core.Note, error) {
	s.mu.RLock()
	n, ok := s.notes[id]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound{ID: id}
	}
	cp := *n
	return &cp, nil
}

func (s *MemoryStore) List(limit, offset int) ([]*core.Note, int, error) {
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
	slice := s.order[offset:end]
	out := make([]*core.Note, 0, len(slice))
	for _, id := range slice {
		cp := *s.notes[id]
		out = append(out, &cp)
	}
	return out, total, nil
}

func (s *MemoryStore) Patch(id string, input core.PatchNoteInput) (*core.Note, error) {
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

func (s *MemoryStore) Delete(id string) error {
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

func (s *MemoryStore) Search(query string) ([]*core.Note, error) {
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
