package middleware

import (
	"net/http"
	"strings"

	"cashmate-api/helpers"

	"github.com/gin-gonic/gin"
)

// AuthJWT memverifikasi Access Token yang dikirim via header Authorization:
// Authorization: Bearer <token>
// Setelah valid, user_id & email dimasukkan ke dalam context Gin.
func AuthJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token tidak ada"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "format Authorization tidak valid"})
			return
		}

		claims, err := helpers.ParseAccessToken(strings.TrimSpace(parts[1]))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// CurrentUserID mengembalikan user_id yang di-set oleh middleware AuthJWT.
// Biasanya dipakai di dalam controller untuk memfilter data milik user.
func CurrentUserID(c *gin.Context) uint {
	if v, ok := c.Get("user_id"); ok {
		if id, ok2 := v.(uint); ok2 {
			return id
		}
	}
	return 0
}
