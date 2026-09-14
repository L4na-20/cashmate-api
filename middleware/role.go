package middleware

import (
	"net/http"

	"cashmate-api/helpers"
	"cashmate-api/models"
	"github.com/gin-gonic/gin"
)

// RoleOwner membatasi akses hanya untuk user dengan role "owner".
// Dipakai setelah middleware AuthJWT (role sudah di-set ke context).
func RoleOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentUserRole(c) != models.RoleOwner {
			helpers.Error(c, http.StatusForbidden, "akses ditolak: hanya Owner yang diizinkan", nil)
			return
		}
		c.Next()
	}
}

// CurrentUserRole mengembalikan role yang di-set oleh middleware AuthJWT.
func CurrentUserRole(c *gin.Context) string {
	if v, ok := c.Get("role"); ok {
		if role, ok2 := v.(string); ok2 {
			return models.NormalizeRole(role)
		}
	}
	return ""
}
