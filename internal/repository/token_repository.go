package repository

import (
	"errors"
	"time"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/domain"
	"gorm.io/gorm"
)

type TokenRepository interface {
	Create(token *domain.RefreshToken) error
	FindByToken(token string) (*domain.RefreshToken, error)
	DeleteByToken(token string) error
	DeleteByUserID(userID uint) error
	DeleteExpired() (int64, error)
}

type tokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) TokenRepository {
	return &tokenRepository{db: db}
}

func (r *tokenRepository) Create(token *domain.RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *tokenRepository) FindByToken(token string) (*domain.RefreshToken, error) {
	var rt domain.RefreshToken
	result := r.db.Where("token = ?", token).First(&rt)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &rt, nil
}

func (r *tokenRepository) DeleteByToken(token string) error {
	return r.db.Where("token = ?", token).Delete(&domain.RefreshToken{}).Error
}

func (r *tokenRepository) DeleteByUserID(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&domain.RefreshToken{}).Error
}

func (r *tokenRepository) DeleteExpired() (int64, error) {
	result := r.db.Where("expires_at < ?", time.Now()).Delete(&domain.RefreshToken{})
	return result.RowsAffected, result.Error
}
