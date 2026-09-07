// Package middleware provides authentication, authorization and CSRF guards.
package middleware

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/rantau/rantau/models"
	"github.com/rantau/rantau/repositories"
)

// SessionName is the key under which session data is stored.
const SessionName = "rantau_session"

// ErrInvalidSessionKey is returned when the session id is missing/invalid.
var ErrInvalidSessionKey = errors.New("no user in session")

// NewSessionStore builds a signed+encrypted cookie store.
func NewSessionStore(key string) *sessions.CookieStore {
	store := sessions.NewCookieStore([]byte(key))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // TLS is terminated off-app in dev; flip with reverse-proxy
	}
	return store
}

// authUserKey is the context key that carries the logged-in user.
type authUserKey struct{}

// AuthContext loads the current user into the request context when a valid
// session exists.
func AuthContext(store *sessions.CookieStore, db *sql.DB) func(http.Handler) http.Handler {
	repo := repositories.NewUserRepository(db)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			sess, err := store.Get(r, SessionName)
			if err == nil {
				if uid, ok := sess.Values["user_id"].(int64); ok && uid > 0 {
					if u, err := repo.FindByID(uid); err == nil {
						ctx = context.WithValue(ctx, authUserKey{}, u)
					}
				}
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// User returns the authenticated user, or nil when not logged in.
func User(r *http.Request) *models.User {
	if u, ok := r.Context().Value(authUserKey{}).(*models.User); ok {
		return u
	}
	return nil
}

// RequireAuth redirects unauthenticated visitors to /login.
func RequireAuth(store *sessions.CookieStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if User(r) == nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CSRF -------------------------------------------------------------------

// SetCSRFToken ensures a token exists in the session and returns it.
func SetCSRFToken(store *sessions.CookieStore, w http.ResponseWriter, r *http.Request) string {
	sess, err := store.Get(r, SessionName)
	if err != nil {
		return ""
	}
	if tok, ok := sess.Values["csrf"].(string); ok && tok != "" {
		return tok
	}
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	tok := base64.RawURLEncoding.EncodeToString(buf)
	sess.Values["csrf"] = tok
	_ = sess.Save(r, w)
	return tok
}

// ValidateCSRF reports whether the submitted token matches the session.
func ValidateCSRF(store *sessions.CookieStore, r *http.Request, submitted string) bool {
	sess, err := store.Get(r, SessionName)
	if err != nil {
		return false
	}
	tok, _ := sess.Values["csrf"].(string)
	return tok != "" && submitted != "" && tok == submitted
}