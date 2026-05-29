package service

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/domain"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/dto"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmailTaken        = errors.New("email is already registered")
	ErrInvalidDateFormat = errors.New("date_of_birth must be in YYYY-MM-DD format")
)

type AuthService interface {
	Register(req *dto.RegisterRequest) (*dto.RegisterResponse, error)
}

type authService struct {
	db       *gorm.DB
	userRepo repository.UserRepository
	logger   *slog.Logger
}

func NewAuthService(db *gorm.DB, userRepo repository.UserRepository, logger *slog.Logger) AuthService {
	return &authService{
		db:       db,
		userRepo: userRepo,
		logger:   logger,
	}
}

func (s *authService) Register(req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)
	req.BirthPlace = strings.TrimSpace(req.BirthPlace)

	dateOfBirth, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		s.logger.Warn("invalid date format received",
			slog.String("date_of_birth", req.DateOfBirth),
		)
		return nil, ErrInvalidDateFormat
	}

	existingUser, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		s.logger.Error("failed to check existing user",
			slog.String("email", req.Email),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		s.logger.Warn("registration attempt with existing email",
			slog.String("email", req.Email),
		)
		return nil, ErrEmailTaken
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		s.logger.Error("failed to hash password",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		Email:          req.Email,
		HashedPassword: string(hashedPassword),
		Profile: domain.UserProfile{
			FullName:    req.FullName,
			BirthPlace:  req.BirthPlace,
			DateOfBirth: dateOfBirth,
		},
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		s.logger.Error("failed to begin transaction",
			slog.String("error", tx.Error.Error()),
		)
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	if err := s.userRepo.CreateWithProfile(tx, user); err != nil {
		tx.Rollback()
		s.logger.Error("failed to create user, transaction rolled back",
			slog.String("email", req.Email),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		s.logger.Error("failed to commit transaction",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("user registered successfully",
		slog.Uint64("user_id", uint64(user.ID)),
		slog.String("email", user.Email),
	)

	return &dto.RegisterResponse{
		UserID: user.ID,
		Email:  user.Email,
	}, nil
}
