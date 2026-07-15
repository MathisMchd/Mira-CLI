package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"mira/internal/apiclient"
	"mira/internal/core"
)

// fakeAPI simule le strict minimum de l'API REST mira (enveloppe JSON,
// routes notes/search) pour tester la CLI sans dépendance à une vraie
// instance (Postgres/Ollama). La CLI ne parle plus qu'à cette API — plus de
// stockage local.
type fakeAPI struct {
	mu    sync.Mutex
	notes []*core.Note
	err   error // si non-nil, toutes les routes répondent 500
}

func (f *fakeAPI) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if f.err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", f.err.Error())
			return
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/notes":
			var input core.CreateNoteInput
			_ = json.NewDecoder(r.Body).Decode(&input)
			f.mu.Lock()
			n := &core.Note{
				ID:               fmt.Sprintf("id-%d", len(f.notes)+1),
				Title:            input.Title,
				Content:          input.Content,
				Tags:             input.Tags,
				EnrichmentStatus: core.EnrichmentPending,
			}
			f.notes = append(f.notes, n)
			f.mu.Unlock()
			writeData(w, http.StatusCreated, n)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/notes":
			f.mu.Lock()
			defer f.mu.Unlock()
			writeData(w, http.StatusOK, f.notes)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/search":
			q := strings.ToLower(r.URL.Query().Get("q"))
			f.mu.Lock()
			defer f.mu.Unlock()
			out := make([]*core.Note, 0)
			for _, n := range f.notes {
				if strings.Contains(strings.ToLower(n.Title), q) || strings.Contains(strings.ToLower(n.Content), q) {
					out = append(out, n)
				}
			}
			writeData(w, http.StatusOK, out)

		default:
			writeError(w, http.StatusNotFound, "NOT_FOUND", "route introuvable")
		}
	}
}

func writeData(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "meta": map[string]any{}})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": message},
		"meta":  map[string]any{},
	})
}

// newTestClient démarre une fausse API pour la durée du test.
func newTestClient(t *testing.T, api *fakeAPI) *apiclient.Client {
	t.Helper()
	srv := httptest.NewServer(api.handler())
	t.Cleanup(srv.Close)
	return apiclient.New(srv.URL)
}

func capture(args []string, client *apiclient.Client) (string, int) {
	var buf bytes.Buffer
	code := run(args, client, &buf)
	return buf.String(), code
}

func TestAdd_valid(t *testing.T) {
	api := &fakeAPI{}
	client := newTestClient(t, api)

	out, code := capture([]string{"mira", "add", "Go", "Un langage compilé"}, client)
	if code != 0 {
		t.Fatalf("attendu code 0, got %d", code)
	}
	if !strings.Contains(out, "Note ajoutée.") {
		t.Errorf("attendu confirmation, got %q", out)
	}
	if len(api.notes) != 1 || api.notes[0].Title != "Go" {
		t.Errorf("note non sauvegardée correctement")
	}
}

func TestAdd_missingArgs(t *testing.T) {
	_, code := capture([]string{"mira", "add", "Go"}, nil)
	if code != 1 {
		t.Errorf("attendu code 1 pour args manquants, got %d", code)
	}
}

func TestAdd_storeError(t *testing.T) {
	api := &fakeAPI{err: errors.New("disk full")}
	client := newTestClient(t, api)

	out, code := capture([]string{"mira", "add", "Go", "contenu"}, client)
	if code != 1 {
		t.Errorf("attendu code 1 en cas d'erreur API, got %d", code)
	}
	if !strings.Contains(out, "disk full") {
		t.Errorf("attendu message d'erreur, got %q", out)
	}
}

// --- list ---

func TestList_empty(t *testing.T) {
	client := newTestClient(t, &fakeAPI{})

	out, code := capture([]string{"mira", "list"}, client)
	if code != 0 {
		t.Fatalf("attendu code 0, got %d", code)
	}
	if !strings.Contains(out, "Aucune note.") {
		t.Errorf("attendu 'Aucune note.', got %q", out)
	}
}

func TestList_withNotes(t *testing.T) {
	api := &fakeAPI{notes: []*core.Note{
		{ID: "1", Title: "A", Content: "alpha"},
		{ID: "2", Title: "B", Content: "beta"},
	}}
	client := newTestClient(t, api)

	out, _ := capture([]string{"mira", "list"}, client)
	if !strings.Contains(out, "A") || !strings.Contains(out, "B") {
		t.Errorf("attendu les deux titres, got %q", out)
	}
}

func TestList_storeError(t *testing.T) {
	client := newTestClient(t, &fakeAPI{err: errors.New("io error")})

	_, code := capture([]string{"mira", "list"}, client)
	if code != 1 {
		t.Errorf("attendu code 1 en cas d'erreur API, got %d", code)
	}
}

func TestSearch_found(t *testing.T) {
	api := &fakeAPI{notes: []*core.Note{
		{ID: "1", Title: "Go", Content: "Un langage compilé"},
		{ID: "2", Title: "Python", Content: "Un langage interprété"},
	}}
	client := newTestClient(t, api)

	out, code := capture([]string{"mira", "search", "compilé"}, client)
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
	client := newTestClient(t, &fakeAPI{})

	out, code := capture([]string{"mira", "search", "rien"}, client)
	if code != 0 {
		t.Fatalf("attendu code 0, got %d", code)
	}
	if !strings.Contains(out, "Aucun résultat.") {
		t.Errorf("attendu 'Aucun résultat.', got %q", out)
	}
}

func TestSearch_missingQuery(t *testing.T) {
	_, code := capture([]string{"mira", "search"}, nil)
	if code != 1 {
		t.Errorf("attendu code 1 pour query manquante, got %d", code)
	}
}

func TestSearch_multiWordQuery(t *testing.T) {
	api := &fakeAPI{notes: []*core.Note{
		{ID: "1", Title: "Note", Content: "bonjour le monde"},
	}}
	client := newTestClient(t, api)

	// args[2:] sont joints par espace → "bonjour le" doit être une sous-chaîne exacte
	out, _ := capture([]string{"mira", "search", "bonjour", "le"}, client)
	if !strings.Contains(out, "Note") {
		t.Errorf("attendu résultat pour requête multi-mots, got %q", out)
	}
}

func TestHelp(t *testing.T) {
	out, code := capture([]string{"mira", "help"}, nil)
	if code != 0 {
		t.Fatalf("attendu code 0, got %d", code)
	}
	if !strings.Contains(out, "Usage:") {
		t.Errorf("attendu l'aide, got %q", out)
	}
}

func TestNoArgs(t *testing.T) {
	out, _ := capture([]string{"mira"}, nil)
	if !strings.Contains(out, "Usage:") {
		t.Errorf("attendu l'aide sans args, got %q", out)
	}
}

func TestUnknownCommand(t *testing.T) {
	out, _ := capture([]string{"mira", "inconnu"}, nil)
	if !strings.Contains(out, "Usage:") {
		t.Errorf("attendu l'aide pour commande inconnue, got %q", out)
	}
}
