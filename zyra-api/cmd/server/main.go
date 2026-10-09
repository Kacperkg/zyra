package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"zyra-api/internal/auth"
	"zyra-api/internal/config"
	"zyra-api/internal/database"
	"zyra-api/internal/repository"
	"zyra-api/internal/routes"
	"zyra-api/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("database connection failed: ", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	if cfg.Migrate {
		if err = database.Migrate(db); err != nil {
			log.Fatal("migration failed: ", err)
		}
	}
	svc := services.New(repository.New(db), auth.New(cfg.JWTSecret))
	svc.DevAdminLogin = cfg.DevAdminLogin
	if cfg.DevAdminLogin {
		if err = svc.BootstrapDevelopmentAdmin(context.Background()); err != nil {
			log.Fatal("development admin bootstrap failed: ", err)
		}
		log.Print("Development-only admin/admin login enabled on loopback listener")
	}
	if cfg.BootstrapEmail != "" {
		if err = svc.Bootstrap(context.Background(), cfg.BootstrapEmail, cfg.BootstrapPassword); err != nil {
			log.Fatal("bootstrap failed: ", err)
		}
	}
	if err = svc.EnsureOwner(context.Background(), cfg.OwnerEmail); err != nil {
		log.Fatal("owner selection failed: ", err)
	}
	server := &http.Server{Addr: cfg.Address, Handler: routes.New(svc), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	stop, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()
	go func() {
		<-stop.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	log.Printf("Zyra API listening on %s", cfg.Address)
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
