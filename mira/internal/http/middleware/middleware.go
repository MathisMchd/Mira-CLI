package middleware

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	resp "mira/internal/http/response"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = fmt.Sprintf("%x", b)
		}
		w.Header().Set("X-Request-ID", id)
		r.Header.Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(rec, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", r.Header.Get("X-Request-ID"),
			)
		})
	}
}

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						"error", err,
						"stack", string(debug.Stack()),
						"request_id", r.Header.Get("X-Request-ID"),
					)
					resp.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// bufResponseWriter bufferise la réponse pour permettre son interception avant envoi.
type bufResponseWriter struct {
	mu          sync.Mutex
	header      http.Header
	buf         bytes.Buffer
	status      int
	timedOut    bool
	wroteHeader bool
}

func newBufResponseWriter() *bufResponseWriter {
	return &bufResponseWriter{
		header: make(http.Header),
		status: http.StatusOK,
	}
}

func (bw *bufResponseWriter) Header() http.Header { return bw.header }

func (bw *bufResponseWriter) WriteHeader(code int) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	if !bw.wroteHeader && !bw.timedOut {
		bw.status = code
		bw.wroteHeader = true
	}
}

func (bw *bufResponseWriter) Write(b []byte) (int, error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	if bw.timedOut {
		return len(b), nil
	}
	bw.wroteHeader = true
	return bw.buf.Write(b)
}

func (bw *bufResponseWriter) flush(w http.ResponseWriter) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	for k, v := range bw.header {
		w.Header()[k] = v
	}
	w.WriteHeader(bw.status)
	_, _ = w.Write(bw.buf.Bytes())
}

func (bw *bufResponseWriter) markTimedOut() {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.timedOut = true
}

// Timeout interrompt la requête après d et renvoie 503 avec l'enveloppe JSON correcte.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()

			bw := newBufResponseWriter()
			done := make(chan struct{})

			go func() {
				next.ServeHTTP(bw, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				bw.flush(w)
			case <-ctx.Done():
				bw.markTimedOut()
				resp.JSONError(w, r, http.StatusServiceUnavailable, "TIMEOUT", "request timed out")
			}
		})
	}
}

// MuxErrors intercepte les 404 et 405 produits par le mux et les reformate en JSON.
func MuxErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bw := newBufResponseWriter()
		next.ServeHTTP(bw, r)
		alreadyJSON := strings.HasPrefix(bw.header.Get("Content-Type"), "application/json")
		switch {
		case bw.status == http.StatusNotFound && !alreadyJSON:
			resp.JSONError(w, r, http.StatusNotFound, "NOT_FOUND", "route introuvable")
		case bw.status == http.StatusMethodNotAllowed && !alreadyJSON:
			if allow := bw.header.Get("Allow"); allow != "" {
				w.Header().Set("Allow", allow)
			}
			resp.JSONError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "méthode non autorisée")
		default:
			bw.flush(w)
		}
	})
}

func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
