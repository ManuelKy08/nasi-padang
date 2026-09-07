// Package handlers wires HTTP routes to application logic.
package handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/rantau/rantau/config"
	"github.com/rantau/rantau/middleware"
	"github.com/rantau/rantau/models"
	"github.com/rantau/rantau/repositories"
)

// App bundles dependencies for every handler.
type App struct {
	DB    *sql.DB
	Store *sessions.CookieStore
	Cfg   config.Config
	Tpl   map[string]view

	Users  *repositories.UserRepository
	Menus  *repositories.MenuRepository
	Carts  *repositories.CartRepository
	Orders *repositories.OrderRepository
}

// view is the compiled template set for one page.
type view struct {
	Tpl    *template.Template
	Layout string
}

// pages pairs each page template with the layout that frames it. Templates are
// compiled per page (not as one giant set) so that every page can define its
// own "content" block without name clashes.
var pages = []struct{ file, layout string }{
	{"auth/login.html", "base_auth"},
	{"dashboard/index.html", "base"},
	{"about/index.html", "base"},
	{"menu/index.html", "base"},
	{"menu/detail.html", "base"},
	{"menu/favorites.html", "base"},
	{"cart/index.html", "base"},
	{"checkout/index.html", "base"},
	{"checkout/receipt.html", "base"},
	{"orders/index.html", "base"},
	{"orders/detail.html", "base"},
	{"profile/index.html", "base"},
	{"admin/index.html", "base"},
	{"admin/menu.html", "base"},
	{"admin/orders.html", "base"},
	{"admin/users.html", "base"},
	{"admin/reports.html", "base"},
}

// NewApp creates the app, connecting the database and compiling templates.
func NewApp(cfg config.Config, db *sql.DB, store *sessions.CookieStore) (*App, error) {
	app := &App{
		DB:     db,
		Store:  store,
		Cfg:    cfg,
		Users:  repositories.NewUserRepository(db),
		Menus:  repositories.NewMenuRepository(db),
		Carts:  repositories.NewCartRepository(db),
		Orders: repositories.NewOrderRepository(db),
	}

	tpl, err := parseTemplates("templates")
	if err != nil {
		return nil, err
	}
	app.Tpl = tpl
	return app, nil
}

func parseTemplates(root string) (map[string]view, error) {
	funcs := template.FuncMap{
		"rupiah":        rupiah,
		"moneyNoRp":     moneyNoRp,
		"datetime":      formatDateTime,
		"dateShort":     formatDateShort,
		"statusLabel":   models.StatusLabel,
		"statusClass":   statusClass,
		"orderTypeLabel": models.OrderTypeLabel,
		"payLabel":      models.PaymentLabel,
		"sub":       func(a, b any) int64 { return anyToInt64(a) - anyToInt64(b) },
		"mul":       func(a, b any) int64 { return anyToInt64(a) * anyToInt64(b) },
		"addInt":    func(a, b any) int { return int(anyToInt64(a) + anyToInt64(b)) },
		"initial":   initialOf,
		"dict":      dict,
		"json":      toJSON,
	}

	baseParts := []string{
		"layouts/base.html",
		"layouts/navbar.html",
		"layouts/footer.html",
		"layouts/cart.html",
		"layouts/toasts.html",
		"layouts/menu_partials.html",
	}

	out := make(map[string]view, len(pages))
	for _, p := range pages {
		parts := baseParts
		if p.layout == "base_auth" {
			parts = []string{"layouts/base_auth.html", "layouts/toasts.html"}
		}
		paths := make([]string, 0, len(parts)+1)
		for _, f := range parts {
			paths = append(paths, root+"/"+f)
		}
		paths = append(paths, root+"/"+p.file)

		tt, err := template.New("page").Funcs(funcs).ParseFiles(paths...)
		if err != nil {
			return nil, err
		}
		out[p.file] = view{Tpl: tt, Layout: p.layout}
	}
	return out, nil
}

func dict(values ...interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	for i := 0; i+1 < len(values); i += 2 {
		m[values[i].(string)] = values[i+1]
	}
	return m
}

// initialOf returns the uppercase first letter of a name for avatars.
func initialOf(name string) string {
	for _, r := range name {
		return strings.ToUpper(string(r))
	}
	return "R"
}

// toJSON marshals a value into a template-safe JSON literal.
func toJSON(v any) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return template.JS("[]")
	}
	return template.JS(b)
}

