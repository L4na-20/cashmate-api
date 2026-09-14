package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cashmate-api/config"
	"cashmate-api/models"
	"cashmate-api/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadConfig()
	config.ConnectDB()

	// AutoMigrate is intentionally limited to local development. Production
	// schema changes must be applied from the versioned SQL migrations.
	if config.AutoMigrateEnabled() {
		if err := config.DB.AutoMigrate(
			&models.Business{},
			&models.User{},
			&models.Wallet{},
			&models.Category{},
			&models.Transaction{},
		); err != nil {
			log.Fatalf("gagal migrasi database: %v", err)
		}
	}

	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()
	if proxies := config.TrustedProxies(); len(proxies) > 0 {
		_ = r.SetTrustedProxies(proxies)
	}
	routes.Setup(r)

	server := &http.Server{
		Addr:         ":" + config.AppPort(),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("[cashmate-api]  server berjalan di http://localhost:%s", config.AppPort())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("gagal menjalankan server: %v", err)
		}
	}()

	// Graceful Shutdown: menunggu sinyal SIGINT / SIGTERM (mis. saat docker stop).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[cashmate-api]  menerima sinyal shutdown, mematikan server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown gagal: %v", err)
	}
	log.Println("[cashmate-api]  server berhenti dengan bersih")
}
