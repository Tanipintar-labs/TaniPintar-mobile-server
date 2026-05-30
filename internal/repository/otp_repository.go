package repository

import (
	"errors"
	"time"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/domain"
	"gorm.io/gorm"
)

type OTPRepository interface {
	Create(entry *domain.OTPEntry) error
	FindLatestByEmail(email string) (*domain.OTPEntry, error)
	IncrementAttempts(id uint) error
	FreezeByID(id uint, until time.Time) error
	DeleteByEmail(email string) error
	DeleteExpired() (int64, error)
}

type otpRepository struct {
	db *gorm.DB
}

func NewOTPRepository(db *gorm.DB) OTPRepository {
	return &otpRepository{db: db}
}

func (r *otpRepository) Create(entry *domain.OTPEntry) error {
	return r.db.Create(entry).Error
}

func (r *otpRepository) FindLatestByEmail(email string) (*domain.OTPEntry, error) {
	var entry domain.OTPEntry
	result := r.db.Where("email = ?", email).Order("created_at DESC").First(&entry)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &entry, nil
}

func (r *otpRepository) IncrementAttempts(id uint) error {
	return r.db.Model(&domain.OTPEntry{}).Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error
}

func (r *otpRepository) FreezeByID(id uint, until time.Time) error {
	return r.db.Model(&domain.OTPEntry{}).Where("id = ?", id).
		Update("frozen_until", until).Error
}

func (r *otpRepository) DeleteByEmail(email string) error {
	return r.db.Where("email = ?", email).Delete(&domain.OTPEntry{}).Error
}

func (r *otpRepository) DeleteExpired() (int64, error) {
	result := r.db.Where("expires_at < ?", time.Now()).Delete(&domain.OTPEntry{})
	return result.RowsAffected, result.Error
}
