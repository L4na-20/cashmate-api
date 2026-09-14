package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cashmate-api/config"
	"cashmate-api/helpers"
	"cashmate-api/models"
	"cashmate-api/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type transactionInput struct {
	WalletID    uint   `json:"wallet_id"`
	Amount      int64  `json:"amount"`
	Type        string `json:"type"`
	CategoryID  uint   `json:"category_id"`
	Description string `json:"description"`
	Date        string `json:"date"`
}

type transactionMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	Total       int64 `json:"total"`
	LastPage    int   `json:"last_page"`
	From        int64 `json:"from"`
	To          int64 `json:"to"`
}

func transactionView(transaction models.Transaction, revealBalance bool) gin.H {
	result := gin.H{
		"id":                 transaction.ID,
		"business_id":        transaction.BusinessID,
		"wallet_id":          transaction.WalletID,
		"category_id":        transaction.CategoryID,
		"created_by_user_id": transaction.CreatedByUserID,
		"amount":             transaction.Amount,
		"type":               transaction.Type,
		"description":        transaction.Description,
		"date":               transaction.Date.Format("2006-01-02"),
		"created_at":         transaction.CreatedAt,
		"updated_at":         transaction.UpdatedAt,
		"updated_by_user_id": transaction.UpdatedByUserID,
		"deleted_by_user_id": transaction.DeletedByUserID,
	}
	if transaction.DeletedAt.Valid {
		result["deleted_at"] = transaction.DeletedAt.Time
	} else {
		result["deleted_at"] = nil
	}
	if transaction.Wallet != nil {
		wallet := gin.H{
			"id":          transaction.Wallet.ID,
			"business_id": transaction.Wallet.BusinessID,
			"name":        transaction.Wallet.Name,
			"currency":    transaction.Wallet.Currency,
		}
		if revealBalance {
			wallet["balance"] = transaction.Wallet.Balance
		}
		result["wallet"] = wallet
	}
	if transaction.Category != nil {
		result["category"] = gin.H{
			"id":          transaction.Category.ID,
			"business_id": transaction.Category.BusinessID,
			"name":        transaction.Category.Name,
			"type":        transaction.Category.Type,
		}
	}
	if transaction.CreatedBy != nil {
		result["created_by"] = gin.H{
			"id":   transaction.CreatedBy.ID,
			"name": transaction.CreatedBy.Name,
			"role": models.NormalizeRole(transaction.CreatedBy.Role),
		}
	}
	return result
}

func preloadTransaction(query *gorm.DB) *gorm.DB {
	return query.
		Preload("Wallet", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Category", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("CreatedBy", func(db *gorm.DB) *gorm.DB { return db.Unscoped() })
}

func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", value, config.BusinessLocation())
}

func todayDate() time.Time {
	now := time.Now().In(config.BusinessLocation())
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, config.BusinessLocation())
}

func validateTransactionInput(input *transactionInput) string {
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))
	input.Description = helpers.Sanitize(input.Description)
	if input.WalletID == 0 || input.CategoryID == 0 || input.Amount <= 0 {
		return "wallet_id, category_id, dan amount (> 0) wajib diisi"
	}
	if input.Type != models.TransactionIncome && input.Type != models.TransactionExpense {
		return "type harus income atau expense"
	}
	return ""
}

func transactionDate(inputDate string, fallback time.Time) (time.Time, error) {
	if inputDate == "" {
		return fallback, nil
	}
	return parseDate(inputDate)
}

