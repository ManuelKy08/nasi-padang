package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rantau/rantau/middleware"
	"github.com/rantau/rantau/models"
	"github.com/rantau/rantau/repositories"
)

// RegisterAdminRoutes wires every admin screen (protected by middleware).
func (a *App) RegisterAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin", a.adminDashboard)
	mux.HandleFunc("GET /admin/menu", a.adminMenu)
	mux.HandleFunc("POST /admin/menu/create", a.requireCSRF(a.adminMenuCreate))
	mux.HandleFunc("POST /admin/menu/edit", a.requireCSRF(a.adminMenuEdit))
	mux.HandleFunc("POST /admin/menu/delete", a.requireCSRF(a.adminMenuDelete))
	mux.HandleFunc("POST /admin/menu/toggle", a.requireCSRF(a.adminMenuToggle))
	mux.HandleFunc("GET /admin/orders", a.adminOrders)
	mux.HandleFunc("POST /admin/orders/{id}/status", a.requireCSRF(a.adminOrderStatus))
	mux.HandleFunc("POST /admin/orders/{id}/delete", a.requireCSRF(a.adminOrderDelete))
	mux.HandleFunc("GET /admin/users", a.adminUsers)
	mux.HandleFunc("POST /admin/users/{id}/role", a.requireCSRF(a.adminUserRole))
	mux.HandleFunc("POST /admin/users/{id}/delete", a.requireCSRF(a.adminUserDelete))
	mux.HandleFunc("GET /admin/reports", a.adminReports)
}

// adminDashboard renders headline statistics.
func (a *App) adminDashboard(w http.ResponseWriter, r *http.Request) {
	menuCount, _ := a.Menus.Count()
	customerCount, err := a.Users.Count()
	if err != nil {
		a.serverError(w, err)
		return
	}
	ordersToday, _ := a.Orders.OrdersToday()
	active, _ := a.Orders.ActiveOrders()
	revenueToday, _ := a.Orders.RevenueToday()
	chart, _ := a.Orders.RevenueLastNDays(7)
	topMenu, _ := a.Orders.TopMenu(5)

	a.render(w, r, "admin/index.html", "Admin", map[string]any{
		"MenuCount":     menuCount,
		"CustomerCount": customerCount,
		"OrdersToday":   ordersToday,
		"Active":        active,
		"RevenueToday":  revenueToday,
		"Chart":         chart,
		"TopMenu":       topMenu,
	})
}

