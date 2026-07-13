package handlers

import (
	"net/http"

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
	if !decodeAndValidate(w, r, &input) {
		return
	}
	n, err := h.store.Create(input)
	if err != nil {
		handleStoreErr(w, r, err)
		return
	}
	resp.JSON(w, r, http.StatusCreated, n)
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := resp.ParsePagination(r)
	notes, total, err := h.store.List(limit, offset)
	if err != nil {
		handleStoreErr(w, r, err)
		return
	}
	resp.JSONList(w, r, http.StatusOK, notes, total, limit, offset)
}

func (h *NoteHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	n, err := h.store.GetByID(r.PathValue("id"))
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
	n, err := h.store.Patch(r.PathValue("id"), input)
	if err != nil {
		handleStoreErr(w, r, err)
		return
	}
	resp.JSON(w, r, http.StatusOK, n)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Delete(r.PathValue("id")); err != nil {
		handleStoreErr(w, r, err)
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
		handleStoreErr(w, r, err)
		return
	}
	resp.JSON(w, r, http.StatusOK, notes)
}
