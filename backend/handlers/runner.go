package handlers

import (
	"backend/models"
	"encoding/json"
	"net/http"
	"strings"
)

func (h *Handlers) RunnerHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
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
