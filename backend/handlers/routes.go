package handlers

import "net/http"

// Routes returns the API router.
func (h *Handlers) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/register", h.RegisterHandler)
	mux.HandleFunc("POST /api/auth/login", h.LoginHandler)
	mux.HandleFunc("POST /api/auth/logout", h.LogoutHandler)
	mux.HandleFunc("GET /api/auth/me", h.RequireUser(h.MeHandler))

	mux.HandleFunc("POST /api/runner", h.RequireRunner(h.RunnerHandler))
	mux.HandleFunc("GET /api/jobs/acquire", h.RequireRunner(h.AcquireJobsHandler))
	mux.HandleFunc("POST /api/jobs/{id}/logs", h.RequireRunner(h.AppendLogsHandler))

	mux.HandleFunc("GET /api/jobs", h.RequireUser(h.ListJobsHandler))
	mux.HandleFunc("POST /api/jobs", h.RequireUser(h.CreateJobHandler))
	mux.HandleFunc("GET /api/jobs/{id}", h.RequireUser(h.GetJobHandler))
	mux.HandleFunc("POST /api/jobs/{id}/cancel", h.RequireUser(h.CancelJobHandler))

	mux.HandleFunc("GET /api/admin/users", h.RequireAdmin(h.ListUsersHandler))
	mux.HandleFunc("PATCH /api/admin/users/{id}", h.RequireAdmin(h.UpdateUserHandler))
	return mux
}
