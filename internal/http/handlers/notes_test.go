package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mira/internal/http/handlers"
	"mira/internal/store"
)

func newHandler() *handlers.NoteHandler {
	return handlers.NewNoteHandler(store.NewMemory())
}

func TestCreate_Success(t *testing.T) {
	h := newHandler()
	body := `{"title":"Go","content":"Un langage compilé"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
	var env map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data field")
	}
	if data["title"] != "Go" {
		t.Errorf("unexpected title: %v", data["title"])
	}
	if id, _ := data["id"].(string); id == "" {
		t.Error("expected non-empty id")
	}
}

func TestCreate_MissingTitle(t *testing.T) {
	h := newHandler()
	body := `{"content":"pas de titre"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
	var env map[string]interface{}
	json.NewDecoder(w.Body).Decode(&env)
	if env["error"] == nil {
		t.Error("expected error field in response")
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	h := newHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(`{invalid}`))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreate_WrongContentType(t *testing.T) {
	h := newHandler()
	body := `{"title":"Go","content":"Un langage compilé"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", w.Code)
	}
	var env map[string]interface{}
	json.NewDecoder(w.Body).Decode(&env)
	errObj, ok := env["error"].(map[string]interface{})
	if !ok {
		t.Fatal("missing error field")
	}
	if errObj["code"] != "UNSUPPORTED_MEDIA_TYPE" {
		t.Errorf("expected UNSUPPORTED_MEDIA_TYPE, got %v", errObj["code"])
	}
}

func TestGetByID_NotFound(t *testing.T) {
	h := newHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	h.GetByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var env map[string]interface{}
	json.NewDecoder(w.Body).Decode(&env)
	errObj, ok := env["error"].(map[string]interface{})
	if !ok {
		t.Fatal("missing error field")
	}
	if errObj["code"] != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND, got %v", errObj["code"])
	}
}
