package middleware

import (
	"net/http"
	"strings"

	"cashmate-api/config"
	"cashmate-api/helpers"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
)

// AuthJWT memverifikasi Access Token yang dikirim via header Authorization:
// Authorization: Bearer <token>. User aktif kemudian dimuat dari database agar
// tenant dan role tidak ditentukan oleh claim JWT yang stale.
func AuthJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			helpers.Error(c, http.StatusUnauthorized, "token tidak ada", nil)
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			helpers.Error(c, http.StatusUnauthorized, "format Authorization tidak valid", nil)
			return
		}

		claims, err := helpers.ParseAccessToken(strings.TrimSpace(parts[1]))
		if err != nil {
			helpers.Error(c, http.StatusUnauthorized, err.Error(), nil)
			return
		}

		var user models.User
		if err := config.DB.Preload("Business").First(&user, claims.UserID).Error; err != nil || user.Business == nil {
			helpers.Error(c, http.StatusUnauthorized, "sesi tidak valid", nil)
			return
		}
		if claims.AuthVersion != user.AuthVersion {
			helpers.Error(c, http.StatusUnauthorized, "sesi telah diinvalidasi", nil)
			return
		}
		role := models.NormalizeRole(user.Role)
		if role == "" {
			helpers.Error(c, http.StatusUnauthorized, "role user tidak valid", nil)
			return
		}

		c.Set("user_id", user.ID)
		c.Set("business_id", user.BusinessID)
		c.Set("email", user.Email)
		c.Set("role", role)
		c.Set("current_user", &user)
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

// CurrentBusinessID returns the tenant ID loaded by AuthJWT.
func CurrentBusinessID(c *gin.Context) uint {
	if v, ok := c.Get("business_id"); ok {
		if id, ok2 := v.(uint); ok2 {
			return id
		}
	}
	return 0
}
