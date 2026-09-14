package controllers

import (
	"net/http"
	"time"

	"cashmate-api/config"
	"cashmate-api/helpers"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
)

// DashboardSummary is Owner-only (enforced by the route and rechecked here)
// and aggregates active transactions for the authenticated Business.
func DashboardSummary(c *gin.Context) {
	if !isOwner(c) {
		helpers.Error(c, http.StatusForbidden, "dashboard hanya tersedia untuk Owner", nil)
		return
	}
	businessID := currentBusinessID(c)
	now := time.Now().In(config.BusinessLocation())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, config.BusinessLocation())
	nextMonth := monthStart.AddDate(0, 1, 0)

	var totalBalance int64
	config.DB.Unscoped().Model(&models.Wallet{}).
		Where("business_id = ?", businessID).
		Select("COALESCE(SUM(balance), 0)").Scan(&totalBalance)

	var income, expense int64
	config.DB.Model(&models.Transaction{}).
		Where("business_id = ? AND type = ? AND date >= ? AND date < ?", businessID, models.TransactionIncome, monthStart, nextMonth).
		Select("COALESCE(SUM(amount), 0)").Scan(&income)
	config.DB.Model(&models.Transaction{}).
		Where("business_id = ? AND type = ? AND date >= ? AND date < ?", businessID, models.TransactionExpense, monthStart, nextMonth).
		Select("COALESCE(SUM(amount), 0)").Scan(&expense)

	var transactionCount int64
	config.DB.Model(&models.Transaction{}).Where("business_id = ?", businessID).Count(&transactionCount)
	var latest []models.Transaction
	if err := preloadTransaction(config.DB.Model(&models.Transaction{}).
		Where("business_id = ?", businessID).
		Order("date DESC, id DESC").Limit(5).Find(&latest)).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal mengambil transaksi terbaru", nil)
		return
	}
	latestViews := make([]gin.H, 0, len(latest))
	for _, transaction := range latest {
		latestViews = append(latestViews, transactionView(transaction, true))
	}
	helpers.Success(c, http.StatusOK, "dashboard berhasil diambil", gin.H{
		"total_balance":         totalBalance,
		"current_month_income":  income,
		"current_month_expense": expense,
		"net_cashflow":          income - expense,
		"transaction_count":     transactionCount,
		"latest_transactions":   latestViews,
	})
}
