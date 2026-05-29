package domain

import "time"

type User struct {
	ID             uint   `gorm:"primaryKey"`
	Email          string `gorm:"uniqueIndex;not null;size:255"`
	HashedPassword string `gorm:"not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Profile        UserProfile `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

type UserProfile struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uint      `gorm:"uniqueIndex;not null"`
	FullName    string    `gorm:"not null;size:255"`
	BirthPlace  string    `gorm:"not null;size:255"`
	DateOfBirth time.Time `gorm:"not null;type:date"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
