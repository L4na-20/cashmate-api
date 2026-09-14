package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"cashmate-api/config"
	"cashmate-api/helpers"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
)

type monthlyAggregate struct {
	Month  string `gorm:"column:month"`
	Type   string `gorm:"column:type"`
	Amount int64  `gorm:"column:total"`
}

func MonthlyReport(c *gin.Context) {
	if !isOwner(c) {
		helpers.Error(c, http.StatusForbidden, "laporan hanya tersedia untuk Owner", nil)
		return
	}
	year := time.Now().In(config.BusinessLocation()).Year()
	if value := c.Query("year"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 9999 {
			helpers.Error(c, http.StatusBadRequest, "year harus berupa tahun yang valid", nil)
			return
		}
		year = parsed
	}

	var rows []monthlyAggregate
	query := config.DB.Model(&models.Transaction{}).
		Where("business_id = ? AND YEAR(date) = ?", currentBusinessID(c), year).
		Select("DATE_FORMAT(date, '%Y-%m') AS month, type, COALESCE(SUM(amount), 0) AS total").
		Group("DATE_FORMAT(date, '%Y-%m'), type").
		Order("month ASC")
	if err := query.Scan(&rows).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal mengambil laporan bulanan", nil)
		return
	}
	grouped := map[string]*struct {
		Income  int64
		Expense int64
	}{}
	for _, row := range rows {
		if grouped[row.Month] == nil {
			grouped[row.Month] = &struct {
				Income  int64
				Expense int64
			}{}
		}
		if row.Type == models.TransactionIncome {
			grouped[row.Month].Income = row.Amount
		} else if row.Type == models.TransactionExpense {
			grouped[row.Month].Expense = row.Amount
		}
	}
	items := make([]gin.H, 0, len(grouped))
	for month := time.Date(year, 1, 1, 0, 0, 0, 0, config.BusinessLocation()); month.Year() == year; month = month.AddDate(0, 1, 0) {
		key := fmt.Sprintf("%04d-%02d", month.Year(), month.Month())
		entry := grouped[key]
		var income, expense int64
		if entry != nil {
			income, expense = entry.Income, entry.Expense
		}
		items = append(items, gin.H{"month": key, "income": income, "expense": expense, "net_cashflow": income - expense})
	}
	helpers.Success(c, http.StatusOK, "laporan bulanan berhasil diambil", items)
}
