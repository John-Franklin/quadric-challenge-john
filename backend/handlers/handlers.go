// Package handlers implements the HTTP API.
package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"backend/sessions"
)

// Handlers holds the dependencies shared by all HTTP handlers.
type Handlers struct {
	DB       *sql.DB
	Sessions *sessions.Store
	// RunnerToken must be sent by runners in the X-Runner-Token header.
	RunnerToken string
	// AdminEmail, if set, makes the account registered with that email an admin.
	AdminEmail string
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeInternalError logs the underlying error without leaking it to the client.
func writeInternalError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

// decodeJSON decodes the request body into v, replying 400 and returning false if it is invalid.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// pathID parses the {id} path value, replying 400 and returning false if it is invalid.
func pathID(w http.ResponseWriter, r *http.Request, resource string) (uint64, bool) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+resource+" id")
		return 0, false
	}
	return id, true
}
