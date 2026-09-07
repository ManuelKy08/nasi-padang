package middleware

import (
	"net/http"

	"github.com/gorilla/sessions"
)

// RequireAdmin restricts a route to the admin role using server-side checks.
// Users must also be authenticated; combine with RequireAuth.
func RequireAdmin(store *sessions.CookieStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := User(r)
			if u == nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			if u.Role != "admin" {
				http.Error(w, "403 — Halaman ini hanya untuk admin.", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}