// adminMenu lists items with create/edit forms.
func (a *App) adminMenu(w http.ResponseWriter, r *http.Request) {
	items, err := a.Menus.List(repositories.MenuFilter{})
	if err != nil {
		a.serverError(w, err)
		return
	}
	cats, err := a.Menus.ListCategories()
	if err != nil {
		a.serverError(w, err)
		return
	}

	var editing *models.MenuItem
	if id, err := strconv.ParseInt(r.URL.Query().Get("edit"), 10, 64); err == nil {
		if m, err := a.Menus.FindByID(id, 0); err == nil {
			editing = m
		}
	}

	a.render(w, r, "admin/menu.html", "Manajemen Menu", map[string]any{
		"Items":      items,
		"Categories": cats,
		"Editing":    editing,
	})
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, rn := range s {
		switch {
		case rn >= 'a' && rn <= 'z', rn >= '0' && rn <= '9':
			b.WriteRune(rn)
		case rn == ' ' || rn == '-' || rn == '/' || rn == '_':
			b.WriteByte('-')
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return out
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// uploadImage stores an uploaded photo into static/images/menu.
func (a *App) uploadImage(img multipart.File, header *multipart.FileHeader) (string, error) {
	if header.Size > 5*1024*1024 {
		return "", fmt.Errorf("file terlalu besar")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".svg":
	default:
		return "", fmt.Errorf("format gambar tidak didukung")
	}

	dir := filepath.Join("static", "images", "menu")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	name := "menu-" + randomHex(8) + ext
	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, img); err != nil {
		return "", err
	}
	return "/static/images/menu/" + name, nil
}

func (a *App) adminMenuCreate(w http.ResponseWriter, r *http.Request) {
	catID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	if err != nil {
		a.setFlash(w, r, "flash_err", "Kategori tidak valid.")
		http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	desc := strings.TrimSpace(r.FormValue("description"))
	price, err := strconv.ParseInt(strings.ReplaceAll(r.FormValue("price"), ".", ""), 10, 64)
	if err != nil || price < 100 || len(name) < 2 {
		a.setFlash(w, r, "flash_err", "Data menu tidak valid.")
		http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
		return
	}

	img := "/static/images/menu/placeholder.svg"
	if f, h, err := r.FormFile("image"); err == nil {
		defer f.Close()
		if p, err := a.uploadImage(f, h); err == nil {
			img = p
		} else {
			log.Printf("upload: %v", err)
		}
	}

	available := r.FormValue("available") == "1"
	slug := slugify(name)
	if n := 1; true {
		test := slug
		for {
			exists, err := a.Menus.SlugExists(test, 0)
			if err != nil || !exists {
				slug = test
				break
			}
			test = fmt.Sprintf("%s-%d", slug, n)
			n++
		}
	}

	_, err = a.Menus.Create(&models.MenuItem{
		CategoryID: catID, Name: name, Slug: slug, Description: desc,
		Price: price, Image: img, Available: available,
	})
	if err != nil {
		a.setFlash(w, r, "flash_err", "Gagal membuat menu.")
	} else {
		a.setFlash(w, r, "flash_ok", name+" berhasil ditambahkan.")
	}
	http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
}

func (a *App) adminMenuEdit(w http.ResponseWriter, r *http.Request) {
	id, err := formID(r, "id")
	if err != nil {
		a.setFlash(w, r, "flash_err", "ID menu tidak valid.")
		http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
		return
	}
	catID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	price, _ := strconv.ParseInt(strings.ReplaceAll(r.FormValue("price"), ".", ""), 10, 64)
	name := strings.TrimSpace(r.FormValue("name"))
	if err != nil || price < 100 || len(name) < 2 {
		a.setFlash(w, r, "flash_err", "Data menu tidak valid.")
		http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
		return
	}

	existing, err := a.Menus.FindByID(id, 0)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	img := existing.Image
	if f, h, err := r.FormFile("image"); err == nil {
		defer f.Close()
		if p, err := a.uploadImage(f, h); err == nil {
			img = p
		}
	}

	existing.CategoryID = catID
	existing.Name = name
	existing.Description = strings.TrimSpace(r.FormValue("description"))
	existing.Price = price
	existing.Slug = slugify(name)
	existing.Image = img
	existing.Available = r.FormValue("available") == "1"

	if err := a.Menus.Update(existing); err != nil {
		a.setFlash(w, r, "flash_err", "Gagal memperbarui menu.")
	} else {
		a.setFlash(w, r, "flash_ok", "Menu berhasil diperbarui.")
	}
	http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
}

func (a *App) adminMenuToggle(w http.ResponseWriter, r *http.Request) {
	id, err := formID(r, "id")
	if err != nil {
		a.jsonError(w, http.StatusBadRequest, "ID tidak valid.")
		return
	}
	m, err := a.Menus.FindByID(id, 0)
	if err != nil {
		a.jsonError(w, http.StatusNotFound, "Menu tidak ditemukan.")
		return
	}
	if err := a.Menus.SetAvailable(id, !m.Available); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "Gagal mengubah status.")
		return
	}
	msg := m.Name + " kini tersedia."
	if m.Available {
		msg = m.Name + " ditandai tidak tersedia."
	}
	a.writeJSON(w, http.StatusOK, resp{OK: true, Data: map[string]any{"message": msg, "available": !m.Available}})
}

func (a *App) adminMenuDelete(w http.ResponseWriter, r *http.Request) {
	id, err := formID(r, "id")
	if err != nil {
		a.setFlash(w, r, "flash_err", "ID tidak valid.")
		http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
		return
	}
	if err := a.Menus.Delete(id); err != nil {
		a.setFlash(w, r, "flash_err", "Gagal menghapus menu.")
	} else {
		a.setFlash(w, r, "flash_ok", "Menu dihapus.")
	}
	http.Redirect(w, r, "/admin/menu", http.StatusSeeOther)
}

// adminOrders shows all orders with status filter.
func (a *App) adminOrders(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	orders, err := a.Orders.ListAll(status, 0)
	if err != nil {
		a.serverError(w, err)
		return
	}
	a.render(w, r, "admin/orders.html", "Semua Pesanan", map[string]any{
		"Orders":  orders,
		"Status":  status,
		"Filters": []string{"all", "pending", "processing", "cooking", "ready", "delivering", "served", "completed", "cancelled"},
	})
}

func (a *App) adminOrderStatus(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	_ = u
	id, err := parseID(r, "id")
	if err != nil {
		a.jsonError(w, http.StatusBadRequest, "ID tidak valid.")
		return
	}
	status := r.FormValue("status")
	if !oneOf(status, models.StatusPending, models.StatusProcessing, models.StatusCooking,
		models.StatusReady, models.StatusDelivering, models.StatusServed,
		models.StatusCompleted, models.StatusCancelled) {
		a.jsonError(w, http.StatusBadRequest, "Status tidak valid.")
		return
	}
	if err := a.Orders.UpdateStatus(id, status); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "Gagal memperbarui status.")
		return
	}
	a.writeJSON(w, http.StatusOK, resp{OK: true, Data: map[string]any{
		"message": "Status pesanan diperbarui.",
		"status":  status,
	}})
}

