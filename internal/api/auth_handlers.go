package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Laaaaksh/leakboard/internal/auth"
	"github.com/Laaaaksh/leakboard/internal/store"
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleSetup creates the single admin account. It only succeeds while no
// user exists yet, so it can be called safely on every fresh install
// without a separate "is this the first run" flag.
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	count, err := s.Store.CountUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "checking existing users")
		return
	}
	if count > 0 {
		writeError(w, http.StatusConflict, "setup already completed")
		return
	}

	var creds credentials
	if err := decodeJSON(r, &creds); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	creds.Email = strings.TrimSpace(creds.Email)
	if creds.Email == "" || len(creds.Password) < 8 {
		writeError(w, http.StatusBadRequest, "email is required and password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(creds.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}
	user, err := s.Store.CreateUser(r.Context(), creds.Email, hash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	s.startSession(w, r, user)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var creds credentials
	if err := decodeJSON(r, &creds); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := s.Store.UserByEmail(r.Context(), strings.TrimSpace(creds.Email))
	if errors.Is(err, store.ErrNotFound) || (err == nil && !auth.CheckPassword(user.PasswordHash, creds.Password)) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	s.startSession(w, r, user)
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, user store.User) {
	token, err := auth.GenerateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	if err := s.Store.CreateSession(r.Context(), token, user.ID, time.Now().Add(auth.SessionDuration)); err != nil {
		writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	auth.SetSessionCookie(w, token, s.Secure)
	writeJSON(w, http.StatusOK, map[string]string{"email": user.Email})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		_ = s.Store.DeleteSession(r.Context(), cookie.Value)
	}
	auth.ClearSessionCookie(w)
	writeJSON(w, http.StatusOK, nil)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		count, err := s.Store.CountUsers(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "checking setup state")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false, "setupRequired": count == 0})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "email": user.Email})
}
