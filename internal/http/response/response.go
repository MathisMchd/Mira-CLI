package response

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type Envelope struct {
	Data  interface{} `json:"data,omitempty"`
	Meta  Meta   `json:"meta"`
	Error *Error `json:"error,omitempty"`
}

type Meta struct {
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp"`
	Total     *int   `json:"total,omitempty"`
	Limit     *int   `json:"limit,omitempty"`
	Offset    *int   `json:"offset,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	write(w, status, Envelope{
		Data: data,
		Meta: meta(r),
	})
}

func JSONList(w http.ResponseWriter, r *http.Request, status int, data interface{}, total, limit, offset int) {
	m := meta(r)
	m.Total = &total
	m.Limit = &limit
	m.Offset = &offset
	write(w, status, Envelope{Data: data, Meta: m})
}

func JSONError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	write(w, status, Envelope{
		Meta:  meta(r),
		Error: &Error{Code: code, Message: message},
	})
}

// ParsePagination extrait limit et offset depuis les query params.
// Valeurs par défaut : limit=10, offset=0. Limit plafonné à 100.
func ParsePagination(r *http.Request) (limit, offset int) {
	limit = 10
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

func meta(r *http.Request) Meta {
	return Meta{
		RequestID: r.Header.Get("X-Request-ID"),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func write(w http.ResponseWriter, status int, env Envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}
