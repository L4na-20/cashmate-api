package middleware

import (
	"net/http"
	"time"

	"cashmate-api/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS menyediakan konfigurasi CORS yang ketat. Origin yang diizinkan
// diambil dari env CORS_ORIGINS (dipisahkan koma).
func CORS() gin.HandlerFunc {
	corsCfg := cors.Config{
		AllowAllOrigins: false,
		AllowOrigins:    config.AllowedOrigins(),
		AllowMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodOptions,
		},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}

	return cors.New(corsCfg)
}
