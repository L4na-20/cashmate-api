package models

import "time"

// TransactionPhoto is an uploaded proof-of-transaction image attached to a
// single Transaction. Files live on disk; only the public URL is persisted.
type TransactionPhoto struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TransactionID uint      `gorm:"not null;index" json:"transaction_id"`
	URL           string    `gorm:"size:255;not null" json:"url"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
