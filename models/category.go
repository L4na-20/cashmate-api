package models

import (
	"time"

	"gorm.io/gorm"
)

type Category struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    *uint          `gorm:"index" json:"user_id"`
	Name      string         `gorm:"size:100;not null" json:"name"`
	Type      string         `gorm:"size:10;not null;index" json:"type"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}
