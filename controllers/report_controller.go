package controllers

import (
	"net/http"

	"cashmate-api/config"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
)

// MonthlyReport mengelompokkan income & expense per bulan milik user.
//
// Query params:
//   - year : tahun (default tahun berjalan, gunakan 0 = semua tahun)
func MonthlyReport(c *gin.Context) {
	userID := mustUserID(c)
	year := c.Query("year")

	var walletIDs []uint
	config.DB.Model(&models.Wallet{}).Where("user_id = ?", userID).Pluck("id", &walletIDs)

	if len(walletIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		return
	}

	query := config.DB.Model(&models.Transaction{}).Where("wallet_id IN ?", walletIDs)
	if year != "" && year != "0" {
		query = query.Where("YEAR(date) = ?", year)
	}

	var rows []struct {
		Month string  `gorm:"column:month"`
		Type  string  `gorm:"column:type"`
		Total float64 `gorm:"column:total"`
	}
	query.Select(
		"DATE_FORMAT(date, '%Y-%m') AS month",
		"type",
		"COALESCE(SUM(amount),0) AS total",
	).
		Group("month, type").
		Order("month ASC").
		Scan(&rows)

	grouped := map[string]*struct {
		Income  float64 `json:"income"`
		Expense float64 `json:"expense"`
		Balance float64 `json:"balance"`
	}{}

	for _, r := range rows {
		if grouped[r.Month] == nil {
			grouped[r.Month] = &struct {
				Income  float64 `json:"income"`
				Expense float64 `json:"expense"`
				Balance float64 `json:"balance"`
			}{}
		}
		if r.Type == "income" {
			grouped[r.Month].Income = r.Total
		} else if r.Type == "expense" {
			grouped[r.Month].Expense = r.Total
		}
	}

	type item struct {
		Month   string  `json:"month"`
		Income  float64 `json:"income"`
		Expense float64 `json:"expense"`
		Balance float64 `json:"balance"`
	}

	items := make([]item, 0, len(grouped))
	for month, g := range grouped {
		g.Balance = g.Income - g.Expense
		items = append(items, item{
			Month:   month,
			Income:  g.Income,
			Expense: g.Expense,
			Balance: g.Balance,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}
