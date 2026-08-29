// Package auth handles password hashing and cookie-based sessions for
// Leakboard's own dashboard login. It intentionally does not implement
// GitHub OAuth login: v1 uses one local admin account per instance, kept
// separate from the GitHub personal access token used to read repos.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Laaaaksh/leakboard/internal/store"
)

// CookieName is the session cookie set on login.
const CookieName = "leakboard_session"

// SessionDuration is how long a session stays valid after login.
const SessionDuration = 30 * 24 * time.Hour

// HashPassword bcrypt-hashes a plaintext password for storage.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword reports whether password matches a bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateToken returns a random, URL-safe session token.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

type contextKey int

const userContextKey contextKey = 0

// UserFromContext returns the authenticated user set by Middleware, or
// (User{}, false) if the request was not authenticated.
func UserFromContext(ctx context.Context) (store.User, bool) {
	u, ok := ctx.Value(userContextKey).(store.User)
	return u, ok
}

// Middleware resolves the session cookie into a user and stores it in the
// request context. It never rejects the request itself: handlers that
// require a logged-in user call RequireUser and respond 401 if none is set,
// which keeps unauthenticated read endpoints (like the login page) simple.
func Middleware(db *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(CookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			user, err := db.UserBySession(r.Context(), cookie.Value)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireUser returns the authenticated user or writes a 401 JSON error and
// reports ok=false.
func RequireUser(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return store.User{}, false
	}
	return u, true
}

// SetSessionCookie sets the session cookie for token, marking it Secure
// when secure is true (the instance is served over HTTPS).
func SetSessionCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(SessionDuration),
	})
}

// ClearSessionCookie expires the session cookie, logging the browser out.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}
