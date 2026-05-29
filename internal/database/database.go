package database

import (
	"fmt"
	"log"
	"time"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
		cfg.Database.SSLRootCert,
	)

	logLevel := logger.Info
	if cfg.AppEnv == "production" {
		logLevel = logger.Silent
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		log.Fatalf("[FATAL] Failed to connect to database: %v", err)
	}

	// Configure connection pooling on the underlying *sql.DB.
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("[FATAL] Failed to get underlying database connection: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("[FATAL] Failed to ping database: %v", err)
	}

	log.Println("[INFO] Database connected successfully")

	RunMigrations(db)

	return db
}

func Close(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("[ERROR] Failed to get underlying database connection for closing: %v", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		log.Printf("[ERROR] Failed to close database connection: %v", err)
		return
	}

	log.Println("[INFO] Database connection closed")
}
