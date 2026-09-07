package handlers

import (
	"net/http"
	"strings"

	"github.com/rantau/rantau/middleware"
	"github.com/rantau/rantau/models"
	"github.com/rantau/rantau/repositories"
)

// RegisterMenuRoutes wires menu browsing, favorites and live search.
func (a *App) RegisterMenuRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /menu", a.menuIndex)
	mux.HandleFunc("GET /menu/{id}", a.menuDetail)
	mux.HandleFunc("GET /api/menu/search", a.menuSearchJSON)
	mux.HandleFunc("POST /favorites/toggle", a.requireCSRF(a.favoriteToggle))
	mux.HandleFunc("GET /favorites", a.favorites)
}

// menuIndex lists menu items with search, category and sort filters.
func (a *App) menuIndex(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	var userID int64
	if u != nil {
		userID = u.ID
	}

	f := repositories.MenuFilter{
		CategorySlug:  r.URL.Query().Get("category"),
		Search:        strings.TrimSpace(r.URL.Query().Get("q")),
		Sort:          r.URL.Query().Get("sort"),
		UserID:        userID,
		AvailableOnly: false,
	}

	items, err := a.Menus.List(f)
	if err != nil {
		a.serverError(w, err)
		return
	}
	cats, err := a.Menus.ListCategories()
	if err != nil {
		a.serverError(w, err)
		return
	}

	a.render(w, r, "menu/index.html", "Menu", map[string]any{
		"Items":      items,
		"Categories": cats,
		"ActiveCat":  f.CategorySlug,
		"Q":          f.Search,
		"Sort":       f.Sort,
		"Open":       a.openNow(),
	})
}

// menuDetail shows a single dish with quantity selector.
func (a *App) menuDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := middleware.User(r)
	var userID int64
	if u != nil {
		userID = u.ID
	}

	m, err := a.Menus.FindByID(id, userID)
	if err != nil {
		if err == repositories.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		a.serverError(w, err)
		return
	}

	related := []models.MenuItem{}
	catSlug := ""
	if cats, err := a.Menus.ListCategories(); err == nil {
		for _, c := range cats {
			if c.ID == m.CategoryID {
				catSlug = c.Slug
				if items, err := a.Menus.List(repositories.MenuFilter{CategorySlug: c.Slug, UserID: userID}); err == nil {
					related = items
				}
				break
			}
		}
	}

	cartQty := 0
	if carts, err := a.Carts.List(userID); err == nil {
		for _, c := range carts {
			if c.MenuItemID == m.ID {
				cartQty = c.Quantity
				break
			}
		}
	}

	a.render(w, r, "menu/detail.html", m.Name, map[string]any{
		"Item":    m,
		"Related": related,
		"CatSlug": catSlug,
		"CartQty": cartQty,
	})
}

// menuSearchJSON supports live search suggestions.
func (a *App) menuSearchJSON(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		a.writeJSON(w, http.StatusOK, resp{OK: true, Data: []models.MenuItem{}})
		return
	}
	u := middleware.User(r)
	var userID int64
	if u != nil {
		userID = u.ID
	}
	items, err := a.Menus.List(repositories.MenuFilter{Search: q, UserID: userID, AvailableOnly: false})
	if err != nil {
		a.writeJSON(w, http.StatusInternalServerError, resp{OK: false, Error: "gagal mencari menu"})
		return
	}
	if len(items) > 8 {
		items = items[:8]
	}
	a.writeJSON(w, http.StatusOK, resp{OK: true, Data: items})
}

// favoriteToggle flips the favorite state for the current user.
func (a *App) favoriteToggle(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	if u == nil {
		a.jsonError(w, http.StatusUnauthorized, "Silakan masuk terlebih dahulu.")
		return
	}
	menuID, err := formID(r, "menu_id")
	if err != nil || menuID <= 0 {
		a.jsonError(w, http.StatusBadRequest, "ID menu tidak valid.")
		return
	}
	if _, err := a.Menus.FindByID(menuID, u.ID); err != nil {
		a.jsonError(w, http.StatusNotFound, "Menu tidak ditemukan.")
		return
	}
	nowFav, err := a.Menus.ToggleFavorite(u.ID, menuID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "Gagal memperbarui favorite.")
		return
	}
	if nowFav {
		a.writeJSON(w, http.StatusOK, resp{OK: true, Data: map[string]any{"favorite": true}})
		return
	}
	a.writeJSON(w, http.StatusOK, resp{OK: true, Data: map[string]any{"favorite": false}})
}

// favorites lists the user's saved dishes.
func (a *App) favorites(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	items, err := a.Menus.ListFavorites(u.ID)
	if err != nil {
		a.serverError(w, err)
		return
	}
	a.render(w, r, "menu/favorites.html", "Favorit", map[string]any{
		"Items": items,
	})
}