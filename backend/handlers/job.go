package handlers

import (
	"backend/models"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

const maxLogChunkBytes = 1 << 20

func (h *Handlers) AcquireJobsHandler(w http.ResponseWriter, r *http.Request) {
	runnerID := strings.TrimSpace(r.Header.Get("X-Runner-ID"))
	if runnerID == "" {
		writeError(w, http.StatusBadRequest, "X-Runner-ID header is required")
		return
	}

	// Registering on every poll keeps last_heard_at fresh and satisfies the jobs.runner_id FK.
	runner := models.Runner{ID: runnerID}
	if err := runner.Register(r.Context(), h.DB); err != nil {
		writeInternalError(w, r, err)
		return
	}

	job, err := models.AcquireJob(r.Context(), h.DB, runnerID)
	if errors.Is(err, models.ErrNotFound) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *Handlers) ListJobsHandler(w http.ResponseWriter, r *http.Request) {
	jobs, err := models.ListJobs(r.Context(), h.DB)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (h *Handlers) CreateJobHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
		Notes  string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if !slices.Contains(models.Actions, req.Action) {
		writeError(w, http.StatusBadRequest, "action must be one of: "+strings.Join(models.Actions, ", "))
		return
	}

	job := models.Job{Action: req.Action, Notes: strings.TrimSpace(req.Notes)}
	if err := job.Register(r.Context(), h.DB); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (h *Handlers) AppendLogsHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid job id", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxLogChunkBytes))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	job := models.Job{ID: id}
	err = job.AppendLogs(r.Context(), h.DB, string(body))
	if errors.Is(err, models.ErrNotFound) {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
