package main

import (
	"log"

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

	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()
	routes.Setup(r)

	addr := ":" + config.AppPort()
	log.Printf("[cashmate-api]  server berjalan di http://localhost%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("gagal menjalankan server: %v", err)
	}
}
