package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"backend/models"
)

const (
	maxLogChunkBytes = 1 << 20
	maxCodeBytes     = 64 << 10

	defaultDigits = 1000
	maxDigits     = 10000
	defaultWords  = 200
	maxWords      = 5000
)

// sizeParam validates an optional per-action count: it must be omitted unless action is
// forAction, defaults to def when omitted, and must be within 1..limit.
func sizeParam(name string, v *int, action, forAction string, def, limit int) (*int, error) {
	if action != forAction {
		if v != nil {
			return nil, fmt.Errorf("%s is only allowed for %s", name, forAction)
		}
		return nil, nil
	}
	if v == nil {
		return &def, nil
	}
	if *v < 1 || *v > limit {
		return nil, fmt.Errorf("%s must be between 1 and %d for %s", name, limit, forAction)
	}
	return v, nil
}

// AcquireJobsHandler hands the oldest pending job to the calling runner, or replies 204 if there is none.
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

// ListJobsHandler returns one page of jobs, selected by the 1-based "page" query parameter.
func (h *Handlers) ListJobsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if raw := query.Get("ids"); raw != "" {
		if query.Has("before") {
			writeError(w, http.StatusBadRequest, "ids and before cannot be combined")
			return
		}
		h.listJobsByIDs(w, r, raw)
		return
	}

	var before uint64
	if raw := query.Get("before"); raw != "" {
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || n == 0 {
			writeError(w, http.StatusBadRequest, "before must be a positive job id")
			return
		}
		before = n
	}

	result, err := models.ListJobs(r.Context(), h.DB, before)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// listJobsByIDs serves GET /api/jobs?ids=1,2,3, which lets clients refresh specific rows
// (e.g. still-active jobs further down an infinite-scroll list).
func (h *Handlers) listJobsByIDs(w http.ResponseWriter, r *http.Request, raw string) {
	parts := strings.Split(raw, ",")
	if len(parts) > models.MaxJobsByIDs {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("at most %d ids per request", models.MaxJobsByIDs))
		return
	}
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "ids must be a comma-separated list of positive job ids")
			return
		}
		ids = append(ids, n)
	}

	jobs, err := models.ListJobsByIDs(r.Context(), h.DB, ids)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string][]models.Job{"jobs": jobs})
}

// GetJobHandler returns a single job including its logs.
func (h *Handlers) GetJobHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "job")
	if !ok {
		return
	}

	job, err := models.GetJob(r.Context(), h.DB, id)
	if errors.Is(err, models.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// CreateJobHandler queues a new job owned by the current user.
func (h *Handlers) CreateJobHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
		Notes  string `json:"notes"`
		Code   string `json:"code"`
		Digits *int   `json:"digits"`
		Words  *int   `json:"words"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !slices.Contains(models.Actions, req.Action) {
		writeError(w, http.StatusBadRequest, "action must be one of: "+strings.Join(models.Actions, ", "))
		return
	}

	if req.Action == models.ActionRunPython {
		if strings.TrimSpace(req.Code) == "" {
			writeError(w, http.StatusBadRequest, "code is required for "+models.ActionRunPython)
			return
		}
		if len(req.Code) > maxCodeBytes {
			writeError(w, http.StatusBadRequest, "code must be at most 64 KiB")
			return
		}
	} else if req.Code != "" {
		writeError(w, http.StatusBadRequest, "code is only allowed for "+models.ActionRunPython)
		return
	}

	digits, err := sizeParam("digits", req.Digits, req.Action, models.ActionCalculatePi, defaultDigits, maxDigits)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	words, err := sizeParam("words", req.Words, req.Action, models.ActionLoremIpsum, defaultWords, maxWords)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	job := models.Job{
		Action: req.Action, Notes: strings.TrimSpace(req.Notes), Code: req.Code,
		Digits: digits, Words: words,
	}
	if err := job.Register(r.Context(), h.DB, currentUser(r)); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

// AppendLogsHandler appends a runner's log chunk to a job.
func (h *Handlers) AppendLogsHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "job")
	if !ok {
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxLogChunkBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	job := models.Job{ID: id}
	err = job.AppendLogs(r.Context(), h.DB, string(body))
	if errors.Is(err, models.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	// The runner stops the job when this reports "cancelled".
	w.Header().Set("Job-Status", job.Status)
	w.WriteHeader(http.StatusNoContent)
}

// CancelJobHandler cancels a pending or running job owned by the current user, or any job for admins.
func (h *Handlers) CancelJobHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "job")
	if !ok {
		return
	}

	job, err := models.CancelJob(r.Context(), h.DB, id, cancelScope(currentUser(r)))
	switch {
	case errors.Is(err, models.ErrNotFound):
		writeError(w, http.StatusNotFound, "job not found")
	case errors.Is(err, models.ErrForbidden):
		writeError(w, http.StatusForbidden, "you can only cancel your own jobs")
	case errors.Is(err, models.ErrNotCancellable):
		writeError(w, http.StatusConflict, "only pending or running jobs can be cancelled")
	case err != nil:
		writeInternalError(w, r, err)
	default:
		writeJSON(w, http.StatusOK, job)
	}
}
