package handlers

import (
	"log"
	"net/http"

	"github.com/rantau/rantau/middleware"
)

// RegisterHomeRoutes wires the dashboard and about pages.
func (a *App) RegisterHomeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /dashboard", a.dashboard)
	mux.HandleFunc("GET /about", a.about)
}

// dashboard renders the personalised landing page.
func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	ctx := map[string]any{}

	if favs, err := a.Menus.Popular(4); err == nil {
		ctx["Popular"] = favs
	} else {
		log.Printf("dashboard popular: %v", err)
	}

	if nasional, err := a.Menus.Suggested([]string{"rendang", "ayam pop", "dendeng balado", "gulai ayam"}); err == nil {
		ctx["Favourites"] = nasional
	} else {
		log.Printf("dashboard sugared: %v", err)
	}

	if orders, err := a.Orders.ListByUser(u.ID, 3); err == nil {
		ctx["RecentOrders"] = orders
	} else {
		log.Printf("dashboard orders: %v", err)
	}

	a.render(w, r, "dashboard/index.html", "Dashboard", ctx)
}

// about shows the brand story.
func (a *App) about(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "about/index.html", "Tentang Kami", map[string]any{})
}