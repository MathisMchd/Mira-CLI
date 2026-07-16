package store

import (
	"context"
	"errors"
	"testing"

	"mira/internal/core"
)

func TestSeed_insertsNotesAndCallsOnCreated(t *testing.T) {
	s := NewMemory()
	var createdIDs []string

	if err := Seed(context.Background(), s, func(noteID string) {
		createdIDs = append(createdIDs, noteID)
	}); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	notes, total, err := s.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != len(seedNotes) {
		t.Errorf("total = %d, attendu %d", total, len(seedNotes))
	}
	if len(createdIDs) != len(seedNotes) {
		t.Errorf("onCreated appelé %d fois, attendu %d", len(createdIDs), len(seedNotes))
	}
	if len(notes) != len(seedNotes) {
		t.Errorf("notes insérées = %d, attendu %d", len(notes), len(seedNotes))
	}
}

func TestSeed_nilCallbackIsSafe(t *testing.T) {
	s := NewMemory()
	if err := Seed(context.Background(), s, nil); err != nil {
		t.Fatalf("Seed avec callback nil: %v", err)
	}
}

var errCreateBoom = errors.New("create boom")

// erroringStore n'implémente réellement que Create (qui échoue toujours) ;
// les autres méthodes de Store ne sont jamais appelées par Seed.
type erroringStore struct{ MemoryStore }

func (*erroringStore) Create(ctx context.Context, input core.CreateNoteInput) (*core.Note, error) {
	return nil, errCreateBoom
}

func TestSeed_propagatesCreateError(t *testing.T) {
	err := Seed(context.Background(), &erroringStore{}, nil)
	if !errors.Is(err, errCreateBoom) {
		t.Errorf("err = %v, attendu %v", err, errCreateBoom)
	}
}
