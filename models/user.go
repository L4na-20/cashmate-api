package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	BusinessID  uint           `gorm:"not null;index:idx_users_business_role" json:"business_id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Email       string         `gorm:"size:100;uniqueIndex;not null" json:"email"`
	Password    string         `gorm:"size:255;not null" json:"-"`
	Role        string         `gorm:"size:20;not null;default:STAFF;index:idx_users_business_role" json:"role"`
	AuthVersion uint64         `gorm:"not null;default:0" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	Business *Business `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
}
