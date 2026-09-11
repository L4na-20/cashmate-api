package controllers

import (
	"cashmate-api/config"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// isWalletOwner memastikan wallet dengan id dimiliki oleh user yang login.
// Mengembalikan true jika wallet milik user tsb.
func isWalletOwner(c *gin.Context, walletID uint) bool {
	userID := mustUserID(c)
	var count int64
	config.DB.Model(&models.Wallet{}).
		Where("id = ? AND user_id = ?", walletID, userID).
		Count(&count)
	return count > 0
}

// isCategoryOwner memastikan kategori (global atau milik user) dapat
// diakses oleh user yang login. Kategori global (user_id NULL) boleh dipakai
// semua user; kategori spesifik hanya oleh pemiliknya.
func isCategoryOwner(c *gin.Context, categoryID uint) bool {
	userID := mustUserID(c)
	var count int64
	config.DB.Model(&models.Category{}).
		Where("id = ? AND (user_id IS NULL OR user_id = ?)", categoryID, userID).
		Count(&count)
	return count > 0
}

// mustUserID mengembalikan user_id dari context yang di-set AuthJWT.
func mustUserID(c *gin.Context) uint {
	v, _ := c.Get("user_id")
	id, _ := v.(uint)
	return id
}

// dbWithUserFilter menambahkan filter user_id pada query untuk memastikan
// data isolation pada setiap query GORM.
func dbWithUserFilter(c *gin.Context) *gorm.DB {
	userID := mustUserID(c)
	return config.DB.Where("user_id = ?", userID)
}
