package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"cashmate-api/config"
	"cashmate-api/helpers"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
)

// walletInput menampung payload create & update wallet.
type walletInput struct {
	Name     string  `json:"name" binding:"required,min=1"`
	Balance  float64 `json:"balance" binding:"min=0"`
	Currency string  `json:"currency" binding:"required,len=3"`
}

// WalletsIndex menampilkan seluruh wallet milik user yang login.
func WalletsIndex(c *gin.Context) {
	userID := mustUserID(c)
	var wallets []models.Wallet

	if err := config.DB.Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&wallets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": wallets})
}

// WalletsShow menampilkan satu wallet milik user (anti-IDOR).
func WalletsShow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id wallet tidak valid"})
		return
	}

	userID := mustUserID(c)
	var wallet models.Wallet
	if err := config.DB.Where("id = ? AND user_id = ?", id, userID).
		First(&wallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": wallet})
}

// WalletsStore membuat wallet baru milik user yang login.
func WalletsStore(c *gin.Context) {
	var input walletInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, currency wajib diisi dan valid"})
		return
	}

	userID := mustUserID(c)
	input.Name = helpers.Sanitize(input.Name)
	if input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nama wallet wajib diisi"})
		return
	}
	input.Currency = strings.ToUpper(helpers.Sanitize(input.Currency))

	wallet := models.Wallet{
		UserID:   userID,
		Name:     input.Name,
		Balance:  input.Balance,
		Currency: input.Currency,
	}

	if err := config.DB.Create(&wallet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "wallet berhasil dibuat",
		"data":    wallet,
	})
}

// WalletsUpdate mengubah wallet hanya jika milik user yang login.
func WalletsUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id wallet tidak valid"})
		return
	}

	userID := mustUserID(c)
	var wallet models.Wallet
	if err := config.DB.Where("id = ? AND user_id = ?", id, userID).
		First(&wallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet tidak ditemukan"})
		return
	}

	var input walletInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, currency wajib diisi dan valid"})
		return
	}

	wallet.Name = helpers.Sanitize(input.Name)
	wallet.Balance = input.Balance
	wallet.Currency = strings.ToUpper(helpers.Sanitize(input.Currency))

	if err := config.DB.Save(&wallet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "wallet berhasil diperbarui",
		"data":    wallet,
	})
}

// WalletsDestroy menghapus wallet (hanya milik user yang login).
func WalletsDestroy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id wallet tidak valid"})
		return
	}

	userID := mustUserID(c)
	var wallet models.Wallet
	if err := config.DB.Where("id = ? AND user_id = ?", id, userID).
		First(&wallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet tidak ditemukan"})
		return
	}

	if err := config.DB.Delete(&wallet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "wallet berhasil dihapus"})
}
