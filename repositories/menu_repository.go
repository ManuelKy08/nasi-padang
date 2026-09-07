package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/rantau/rantau/models"
)

// MenuRepository handles categories, menu items and favorites.
type MenuRepository struct{ db *sql.DB }

func NewMenuRepository(db *sql.DB) *MenuRepository { return &MenuRepository{db: db} }

// ListCategories returns all categories ordered by id.
func (r *MenuRepository) ListCategories() ([]models.Category, error) {
	rows, err := r.db.Query(`SELECT id, name, slug FROM categories ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// MenuFilter describes how to narrow down a menu list query.
type MenuFilter struct {
	CategorySlug  string
	Search        string
	Sort          string // "", "name", "price_asc", "price_desc", "popular"
	UserID        int64
	AvailableOnly bool
}

const menuSelect = `
	SELECT m.id, m.category_id, c.name, m.name, m.slug, COALESCE(m.description, ''), m.price,
	       COALESCE(m.image, ''), m.available, m.created_at,
	       CASE WHEN f.id IS NULL THEN 0 ELSE 1 END`

func (r *MenuRepository) scanMenu(sc interface{ Scan(...any) error }) (models.MenuItem, error) {
	var m models.MenuItem
	err := sc.Scan(&m.ID, &m.CategoryID, &m.CategoryName, &m.Name, &m.Slug,
		&m.Description, &m.Price, &m.Image, &m.Available, &m.CreatedAt, &m.IsFavorite)
	return m, err
}

// List returns menu items filtered by category/search/sort.
func (r *MenuRepository) List(f MenuFilter) ([]models.MenuItem, error) {
	from := ` FROM menu_items m
	JOIN categories c ON c.id = m.category_id
	LEFT JOIN favorites f ON f.menu_item_id = m.id AND f.user_id = ?`
	where := []string{"1 = 1"}
	args := []any{f.UserID}

	if f.CategorySlug != "" && f.CategorySlug != "semua" {
		where = append(where, "c.slug = ?")
		args = append(args, f.CategorySlug)
	}
	if f.Search != "" {
		like := "%" + strings.ToLower(f.Search) + "%"
		where = append(where, "(LOWER(m.name) LIKE ? OR LOWER(c.name) LIKE ? OR LOWER(m.description) LIKE ?)")
		args = append(args, like, like, like)
	}
	if f.AvailableOnly {
		where = append(where, "m.available = 1")
	}

	order := "m.name ASC"
	switch f.Sort {
	case "name":
		order = "m.name ASC"
	case "price_asc":
		order = "m.price ASC"
	case "price_desc":
		order = "m.price DESC"
	case "popular":
		order = "(SELECT COALESCE(SUM(oi.quantity), 0) FROM order_items oi WHERE oi.menu_item_id = m.id) DESC"
	}

	q := menuSelect + from + " WHERE " + strings.Join(where, " AND ") + " ORDER BY " + order
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.MenuItem
	for rows.Next() {
		m, err := r.scanMenu(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// FindByID returns a single menu item with its category.
func (r *MenuRepository) FindByID(id int64, userID int64) (*models.MenuItem, error) {
	q := menuSelect + `
		FROM menu_items m
		JOIN categories c ON c.id = m.category_id
		LEFT JOIN favorites f ON f.menu_item_id = m.id AND f.user_id = ?
		WHERE m.id = ?`
	m, err := r.scanMenu(r.db.QueryRow(q, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Create inserts a new menu item.
func (r *MenuRepository) Create(m *models.MenuItem) (int64, error) {
	res, err := r.db.Exec(`
		INSERT INTO menu_items (category_id, name, slug, description, price, image, available, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW())`,
		m.CategoryID, m.Name, m.Slug, m.Description, m.Price, m.Image, m.Available)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Update edits menu fields.
func (r *MenuRepository) Update(m *models.MenuItem) error {
	_, err := r.db.Exec(`
		UPDATE menu_items SET category_id = ?, name = ?, slug = ?, description = ?,
			price = ?, image = ?, available = ? WHERE id = ?`,
		m.CategoryID, m.Name, m.Slug, m.Description, m.Price, m.Image, m.Available, m.ID)
	return err
}

// SetAvailable flips availability.
func (r *MenuRepository) SetAvailable(id int64, available bool) error {
	_, err := r.db.Exec(`UPDATE menu_items SET available = ? WHERE id = ?`, available, id)
	return err
}

// Delete removes a menu item.
func (r *MenuRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM menu_items WHERE id = ?`, id)
	return err
}

