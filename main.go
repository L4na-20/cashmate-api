package main

import (
	"context"
	"fmt"
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

	// AutoMigrate membuat tabel secara otomatis jika belum ada.
	if err := config.DB.AutoMigrate(
		&models.User{},
		&models.Wallet{},
		&models.Category{},
		&models.Transaction{},
	); err != nil {
		log.Fatalf("gagal migrasi database: %v", err)
	}

	// Bootstrap role owner: jika belum ada satupun owner (mis. database lama
	// yang sudah terisi sebelum fitur role), user pertama di-promosikan.
	var ownerCount int64
	config.DB.Model(&models.User{}).Where("role = ?", "owner").Count(&ownerCount)
	if ownerCount == 0 {
		var first models.User
		if err := config.DB.Order("id ASC").First(&first).Error; err == nil {
			if err := config.DB.Model(&first).Update("role", "owner").Error; err == nil {
				fmt.Fprintf(os.Stdout, "[cashmate-api]  user #%d (%s) dipromosikan menjadi owner\n", first.ID, first.Email)
			}
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
