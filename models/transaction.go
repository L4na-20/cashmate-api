package models

import (
	"time"

	"gorm.io/gorm"
)

type Transaction struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	BusinessID      uint           `gorm:"not null;index:idx_transactions_business_date" json:"business_id"`
	WalletID        uint           `gorm:"not null;index" json:"wallet_id"`
	CategoryID      uint           `gorm:"not null;index" json:"category_id"`
	CreatedByUserID uint           `gorm:"not null;index:idx_transactions_creator_date" json:"created_by_user_id"`
	Amount          int64          `gorm:"type:bigint unsigned;not null" json:"amount"`
	Type            string         `gorm:"size:10;not null;index" json:"type"`
	Description     string         `gorm:"size:255" json:"description"`
	Date            time.Time      `gorm:"type:date;index:idx_transactions_business_date" json:"date"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	UpdatedByUserID *uint          `gorm:"index" json:"updated_by_user_id,omitempty"`
	DeletedByUserID *uint          `gorm:"index" json:"deleted_by_user_id,omitempty"`

	Business  *Business          `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
	Wallet    *Wallet            `gorm:"foreignKey:WalletID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"wallet,omitempty"`
	Category  *Category          `gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"category,omitempty"`
	CreatedBy *User              `gorm:"foreignKey:CreatedByUserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"created_by,omitempty"`
	Photos    []TransactionPhoto `gorm:"foreignKey:TransactionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"photos,omitempty"`
	UpdatedBy *User              `gorm:"foreignKey:UpdatedByUserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
	DeletedBy *User              `gorm:"foreignKey:DeletedByUserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
}