func (a *App) adminOrderDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		a.setFlash(w, r, "flash_err", "ID tidak valid.")
		http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
		return
	}
	// Remove dependent rows then the order itself.
	if _, err := a.DB.Exec(`DELETE FROM order_items WHERE order_id = ?`, id); err != nil {
		a.setFlash(w, r, "flash_err", "Gagal menghapus pesanan.")
	} else if _, err := a.DB.Exec(`DELETE FROM orders WHERE id = ?`, id); err != nil {
		a.setFlash(w, r, "flash_err", "Gagal menghapus pesanan.")
	} else {
		a.setFlash(w, r, "flash_ok", "Pesanan dihapus.")
	}
	http.Redirect(w, r, "/admin/orders?status="+r.URL.Query().Get("status"), http.StatusSeeOther)
}

// adminUsers lists customers and lets admin adjust roles.
func (a *App) adminUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.Users.List()
	if err != nil {
		a.serverError(w, err)
		return
	}
	a.render(w, r, "admin/users.html", "Manajemen Pengguna", map[string]any{
		"Users": users,
	})
}

func (a *App) adminUserRole(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		a.jsonError(w, http.StatusBadRequest, "ID tidak valid.")
		return
	}
	role := r.FormValue("role")
	if !oneOf(role, models.RoleCustomer, models.RoleAdmin) {
		a.jsonError(w, http.StatusBadRequest, "Role tidak valid.")
		return
	}
	if err := a.Users.UpdateRole(id, role); err != nil {
		a.jsonError(w, http.StatusInternalServerError, "Gagal memperbarui role.")
		return
	}
	a.writeJSON(w, http.StatusOK, resp{OK: true, Data: map[string]any{"message": "Role diperbarui."}})
}

func (a *App) adminUserDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		a.setFlash(w, r, "flash_err", "ID tidak valid.")
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}
	if err := a.Users.Delete(id); err != nil {
		a.setFlash(w, r, "flash_err", "Gagal menghapus pengguna.")
	} else {
		a.setFlash(w, r, "flash_ok", "Pengguna dihapus.")
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// adminReports shows revenue analytics.
func (a *App) adminReports(w http.ResponseWriter, r *http.Request) {
	today, _ := a.Orders.RevenueToday()
	week, _ := a.Orders.RevenueWeek()
	month, _ := a.Orders.RevenueMonth()
	ordersToday, _ := a.Orders.OrdersToday()
	totalOrders, _ := a.Orders.CountTotal()
	avg, _ := a.Orders.AverageOrderValue()
	topMenu, _ := a.Orders.TopMenu(5)
	topCat, _ := a.Orders.TopCategories(5)
	chart, _ := a.Orders.RevenueLastNDays(7)
	dist, _ := a.Orders.StatusDistribution()

	a.render(w, r, "admin/reports.html", "Laporan Penjualan", map[string]any{
		"Today":      today,
		"Week":       week,
		"Month":      month,
		"OrdersToday": ordersToday,
		"TotalOrders": totalOrders,
		"Avg":        avg,
		"TopMenu":    topMenu,
		"TopCat":     topCat,
		"Chart":      chart,
		"Dist":       dist,
	})
}