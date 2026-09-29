package handlers

import (
	"errors"
	"net/http"

	"backend/models"
)

// ListUsersHandler returns every user.
func (h *Handlers) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := models.ListUsers(r.Context(), h.DB)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// UpdateUserHandler grants or revokes a user's admin access.
func (h *Handlers) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "user")
	if !ok {
		return
	}
	var req struct {
		IsAdmin *bool `json:"is_admin"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.IsAdmin == nil {
		writeError(w, http.StatusBadRequest, "is_admin is required")
		return
	}
	// Prevents the last admin from locking everyone out of the admin page.
	if !*req.IsAdmin && id == currentUser(r).ID {
		writeError(w, http.StatusBadRequest, "you cannot remove your own admin access")
		return
	}

	user, err := models.SetUserAdmin(r.Context(), h.DB, id, *req.IsAdmin)
	if errors.Is(err, models.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}
