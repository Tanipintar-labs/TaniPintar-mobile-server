package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port   string
	AppEnv string
}

// Load reads environment variables from .env file and returns a Config struct.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file: ", err)
	}

	cfg := &Config{
		Port:   getEnv("PORT", "8080"),
		AppEnv: getEnv("APP_ENV", "development"),
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

func (c *Config) validate() {
	if c.AppEnv != "production" {
		log.Printf("[WARNING] APP_ENV is set to %q — not recommended for production deployment", c.AppEnv)
	}
}

