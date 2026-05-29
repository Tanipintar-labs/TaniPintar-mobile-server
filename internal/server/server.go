package server

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/config"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/handler"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/middleware"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/repository"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
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

func RegisterRoutes(router *gin.Engine, db *gorm.DB) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	router.GET("/ping", handler.HealthCheck())

	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(db, userRepo, logger)
	authHandler := handler.NewAuthHandler(authService, logger)

	api := router.Group("/api")
	{
		api.POST("/register",
			middleware.RateLimiter(rate.Every(12*time.Second), 5),
			authHandler.Register(),
		)
	}
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
