package domain

import "time"

type RefreshToken struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"index;not null"`
	Token     string    `gorm:"uniqueIndex;not null;size:64"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}
