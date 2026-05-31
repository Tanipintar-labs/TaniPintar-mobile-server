package repository

import (
	"errors"
	"time"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/domain"
	"gorm.io/gorm"
)

var ErrEmailAlreadyExists = errors.New("email already registered")

type UserRepository interface {
	FindByEmail(email string) (*domain.User, error)
	FindByID(id uint) (*domain.User, error)
	CreateWithProfile(tx *gorm.DB, user *domain.User) error
	UpdateWithProfile(tx *gorm.DB, user *domain.User, profile *domain.UserProfile) error
	SetVerified(userID uint) error
	DeleteUnverifiedOlderThan(duration time.Duration) (int64, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	result := r.db.Preload("Profile").Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

func (r *userRepository) FindByID(id uint) (*domain.User, error) {
	var user domain.User
	result := r.db.Preload("Profile").First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

func (r *userRepository) CreateWithProfile(tx *gorm.DB, user *domain.User) error {
	return tx.Create(user).Error
}

func (r *userRepository) UpdateWithProfile(tx *gorm.DB, user *domain.User, profile *domain.UserProfile) error {
	if err := tx.Model(user).Updates(map[string]interface{}{
		"hashed_password": user.HashedPassword,
	}).Error; err != nil {
		return err
	}

	return tx.Model(profile).Where("user_id = ?", user.ID).Updates(map[string]interface{}{
		"full_name":     profile.FullName,
		"birth_place":   profile.BirthPlace,
		"date_of_birth": profile.DateOfBirth,
	}).Error
}

func (r *userRepository) SetVerified(userID uint) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Update("is_verified", true).Error
}

func (r *userRepository) DeleteUnverifiedOlderThan(duration time.Duration) (int64, error) {
	cutoff := time.Now().Add(-duration)
	result := r.db.Unscoped().Where("is_verified = ? AND created_at < ?", false, cutoff).Delete(&domain.User{})
	return result.RowsAffected, result.Error
}