func TransactionsIndex(c *gin.Context) {
	businessID := currentBusinessID(c)
	owner := isOwner(c)
	status := parseResourceStatus(c)
	if !owner {
		status = "active"
	}
	query := config.DB.Model(&models.Transaction{}).Where("business_id = ?", businessID)
	if owner && status != "active" {
		query = query.Unscoped()
	}
	if status == "disabled" {
		query = query.Where("deleted_at IS NOT NULL")
	} else if status == "active" {
		query = query.Where("deleted_at IS NULL")
	}
	if !owner {
		query = query.Where("created_by_user_id = ? AND date = ?", currentUserID(c), todayDate().Format("2006-01-02"))
	}

	if value := c.Query("wallet_id"); value != "" {
		id, err := strconv.ParseUint(value, 10, 64)
		if err != nil || id == 0 {
			helpers.Error(c, http.StatusBadRequest, "wallet_id filter tidak valid", nil)
			return
		}
		query = query.Where("wallet_id = ?", id)
	}
	if value := c.Query("category_id"); value != "" {
		id, err := strconv.ParseUint(value, 10, 64)
		if err != nil || id == 0 {
			helpers.Error(c, http.StatusBadRequest, "category_id filter tidak valid", nil)
			return
		}
		query = query.Where("category_id = ?", id)
	}
	if value := strings.ToLower(strings.TrimSpace(c.Query("type"))); value != "" {
		if value != models.TransactionIncome && value != models.TransactionExpense {
			helpers.Error(c, http.StatusBadRequest, "type filter tidak valid", nil)
			return
		}
		query = query.Where("type = ?", value)
	}
	if owner {
		if value := c.Query("creator_id"); value != "" {
			id, err := strconv.ParseUint(value, 10, 64)
			if err != nil || id == 0 {
				helpers.Error(c, http.StatusBadRequest, "creator_id filter tidak valid", nil)
				return
			}
			query = query.Where("created_by_user_id = ?", id)
		}
		if value := c.Query("from_date"); value != "" {
			if _, err := parseDate(value); err != nil {
				helpers.Error(c, http.StatusBadRequest, "from_date harus YYYY-MM-DD", nil)
				return
			}
			query = query.Where("date >= ?", value)
		}
		if value := c.Query("to_date"); value != "" {
			if _, err := parseDate(value); err != nil {
				helpers.Error(c, http.StatusBadRequest, "to_date harus YYYY-MM-DD", nil)
				return
			}
			query = query.Where("date <= ?", value)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal menghitung transaksi", nil)
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
	if err := preloadTransaction(query.Order("date DESC, id DESC").Offset((page - 1) * perPage).Limit(perPage)).Find(&transactions).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal mengambil transaksi", nil)
		return
	}
	views := make([]gin.H, 0, len(transactions))
	for _, transaction := range transactions {
		views = append(views, transactionView(transaction, owner))
	}
	lastPage := 0
	if total > 0 {
		lastPage = int((total + int64(perPage) - 1) / int64(perPage))
	}
	from, to := int64(0), int64(0)
	if total > 0 {
		from = int64((page-1)*perPage + 1)
		to = int64(page * perPage)
		if to > total {
			to = total
		}
	}
	helpers.SuccessWithMeta(c, http.StatusOK, "transaksi berhasil diambil", views, transactionMeta{
		CurrentPage: page, PerPage: perPage, Total: total, LastPage: lastPage, From: from, To: to,
	})
}

func TransactionsStore(c *gin.Context) {
	var input transactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.Error(c, http.StatusBadRequest, "payload transaksi tidak valid", nil)
		return
	}
	if msg := validateTransactionInput(&input); msg != "" {
		helpers.Error(c, http.StatusBadRequest, msg, nil)
		return
	}
	if !isOwner(c) && input.Date != "" {
		helpers.Error(c, http.StatusForbidden, "Staff tidak dapat menentukan tanggal transaksi", nil)
		return
	}
	date, err := transactionDate(input.Date, todayDate())
	if err != nil {
		helpers.Error(c, http.StatusBadRequest, "date harus YYYY-MM-DD", nil)
		return
	}
	transaction := models.Transaction{
		BusinessID:      currentBusinessID(c),
		WalletID:        input.WalletID,
		CategoryID:      input.CategoryID,
		CreatedByUserID: currentUserID(c),
		Amount:          input.Amount,
		Type:            input.Type,
		Description:     input.Description,
		Date:            date,
	}
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if _, _, err := services.ValidateReferences(tx, transaction.BusinessID, transaction.WalletID, transaction.CategoryID, transaction.Type); err != nil {
			return err
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}
		return services.ApplyWalletDelta(tx, transaction.BusinessID, transaction.WalletID, services.Delta(transaction), false)
	})
	if errors.Is(err, services.ErrResourceUnavailable) {
		helpers.Error(c, http.StatusUnprocessableEntity, "wallet atau kategori tidak tersedia untuk transaksi", nil)
		return
	}
	if err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal menyimpan transaksi", nil)
		return
	}
	transaction = loadTransaction(transaction.ID)
	helpers.Success(c, http.StatusCreated, "transaksi berhasil disimpan", transactionView(transaction, isOwner(c)))
}

func loadTransaction(id uint) models.Transaction {
	var transaction models.Transaction
	preloadTransaction(config.DB).First(&transaction, id)
	return transaction
}

func TransactionsUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		helpers.Error(c, http.StatusBadRequest, "id transaksi tidak valid", nil)
		return
	}
	var input transactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.Error(c, http.StatusBadRequest, "payload transaksi tidak valid", nil)
		return
	}
	if msg := validateTransactionInput(&input); msg != "" {
		helpers.Error(c, http.StatusBadRequest, msg, nil)
		return
	}
	var updated models.Transaction
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.Transaction
		if err := tx.Where("id = ? AND business_id = ?", id, currentBusinessID(c)).First(&existing).Error; err != nil {
			return services.ErrTransactionNotFound
		}
		date, err := transactionDate(input.Date, existing.Date)
		if err != nil {
			return err
		}
		if _, _, err := services.ValidateReferences(tx, existing.BusinessID, input.WalletID, input.CategoryID, input.Type); err != nil {
			return err
		}
		if err := services.ApplyWalletDelta(tx, existing.BusinessID, existing.WalletID, -services.Delta(existing), true); err != nil {
			return err
		}
		userID := currentUserID(c)
		updates := map[string]any{
			"wallet_id":          input.WalletID,
			"category_id":        input.CategoryID,
			"amount":             input.Amount,
			"type":               input.Type,
			"description":        input.Description,
			"date":               date,
			"updated_by_user_id": userID,
		}
		if err := tx.Model(&existing).Updates(updates).Error; err != nil {
			return err
		}
		if err := services.ApplyWalletDelta(tx, existing.BusinessID, input.WalletID, services.Delta(models.Transaction{Amount: input.Amount, Type: input.Type}), false); err != nil {
			return err
		}
		updated = existing
		updated.WalletID, updated.CategoryID, updated.Amount, updated.Type = input.WalletID, input.CategoryID, input.Amount, input.Type
		updated.Description, updated.Date, updated.UpdatedByUserID = input.Description, date, &userID
		return nil
	})
	if errors.Is(err, services.ErrTransactionNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
		helpers.Error(c, http.StatusNotFound, "transaksi tidak ditemukan", nil)
		return
	}
	if errors.Is(err, services.ErrResourceUnavailable) {
		helpers.Error(c, http.StatusUnprocessableEntity, "wallet atau kategori tidak tersedia untuk transaksi", nil)
		return
	}
	if _, ok := err.(*time.ParseError); ok {
		helpers.Error(c, http.StatusBadRequest, "date harus YYYY-MM-DD", nil)
		return
	}
	if err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal memperbarui transaksi", nil)
		return
	}
	updated = loadTransaction(updated.ID)
	helpers.Success(c, http.StatusOK, "transaksi berhasil diperbarui", transactionView(updated, true))
}

func TransactionsDestroy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		helpers.Error(c, http.StatusBadRequest, "id transaksi tidak valid", nil)
		return
	}
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var transaction models.Transaction
		if err := tx.Where("id = ? AND business_id = ?", id, currentBusinessID(c)).First(&transaction).Error; err != nil {
			return services.ErrTransactionNotFound
		}
		if err := services.ApplyWalletDelta(tx, transaction.BusinessID, transaction.WalletID, -services.Delta(transaction), true); err != nil {
			return err
		}
		if err := tx.Model(&transaction).UpdateColumn("deleted_by_user_id", currentUserID(c)).Error; err != nil {
			return err
		}
		return tx.Delete(&transaction).Error
	})
	if errors.Is(err, services.ErrTransactionNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
		helpers.Error(c, http.StatusNotFound, "transaksi tidak ditemukan", nil)
		return
	}
	if errors.Is(err, services.ErrResourceUnavailable) {
		helpers.Error(c, http.StatusUnprocessableEntity, "wallet transaksi tidak tersedia", nil)
		return
	}
	if err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal melakukan void transaksi", nil)
		return
	}
	helpers.Success(c, http.StatusOK, "transaksi berhasil di-void", nil)
}

func TransactionsRestore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		helpers.Error(c, http.StatusBadRequest, "id transaksi tidak valid", nil)
		return
	}
	var restored models.Transaction
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("id = ? AND business_id = ?", id, currentBusinessID(c)).First(&restored).Error; err != nil {
			return services.ErrTransactionNotFound
		}
		if !restored.DeletedAt.Valid {
			return services.ErrTransactionNotFound
		}
		if err := services.ApplyWalletDelta(tx, restored.BusinessID, restored.WalletID, services.Delta(restored), true); err != nil {
			return err
		}
		userID := currentUserID(c)
		return tx.Unscoped().Model(&restored).Updates(map[string]any{
			"deleted_at":         nil,
			"deleted_by_user_id": nil,
			"updated_by_user_id": userID,
		}).Error
	})
	if errors.Is(err, services.ErrTransactionNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
		helpers.Error(c, http.StatusNotFound, "transaksi tidak ditemukan atau sudah aktif", nil)
		return
	}
	if errors.Is(err, services.ErrResourceUnavailable) {
		helpers.Error(c, http.StatusUnprocessableEntity, "wallet transaksi tidak tersedia", nil)
		return
	}
	if err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal memulihkan transaksi", nil)
		return
	}
	restored = loadTransaction(restored.ID)
	helpers.Success(c, http.StatusOK, "transaksi berhasil dipulihkan", transactionView(restored, true))
}
