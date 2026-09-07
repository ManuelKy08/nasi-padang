package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/rantau/rantau/middleware"
	"github.com/rantau/rantau/models"
	"github.com/rantau/rantau/repositories"
)

// RegisterCheckoutRoutes wires checkout and receipt flow.
func (a *App) RegisterCheckoutRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /checkout", a.checkoutPage)
	mux.HandleFunc("POST /checkout", a.requireCSRF(a.checkoutCreate))
	mux.HandleFunc("GET /receipt/{number}", a.receiptPage)
}

// checkoutPage shows the order summary + step widget.
func (a *App) checkoutPage(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	items, err := a.Carts.List(u.ID)
	if err != nil {
		a.serverError(w, err)
		return
	}
	if len(items) == 0 {
		a.setFlash(w, r, "flash_err", "Pesanan kamu masih kosong.")
		http.Redirect(w, r, "/menu", http.StatusSeeOther)
		return
	}
	subtotal, err := a.Carts.Subtotal(u.ID)
	if err != nil {
		a.serverError(w, err)
		return
	}

	a.render(w, r, "checkout/index.html", "Checkout", map[string]any{
		"Items":       items,
		"Subtotal":    subtotal,
		"DeliveryFee": a.Cfg.DeliveryFee,
		"User":        u,
	})
}

// checkoutCreate validates input, creates the order and clears the cart.
func (a *App) checkoutCreate(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	items, err := a.Carts.List(u.ID)
	if err != nil {
		a.serverError(w, err)
		return
	}
	if len(items) == 0 {
		a.setFlash(w, r, "flash_err", "Pesanan kamu masih kosong.")
		http.Redirect(w, r, "/menu", http.StatusSeeOther)
		return
	}

	orderType := r.FormValue("order_type")
	payment := r.FormValue("payment_method")
	if !oneOf(orderType, models.OrderTypeDineIn, models.OrderTypeTakeAway, models.OrderTypeDelivery) {
		a.setFlash(w, r, "flash_err", "Tipe pesanan tidak valid.")
		http.Redirect(w, r, "/checkout", http.StatusSeeOther)
		return
	}
	if !oneOf(payment, models.PayCash, models.PayBank, models.PayEWallet) {
		a.setFlash(w, r, "flash_err", "Metode pembayaran tidak valid.")
		http.Redirect(w, r, "/checkout", http.StatusSeeOther)
		return
	}

	table := strings.TrimSpace(r.FormValue("table_number"))
	pickupName := strings.TrimSpace(r.FormValue("pickup_name"))
	pickupPhone := strings.TrimSpace(r.FormValue("pickup_phone"))
	address := strings.TrimSpace(r.FormValue("address"))
	notes := strings.TrimSpace(r.FormValue("notes"))

	switch orderType {
	case models.OrderTypeDineIn:
		if table == "" {
			a.setFlash(w, r, "flash_err", "Nomor meja wajib diisi.")
			http.Redirect(w, r, "/checkout", http.StatusSeeOther)
			return
		}
	case models.OrderTypeTakeAway:
		if pickupName == "" || !phoneRe.MatchString(pickupPhone) {
			a.setFlash(w, r, "flash_err", "Nama pemesan dan nomor HP yang valid wajib diisi.")
			http.Redirect(w, r, "/checkout", http.StatusSeeOther)
			return
		}
	case models.OrderTypeDelivery:
		if pickupName == "" || !phoneRe.MatchString(pickupPhone) || address == "" {
			a.setFlash(w, r, "flash_err", "Nama, nomor HP, dan alamat wajib diisi untuk delivery.")
			http.Redirect(w, r, "/checkout", http.StatusSeeOther)
			return
		}
	}

	var subtotal int64
	var orderItems []models.OrderItem
	for _, it := range items {
		if !it.Menu.Available {
			a.setFlash(w, r, "flash_err", "Menu "+it.Menu.Name+" sedang tidak tersedia.")
			http.Redirect(w, r, "/cart", http.StatusSeeOther)
			return
		}
		subtotal += it.Menu.Price * int64(it.Quantity)
		orderItems = append(orderItems, models.OrderItem{
			OrderID:    0,
			MenuItemID: it.MenuItemID,
			MenuName:   it.Menu.Name,
			Quantity:   it.Quantity,
			Price:      it.Menu.Price,
		})
	}

	deliveryFee := int64(0)
	if orderType == models.OrderTypeDelivery {
		deliveryFee = a.Cfg.DeliveryFee
	}

	number, err := a.Orders.NextOrderNumber(time.Now())
	if err != nil {
		a.serverError(w, err)
		return
	}

	order := &models.Order{
		UserID:        u.ID,
		OrderNumber:   number,
		OrderType:     orderType,
		Subtotal:      subtotal,
		DeliveryFee:   deliveryFee,
		Total:         subtotal + deliveryFee,
		PaymentMethod: payment,
		Status:        models.StatusPending,
		Address:       address,
		TableNumber:   table,
		Notes:         notes,
		Items:         orderItems,
	}

	if err := a.Orders.Create(order); err != nil {
		a.serverError(w, err)
		return
	}

	a.setFlash(w, r, "flash_ok", "Pesanan "+number+" berhasil dibuat.")
	http.Redirect(w, r, "/receipt/"+number, http.StatusSeeOther)
}

// receiptPage renders the printable digital receipt.
func (a *App) receiptPage(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	number := r.PathValue("number")
	order, err := a.Orders.FindByNumber(number, u.ID)
	if err != nil {
		if err == repositories.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		a.serverError(w, err)
		return
	}
	a.render(w, r, "checkout/receipt.html", "Struk Pesanan", map[string]any{
		"Order": order,
		"Items": order.Items,
	})
}

func oneOf(v string, opts ...string) bool {
	for _, o := range opts {
		if v == o {
			return true
		}
	}
	return false
}