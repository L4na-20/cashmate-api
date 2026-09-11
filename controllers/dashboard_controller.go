package controllers

import (
	"net/http"

	"cashmate-api/config"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
)

// DashboardSummary mengembalikan ringkasan keuangan user yang login.
func DashboardSummary(c *gin.Context) {
	userID := mustUserID(c)

	var walletIDs []uint
	config.DB.Model(&models.Wallet{}).Where("user_id = ?", userID).Pluck("id", &walletIDs)
	if len(walletIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"total_income":        0,
			"total_expense":       0,
			"balance":             0,
			"transaction_count":   map[string]int64{"income": 0, "expense": 0},
			"recent_transactions": []models.Transaction{},
		})
		return
	}

	var incomeAgg, expenseAgg struct {
		Total float64
	}
	config.DB.Model(&models.Transaction{}).
		Where("type = ? AND wallet_id IN ?", "income", walletIDs).
		Select("COALESCE(SUM(amount),0) AS total").Scan(&incomeAgg)
	config.DB.Model(&models.Transaction{}).
		Where("type = ? AND wallet_id IN ?", "expense", walletIDs).
		Select("COALESCE(SUM(amount),0) AS total").Scan(&expenseAgg)

	// Total saldo seluruh wallet milik user.
	var totalBalance float64
	config.DB.Model(&models.Wallet{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(balance),0)").Scan(&totalBalance)

	var recent []models.Transaction
	config.DB.Preload("Category").Preload("Wallet").
		Where("wallet_id IN ?", walletIDs).
		Order("date DESC, id DESC").Limit(5).Find(&recent)

	c.JSON(http.StatusOK, gin.H{
		"total_income":       incomeAgg.Total,
		"total_expense":      expenseAgg.Total,
		"balance":            totalBalance,
		"transaction_count": struct {
			Income  int64 `json:"income"`
			Expense int64 `json:"expense"`
		}{
			incomeCountUser(walletIDs),
			expenseCountUser(walletIDs),
		},
		"recent_transactions": recent,
	})
}

func incomeCountUser(walletIDs []uint) int64 {
	var n int64
	config.DB.Model(&models.Transaction{}).
		Where("type = ? AND wallet_id IN ?", "income", walletIDs).Count(&n)
	return n
}

func expenseCountUser(walletIDs []uint) int64 {
	var n int64
	config.DB.Model(&models.Transaction{}).
		Where("type = ? AND wallet_id IN ?", "expense", walletIDs).Count(&n)
	return n
}
