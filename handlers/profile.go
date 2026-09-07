package handlers

import (
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/rantau/rantau/middleware"
)

// RegisterProfileRoutes wires the profile page.
func (a *App) RegisterProfileRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /profile", a.profilePage)
	mux.HandleFunc("POST /profile", a.requireCSRF(a.profileUpdate))
}

func (a *App) profilePage(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	count, total, err := a.Orders.UserStats(u.ID)
	if err != nil {
		a.serverError(w, err)
		return
	}
	a.render(w, r, "profile/index.html", "Profil Saya", map[string]any{
		"OrderCount": count,
		"TotalSpent": total,
	})
}

func (a *App) profileUpdate(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)

	name := strings.TrimSpace(r.FormValue("name"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	address := strings.TrimSpace(r.FormValue("address"))
	password := r.FormValue("password")
	newPassword := r.FormValue("new_password")

	if len(name) < 3 {
		a.setFlash(w, r, "flash_err", "Nama minimal 3 karakter.")
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	if !phoneRe.MatchString(phone) {
		a.setFlash(w, r, "flash_err", "Nomor HP tidak valid.")
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	u.Name = name
	u.Phone = phone
	u.Address = address
	if err := a.Users.Update(u); err != nil {
		a.serverError(w, err)
		return
	}

	if newPassword != "" {
		if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
			a.setFlash(w, r, "flash_err", "Password lama salah.")
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		}
		if len(newPassword) < 8 {
			a.setFlash(w, r, "flash_err", "Password baru minimal 8 karakter.")
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			a.serverError(w, err)
			return
		}
		if err := a.Users.UpdatePassword(u.ID, string(hash)); err != nil {
			a.serverError(w, err)
			return
		}
	}

	a.setFlash(w, r, "flash_ok", "Profil berhasil diperbarui.")
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}