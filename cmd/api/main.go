package main

import (
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/config"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/database"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/server"
)

func main() {
	cfg := config.Load()

	db := database.Connect(cfg)
	defer database.Close(db)

	router := server.NewRouter(cfg)
	server.RegisterRoutes(router)

	server.Run(router, cfg)
}