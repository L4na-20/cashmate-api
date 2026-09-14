package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RoleOwner membatasi akses hanya untuk user dengan role "owner".
// Dipakai setelah middleware AuthJWT (role sudah di-set ke context).
func RoleOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "owner" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "akses ditolak: hanya owner yang diizinkan"})
			return
		}
		c.Next()
	}
}

// CurrentUserRole mengembalikan role yang di-set oleh middleware AuthJWT.
func CurrentUserRole(c *gin.Context) string {
	if v, ok := c.Get("role"); ok {
		if role, ok2 := v.(string); ok2 {
			return role
		}
	}
	return ""
}