package handlers

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"backend/models"
	"backend/sessions"
)

const (
	sessionCookie     = "session"
	minPasswordLength = 8
	// bcrypt ignores everything past 72 bytes, so longer passwords are rejected rather than truncated.
	maxPasswordBytes = 72
)

type userKey struct{}

// currentUser returns the user attached by RequireUser.
func currentUser(r *http.Request) *models.User {
	return r.Context().Value(userKey{}).(*models.User)
}

// cancelScope returns nil for admins (may cancel any job) or the user's own ID.
func cancelScope(u *models.User) *uint64 {
	if u.IsAdmin {
		return nil
	}
	return &u.ID
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func decodeCredentials(w http.ResponseWriter, r *http.Request) (credentials, bool) {
	var c credentials
	if !decodeJSON(w, r, &c) {
		return c, false
	}
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	return c, true
}

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email && len(email) <= 254
}

// RegisterHandler creates an account and logs it in.
func (h *Handlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	c, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	if !validEmail(c.Email) {
		writeError(w, http.StatusBadRequest, "a valid email is required")
		return
	}
	if len(c.Password) < minPasswordLength || len(c.Password) > maxPasswordBytes {
		writeError(w, http.StatusBadRequest, "password must be 8 to 72 characters")
		return
	}

	isAdmin := h.AdminEmail != "" && c.Email == h.AdminEmail
	user, err := models.CreateUser(r.Context(), h.DB, c.Email, c.Password, isAdmin)
	if errors.Is(err, models.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "an account with this email already exists")
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	if err := h.startSession(w, r, user.ID); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

// LoginHandler starts a session for valid credentials.
func (h *Handlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	c, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	user, err := models.Authenticate(r.Context(), h.DB, c.Email, c.Password)
	if errors.Is(err, models.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	if err := h.startSession(w, r, user.ID); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// LogoutHandler ends the current session, if any.
func (h *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		if err := h.Sessions.Delete(r.Context(), cookie.Value); err != nil {
			writeInternalError(w, r, err)
			return
		}
	}
	setSessionCookie(w, r, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

// MeHandler returns the logged-in user.
func (h *Handlers) MeHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentUser(r))
}

func (h *Handlers) startSession(w http.ResponseWriter, r *http.Request, userID uint64) error {
	token, err := h.Sessions.Create(r.Context(), userID)
	if err != nil {
		return err
	}
	setSessionCookie(w, r, token, int(sessions.TTL.Seconds()))
	return nil
}

// SameSite=Lax keeps the cookie off cross-site POSTs, which covers CSRF for this JSON API.
func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/api",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

// RequireUser rejects requests without a valid session and attaches the user to the context.
// The user is reloaded on every request so admin changes apply immediately.
func (h *Handlers) RequireUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		userID, err := h.Sessions.UserID(r.Context(), cookie.Value)
		if errors.Is(err, sessions.ErrNoSession) {
			writeError(w, http.StatusUnauthorized, "session expired, please log in again")
			return
		}
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		user, err := models.GetUser(r.Context(), h.DB, userID)
		if errors.Is(err, models.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	}
}

// RequireAdmin is RequireUser that additionally rejects non-admins.
func (h *Handlers) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return h.RequireUser(func(w http.ResponseWriter, r *http.Request) {
		if !currentUser(r).IsAdmin {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}
		next(w, r)
	})
}

// RequireRunner checks the shared runner token sent in X-Runner-Token.
func (h *Handlers) RequireRunner(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Runner-Token")
		if subtle.ConstantTimeCompare([]byte(token), []byte(h.RunnerToken)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid runner token")
			return
		}
		next(w, r)
	}
}
