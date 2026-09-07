package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rantau/rantau/models"
)

// OrderRepository handles orders, order items and reporting statistics.
type OrderRepository struct{ db *sql.DB }

func NewOrderRepository(db *sql.DB) *OrderRepository { return &OrderRepository{db: db} }

// NextOrderNumber computes the sequential daily order number like
// RN-20260907-001.
func (r *OrderRepository) NextOrderNumber(now time.Time) (string, error) {
	var maxNo int
	dayPrefix := now.Format("20060102")
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM orders WHERE order_number LIKE ?`,
		"RN-"+dayPrefix+"-%").Scan(&maxNo)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("RN-%s-%03d", dayPrefix, maxNo+1), nil
}

// Create inserts an order and its items in a transaction, then clears the cart.
func (r *OrderRepository) Create(o *models.Order) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO orders (user_id, order_number, order_type, subtotal, delivery_fee,
			total, payment_method, status, address, table_number, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())`,
		o.UserID, o.OrderNumber, o.OrderType, o.Subtotal, o.DeliveryFee,
		o.Total, o.PaymentMethod, o.Status, o.Address, o.TableNumber, o.Notes)
	if err != nil {
		return err
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	o.ID = orderID

	for _, it := range o.Items {
		if _, err := tx.Exec(`
			INSERT INTO order_items (order_id, menu_item_id, quantity, price)
			VALUES (?, ?, ?, ?)`,
			o.ID, it.MenuItemID, it.Quantity, it.Price); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`DELETE FROM carts WHERE user_id = ?`, o.UserID); err != nil {
		return err
	}

	return tx.Commit()
}

const orderSelect = `
	SELECT o.id, o.user_id, u.name, o.order_number, o.order_type, o.subtotal,
	       o.delivery_fee, o.total, o.payment_method, o.status,
	       COALESCE(o.address, ''), COALESCE(o.table_number, ''),
	       COALESCE(o.notes, ''), o.created_at
	FROM orders o JOIN users u ON u.id = o.user_id`

func (r *OrderRepository) scanOrder(sc interface{ Scan(...any) error }) (models.Order, error) {
	var o models.Order
	err := sc.Scan(&o.ID, &o.UserID, &o.UserName, &o.OrderNumber, &o.OrderType, &o.Subtotal,
		&o.DeliveryFee, &o.Total, &o.PaymentMethod, &o.Status, &o.Address,
		&o.TableNumber, &o.Notes, &o.CreatedAt)
	return o, err
}

func (r *OrderRepository) loadItems(o *models.Order) error {
	rows, err := r.db.Query(`
		SELECT oi.id, oi.order_id, oi.menu_item_id, m.name, oi.quantity, oi.price
		FROM order_items oi JOIN menu_items m ON m.id = oi.menu_item_id
		WHERE oi.order_id = ? ORDER BY oi.id`, o.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var it models.OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.MenuItemID, &it.MenuName, &it.Quantity, &it.Price); err != nil {
			return err
		}
		o.Items = append(o.Items, it)
	}
	return rows.Err()
}

