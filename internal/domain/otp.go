package domain

import "time"

type OTPEntry struct {
	ID          uint   `gorm:"primaryKey"`
	Email       string `gorm:"index;not null;size:255"`
	Code        string `gorm:"not null;size:6"`
	Attempts    int    `gorm:"default:0"`
	FrozenUntil *time.Time
	ExpiresAt   time.Time `gorm:"not null"`
	CreatedAt   time.Time
}