// rupiah formats an integer Rupiah value with thousand separators.
func rupiah(n int64) string {
	return "Rp " + moneyNoRp(n)
}

func moneyNoRp(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	groups := 0
	for i := len(s) - 1; i >= 0; i-- {
		if groups == 3 {
			b.WriteString(".")
			groups = 0
		}
		b.WriteByte(s[i])
		groups++
	}
	out := reverse(b.String())
	if neg {
		out = "-" + out
	}
	return out
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// anyToInt64 coerces the integer-ish values templates pass around into int64.
func anyToInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case uint:
		return int64(n)
	case uint64:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func formatDateTime(t time.Time) string {
	return t.Format("02 Jan 2006 · 15:04")
}

func formatDateShort(t time.Time) string {
	return t.Format("02 Jan 2006")
}

// statusClass maps an order status to a CSS badge class.
func statusClass(s string) string {
	switch s {
	case "pending":
		return "st-pending"
	case "processing":
		return "st-processing"
	case "cooking":
		return "st-cooking"
	case "ready":
		return "st-ready"
	case "served":
		return "st-served"
	case "delivering":
		return "st-delivering"
	case "completed":
		return "st-completed"
	case "cancelled":
		return "st-cancelled"
	}
	return "st-pending"
}

// render composes common view data and executes the layout for a page.
// page selects the compiled template set (e.g. "dashboard/index.html").
func (a *App) render(w http.ResponseWriter, r *http.Request, page, title string, data map[string]any) {
	v, ok := a.Tpl[page]
	if !ok {
		http.Error(w, "view not found", http.StatusInternalServerError)
		return
	}

	d := make(map[string]any, len(data)+8)
	for k, val := range data {
		d[k] = val
	}

	d["PageTitle"] = title
	d["CSRF"] = middleware.SetCSRFToken(a.Store, w, r)
	d["User"] = middleware.User(r)

	flashOK, err := a.popFlash(w, r, "flash_ok")
	if err == nil {
		d["FlashOk"] = flashOK
	}
	fe, _ := a.popFlash(w, r, "flash_err")
	d["FlashErr"] = fe

	cartCount := 0
	if u := middleware.User(r); u != nil {
		if n, err := a.Carts.Count(u.ID); err == nil {
			cartCount = n
		}
	}
	d["CartCount"] = cartCount
	d["OpenStatus"] = a.openNow()
	d["OpenHour"] = a.Cfg.OpenHour
	d["CloseHour"] = a.Cfg.CloseHour
	d["Path"] = r.URL.Path

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := v.Tpl.ExecuteTemplate(w, v.Layout, d); err != nil {
		log.Printf("render %q: %v", v.Layout, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

// openNow reports whether the kitchen is open based on server clock.
func (a *App) openNow() bool {
	h := time.Now().Hour()
	return h >= a.Cfg.OpenHour && h < a.Cfg.CloseHour
}

// Flash helpers ----------------------------------------------------------

func (a *App) setFlash(w http.ResponseWriter, r *http.Request, key, msg string) {
	sess, err := a.Store.Get(r, middleware.SessionName)
	if err != nil {
		return
	}
	sess.Values[key] = msg
	_ = sess.Save(r, w)
}

func (a *App) popFlash(w http.ResponseWriter, r *http.Request, key string) (string, error) {
	sess, err := a.Store.Get(r, middleware.SessionName)
	if err != nil {
		return "", err
	}
	msg, _ := sess.Values[key].(string)
	delete(sess.Values, key)
	_ = sess.Save(r, w)
	return msg, nil
}

// JSON helpers -------------------------------------------------------------

type resp struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Data  any    `json:"data,omitempty"`
}

func (a *App) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

func (a *App) jsonError(w http.ResponseWriter, status int, msg string) {
	a.writeJSON(w, status, resp{OK: false, Error: msg})
}

// Parse helpers ------------------------------------------------------------

func parseID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func formID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.FormValue(name), 10, 64)
}

// requireCSRF rejects POSTs without a valid CSRF token.
func (a *App) requireCSRF(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := r.FormValue("csrf_token")
		if tok == "" {
			tok = r.Header.Get("X-CSRF-Token")
		}
		if !middleware.ValidateCSRF(a.Store, r, tok) {
			http.Error(w, "CSRF token tidak valid.", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}