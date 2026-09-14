package models

import (
	"time"

	"gorm.io/gorm"
)

type Wallet struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	BusinessID uint           `gorm:"not null;index:idx_wallets_business_deleted" json:"business_id"`
	Name       string         `gorm:"size:100;not null" json:"name"`
	Balance    int64          `gorm:"type:bigint;not null;default:0" json:"balance"`
	Currency   string         `gorm:"size:3;not null;default:IDR" json:"currency"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	Business *Business `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
}
