package server

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/config"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/handler"
	"github.com/gin-gonic/gin"
)

func NewRouter(cfg *config.Config) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.Default()
	return router
}

func RegisterRoutes(router *gin.Engine) {
	router.GET("/ping", handler.HealthCheck())
}

func Run(router *gin.Engine, cfg *config.Config) {
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("[INFO] Server is starting on port %s (env: %s)", cfg.Port, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Failed to start server: %v", err)
		}
	}()

	<-ctx.Done()
	stop()
	log.Println("[INFO] Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[WARNING] Server forced to shutdown: %v", err)
	}

	log.Println("[INFO] Server exited gracefully")
}
