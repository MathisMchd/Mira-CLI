package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	resp "mira/internal/http/response"
	"mira/internal/store"
)

const maxBodyBytes = 1 << 20 // 1 MB

type validator interface {
	Validate() error
}

// decodeAndValidate vérifie Content-Type, limite la taille du body (413),
// décode le JSON (400) et valide le payload (422).
func decodeAndValidate(w http.ResponseWriter, r *http.Request, v validator) bool {
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(strings.TrimSpace(ct), "application/json") {
		resp.JSONError(w, r, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE",
			"Content-Type doit être application/json")
		return false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			resp.JSONError(w, r, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE",
				"corps de requête trop volumineux (max 1 Mo)")
			return false
		}
		resp.JSONError(w, r, http.StatusBadRequest, "INVALID_JSON", "corps JSON invalide")
		return false
	}

	if err := v.Validate(); err != nil {
		resp.JSONError(w, r, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return false
	}
	return true
}

// handleStoreErr mappe les erreurs du store vers les réponses HTTP appropriées.
func handleStoreErr(w http.ResponseWriter, r *http.Request, err error) {
	if store.IsNotFound(err) {
		resp.JSONError(w, r, http.StatusNotFound, "NOT_FOUND", "note introuvable")
		return
	}
	resp.JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "erreur interne")
}
