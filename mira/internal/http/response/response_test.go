package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func decode(t *testing.T, rec *httptest.ResponseRecorder) Envelope {
	t.Helper()
	var env Envelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("décodage réponse: %v", err)
	}
	return env
}

func TestJSON_setsDataAndMeta(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.Header.Set("X-Request-ID", "req-1")

	JSON(rec, r, http.StatusOK, map[string]string{"hello": "world"})

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, attendu %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, attendu application/json", ct)
	}

	env := decode(t, rec)
	if env.Error != nil {
		t.Errorf("Error non-nil sur une réponse succès: %+v", env.Error)
	}
	if env.Meta.RequestID != "req-1" {
		t.Errorf("request_id = %q, attendu req-1", env.Meta.RequestID)
	}
	if env.Meta.Timestamp == "" {
		t.Error("timestamp vide")
	}
	if env.Meta.Total != nil || env.Meta.Limit != nil || env.Meta.Offset != nil {
		t.Errorf("meta de pagination inattendue sur JSON simple: %+v", env.Meta)
	}
}

func TestJSONList_setsPaginationMeta(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/x", nil)

	JSONList(rec, r, http.StatusOK, []int{1, 2, 3}, 42, 10, 5)

	env := decode(t, rec)
	if env.Meta.Total == nil || *env.Meta.Total != 42 {
		t.Errorf("total = %v, attendu 42", env.Meta.Total)
	}
	if env.Meta.Limit == nil || *env.Meta.Limit != 10 {
		t.Errorf("limit = %v, attendu 10", env.Meta.Limit)
	}
	if env.Meta.Offset == nil || *env.Meta.Offset != 5 {
		t.Errorf("offset = %v, attendu 5", env.Meta.Offset)
	}
}

func TestJSONError_setsErrorNoData(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/x", nil)

	JSONError(rec, r, http.StatusNotFound, "NOT_FOUND", "note introuvable")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, attendu 404", rec.Code)
	}
	env := decode(t, rec)
	if env.Error == nil {
		t.Fatal("Error nil sur une réponse d'erreur")
	}
	if env.Error.Code != "NOT_FOUND" || env.Error.Message != "note introuvable" {
		t.Errorf("error = %+v, attendu {NOT_FOUND note introuvable}", env.Error)
	}
	if env.Data != nil {
		t.Errorf("data non-nil sur une réponse d'erreur: %v", env.Data)
	}
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantLimit  int
		wantOffset int
	}{
		{"absent -> défauts", "", 10, 0},
		{"valeurs valides", "limit=25&offset=50", 25, 50},
		{"limit non numérique -> défaut", "limit=abc", 10, 0},
		{"limit négatif -> défaut", "limit=-5", 10, 0},
		{"limit à zéro -> défaut", "limit=0", 10, 0},
		{"limit au-delà de 100 -> défaut", "limit=101", 10, 0},
		{"limit à 100 -> accepté", "limit=100", 100, 0},
		{"offset négatif -> défaut", "offset=-1", 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/x?"+tt.query, nil)
			gotLimit, gotOffset := ParsePagination(r)
			if gotLimit != tt.wantLimit || gotOffset != tt.wantOffset {
				t.Errorf("ParsePagination(%q) = (%d, %d), attendu (%d, %d)",
					tt.query, gotLimit, gotOffset, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}

func TestParsePagination_urlValuesRoundtrip(t *testing.T) {
	v := url.Values{"limit": {"7"}, "offset": {"3"}}
	r := httptest.NewRequest(http.MethodGet, "/x?"+v.Encode(), nil)
	limit, offset := ParsePagination(r)
	if limit != 7 || offset != 3 {
		t.Errorf("ParsePagination = (%d, %d), attendu (7, 3)", limit, offset)
	}
}
