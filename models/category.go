package models

import (
	"time"

	"gorm.io/gorm"
)

type Category struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	BusinessID *uint          `gorm:"index:idx_categories_business_type" json:"business_id"`
	Name       string         `gorm:"size:100;not null" json:"name"`
	Type       string         `gorm:"size:10;not null;index:idx_categories_business_type" json:"type"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	Business *Business `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
}
