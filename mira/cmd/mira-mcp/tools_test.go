package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira/internal/apiclient"
	"mira/internal/core"
)

// fakeMiraAPI simule le strict nécessaire de l'API REST mira (enveloppe
// JSON, routes notes/search) pour tester les handlers de tools MCP sans
// dépendance à une vraie instance Postgres/Ollama.
type fakeMiraAPI struct {
	mu    sync.Mutex
	notes []*core.Note
}

func (f *fakeMiraAPI) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
				CreatedAt:        time.Now().UTC(),
				UpdatedAt:        time.Now().UTC(),
			}
			f.notes = append(f.notes, n)
			f.mu.Unlock()
			writeEnvelope(w, http.StatusCreated, n)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/notes":
			f.mu.Lock()
			defer f.mu.Unlock()
			writeEnvelope(w, http.StatusOK, f.notes)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/notes/"):
			id := strings.TrimPrefix(r.URL.Path, "/api/v1/notes/")
			f.mu.Lock()
			defer f.mu.Unlock()
			for _, n := range f.notes {
				if n.ID == id {
					writeEnvelope(w, http.StatusOK, n)
					return
				}
			}
			writeErrEnvelope(w, http.StatusNotFound, "NOT_FOUND", "note introuvable")

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
			writeEnvelope(w, http.StatusOK, out)

		default:
			writeErrEnvelope(w, http.StatusNotFound, "NOT_FOUND", "route introuvable")
		}
	}
}

func writeEnvelope(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "meta": map[string]any{}})
}

func writeErrEnvelope(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": message},
		"meta":  map[string]any{},
	})
}

func newTestToolServer(t *testing.T, api *fakeMiraAPI) *toolServer {
	t.Helper()
	srv := httptest.NewServer(api.handler())
	t.Cleanup(srv.Close)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &toolServer{client: apiclient.New(srv.URL, ""), logger: logger, timeout: 2 * time.Second}
}

func TestSearchNotes(t *testing.T) {
	api := &fakeMiraAPI{}
	ts := newTestToolServer(t, api)
	ctx := context.Background()

	if _, err := ts.client.Create(ctx, core.CreateNoteInput{Title: "Go", Content: "channels et goroutines"}); err != nil {
		t.Fatalf("seed create: %v", err)
	}
	if _, err := ts.client.Create(ctx, core.CreateNoteInput{Title: "Cuisine", Content: "brioche maison"}); err != nil {
		t.Fatalf("seed create: %v", err)
	}

	_, out, err := ts.SearchNotes(ctx, nil, SearchNotesInput{Query: "goroutines"})
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}
	if len(out.Notes) != 1 || out.Notes[0].Title != "Go" {
		t.Errorf("résultats = %+v, attendu 1 note 'Go'", out.Notes)
	}
}

func TestSearchNotes_emptyQueryIsRejectedWithoutNetworkCall(t *testing.T) {
	api := &fakeMiraAPI{}
	ts := newTestToolServer(t, api)

	_, _, err := ts.SearchNotes(context.Background(), nil, SearchNotesInput{Query: "   "})
	if err == nil {
		t.Fatal("attendu une erreur pour une query vide")
	}
	if !strings.Contains(err.Error(), "query") {
		t.Errorf("message d'erreur = %q, attendu qu'il mentionne 'query'", err.Error())
	}
}

func TestGetNote(t *testing.T) {
	api := &fakeMiraAPI{}
	ts := newTestToolServer(t, api)
	ctx := context.Background()

	created, err := ts.client.Create(ctx, core.CreateNoteInput{Title: "Go", Content: "un langage"})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}

	_, out, err := ts.GetNote(ctx, nil, GetNoteInput{ID: created.ID})
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if out.ID != created.ID || out.Content != "un langage" {
		t.Errorf("out = %+v, attendu contenu complet de la note créée", out)
	}
}

func TestGetNote_emptyIDIsRejected(t *testing.T) {
	ts := newTestToolServer(t, &fakeMiraAPI{})
	_, _, err := ts.GetNote(context.Background(), nil, GetNoteInput{ID: " "})
	if err == nil {
		t.Fatal("attendu une erreur pour un id vide")
	}
}

func TestGetNote_notFoundPropagatesCleanError(t *testing.T) {
	ts := newTestToolServer(t, &fakeMiraAPI{})
	_, _, err := ts.GetNote(context.Background(), nil, GetNoteInput{ID: "does-not-exist"})
	if err == nil {
		t.Fatal("attendu une erreur pour un id inexistant")
	}
	if !strings.Contains(err.Error(), "introuvable") {
		t.Errorf("message = %q, attendu qu'il mentionne 'introuvable'", err.Error())
	}
}

func TestAddNote(t *testing.T) {
	ts := newTestToolServer(t, &fakeMiraAPI{})

	_, out, err := ts.AddNote(context.Background(), nil, AddNoteInput{Title: "Go", Content: "contenu", Tags: []string{"go"}})
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}
	if out.ID == "" || out.Title != "Go" {
		t.Errorf("out = %+v", out)
	}
	if out.EnrichmentStatus != core.EnrichmentPending {
		t.Errorf("enrichment_status = %q, attendu pending juste après création", out.EnrichmentStatus)
	}
}

func TestAddNote_validationErrorNoNetworkCall(t *testing.T) {
	api := &fakeMiraAPI{}
	ts := newTestToolServer(t, api)

	_, _, err := ts.AddNote(context.Background(), nil, AddNoteInput{Title: "", Content: ""})
	if err == nil {
		t.Fatal("attendu une erreur de validation")
	}
	if len(api.notes) != 0 {
		t.Error("la note invalide n'aurait pas dû atteindre l'API")
	}
}

func TestListRecentNotes(t *testing.T) {
	ts := newTestToolServer(t, &fakeMiraAPI{})
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := ts.client.Create(ctx, core.CreateNoteInput{Title: "n", Content: "c"}); err != nil {
			t.Fatalf("seed create: %v", err)
		}
	}

	_, out, err := ts.ListRecentNotes(ctx, nil, ListRecentNotesInput{})
	if err != nil {
		t.Fatalf("ListRecentNotes: %v", err)
	}
	if len(out.Notes) != 3 {
		t.Errorf("len(notes) = %d, attendu 3", len(out.Notes))
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name           string
		v, def, lo, hi int
		want           int
	}{
		{"absent -> défaut", 0, 10, 1, 50, 10},
		{"négatif -> défaut", -5, 10, 1, 50, 10},
		{"sous le minimum -> minimum", 0, 10, 5, 50, 10}, // v<=0 retombe sur def avant le clamp bas
		{"au-dessus du maximum -> maximum", 999, 10, 1, 50, 50},
		{"dans la plage -> inchangé", 25, 10, 1, 50, 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clamp(tt.v, tt.def, tt.lo, tt.hi); got != tt.want {
				t.Errorf("clamp(%d,%d,%d,%d) = %d, attendu %d", tt.v, tt.def, tt.lo, tt.hi, got, tt.want)
			}
		})
	}
}

func TestWithRecovery_catchesPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	panicking := func(ctx context.Context, req *mcp.CallToolRequest, in string) (*mcp.CallToolResult, string, error) {
		panic("boom")
	}
	wrapped := withRecovery(logger, "test_tool", panicking)

	_, _, err := wrapped(context.Background(), nil, "input")
	if err == nil {
		t.Fatal("attendu une erreur après panic, pas une propagation du panic")
	}
	if !strings.Contains(err.Error(), "test_tool") {
		t.Errorf("erreur = %q, attendu qu'elle mentionne le nom du tool", err.Error())
	}
}
