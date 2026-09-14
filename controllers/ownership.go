package controllers

import (
	"cashmate-api/config"
	"cashmate-api/middleware"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func currentUserID(c *gin.Context) uint {
	return middleware.CurrentUserID(c)
}

func currentBusinessID(c *gin.Context) uint {
	return middleware.CurrentBusinessID(c)
}

// mustUserID is retained as a small compatibility helper for controllers that
// are migrated incrementally.
func mustUserID(c *gin.Context) uint { return currentUserID(c) }

func mustBusinessID(c *gin.Context) uint { return currentBusinessID(c) }

func isOwner(c *gin.Context) bool {
	return middleware.CurrentUserRole(c) == models.RoleOwner
}

func scopedBusiness(db *gorm.DB, c *gin.Context) *gorm.DB {
	return db.Where("business_id = ?", currentBusinessID(c))
}

func findBusinessWallet(db *gorm.DB, c *gin.Context, walletID uint, includeDeleted bool) (models.Wallet, error) {
	var wallet models.Wallet
	query := db
	if includeDeleted {
		query = query.Unscoped()
	}
	err := query.Where("id = ? AND business_id = ?", walletID, currentBusinessID(c)).First(&wallet).Error
	return wallet, err
}

func findBusinessCategory(db *gorm.DB, c *gin.Context, categoryID uint, includeDeleted bool) (models.Category, error) {
	var category models.Category
	query := db
	if includeDeleted {
		query = query.Unscoped()
	}
	err := query.Where("id = ? AND (business_id = ? OR business_id IS NULL)", categoryID, currentBusinessID(c)).First(&category).Error
	return category, err
}

func isWalletAvailable(c *gin.Context, walletID uint) bool {
	_, err := findBusinessWallet(config.DB, c, walletID, false)
	return err == nil
}

func isCategoryAvailable(c *gin.Context, categoryID uint) bool {
	_, err := findBusinessCategory(config.DB, c, categoryID, false)
	return err == nil
}

// dbWithUserFilter is retained for callers outside the migrated controllers;
// new code should use scopedBusiness instead.
func dbWithUserFilter(c *gin.Context) *gorm.DB {
	return scopedBusiness(config.DB, c)
}
