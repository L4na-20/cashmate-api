package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"cashmate-api/config"
	"cashmate-api/helpers"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// transactionInput menampung payload create & update transaksi.
type transactionInput struct {
	WalletID    uint    `json:"wallet_id" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Type        string  `json:"type" binding:"required,oneof=income expense"`
	CategoryID  uint    `json:"category_id" binding:"required"`
	Description string  `json:"description"`
	Date        string  `json:"date"` // YYYY-MM-DD
}

// TransactionsIndex menampilkan histori transaksi milik user yang login
// (difilter melalui wallet milik user).
func TransactionsIndex(c *gin.Context) {
	userID := mustUserID(c)
	// Kumpulan wallet id milik user (dipakai untuk filter transaksi).
	var walletIDs []uint
	config.DB.Model(&models.Wallet{}).Where("user_id = ?", userID).Pluck("id", &walletIDs)
	if len(walletIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []models.Transaction{}, "total": 0})
		return
	}

	buildFiltered := func() *gorm.DB {
		q := config.DB.Model(&models.Transaction{}).
			Where("wallet_id IN ?", walletIDs)
		if v := c.Query("wallet_id"); v != "" {
			if wid, err := strconv.ParseUint(v, 10, 64); err == nil && isWalletOwner(c, uint(wid)) {
				q = q.Where("wallet_id = ?", wid)
			} else {
				// Jika wallet_id tidak milik user, kembalikan tanpa hasil.
				q = q.Where("1 = 0")
			}
		}
		if v := c.Query("from_date"); v != "" {
			q = q.Where("date >= ?", v)
		}
		if v := c.Query("to_date"); v != "" {
			q = q.Where("date <= ?", v)
		}
		if v := c.Query("category_id"); v != "" {
			if cid, err := strconv.ParseUint(v, 10, 64); err == nil && isCategoryOwner(c, uint(cid)) {
				q = q.Where("category_id = ?", cid)
			} else {
				q = q.Where("1 = 0")
			}
		}
		if v := c.Query("type"); v != "" {
			v = strings.ToLower(v)
			if v == "income" || v == "expense" {
				q = q.Where("type = ?", v)
			}
		}
		return q
	}

	var total int64
	if err := buildFiltered().Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "15"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 15
	}

	var transactions []models.Transaction
	if err := buildFiltered().
		Preload("Category").
		Preload("Wallet").
		Order("date DESC, id DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	from := (page-1)*perPage + 1
	to := page * perPage
	if total == 0 {
		from, to = 0, 0
	} else if int64(to) > total {
		to = int(total)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":         transactions,
		"total":        total,
		"from":         int64(from),
		"to":           int64(to),
		"per_page":     perPage,
		"current_page": page,
		"last_page":    (int(total) + perPage - 1) / perPage,
	})
}

// validateTransaction memvalidasi input & memastikan wallet & kategori
// yang direferensikan adalah milik user yang login.
func validateTransaction(c *gin.Context, input *transactionInput) (models.Transaction, string) {
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))
	if input.Type != "income" && input.Type != "expense" {
		return models.Transaction{}, "tipe harus 'income' atau 'expense'"
	}
	if input.Amount <= 0 {
		return models.Transaction{}, "nominal harus lebih besar dari 0"
	}

	// Pastikan wallet milik user yang login (anti-IDOR).
	if !isWalletOwner(c, input.WalletID) {
		return models.Transaction{}, "wallet tidak ditemukan"
	}

	// Pastikan kategori (global atau milik user) dapat diakses.
	if !isCategoryOwner(c, input.CategoryID) {
		return models.Transaction{}, "kategori tidak ditemukan"
	}

	var category models.Category
	config.DB.First(&category, input.CategoryID)
	if category.ID == 0 {
		return models.Transaction{}, "kategori tidak ditemukan"
	}
	if category.Type != input.Type {
		return models.Transaction{}, "tipe transaksi tidak cocok dengan tipe kategori"
	}

	txDate := time.Now()
	if input.Date != "" {
		parsed, err := time.Parse("2006-01-02", input.Date)
		if err != nil {
			return models.Transaction{}, "format tanggal harus YYYY-MM-DD"
		}
		txDate = parsed
	}

	return models.Transaction{
		WalletID:    input.WalletID,
		Amount:      input.Amount,
		Type:        input.Type,
		CategoryID:  input.CategoryID,
		Description: helpers.Sanitize(input.Description),
		Date:        txDate,
	}, ""
}

// TransactionsStore mencatat transaksi baru & memperbarui saldo wallet.
func TransactionsStore(c *gin.Context) {
	var input transactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parameter tidak lengkap: wallet_id, amount, type, category_id wajib diisi"})
		return
	}

	transaction, msg := validateTransaction(c, &input)
	if msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	// Simpan transaksi dalam transaksi DB agar konsisten dengan saldo.
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}
		var wallet models.Wallet
		if err := tx.Where("id = ? AND user_id = ?", transaction.WalletID, mustUserID(c)).
			First(&wallet).Error; err != nil {
			return err
		}
		delta := transaction.Amount
		if transaction.Type == "expense" {
			delta = -delta
		}
		return tx.Model(&wallet).Update("balance", wallet.Balance+delta).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	config.DB.Preload("Category").Preload("Wallet").First(&transaction, transaction.ID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "transaksi berhasil disimpan",
		"data":    transaction,
	})
}

// TransactionsUpdate mengubah transaksi milik user (anti-IDOR).
func TransactionsUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id transaksi tidak valid"})
		return
	}

	userID := mustUserID(c)
	// Temukan wallet milik user yang dipakai transaksi ini.
	var existing models.Transaction
	if err := config.DB.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaksi tidak ditemukan"})
		return
	}
	if !isWalletOwner(c, existing.WalletID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaksi tidak ditemukan"})
		return
	}

	var input transactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parameter tidak lengkap: wallet_id, amount, type, category_id wajib diisi"})
		return
	}

	transaction, msg := validateTransaction(c, &input)
	if msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	transaction.ID = existing.ID
	transaction.CreatedAt = existing.CreatedAt

	// Kembalikan saldo lama terlebih dahulu, lalu terapkan yang baru.
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		oldDelta := existing.Amount
		if existing.Type == "expense" {
			oldDelta = -oldDelta
		}
		if err := tx.Model(&models.Wallet{}).
			Where("id = ? AND user_id = ?", existing.WalletID, userID).
			Update("balance", gorm.Expr("balance - ?", oldDelta)).Error; err != nil {
			return err
		}

		if err := tx.Save(&transaction).Error; err != nil {
			return err
		}

		newDelta := transaction.Amount
		if transaction.Type == "expense" {
			newDelta = -newDelta
		}
		return tx.Model(&models.Wallet{}).
			Where("id = ? AND user_id = ?", transaction.WalletID, userID).
			Update("balance", gorm.Expr("balance + ?", newDelta)).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	config.DB.Preload("Category").Preload("Wallet").First(&transaction, transaction.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "transaksi berhasil diperbarui",
		"data":    transaction,
	})
}

// TransactionsDestroy menghapus transaksi milik user & balik saldo.
func TransactionsDestroy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id transaksi tidak valid"})
		return
	}

	userID := mustUserID(c)
	var transaction models.Transaction
	if err := config.DB.First(&transaction, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaksi tidak ditemukan"})
		return
	}
	if !isWalletOwner(c, transaction.WalletID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaksi tidak ditemukan"})
		return
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		delta := transaction.Amount
		if transaction.Type == "expense" {
			delta = -delta
		}
		if err := tx.Model(&models.Wallet{}).
			Where("id = ? AND user_id = ?", transaction.WalletID, userID).
			Update("balance", gorm.Expr("balance - ?", delta)).Error; err != nil {
			return err
		}
		return tx.Delete(&transaction).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "transaksi berhasil dihapus"})
}
