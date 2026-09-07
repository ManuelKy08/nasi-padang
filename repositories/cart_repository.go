package repositories

import (
	"database/sql"

	"github.com/rantau/rantau/models"
)

// CartRepository handles the user's cart.
type CartRepository struct{ db *sql.DB }

func NewCartRepository(db *sql.DB) *CartRepository { return &CartRepository{db: db} }

// Add upserts a cart line. Keeps only available items.
func (r *CartRepository) Add(userID, menuID int64, qty int) error {
	var available bool
	if err := r.db.QueryRow(`SELECT available FROM menu_items WHERE id = ?`, menuID).Scan(&available); err != nil {
		return err
	}
	if !available {
		return sql.ErrNoRows
	}

	for i := 0; i < 2; i++ {
		res, err := r.db.Exec(
			`INSERT INTO carts (user_id, menu_item_id, quantity, created_at)
			 VALUES (?, ?, ?, NOW())
			 ON DUPLICATE KEY UPDATE quantity = quantity + ?`,
			userID, menuID, qty, qty)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n >= 0 {
			// Cap sanity: never allow negative stock weirdness.
			_, _ = r.db.Exec(`UPDATE carts SET quantity = GREATEST(quantity, 1) WHERE user_id = ? AND menu_item_id = ?`, userID, menuID)
			return nil
		}
	}
	return nil
}

// UpdateQuantity sets the quantity for a cart line owned by the user.
func (r *CartRepository) UpdateQuantity(userID, cartID int64, qty int) error {
	if qty < 1 {
		return r.Remove(userID, cartID)
	}
	_, err := r.db.Exec(`UPDATE carts SET quantity = ? WHERE id = ? AND user_id = ?`, qty, cartID, userID)
	return err
}

// Remove deletes a cart line.
func (r *CartRepository) Remove(userID, cartID int64) error {
	_, err := r.db.Exec(`DELETE FROM carts WHERE id = ? AND user_id = ?`, cartID, userID)
	return err
}

// Clear empties the user's cart.
func (r *CartRepository) Clear(userID int64) error {
	_, err := r.db.Exec(`DELETE FROM carts WHERE user_id = ?`, userID)
	return err
}

// List returns cart lines joined with menu data.
func (r *CartRepository) List(userID int64) ([]models.CartItem, error) {
	rows, err := r.db.Query(`
		SELECT ca.id, ca.user_id, ca.menu_item_id, ca.quantity, ca.created_at,
		       m.id, m.category_id, m.name, m.slug, m.description, m.price, m.image, m.available
		FROM carts ca
		JOIN menu_items m ON m.id = ca.menu_item_id
		WHERE ca.user_id = ?
		ORDER BY ca.created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.CartItem
	for rows.Next() {
		var c models.CartItem
		if err := rows.Scan(&c.ID, &c.UserID, &c.MenuItemID, &c.Quantity, &c.CreatedAt,
			&c.Menu.ID, &c.Menu.CategoryID, &c.Menu.Name, &c.Menu.Slug, &c.Menu.Description, &c.Menu.Price, &c.Menu.Image, &c.Menu.Available); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

// Count returns the number of distinct lines in the cart.
func (r *CartRepository) Count(userID int64) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM carts WHERE user_id = ?`, userID).Scan(&n)
	return n, err
}

// Subtotal sums the cart value.
func (r *CartRepository) Subtotal(userID int64) (int64, error) {
	var total int64
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(ca.quantity * m.price), 0)
		FROM carts ca JOIN menu_items m ON m.id = ca.menu_item_id
		WHERE ca.user_id = ?`, userID).Scan(&total)
	return total, err
}