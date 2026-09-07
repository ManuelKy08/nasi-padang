package handlers

import (
	"net/http"
	"strconv"

	"github.com/rantau/rantau/middleware"
)

// RegisterCartRoutes wires cart endpoints (asynchronous and page).
func (a *App) RegisterCartRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /cart", a.cartPage)
	mux.HandleFunc("POST /cart/add", a.requireCSRF(a.cartAdd))
	mux.HandleFunc("POST /cart/update", a.requireCSRF(a.cartUpdate))
	mux.HandleFunc("POST /cart/remove", a.requireCSRF(a.cartRemove))
	mux.HandleFunc("POST /cart/clear", a.requireCSRF(a.cartClear))
	mux.HandleFunc("GET /cart/data", a.cartData)
}

// cartData returns the raw cart for the drawer.
func (a *App) cartData(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	if u == nil {
		a.jsonError(w, http.StatusUnauthorized, "Silakan masuk terlebih dahulu.")
		return
	}
	items, err := a.Carts.List(u.ID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "Gagal memuat pesanan.")
		return
	}
	total, err := a.Carts.Subtotal(u.ID)
	if err != nil {
		a.jsonError(w, http.StatusInternalServerError, "Gagal menghitung total.")
		return
	}
	a.writeJSON(w, http.StatusOK, resp{OK: true, Data: map[string]any{
		"items":            items,
		"subtotal":         total,
		"subtotal_formatted": rupiah(total),
		"count":            len(items),
	}})
}

// cartPage renders the full cart view.
func (a *App) cartPage(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	items, err := a.Carts.List(u.ID)
	if err != nil {
		a.serverError(w, err)
		return
	}
	total, err := a.Carts.Subtotal(u.ID)
	if err != nil {
		a.serverError(w, err)
		return
	}
	a.render(w, r, "cart/index.html", "Pesanan Saya", map[string]any{
		"Items":    items,
		"Subtotal": total,
	})
}

// cartAdd adds an item to the cart.
func (a *App) cartAdd(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	if u == nil {
		a.jsonError(w, http.StatusUnauthorized, "Silakan masuk terlebih dahulu.")
		return
	}
	menuID, err := strconv.ParseInt(r.FormValue("menu_id"), 10, 64)
	if err != nil || menuID <= 0 {
		a.jsonError(w, http.StatusBadRequest, "ID menu tidak valid.")
		return
	}
	qty, err := strconv.Atoi(r.FormValue("qty"))
	if err != nil || qty < 1 {
		qty = 1
	}
	if qty > 50 {
		qty = 50
	}

	m, err := a.Menus.FindByID(menuID, u.ID)
	if err != nil || !m.Available {
		a.jsonError(w, http.StatusBadRequest, "Menu sedang tidak tersedia.")
		return
	}

	if err := a.Carts.Add(u.ID, menuID, qty); err != nil {
		a.jsonError(w, http.StatusBadRequest, "Menu sedang tidak tersedia.")
		return
	}

	count, _ := a.Carts.Count(u.ID)
	a.writeJSON(w, http.StatusOK, resp{
		OK: true,
		Data: map[string]any{
			"message": m.Name + " ditambahkan ke pesanan.",
			"count":   count,
		},
	})
}

// cartUpdate changes quantity of an existing line.
func (a *App) cartUpdate(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	itemID, err := formID(r, "item_id")
	if err != nil {
		a.jsonError(w, http.StatusBadRequest, "ID item tidak valid.")
		return
	}
	qty, err := strconv.Atoi(r.FormValue("qty"))
	if err != nil || qty < 1 {
		qty = 1
	}
	if qty > 50 {
		qty = 50
	}
	if err := a.Carts.UpdateQuantity(u.ID, itemID, qty); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "Gagal memperbarui jumlah.")
		return
	}
	total, _ := a.Carts.Subtotal(u.ID)
	count, _ := a.Carts.Count(u.ID)
	a.writeJSON(w, http.StatusOK, resp{OK: true, Data: map[string]any{
		"subtotal":         total,
		"subtotal_formatted": rupiah(total),
		"count":            count,
	}})
}

// cartRemove deletes a line.
func (a *App) cartRemove(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	itemID, err := formID(r, "item_id")
	if err != nil {
		a.jsonError(w, http.StatusBadRequest, "ID item tidak valid.")
		return
	}
	if err := a.Carts.Remove(u.ID, itemID); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "Gagal menghapus item.")
		return
	}
	total, _ := a.Carts.Subtotal(u.ID)
	count, _ := a.Carts.Count(u.ID)
	a.writeJSON(w, http.StatusOK, resp{OK: true, Data: map[string]any{
		"subtotal":         total,
		"subtotal_formatted": rupiah(total),
		"count":            count,
		"message":          "Item dihapus dari pesanan.",
	}})
}

// cartClear empties the cart.
func (a *App) cartClear(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	if err := a.Carts.Clear(u.ID); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "Gagal mengosongkan keranjang.")
		return
	}
	a.writeJSON(w, http.StatusOK, resp{OK: true, Data: map[string]any{
		"count":    0,
		"subtotal": int64(0),
		"subtotal_formatted": rupiah(0),
	}})
}