// Count returns the number of menu items.
func (r *MenuRepository) Count() (int64, error) {
	var n int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM menu_items`).Scan(&n)
	return n, err
}

// SlugExists reports whether a slug clashes with an existing item.
func (r *MenuRepository) SlugExists(slug string, exceptID int64) (bool, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM menu_items WHERE slug = ? AND id <> ?`, slug, exceptID).Scan(&n)
	return n > 0, err
}

// MenuRepository favourite helpers -------------------------------------------------

// ToggleFavorite flips the favorite state and reports the new state.
func (r *MenuRepository) ToggleFavorite(userID, menuID int64) (bool, error) {
	res, err := r.db.Exec(`DELETE FROM favorites WHERE user_id = ? AND menu_item_id = ?`, userID, menuID)
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return false, nil
	}
	_, err = r.db.Exec(`INSERT INTO favorites (user_id, menu_item_id, created_at) VALUES (?, ?, NOW())`, userID, menuID)
	return true, err
}

// ListFavorites returns the user's favorite menu items.
func (r *MenuRepository) ListFavorites(userID int64) ([]models.MenuItem, error) {
	rows, err := r.db.Query(`
		SELECT m.id, m.category_id, c.name, m.name, m.slug, m.description, m.price,
		       m.image, m.available, m.created_at, 1
		FROM favorites f
		JOIN menu_items m ON m.id = f.menu_item_id
		JOIN categories c ON c.id = m.category_id
		WHERE f.user_id = ?
		ORDER BY f.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.MenuItem
	for rows.Next() {
		m, err := r.scanMenu(rows)
		if err != nil {
			return nil, err
		}
		m.IsFavorite = true
		items = append(items, m)
	}
	return items, rows.Err()
}

// CountFavorites returns how many favorites a user has.
func (r *MenuRepository) CountFavorites(userID int64) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM favorites WHERE user_id = ?`, userID).Scan(&n)
	return n, err
}

// Popular returns the most ordered menu items.
func (r *MenuRepository) Popular(limit int) ([]models.MenuItem, error) {
	rows, err := r.db.Query(`
		SELECT m.id, m.category_id, c.name, m.name, m.slug, m.description, m.price,
		       m.image, m.available, m.created_at, 0,
		       COALESCE(SUM(oi.quantity), 0)
		FROM menu_items m
		JOIN categories c ON c.id = m.category_id
		LEFT JOIN order_items oi ON oi.menu_item_id = m.id
		GROUP BY m.id, c.name
		ORDER BY 12 DESC, m.name ASC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.MenuItem
	for rows.Next() {
		var m models.MenuItem
		if err := rows.Scan(&m.ID, &m.CategoryID, &m.CategoryName, &m.Name, &m.Slug,
			&m.Description, &m.Price, &m.Image, &m.Available, &m.CreatedAt, &m.IsFavorite, &m.TotalQty); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// Suggested returns a short list for the dashboard "MENU FAVORIT" section.
func (r *MenuRepository) Suggested(names []string) ([]models.MenuItem, error) {
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(names)), ",")
	q := fmt.Sprintf(`
		SELECT m.id, m.category_id, c.name, m.name, m.slug, m.description, m.price,
		       m.image, m.available, m.created_at, 0
		FROM menu_items m
		JOIN categories c ON c.id = m.category_id
		WHERE LOWER(m.name) IN (%s)
		ORDER BY FIELD(LOWER(m.name), %s)`, placeholders, placeholders)
	args := make([]any, 0, len(names))
	for _, n := range names {
		args = append(args, strings.ToLower(n))
	}
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.MenuItem
	for rows.Next() {
		m, err := r.scanMenu(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}