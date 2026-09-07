package repositories

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/rantau/rantau/models"
)

// ErrNotFound is returned when a query find nothing.
var ErrNotFound = errors.New("record not found")

// UserRepository wraps all user persistence.
type UserRepository struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }

// Create inserts a new user and returns its ID.
func (r *UserRepository) Create(u *models.User) (int64, error) {
	res, err := r.db.Exec(`
		INSERT INTO users (name, email, phone, password, role, address, created_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW())`,
		u.Name, strings.ToLower(strings.TrimSpace(u.Email)), u.Phone, u.Password, u.Role, u.Address)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// FindByEmail returns a user by email (case-insensitive lookup on stored lowercase).
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	u := &models.User{}
	err := r.db.QueryRow(`
		SELECT id, name, email, phone, password, role, COALESCE(address, ''), created_at
		FROM users WHERE email = ?`, strings.ToLower(strings.TrimSpace(email))).
		Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.Password, &u.Role, &u.Address, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// FindByID returns a user by primary key.
func (r *UserRepository) FindByID(id int64) (*models.User, error) {
	u := &models.User{}
	err := r.db.QueryRow(`
		SELECT id, name, email, phone, password, role, COALESCE(address, ''), created_at
		FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.Password, &u.Role, &u.Address, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// EmailExists reports whether the email is already taken (ignoring the given user id).
func (r *UserRepository) EmailExists(email string, exceptID int64) (bool, error) {
	var n int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE email = ? AND id <> ?`,
		strings.ToLower(strings.TrimSpace(email)), exceptID).Scan(&n)
	return n > 0, err
}

// Update edits profile fields for a user.
func (r *UserRepository) Update(u *models.User) error {
	_, err := r.db.Exec(
		`UPDATE users SET name = ?, phone = ?, address = ? WHERE id = ?`,
		u.Name, u.Phone, u.Address, u.ID)
	return err
}

// UpdatePassword sets a new bcrypt hash.
func (r *UserRepository) UpdatePassword(id int64, hash string) error {
	_, err := r.db.Exec(`UPDATE users SET password = ? WHERE id = ?`, hash, id)
	return err
}

// UpdateRole changes the role of a user.
func (r *UserRepository) UpdateRole(id int64, role string) error {
	_, err := r.db.Exec(`UPDATE users SET role = ? WHERE id = ?`, role, id)
	return err
}

// Delete removes a user.
func (r *UserRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

// List returns all users ordered newest first, with order statistics.
func (r *UserRepository) List() ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.name, u.email, u.phone, u.role, u.address, u.created_at,
		       COALESCE(o.cnt, 0) AS cnt, COALESCE(o.ttl, 0) AS ttl
		FROM users u
		LEFT JOIN (SELECT user_id, COUNT(*) cnt, COALESCE(SUM(total), 0) ttl
		           FROM orders GROUP BY user_id) o ON o.user_id = u.id
		ORDER BY u.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.Role, &u.Address, &u.CreatedAt, &u.OrderCount, &u.TotalSpent); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Count returns total users.
func (r *UserRepository) Count() (int64, error) {
	var n int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}