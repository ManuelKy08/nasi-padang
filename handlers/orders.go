package handlers

import (
	"net/http"

	"github.com/rantau/rantau/middleware"
	"github.com/rantau/rantau/models"
	"github.com/rantau/rantau/repositories"
)

// RegisterOrderRoutes wires order history and tracking.
func (a *App) RegisterOrderRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /orders", a.orderHistory)
	mux.HandleFunc("GET /orders/{id}", a.orderDetail)
}

// orderHistory shows all the current user's orders.
func (a *App) orderHistory(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	orders, err := a.Orders.ListByUser(u.ID, 0)
	if err != nil {
		a.serverError(w, err)
		return
	}
	a.render(w, r, "orders/index.html", "Riwayat Pesanan", map[string]any{
		"Orders": orders,
	})
}

// TrackStep is one stage in the order timeline.
type TrackStep struct {
	Label  string
	Done   bool
	Active bool
}

// orderDetail renders tracking for a single order.
func (a *App) orderDetail(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	order, err := a.Orders.FindByID(id, u.ID)
	if err != nil {
		if err == repositories.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		a.serverError(w, err)
		return
	}

	statusIndex := trackIndex(order.Status)
	steps := trackSteps(order.OrderType)
	for i := range steps {
		if order.Status == models.StatusCancelled {
			break
		}
		if i < statusIndex {
			steps[i].Done = true
		}
		if i == statusIndex {
			steps[i].Active = true
		}
	}

	a.render(w, r, "orders/detail.html", "Lacak Pesanan", map[string]any{
		"Order": order,
		"Steps": steps,
	})
}

func trackSteps(orderType string) []TrackStep {
	if orderType == models.OrderTypeDineIn {
		return []TrackStep{
			{Label: "Pesanan Dibuat"},
			{Label: "Diproses"},
			{Label: "Sedang Dimasak"},
			{Label: "Disajikan"},
			{Label: "Selesai"},
		}
	}
	return []TrackStep{
		{Label: "Pesanan Dibuat"},
		{Label: "Diproses"},
		{Label: "Sedang Dimasak"},
		{Label: "Siap"},
		{Label: "Diantar"},
		{Label: "Selesai"},
	}
}

func trackIndex(status string) int {
	switch status {
	case models.StatusPending:
		return 0
	case models.StatusProcessing:
		return 1
	case models.StatusCooking:
		return 2
	case models.StatusReady:
		return 3
	case models.StatusDelivering:
		return 4
	case models.StatusServed:
		return 3
	case models.StatusCompleted:
		return len(trackSteps(models.OrderTypeTakeAway)) - 1
	case models.StatusCancelled:
		return 0
	}
	return 0
}