package handlers

import (
	"log"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/rantau/rantau/middleware"
	"github.com/rantau/rantau/models"
)

var (
	emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRe = regexp.MustCompile(`^\+?[0-9]{9,15}$`)
)

// RegisterRoutes wires auth routes into the mux.
func (a *App) RegisterAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", a.rootRedirect)
	mux.HandleFunc("GET /login", a.consistentAuth(a.loginGet))
	mux.HandleFunc("POST /login", a.consistentAuth(a.loginPost))
	mux.HandleFunc("GET /register", a.consistentAuth(a.registerGet))
	mux.HandleFunc("POST /register", a.consistentAuth(a.registerPost))
	mux.HandleFunc("POST /logout", a.requireCSRF(a.logout))
}

// rootRedirect is the public catch-all: / sends users to the right entry.
func (a *App) rootRedirect(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if middleware.User(r) != nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// consistentAuth redirects logged-in users away from login/register pages.
func (a *App) consistentAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if middleware.User(r) != nil {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (a *App) loginGet(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "auth/login.html", "Masuk", map[string]any{})
}

func (a *App) loginPost(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	u, err := a.Users.FindByEmail(email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		a.setFlash(w, r, "flash_err", "Email atau password salah. Silakan coba lagi.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	sess, err := a.Store.Get(r, middleware.SessionName)
	if err != nil {
		http.Error(w, "session unavailable", http.StatusInternalServerError)
		return
	}
	sess.Values["user_id"] = u.ID
	sess.Values["role"] = u.Role
	_ = sess.Save(r, w)

	redirect := "/dashboard"
	if u.Role == models.RoleAdmin {
		redirect = "/admin"
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (a *App) registerGet(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "auth/login.html", "Daftar", map[string]any{})
}

func (a *App) registerPost(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")

	if name == "" || len(name) < 3 {
		a.setFlash(w, r, "flash_err", "Nama lengkap minimal 3 karakter.")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}
	if !emailRe.MatchString(email) {
		a.setFlash(w, r, "flash_err", "Alamat email tidak valid.")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}
	if !phoneRe.MatchString(phone) {
		a.setFlash(w, r, "flash_err", "Nomor HP tidak valid.")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}
	if len(password) < 8 {
		a.setFlash(w, r, "flash_err", "Password minimal 8 karakter.")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}
	if password != confirm {
		a.setFlash(w, r, "flash_err", "Konfirmasi password tidak sama.")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	exists, err := a.Users.EmailExists(email, 0)
	if err != nil {
		a.serverError(w, err)
		return
	}
	if exists {
		a.setFlash(w, r, "flash_err", "Email sudah terdaftar. Silakan masuk.")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		a.serverError(w, err)
		return
	}

	u := &models.User{
		Name:     name,
		Email:    email,
		Phone:    phone,
		Password: string(hash),
		Role:     models.RoleCustomer,
	}
	if _, err := a.Users.Create(u); err != nil {
		a.serverError(w, err)
		return
	}

	a.setFlash(w, r, "flash_ok", "Akun berhasil dibuat. Silakan masuk.")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	sess, err := a.Store.Get(r, middleware.SessionName)
	if err == nil {
		sess.Options.MaxAge = -1
		_ = sess.Save(r, w)
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *App) serverError(w http.ResponseWriter, err error) {
	log.Printf("handlers error: %v", err)
	http.Error(w, "Terjadi kesalahan internal. Silakan coba lagi.", http.StatusInternalServerError)
}