// FindByID returns an order. Pass userID=0 to bypass ownership (admin).
func (r *OrderRepository) FindByID(id int64, userID int64) (*models.Order, error) {
	var o models.Order
	var err error
	if userID > 0 {
		o, err = r.scanOrder(r.db.QueryRow(orderSelect+` WHERE o.id = ? AND o.user_id = ?`, id, userID))
	} else {
		o, err = r.scanOrder(r.db.QueryRow(orderSelect+` WHERE o.id = ?`, id))
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadItems(&o); err != nil {
		return nil, err
	}
	return &o, nil
}

// FindByNumber returns an order by its RN-number.
func (r *OrderRepository) FindByNumber(number string, userID int64) (*models.Order, error) {
	var o models.Order
	var err error
	if userID > 0 {
		o, err = r.scanOrder(r.db.QueryRow(orderSelect+` WHERE o.order_number = ? AND o.user_id = ?`, number, userID))
	} else {
		o, err = r.scanOrder(r.db.QueryRow(orderSelect+` WHERE o.order_number = ?`, number))
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadItems(&o); err != nil {
		return nil, err
	}
	return &o, nil
}

// ListByUser returns the order history for a user (newest first).
func (r *OrderRepository) ListByUser(userID int64, limit int) ([]models.Order, error) {
	q := orderSelect + ` WHERE o.user_id = ? ORDER BY o.created_at DESC`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := r.db.Query(q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Order
	for rows.Next() {
		o, err := r.scanOrder(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// ListAll returns every order, optionally filtered by status (admin).
func (r *OrderRepository) ListAll(status string, limit int) ([]models.Order, error) {
	q := orderSelect
	args := []any{}
	if status != "" && status != "all" {
		q += ` WHERE o.status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY o.created_at DESC`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Order
	for rows.Next() {
		o, err := r.scanOrder(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// UpdateStatus changes an order's status.
func (r *OrderRepository) UpdateStatus(id int64, status string) error {
	_, err := r.db.Exec(`UPDATE orders SET status = ? WHERE id = ?`, status, id)
	return err
}

// Stats ---------------------------------------------------------------------

type DayRevenue struct {
	Day     string
	Revenue int64
}

// RevenueLastNDays returns revenue grouped by day for charts (newest-last).
func (r *OrderRepository) RevenueLastNDays(n int) ([]DayRevenue, error) {
	rows, err := r.db.Query(`
		SELECT DATE(created_at) d, COALESCE(SUM(total), 0)
		FROM orders
		WHERE created_at >= CURDATE() - INTERVAL ? DAY AND status <> 'cancelled'
		GROUP BY DATE(created_at) ORDER BY d`, n-1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]DayRevenue, 0, n)
	for rows.Next() {
		var dr DayRevenue
		var d time.Time
		if err := rows.Scan(&d, &dr.Revenue); err != nil {
			return nil, err
		}
		dr.Day = d.Format("2006-01-02")
		list = append(list, dr)
	}
	return list, rows.Err()
}

func (r *OrderRepository) revenueBetween(from, to string) (int64, error) {
	var total int64
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(total), 0) FROM orders
		WHERE created_at >= ? AND created_at < ? AND status <> 'cancelled'`, from, to).Scan(&total)
	return total, err
}

// RevenueToday returns today's turnover.
func (r *OrderRepository) RevenueToday() (int64, error) {
	return r.revenueBetween("CURDATE()", "CURDATE() + INTERVAL 1 DAY")
}

// RevenueWeek returns the last 7 days' turnover.
func (r *OrderRepository) RevenueWeek() (int64, error) {
	return r.revenueBetween("CURDATE() - INTERVAL 6 DAY", "CURDATE() + INTERVAL 1 DAY")
}

// RevenueMonth returns this calendar month's turnover.
func (r *OrderRepository) RevenueMonth() (int64, error) {
	return r.revenueBetween("DATE_FORMAT(CURDATE(), '%Y-%m-01')", "DATE_FORMAT(CURDATE() + INTERVAL 1 MONTH, '%Y-%m-01')")
}

// OrdersToday counts orders created today.
func (r *OrderRepository) OrdersToday() (int64, error) {
	var n int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM orders WHERE created_at >= CURDATE()`).Scan(&n)
	return n, err
}

// ActiveOrders counts orders still being processed.
func (r *OrderRepository) ActiveOrders() (int64, error) {
	var n int64
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM orders
		WHERE status IN ('pending','processing','cooking','ready','delivering','served')`).Scan(&n)
	return n, err
}

// CountTotal counts all non-cancelled orders.
func (r *OrderRepository) CountTotal() (int64, error) {
	var n int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM orders WHERE status <> 'cancelled'`).Scan(&n)
	return n, err
}

// AverageOrderValue returns total revenue / total orders.
func (r *OrderRepository) AverageOrderValue() (int64, error) {
	var avg sql.NullFloat64
	err := r.db.QueryRow(`
		SELECT AVG(total) FROM orders WHERE status <> 'cancelled'`).Scan(&avg)
	if err != nil {
		return 0, err
	}
	return int64(avg.Float64), err
}

// TopMenu lists the most sold items.
func (r *OrderRepository) TopMenu(limit int) ([]models.MenuItem, error) {
	rows, err := r.db.Query(`
		SELECT m.id, oi.menu_item_id, c.name, m.name, m.slug, m.description, m.price,
		       m.image, m.available, MAX(o.created_at), 0, COALESCE(SUM(oi.quantity), 0)
		FROM order_items oi
		JOIN menu_items m ON m.id = oi.menu_item_id
		JOIN categories c ON c.id = m.category_id
		JOIN orders o ON o.id = oi.order_id AND o.status <> 'cancelled'
		GROUP BY oi.menu_item_id
		ORDER BY 12 DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.MenuItem
	for rows.Next() {
		var m models.MenuItem
		var menuItemID int64
		if err := rows.Scan(&m.ID, &menuItemID, &m.CategoryName, &m.Name, &m.Slug,
			&m.Description, &m.Price, &m.Image, &m.Available, &m.CreatedAt, &m.IsFavorite, &m.TotalQty); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// TopCategories lists categories by quantity sold.
func (r *OrderRepository) TopCategories(limit int) ([]struct {
	Category string
	Quantity int
}, error) {
	rows, err := r.db.Query(`
		SELECT c.name, COALESCE(SUM(oi.quantity), 0)
		FROM order_items oi
		JOIN menu_items m ON m.id = oi.menu_item_id
		JOIN categories c ON c.id = m.category_id
		JOIN orders o ON o.id = oi.order_id AND o.status <> 'cancelled'
		GROUP BY c.id ORDER BY 2 DESC, c.name LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []struct {
		Category string
		Quantity int
	}
	for rows.Next() {
		var c string
		var n int
		if err := rows.Scan(&c, &n); err != nil {
			return nil, err
		}
		out = append(out, struct {
			Category string
			Quantity int
		}{c, n})
	}
	return out, rows.Err()
}

// StatusDistribution counts orders per status for admin.
func (r *OrderRepository) StatusDistribution() (map[string]int64, error) {
	rows, err := r.db.Query(`SELECT status, COUNT(*) FROM orders GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int64{}
	for rows.Next() {
		var s string
		var n int64
		if err := rows.Scan(&s, &n); err != nil {
			return nil, err
		}
		out[s] = n
	}
	return out, rows.Err()
}

// UserStats returns an individual user's ordering statistics.
func (r *OrderRepository) UserStats(userID int64) (count int, total int64, err error) {
	err = r.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(total), 0) FROM orders
		WHERE user_id = ? AND status <> 'cancelled'`, userID).Scan(&count, &total)
	return
}