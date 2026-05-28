package main

import (
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/config"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/server"
)

func main() {
	cfg := config.Load()

	router := server.NewRouter(cfg)
	server.RegisterRoutes(router)

	server.Run(router, cfg)
}