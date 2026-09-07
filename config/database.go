// Package config loads application configuration from environment variables
// and provides a helper to open the database connection pool.
package config

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// Config holds every runtime setting the application needs.
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	SessionKey string
	Port       string

	// Restaurant operating hours (server clock based).
	OpenHour  int
	CloseHour int

	// Delivery pricing (in Rupiah).
	DeliveryFee int64
}

// Load reads .env (when present) and merges with sane defaults so the
// application still runs out of the box.
func Load() Config {
	_ = godotenv.Load()

	cfg := Config{
		DBHost:      getenv("DB_HOST", "127.0.0.1"),
		DBPort:      getenv("DB_PORT", "3306"),
		DBUser:      getenv("DB_USER", "rantau"),
		DBPassword:  getenv("DB_PASSWORD", "rantau_secret_2026"),
		DBName:      getenv("DB_NAME", "rantau"),
		SessionKey:  getenv("SESSION_KEY", "change-me-to-a-long-random-secret-key-2026"),
		Port:        getenv("PORT", ":8080"),
		OpenHour:    getenvInt("OPEN_HOUR", 10),
		CloseHour:   getenvInt("CLOSE_HOUR", 22),
		DeliveryFee: int64(getenvInt("DELIVERY_FEE", 5000)),
	}

	if cfg.SessionKey == "change-me-to-a-long-random-secret-key-2026" {
		fmt.Fprintln(os.Stderr, "WARNING: using default SESSION_KEY, set a secure one in .env for production.")
	}

	return cfg
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// DSN builds a MySQL DSN with UTF-8 and timezone handling.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local&time_zone='%%2B00:00'",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

// Connect opens the database and verifies the connection with a ping.
func (c Config) Connect() (*sql.DB, error) {
	db, err := sql.Open("mysql", c.DSN())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}