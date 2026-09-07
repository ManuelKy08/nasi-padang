// RANTAU — Rumah Makan Padang
// An online ordering & restaurant management system written in Go.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rantau/rantau/config"
	"github.com/rantau/rantau/handlers"
	"github.com/rantau/rantau/middleware"
)

func main() {
	cfg := config.Load()

	db, err := cfg.Connect()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	store := middleware.NewSessionStore(cfg.SessionKey)

	app, err := handlers.NewApp(cfg, db, store)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	mux := http.NewServeMux()

	// Public auth routes + root redirect.
	app.RegisterAuthRoutes(mux)

	// Protected customer routes.
	customerRoutes := http.NewServeMux()
	app.RegisterHomeRoutes(customerRoutes)
	app.RegisterMenuRoutes(customerRoutes)
	app.RegisterCartRoutes(customerRoutes)
	app.RegisterCheckoutRoutes(customerRoutes)
	app.RegisterOrderRoutes(customerRoutes)
	app.RegisterProfileRoutes(customerRoutes)

	// Admin-only routes on a separate mux so non-admins get a 403.
	adminRoutes := http.NewServeMux()
	app.RegisterAdminRoutes(adminRoutes)

	protectCustomer := middleware.RequireAuth(store)
	customerPaths := []string{
		"/dashboard", "/about", "/menu", "/cart", "/checkout", "/receipt/",
		"/orders", "/profile", "/favorites", "/api/",
	}
	for _, p := range customerPaths {
		mux.Handle(p, protectCustomer(customerRoutes))
		if !strings.HasSuffix(p, "/") {
			mux.Handle(p+"/", protectCustomer(customerRoutes))
		}
	}

	adminProtect := middleware.RequireAdmin(store)
	mux.Handle("/admin", adminProtect(adminRoutes))
	mux.Handle("/admin/", adminProtect(adminRoutes))

	// Static assets (public).
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Global chain: session -> auth context.
	handler := middleware.AuthContext(store, db)(mux)

	srv := &http.Server{
		Addr:              cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("RANTAU — Rumah Makan Padang running on %s", cfg.Port)
	log.Printf("DB: %s", cfg.DBName)

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-stop
	log.Println("shutting down…")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("server stopped.")
}