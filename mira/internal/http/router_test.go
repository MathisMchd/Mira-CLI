package apihttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mira/internal/core"
	"mira/internal/enrichment"
	apihttp "mira/internal/http"
	"mira/internal/store"
)

type fakeEmbedder struct{}

func (fakeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}
func (fakeEmbedder) Name() string { return "fake" }

type fakeEnricher struct{}

func (fakeEnricher) Enrich(ctx context.Context, note core.Note) (core.EnrichmentResult, error) {
	return core.EnrichmentResult{Tags: note.Tags, Summary: note.Content, Score: 0.5}, nil
}

func newTestRouter() http.Handler {
	s := store.NewMemory()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dispatcher := enrichment.NewDispatcher(s, fakeEnricher{}, logger, 1, 10, time.Second)
	return apihttp.NewRouter(s, dispatcher, fakeEmbedder{}, logger)
}

// TestRouter_endToEndCreateListGet exerce la chaîne complète (middlewares +
// mux + handlers) plutôt que les handlers isolément : c'est le seul test qui
// vérifie que le câblage dans NewRouter (ordre des middlewares, montage des
// routes) fonctionne réellement ensemble.
func TestRouter_endToEndCreateListGet(t *testing.T) {
	router := newTestRouter()

	body, _ := json.Marshal(map[string]any{"title": "Go", "content": "un langage compilé"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/notes: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID absent de la réponse : middleware RequestID non appliqué")
	}

	var created struct {
		Data struct{ ID string } `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("décodage réponse création: %v", err)
	}
	if created.Data.ID == "" {
		t.Fatal("id vide après création")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/notes: status = %d", listRec.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/notes/"+created.Data.ID, nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/notes/{id}: status = %d, body = %s", getRec.Code, getRec.Body.String())
	}
}

func TestRouter_unknownRouteReturnsJSONNotFound(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, attendu 404", rec.Code)
	}
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("réponse non-JSON: %v (%s)", err, rec.Body.String())
	}
	if body.Error.Message != "route introuvable" {
		t.Errorf("message = %q, attendu 'route introuvable' pour une route qui n'existe vraiment pas", body.Error.Message)
	}
}

func TestRouter_knownRouteWrongMethodReturns405(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/notes", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, attendu 405", rec.Code)
	}
}

func TestRouter_rootRedirectsToApp(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, attendu 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/app/" {
		t.Errorf("Location = %q, attendu /app/", loc)
	}
}

func TestRouter_missingNoteReturnsJSONNotFoundNotRouteMessage(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, attendu 404", rec.Code)
	}
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Message != "note introuvable" {
		t.Errorf("message = %q, attendu 'note introuvable' (pas 'route introuvable') pour une route valide mais un id inexistant", body.Error.Message)
	}
}
