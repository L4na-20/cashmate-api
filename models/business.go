package models

import (
	"time"

	"gorm.io/gorm"
)

// Business is the tenant boundary for every CashMate operational record.
type Business struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:150;not null" json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	Users        []User        `gorm:"foreignKey:BusinessID" json:"-"`
	Wallets      []Wallet      `gorm:"foreignKey:BusinessID" json:"-"`
	Categories   []Category    `gorm:"foreignKey:BusinessID" json:"-"`
	Transactions []Transaction `gorm:"foreignKey:BusinessID" json:"-"`
}
