package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"mira/internal/core"
	"mira/internal/enrichment"
	resp "mira/internal/http/response"
	"mira/internal/store"
)

// searchEmbedTimeout borne le temps consacré au calcul de l'embedding de la
// requête de recherche (appel HTTP à Ollama), indépendamment du timeout
// global de la requête HTTP (middleware.Timeout).
const searchEmbedTimeout = 5 * time.Second

type NoteHandler struct {
	store      store.Store
	dispatcher *enrichment.Dispatcher
	embedder   enrichment.Embedder
	logger     *slog.Logger
}

func NewNoteHandler(s store.Store, dispatcher *enrichment.Dispatcher, embedder enrichment.Embedder, logger *slog.Logger) *NoteHandler {
	return &NoteHandler{store: s, dispatcher: dispatcher, embedder: embedder, logger: logger}
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input core.CreateNoteInput
	if !decodeAndValidate(w, r, &input) {
		return
	}
	n, err := h.store.Create(r.Context(), input)
	if err != nil {
		handleStoreErr(w, r, err)
		return
	}
	h.dispatcher.Enqueue(n.ID)
	resp.JSON(w, r, http.StatusCreated, n)
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := resp.ParsePagination(r)
	notes, total, err := h.store.List(r.Context(), limit, offset)
	if err != nil {
		handleStoreErr(w, r, err)
		return
	}
	resp.JSONList(w, r, http.StatusOK, notes, total, limit, offset)
}

func (h *NoteHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	n, err := h.store.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		handleStoreErr(w, r, err)
		return
	}
	resp.JSON(w, r, http.StatusOK, n)
}

func (h *NoteHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var input core.PatchNoteInput
	if !decodeAndValidate(w, r, &input) {
		return
	}
	n, err := h.store.Patch(r.Context(), r.PathValue("id"), input)
	if err != nil {
		handleStoreErr(w, r, err)
		return
	}
	h.dispatcher.Enqueue(n.ID)
	resp.JSON(w, r, http.StatusOK, n)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Delete(r.Context(), r.PathValue("id")); err != nil {
		handleStoreErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Search effectue une recherche hybride texte intégral + similarité
// vectorielle. Si le calcul de l'embedding de la requête échoue (ex. Ollama
// indisponible), la recherche continue en mode texte intégral seul plutôt
// que d'échouer entièrement.
func (h *NoteHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		resp.JSONError(w, r, http.StatusBadRequest, "MISSING_PARAM", "paramètre q requis")
		return
	}

	var queryEmbedding []float32
	embedCtx, cancel := context.WithTimeout(r.Context(), searchEmbedTimeout)
	embedding, err := h.embedder.Embed(embedCtx, q)
	cancel()
	if err != nil {
		h.logger.Warn("embedding de la requête indisponible, repli sur la recherche plein texte",
			"query", q, "error", err)
	} else {
		queryEmbedding = embedding
	}

	notes, err := h.store.Search(r.Context(), q, queryEmbedding, 0)
	if err != nil {
		handleStoreErr(w, r, err)
		return
	}
	resp.JSON(w, r, http.StatusOK, notes)
}
