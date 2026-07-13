package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"mira/internal/core"
	resp "mira/internal/http/response"
	"mira/internal/store"
)

type NoteHandler struct {
	store store.Store
}

func NewNoteHandler(s store.Store) *NoteHandler {
	return &NoteHandler{store: s}
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input core.CreateNoteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		resp.JSONError(w, r, http.StatusBadRequest, "INVALID_JSON", "corps JSON invalide")
		return
	}
	if err := input.Validate(); err != nil {
		resp.JSONError(w, r, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	n, err := h.store.Create(input)
	if err != nil {
		resp.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "erreur interne")
		return
	}
	resp.JSON(w, r, http.StatusCreated, n)
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	notes, total, err := h.store.List(limit, offset)
	if err != nil {
		resp.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "erreur interne")
		return
	}
	resp.JSONList(w, r, http.StatusOK, notes, total, limit, offset)
}

func (h *NoteHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	n, err := h.store.GetByID(id)
	if err != nil {
		if store.IsNotFound(err) {
			resp.JSONError(w, r, http.StatusNotFound, "NOT_FOUND", "note introuvable")
			return
		}
		resp.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "erreur interne")
		return
	}
	resp.JSON(w, r, http.StatusOK, n)
}

func (h *NoteHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var input core.PatchNoteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		resp.JSONError(w, r, http.StatusBadRequest, "INVALID_JSON", "corps JSON invalide")
		return
	}
	if err := input.Validate(); err != nil {
		resp.JSONError(w, r, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	n, err := h.store.Patch(id, input)
	if err != nil {
		if store.IsNotFound(err) {
			resp.JSONError(w, r, http.StatusNotFound, "NOT_FOUND", "note introuvable")
			return
		}
		resp.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "erreur interne")
		return
	}
	resp.JSON(w, r, http.StatusOK, n)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.Delete(id); err != nil {
		if store.IsNotFound(err) {
			resp.JSONError(w, r, http.StatusNotFound, "NOT_FOUND", "note introuvable")
			return
		}
		resp.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "erreur interne")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *NoteHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		resp.JSONError(w, r, http.StatusBadRequest, "MISSING_PARAM", "paramètre q requis")
		return
	}
	notes, err := h.store.Search(q)
	if err != nil {
		resp.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "erreur interne")
		return
	}
	resp.JSON(w, r, http.StatusOK, notes)
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 10
	offset = 0
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
