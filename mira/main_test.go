package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"mira/internal/notes"
)

type fakeStore struct {
	saved []*notes.Note
	err   error
}

func (f *fakeStore) Save(n *notes.Note) error {
	if f.err != nil {
		return f.err
	}
	f.saved = append(f.saved, n)
	return nil
}

func (f *fakeStore) All() ([]*notes.Note, error) {
	return f.saved, f.err
}

func capture(args []string, store notes.NoteStore) (string, int) {
	var buf bytes.Buffer
	code := run(args, store, &buf)
	return buf.String(), code
}

func TestAdd_valid(t *testing.T) {
	store := &fakeStore{}
	out, code := capture([]string{"mira", "add", "Go", "Un langage compilé"}, store)
	if code != 0 {
		t.Fatalf("attendu code 0, got %d", code)
	}
	if !strings.Contains(out, "Note ajoutée.") {
		t.Errorf("attendu confirmation, got %q", out)
	}
	if len(store.saved) != 1 || store.saved[0].Title != "Go" {
		t.Errorf("note non sauvegardée correctement")
	}
}

func TestAdd_missingArgs(t *testing.T) {
	_, code := capture([]string{"mira", "add", "Go"}, &fakeStore{})
	if code != 1 {
		t.Errorf("attendu code 1 pour args manquants, got %d", code)
	}
}

func TestAdd_storeError(t *testing.T) {
	store := &fakeStore{err: errors.New("disk full")}
	out, code := capture([]string{"mira", "add", "Go", "contenu"}, store)
	if code != 1 {
		t.Errorf("attendu code 1 en cas d'erreur store, got %d", code)
	}
	if !strings.Contains(out, "disk full") {
		t.Errorf("attendu message d'erreur, got %q", out)
	}
}

// --- list ---

func TestList_empty(t *testing.T) {
	out, code := capture([]string{"mira", "list"}, &fakeStore{})
	if code != 0 {
		t.Fatalf("attendu code 0, got %d", code)
	}
	if !strings.Contains(out, "Aucune note.") {
		t.Errorf("attendu 'Aucune note.', got %q", out)
	}
}

func TestList_withNotes(t *testing.T) {
	store := &fakeStore{saved: []*notes.Note{
		{Title: "A", Content: "alpha"},
		{Title: "B", Content: "beta"},
	}}
	out, _ := capture([]string{"mira", "list"}, store)
	if !strings.Contains(out, "A") || !strings.Contains(out, "B") {
		t.Errorf("attendu les deux titres, got %q", out)
	}
}

func TestList_storeError(t *testing.T) {
	store := &fakeStore{err: errors.New("io error")}
	_, code := capture([]string{"mira", "list"}, store)
	if code != 1 {
		t.Errorf("attendu code 1 en cas d'erreur store, got %d", code)
	}
}

func TestSearch_found(t *testing.T) {
	store := &fakeStore{saved: []*notes.Note{
		{Title: "Go", Content: "Un langage compilé"},
		{Title: "Python", Content: "Un langage interprété"},
	}}
	out, code := capture([]string{"mira", "search", "compilé"}, store)
	if code != 0 {
		t.Fatalf("attendu code 0, got %d", code)
	}
	if !strings.Contains(out, "Go") {
		t.Errorf("attendu 'Go' dans les résultats, got %q", out)
	}
	if strings.Contains(out, "Python") {
		t.Errorf("'Python' ne devrait pas apparaître, got %q", out)
	}
}

func TestSearch_notFound(t *testing.T) {
	out, code := capture([]string{"mira", "search", "rien"}, &fakeStore{})
	if code != 0 {
		t.Fatalf("attendu code 0, got %d", code)
	}
	if !strings.Contains(out, "Aucun résultat.") {
		t.Errorf("attendu 'Aucun résultat.', got %q", out)
	}
}

func TestSearch_missingQuery(t *testing.T) {
	_, code := capture([]string{"mira", "search"}, &fakeStore{})
	if code != 1 {
		t.Errorf("attendu code 1 pour query manquante, got %d", code)
	}
}

func TestSearch_multiWordQuery(t *testing.T) {
	store := &fakeStore{saved: []*notes.Note{
		{Title: "Note", Content: "bonjour le monde"},
	}}
	// args[2:] sont joints par espace → "bonjour le" doit être une sous-chaîne exacte
	out, _ := capture([]string{"mira", "search", "bonjour", "le"}, store)
	if !strings.Contains(out, "Note") {
		t.Errorf("attendu résultat pour requête multi-mots, got %q", out)
	}
}

func TestHelp(t *testing.T) {
	out, code := capture([]string{"mira", "help"}, &fakeStore{})
	if code != 0 {
		t.Fatalf("attendu code 0, got %d", code)
	}
	if !strings.Contains(out, "Usage:") {
		t.Errorf("attendu l'aide, got %q", out)
	}
}

func TestNoArgs(t *testing.T) {
	out, _ := capture([]string{"mira"}, &fakeStore{})
	if !strings.Contains(out, "Usage:") {
		t.Errorf("attendu l'aide sans args, got %q", out)
	}
}

func TestUnknownCommand(t *testing.T) {
	out, _ := capture([]string{"mira", "inconnu"}, &fakeStore{})
	if !strings.Contains(out, "Usage:") {
		t.Errorf("attendu l'aide pour commande inconnue, got %q", out)
	}
}
