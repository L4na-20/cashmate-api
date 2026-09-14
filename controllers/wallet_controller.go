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

type walletInput struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Balance  *int64 `json:"balance"`
}

type walletView struct {
	ID         uint           `json:"id"`
	BusinessID uint           `json:"business_id"`
	Name       string         `json:"name"`
	Balance    *int64         `json:"balance,omitempty"`
	Currency   string         `json:"currency"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at,omitempty"`
}

func toWalletView(wallet models.Wallet, revealBalance bool) walletView {
	view := walletView{
		ID:         wallet.ID,
		BusinessID: wallet.BusinessID,
		Name:       wallet.Name,
		Currency:   wallet.Currency,
		CreatedAt:  wallet.CreatedAt,
		UpdatedAt:  wallet.UpdatedAt,
		DeletedAt:  wallet.DeletedAt,
	}
	if revealBalance {
		balance := wallet.Balance
		view.Balance = &balance
	}
	return view
}

func parseResourceStatus(c *gin.Context) string {
	status := strings.ToLower(strings.TrimSpace(c.DefaultQuery("status", "active")))
	if status != "active" && status != "disabled" && status != "all" {
		return "active"
	}
	return status
}

func validateWalletInput(input walletInput, requireName bool) (string, bool) {
	input.Name = helpers.Sanitize(input.Name)
	if requireName && input.Name == "" {
		return "nama wallet wajib diisi", false
	}
	if input.Currency != "" && strings.ToUpper(strings.TrimSpace(input.Currency)) != models.CurrencyIDR {
		return "currency MVP hanya IDR", false
	}
	if input.Balance != nil {
		return "balance tidak boleh diubah melalui wallet API", false
	}
	return "", true
}

// WalletsIndex lists wallets in the authenticated Business. Staff always get
// active wallets without balance information.
func WalletsIndex(c *gin.Context) {
	owner := isOwner(c)
	status := parseResourceStatus(c)
	if !owner {
		status = "active"
	}

	query := scopedBusiness(gormDB(), c)
	if owner && status != "active" {
		query = query.Unscoped()
	}
	if status == "disabled" {
		query = query.Where("deleted_at IS NOT NULL")
	} else if status == "active" {
		query = query.Where("deleted_at IS NULL")
	}

	var wallets []models.Wallet
	if err := query.Order("created_at ASC, id ASC").Find(&wallets).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal mengambil wallet", nil)
		return
	}
	views := make([]walletView, 0, len(wallets))
	for _, wallet := range wallets {
		views = append(views, toWalletView(wallet, owner))
	}
	helpers.Success(c, http.StatusOK, "wallet berhasil diambil", views)
}

// WalletsShow returns a single Business wallet with role-safe fields.
func WalletsShow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		helpers.Error(c, http.StatusBadRequest, "id wallet tidak valid", nil)
		return
	}
	query := scopedBusiness(gormDB(), c).Where("id = ?", id)
	if isOwner(c) && c.Query("include_deleted") == "1" {
		query = query.Unscoped()
	}
	var wallet models.Wallet
	if err := query.First(&wallet).Error; err != nil {
		helpers.Error(c, http.StatusNotFound, "wallet tidak ditemukan", nil)
		return
	}
	helpers.Success(c, http.StatusOK, "wallet berhasil diambil", toWalletView(wallet, isOwner(c)))
}

// WalletsStore creates a zero-balance IDR wallet for the current Business.
func WalletsStore(c *gin.Context) {
	var input walletInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.Error(c, http.StatusBadRequest, "payload wallet tidak valid", nil)
		return
	}
	if msg, ok := validateWalletInput(input, true); !ok {
		helpers.Error(c, http.StatusBadRequest, msg, nil)
		return
	}
	input.Name = helpers.Sanitize(input.Name)
	wallet := models.Wallet{BusinessID: currentBusinessID(c), Name: input.Name, Currency: models.CurrencyIDR}
	if err := gormDB().Create(&wallet).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal membuat wallet", nil)
		return
	}
	helpers.Success(c, http.StatusCreated, "wallet berhasil dibuat", toWalletView(wallet, true))
}

// WalletsUpdate changes wallet metadata only; balance is ledger-managed.
func WalletsUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		helpers.Error(c, http.StatusBadRequest, "id wallet tidak valid", nil)
		return
	}
	var input walletInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.Error(c, http.StatusBadRequest, "payload wallet tidak valid", nil)
		return
	}
	if msg, ok := validateWalletInput(input, true); !ok {
		helpers.Error(c, http.StatusBadRequest, msg, nil)
		return
	}
	var wallet models.Wallet
	if err := scopedBusiness(gormDB(), c).Where("id = ?", id).First(&wallet).Error; err != nil {
		helpers.Error(c, http.StatusNotFound, "wallet tidak ditemukan", nil)
		return
	}
	if err := scopedBusiness(gormDB(), c).Model(&wallet).Updates(map[string]any{
		"name":     helpers.Sanitize(input.Name),
		"currency": models.CurrencyIDR,
	}).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal memperbarui wallet", nil)
		return
	}
	wallet.Name = helpers.Sanitize(input.Name)
	wallet.Currency = models.CurrencyIDR
	helpers.Success(c, http.StatusOK, "wallet berhasil diperbarui", toWalletView(wallet, true))
}

func WalletsDestroy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		helpers.Error(c, http.StatusBadRequest, "id wallet tidak valid", nil)
		return
	}
	var wallet models.Wallet
	if err := scopedBusiness(gormDB(), c).Where("id = ?", id).First(&wallet).Error; err != nil {
		helpers.Error(c, http.StatusNotFound, "wallet tidak ditemukan", nil)
		return
	}
	if err := gormDB().Delete(&wallet).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal menonaktifkan wallet", nil)
		return
	}
	helpers.Success(c, http.StatusOK, "wallet berhasil dinonaktifkan", nil)
}

func WalletsRestore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		helpers.Error(c, http.StatusBadRequest, "id wallet tidak valid", nil)
		return
	}
	var wallet models.Wallet
	if err := scopedBusiness(gormDB().Unscoped(), c).Where("id = ?", id).First(&wallet).Error; err != nil {
		helpers.Error(c, http.StatusNotFound, "wallet tidak ditemukan", nil)
		return
	}
	if err := gormDB().Unscoped().Model(&wallet).Updates(map[string]any{"deleted_at": nil}).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal memulihkan wallet", nil)
		return
	}
	helpers.Success(c, http.StatusOK, "wallet berhasil dipulihkan", nil)
}

func gormDB() *gorm.DB {
	return config.DB
}

func configDB() *gorm.DB { return config.DB }
