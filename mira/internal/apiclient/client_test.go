package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"mira/internal/core"
)

func writeEnvelope(t *testing.T, w http.ResponseWriter, status int, data any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "meta": map[string]any{}})
}

func writeErrEnvelope(t *testing.T, w http.ResponseWriter, status int, code, msg string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": msg},
		"meta":  map[string]any{},
	})
}

func TestCreate_success(t *testing.T) {
	var gotBody core.CreateNoteInput
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/notes" {
			t.Errorf("requête inattendue: %s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, attendu application/json", ct)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		writeEnvelope(t, w, http.StatusCreated, core.Note{ID: "n1", Title: gotBody.Title})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	n, err := c.Create(context.Background(), core.CreateNoteInput{Title: "Go", Content: "..."})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n.ID != "n1" || n.Title != "Go" {
		t.Errorf("note = %+v, attendu ID=n1 Title=Go", n)
	}
	if gotBody.Title != "Go" {
		t.Errorf("corps envoyé = %+v", gotBody)
	}
}

func TestCreate_apiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeErrEnvelope(t, w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "title is required")
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	_, err := c.Create(context.Background(), core.CreateNoteInput{})

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("attendu *APIError, got %T (%v)", err, err)
	}
	if apiErr.Code != "VALIDATION_ERROR" {
		t.Errorf("code = %q, attendu VALIDATION_ERROR", apiErr.Code)
	}
	if got := apiErr.Error(); got != "VALIDATION_ERROR: title is required" {
		t.Errorf("Error() = %q", got)
	}
}

func TestList_buildsQueryString(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		writeEnvelope(t, w, http.StatusOK, []core.Note{})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	if _, err := c.List(context.Background(), 25, 50); err != nil {
		t.Fatalf("List: %v", err)
	}
	if gotQuery.Get("limit") != "25" || gotQuery.Get("offset") != "50" {
		t.Errorf("query = %v, attendu limit=25 offset=50", gotQuery)
	}
}

func TestSearch_limitOmittedWhenZero(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		writeEnvelope(t, w, http.StatusOK, []core.Note{})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	if _, err := c.Search(context.Background(), "go", 0); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if gotQuery.Get("q") != "go" {
		t.Errorf("q = %q, attendu go", gotQuery.Get("q"))
	}
	if gotQuery.Has("limit") {
		t.Errorf("limit ne devrait pas être envoyé quand 0, query = %v", gotQuery)
	}
}

func TestSearch_limitSentWhenPositive(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		writeEnvelope(t, w, http.StatusOK, []core.Note{})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	if _, err := c.Search(context.Background(), "go", 5); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if gotQuery.Get("limit") != "5" {
		t.Errorf("limit = %q, attendu 5", gotQuery.Get("limit"))
	}
}

func TestGetByID_pathEscaped(t *testing.T) {
	var gotEscapedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEscapedPath = r.URL.EscapedPath() // forme brute envoyée sur le fil, avant décodage par net/http
		writeEnvelope(t, w, http.StatusOK, core.Note{ID: "abc def"})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	n, err := c.GetByID(context.Background(), "abc def")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if gotEscapedPath != "/api/v1/notes/abc%20def" {
		t.Errorf("escaped path = %q, attendu /api/v1/notes/abc%%20def", gotEscapedPath)
	}
	if n.ID != "abc def" {
		t.Errorf("note = %+v", n)
	}
}

func TestGetByID_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeErrEnvelope(t, w, http.StatusNotFound, "NOT_FOUND", "note introuvable")
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	_, err := c.GetByID(context.Background(), "missing")

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "NOT_FOUND" {
		t.Fatalf("attendu APIError NOT_FOUND, got %v", err)
	}
}

func TestDo_setsAuthorizationHeaderWhenKeySet(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeEnvelope(t, w, http.StatusOK, []core.Note{})
	}))
	defer srv.Close()

	c := New(srv.URL, "secret-key")
	if _, err := c.List(context.Background(), 10, 0); err != nil {
		t.Fatalf("List: %v", err)
	}
	if gotAuth != "Bearer secret-key" {
		t.Errorf("Authorization = %q, attendu 'Bearer secret-key'", gotAuth)
	}
}

func TestDo_noAuthorizationHeaderWhenKeyEmpty(t *testing.T) {
	var sawHeader bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawHeader = r.Header.Get("Authorization") != ""
		writeEnvelope(t, w, http.StatusOK, []core.Note{})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	if _, err := c.List(context.Background(), 10, 0); err != nil {
		t.Fatalf("List: %v", err)
	}
	if sawHeader {
		t.Error("en-tête Authorization envoyé alors qu'aucune clé n'est configurée")
	}
}

func TestDo_connectivityErrorIsFriendly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	badURL := srv.URL
	srv.Close() // plus personne n'écoute sur ce port

	c := New(badURL, "")
	_, err := c.List(context.Background(), 10, 0)
	if err == nil {
		t.Fatal("attendu une erreur de connectivité")
	}
	if !strings.Contains(err.Error(), "impossible de joindre l'API") {
		t.Errorf("message = %q, attendu qu'il contienne 'impossible de joindre l'API'", err.Error())
	}
}

func TestDo_noContentReturnsNilError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	n, err := c.GetByID(context.Background(), "x")
	if err != nil {
		t.Fatalf("204 devrait être traité sans erreur, got %v", err)
	}
	if n == nil {
		t.Fatal("attendu un pointeur non-nil même sur 204 (note zéro-valeur)")
	}
}
