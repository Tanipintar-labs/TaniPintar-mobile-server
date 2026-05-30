package worker

import (
	"log/slog"
	"time"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/repository"
	"github.com/robfig/cron/v3"
)

type CleanupWorker struct {
	cron     *cron.Cron
	userRepo repository.UserRepository
	otpRepo  repository.OTPRepository
	logger   *slog.Logger
}

func NewCleanupWorker(userRepo repository.UserRepository, otpRepo repository.OTPRepository, logger *slog.Logger) *CleanupWorker {
	return &CleanupWorker{
		cron:     cron.New(),
		userRepo: userRepo,
		otpRepo:  otpRepo,
		logger:   logger,
	}
}

func (w *CleanupWorker) Start() {
	w.cron.AddFunc("@every 1h", func() {
		w.cleanupUnverifiedAccounts()
		w.cleanupExpiredOTPs()
	})

	w.cron.Start()
	w.logger.Info("cleanup worker started (runs every 1 hour)")
}

func (w *CleanupWorker) Stop() {
	ctx := w.cron.Stop()
	<-ctx.Done()
	w.logger.Info("cleanup worker stopped")
}

func (w *CleanupWorker) cleanupUnverifiedAccounts() {
	count, err := w.userRepo.DeleteUnverifiedOlderThan(24 * time.Hour)
	if err != nil {
		w.logger.Error("failed to cleanup unverified accounts",
			slog.String("error", err.Error()),
		)
		return
	}

	if count > 0 {
		w.logger.Info("cleaned up unverified ghost accounts",
			slog.Int64("deleted_count", count),
		)
	}
}

func (w *CleanupWorker) cleanupExpiredOTPs() {
	count, err := w.otpRepo.DeleteExpired()
	if err != nil {
		w.logger.Error("failed to cleanup expired OTPs",
			slog.String("error", err.Error()),
		)
		return
	}

	if count > 0 {
		w.logger.Info("cleaned up expired OTP entries",
			slog.Int64("deleted_count", count),
		)
	}
}
