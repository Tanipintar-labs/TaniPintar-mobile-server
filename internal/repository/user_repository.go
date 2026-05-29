package repository

import (
	"errors"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/domain"
	"gorm.io/gorm"
)

var ErrEmailAlreadyExists = errors.New("email already registered")

type UserRepository interface {
	FindByEmail(email string) (*domain.User, error)
	CreateWithProfile(tx *gorm.DB, user *domain.User) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("email = ?", email).First(&user)
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

func (r *userRepository) BeginTx() *gorm.DB {
	return r.db.Begin()
}

func (r *userRepository) DB() *gorm.DB {
	return r.db
}
