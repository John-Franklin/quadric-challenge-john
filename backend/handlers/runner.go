package handlers

import (
	"net/http"
	"strings"

	"backend/models"
)

// RunnerHandler registers a runner, or refreshes it if it is already known.
func (h *Handlers) RunnerHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	runner := models.Runner{ID: strings.TrimSpace(req.ID)}
	if runner.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	if err := runner.Register(r.Context(), h.DB); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, runner)
}
