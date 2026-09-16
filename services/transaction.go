package services

import (
	"errors"

	"cashmate-api/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrResourceUnavailable = errors.New("resource tidak tersedia")
var ErrTransactionNotFound = errors.New("transaksi tidak ditemukan")
var ErrPhotoNotFound = errors.New("foto transaksi tidak ditemukan")

// Delta returns the signed balance effect of a transaction.
func Delta(transaction models.Transaction) int64 {
	if transaction.Type == models.TransactionExpense {
		return -transaction.Amount
	}
	return transaction.Amount
}

// ApplyWalletDelta performs the only balance mutation allowed by the domain.
// The SQL expression avoids stale read-modify-write races.
func ApplyWalletDelta(tx *gorm.DB, businessID, walletID uint, delta int64, includeDeleted bool) error {
	query := tx.Model(&models.Wallet{})
	if includeDeleted {
		query = query.Unscoped()
	}
	result := query.Where("id = ? AND business_id = ?", walletID, businessID).
		UpdateColumn("balance", gorm.Expr("balance + ?", delta))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrResourceUnavailable
	}
	return nil
}

func lockActiveWallet(tx *gorm.DB, businessID, walletID uint) (models.Wallet, error) {
	var wallet models.Wallet
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND business_id = ?", walletID, businessID).
		First(&wallet).Error
	return wallet, err
}

func lockAvailableCategory(tx *gorm.DB, businessID, categoryID uint) (models.Category, error) {
	var category models.Category
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND (business_id = ? OR business_id IS NULL)", categoryID, businessID).
		First(&category).Error
	return category, err
}

// ValidateReferences locks the active wallet/category and checks matching
// transaction type inside the same DB transaction as the write.
func ValidateReferences(tx *gorm.DB, businessID, walletID, categoryID uint, transactionType string) (models.Wallet, models.Category, error) {
	wallet, err := lockActiveWallet(tx, businessID, walletID)
	if err != nil {
		return models.Wallet{}, models.Category{}, ErrResourceUnavailable
	}
	category, err := lockAvailableCategory(tx, businessID, categoryID)
	if err != nil || category.Type != transactionType {
		return models.Wallet{}, models.Category{}, ErrResourceUnavailable
	}
	return wallet, category, nil
}
