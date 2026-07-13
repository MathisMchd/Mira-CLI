package main

import (
	"errors"
	"testing"
)

func newStore() *MemoryStore {
	return &MemoryStore{notes: make(map[string]*Note)}
}

func TestSave_valid(t *testing.T) {
	store := newStore()
	note := &Note{Title: "Go", Content: "Un langage compilé"}
	if err := store.Save(note); err != nil {
		t.Errorf("attendu nil, got %v", err)
	}
}

func TestSave_emptyTitle(t *testing.T) {
	store := newStore()
	note := &Note{Title: "", Content: "Contenu sans titre"}
	if err := store.Save(note); !errors.Is(err, ErrValidation) {
		t.Errorf("attendu ErrValidation, got %v", err)
	}
}

func TestSave_duplicate(t *testing.T) {
	store := newStore()
	note := &Note{Title: "Go", Content: "Premier"}
	store.Save(note)
	if err := store.Save(&Note{Title: "Go", Content: "Second"}); !errors.Is(err, ErrDuplicate) {
		t.Errorf("attendu ErrDuplicate, got %v", err)
	}
}

func TestGet_notFound(t *testing.T) {
	store := newStore()
	if _, err := store.Get("inexistant"); !errors.Is(err, ErrNotFound) {
		t.Errorf("attendu ErrNotFound, got %v", err)
	}
}
