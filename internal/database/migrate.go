package database

import (
	"log"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/domain"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) {
	err := db.AutoMigrate(
		&domain.User{},
		&domain.UserProfile{},
		&domain.OTPEntry{},
		&domain.RefreshToken{},
	)
	if err != nil {
		log.Fatalf("[FATAL] Failed to run database migrations: %v", err)
	}

	log.Println("[INFO] Database migrations completed successfully")
}
