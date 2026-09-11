package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS menyediakan konfigurasi CORS yang ketat. Hanya origin yang
// ada di daftar yang diizinkan mengakses API.
func CORS() gin.HandlerFunc {
	config := cors.Config{
		AllowAllOrigins: false,
		AllowOrigins: []string{
			"http://localhost:8000",
			"http://127.0.0.1:8000",
			"http://localhost:3000",
		},
		AllowMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodOptions,
		},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}

	return cors.New(config)
}
