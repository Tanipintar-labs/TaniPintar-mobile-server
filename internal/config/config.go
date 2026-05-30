package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type DatabaseConfig struct {
	Host        string
	Port        string
	User        string
	Password    string
	Name        string
	SSLMode     string
	SSLRootCert string
}

type SmtpConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type Config struct {
	Port     string
	AppEnv   string
	Database DatabaseConfig
	Smtp     SmtpConfig
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatal("[FATAL] Error loading .env file: ", err)
	}

	cfg := &Config{
		Port:   getEnv("PORT", "8080"),
		AppEnv: getEnv("APP_ENV", "development"),
		Database: DatabaseConfig{
			Host:        requireEnv("DB_HOST"),
			Port:        getEnv("DB_PORT", "5432"),
			User:        requireEnv("DB_USER"),
			Password:    requireEnv("DB_PASSWORD"),
			Name:        requireEnv("DB_NAME"),
			SSLMode:     getEnv("DB_SSLMODE", "verify-full"),
			SSLRootCert: requireEnv("DB_SSL_ROOT_CERT"),
		},
		Smtp: SmtpConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getEnv("SMTP_PORT", "587"),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@tanipintar.com"),
		},
	}

	cfg.validate()

	return cfg
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)

	if !exists {
		log.Printf("[WARNING] Environment variable %s is not set, using default: %q", key, fallback)
		return fallback
	}

	if value == "" {
		log.Printf("[WARNING] Environment variable %s is empty, using default: %q", key, fallback)
		return fallback
	}

	return value
}

func requireEnv(key string) string {
	value, exists := os.LookupEnv(key)

	if !exists {
		log.Fatalf("[FATAL] Required environment variable %s is not set", key)
	}

	if value == "" {
		log.Fatalf("[FATAL] Required environment variable %s is empty", key)
	}

	return value
}

func (c *Config) validate() {
	if c.AppEnv != "production" {
		log.Printf("[WARNING] APP_ENV is set to %q — not recommended for production deployment", c.AppEnv)
	}

	if c.Database.SSLMode != "verify-full" && c.Database.SSLMode != "verify-ca" {
		log.Printf("[WARNING] DB_SSLMODE is set to %q — consider using \"verify-full\" for maximum security", c.Database.SSLMode)
	}

	if _, err := os.Stat(c.Database.SSLRootCert); os.IsNotExist(err) {
		log.Fatalf("[FATAL] SSL root certificate file not found: %s", c.Database.SSLRootCert)
	}
}
