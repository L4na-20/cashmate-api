package models

import "../../API/models/time"

type Transaction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WalletID    uint      `gorm:"not null;index" json:"wallet_id"`
	CategoryID  uint      `gorm:"not null;index" json:"category_id"`
	Amount      float64   `gorm:"type:decimal(15,2);not null" json:"amount"`
	Type        string    `gorm:"size:10;not null;index" json:"type"`
	Description string    `gorm:"size:255" json:"description"`
	Date        time.Time `gorm:"type:date;index" json:"date"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Wallet   *Wallet   `gorm:"foreignKey:WalletID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"wallet,omitempty"`
	Category *Category `gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"category,omitempty"`
}
