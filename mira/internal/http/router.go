package apihttp

import (
	"log/slog"
	"net/http"
	"time"

	"mira/internal/http/handlers"
	"mira/internal/http/middleware"
	"mira/internal/store"
)

func NewRouter(s store.Store, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	notes := handlers.NewNoteHandler(s)

	mux.HandleFunc("POST /api/v1/notes", notes.Create)
	mux.HandleFunc("GET /api/v1/notes", notes.List)
	mux.HandleFunc("GET /api/v1/notes/{id}", notes.GetByID)
	mux.HandleFunc("PATCH /api/v1/notes/{id}", notes.Patch)
	mux.HandleFunc("DELETE /api/v1/notes/{id}", notes.Delete)
	mux.HandleFunc("GET /api/v1/search", notes.Search)
	mux.Handle("GET /docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir("docs"))))

	return middleware.Chain(mux,
		middleware.MuxErrors,
		middleware.Recovery(logger),
		middleware.Timeout(10*time.Second),
		middleware.Logging(logger),
		middleware.RequestID,
	)
}
