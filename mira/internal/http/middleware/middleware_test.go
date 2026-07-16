package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRequestID_generatesWhenAbsent(t *testing.T) {
	var seenInHandler string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenInHandler = r.Header.Get("X-Request-ID")
	})

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	RequestID(next).ServeHTTP(rec, r)

	if seenInHandler == "" {
		t.Fatal("aucun X-Request-ID transmis au handler")
	}
	if rec.Header().Get("X-Request-ID") != seenInHandler {
		t.Errorf("X-Request-ID réponse (%q) != celui vu par le handler (%q)",
			rec.Header().Get("X-Request-ID"), seenInHandler)
	}
}

func TestRequestID_preservesExisting(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.Header.Set("X-Request-ID", "client-provided")
	RequestID(next).ServeHTTP(rec, r)

	if got := rec.Header().Get("X-Request-ID"); got != "client-provided" {
		t.Errorf("X-Request-ID = %q, attendu client-provided", got)
	}
}

func TestRecovery_catchesPanicAndReturns500(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/x", nil)

	// Ne doit pas paniquer jusqu'ici : Recovery intercepte.
	Recovery(discardLogger())(next).ServeHTTP(rec, r)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, attendu 500", rec.Code)
	}
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("réponse non-JSON après panic: %v (%s)", err, rec.Body.String())
	}
	if body.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("code erreur = %q, attendu INTERNAL_ERROR", body.Error.Code)
	}
}

func TestRecovery_passesThroughWithoutPanic(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("ok"))
	})

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	Recovery(discardLogger())(next).ServeHTTP(rec, r)

	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, attendu %d", rec.Code, http.StatusTeapot)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, attendu ok", rec.Body.String())
	}
}

func TestTimeout_passesThroughFastHandler(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("fast"))
	})

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	Timeout(200*time.Millisecond)(next).ServeHTTP(rec, r)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, attendu 201", rec.Code)
	}
	if rec.Body.String() != "fast" {
		t.Errorf("body = %q, attendu fast", rec.Body.String())
	}
	if rec.Header().Get("X-Custom") != "yes" {
		t.Error("en-tête personnalisé perdu par Timeout")
	}
}

func TestTimeout_returns503OnSlowHandler(t *testing.T) {
	blockUntilDone := make(chan struct{})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-blockUntilDone:
		}
	})
	defer close(blockUntilDone)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	Timeout(20*time.Millisecond)(next).ServeHTTP(rec, r)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, attendu 503", rec.Code)
	}
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Code != "TIMEOUT" {
		t.Errorf("code erreur = %q, attendu TIMEOUT", body.Error.Code)
	}
}

func TestMuxErrors_reformats404(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/inconnu", nil)
	MuxErrors(next).ServeHTTP(rec, r)

	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("réponse non-JSON: %v (%s)", err, rec.Body.String())
	}
	if body.Error.Code != "NOT_FOUND" {
		t.Errorf("code erreur = %q, attendu NOT_FOUND", body.Error.Code)
	}
}

func TestMuxErrors_reformats405AndPreservesAllow(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "GET, POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/x", nil)
	MuxErrors(next).ServeHTTP(rec, r)

	if rec.Header().Get("Allow") != "GET, POST" {
		t.Errorf("Allow = %q, attendu GET, POST", rec.Header().Get("Allow"))
	}
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Code != "METHOD_NOT_ALLOWED" {
		t.Errorf("code erreur = %q, attendu METHOD_NOT_ALLOWED", body.Error.Code)
	}
}

func TestMuxErrors_passesThroughOtherStatuses(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":"ok"}`))
	})

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	MuxErrors(next).ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, attendu 200", rec.Code)
	}
	if rec.Body.String() != `{"data":"ok"}` {
		t.Errorf("body altéré par MuxErrors: %q", rec.Body.String())
	}
}

func TestChain_appliesInOrder(t *testing.T) {
	var order []string
	mk := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+":before")
				next.ServeHTTP(w, r)
				order = append(order, name+":after")
			})
		}
	}
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	h := Chain(final, mk("a"), mk("b"))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))

	want := []string{"a:before", "b:before", "handler", "b:after", "a:after"}
	if len(order) != len(want) {
		t.Fatalf("ordre = %v, attendu %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("ordre[%d] = %q, attendu %q (complet: %v)", i, order[i], want[i], order)
		}
	}
}
