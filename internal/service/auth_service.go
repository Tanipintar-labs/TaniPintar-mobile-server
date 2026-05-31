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
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/util"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/worker"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmailTaken         = errors.New("email is already registered and verified")
	ErrInvalidDateFormat  = errors.New("date_of_birth must be in YYYY-MM-DD format")
	ErrOTPExpired         = errors.New("OTP has expired, please request a new one")
	ErrOTPInvalid         = errors.New("invalid OTP code")
	ErrOTPFrozen          = errors.New("too many failed attempts, please try again later")
	ErrOTPNotFound        = errors.New("no OTP found for this email, please register first")
	ErrUserNotFound       = errors.New("no unverified account found for this email")
	ErrAlreadyVerified    = errors.New("email is already verified")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailNotVerified   = errors.New("email is not verified")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

type AuthService interface {
	Register(req *dto.RegisterRequest) (*dto.RegisterResponse, error)
	VerifyOTP(req *dto.VerifyOTPRequest) error
	ResendOTP(req *dto.ResendOTPRequest) error
	Login(req *dto.LoginRequest) (*dto.LoginResponse, error)
	RefreshToken(req *dto.RefreshTokenRequest) (*dto.LoginResponse, error)
	GetUserProfile(userID uint) (*dto.UserProfileResponse, error)
}

type authService struct {
	db           *gorm.DB
	userRepo     repository.UserRepository
	otpRepo      repository.OTPRepository
	tokenRepo    repository.TokenRepository
	tokenService TokenService
	emailSender  worker.EmailSender
	logger       *slog.Logger
}

func NewAuthService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	otpRepo repository.OTPRepository,
	tokenRepo repository.TokenRepository,
	tokenService TokenService,
	emailSender worker.EmailSender,
	logger *slog.Logger,
) AuthService {
	return &authService{
		db:           db,
		userRepo:     userRepo,
		otpRepo:      otpRepo,
		tokenRepo:    tokenRepo,
		tokenService: tokenService,
		emailSender:  emailSender,
		logger:       logger,
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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		s.logger.Error("failed to hash password", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to hash password: %w", err)
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
		if existingUser.IsVerified {
			s.logger.Warn("registration attempt with verified email",
				slog.String("email", req.Email),
			)
			return nil, ErrEmailTaken
		}

		return s.handleUnverifiedReRegistration(existingUser, req, string(hashedPassword), dateOfBirth)
	}

	return s.handleNewRegistration(req, string(hashedPassword), dateOfBirth)
}

func (s *authService) handleNewRegistration(req *dto.RegisterRequest, hashedPassword string, dateOfBirth time.Time) (*dto.RegisterResponse, error) {
	user := &domain.User{
		Email:          req.Email,
		HashedPassword: hashedPassword,
		IsVerified:     false,
		Profile: domain.UserProfile{
			FullName:    req.FullName,
			BirthPlace:  req.BirthPlace,
			DateOfBirth: dateOfBirth,
		},
	}

	tx := s.db.Begin()
	if tx.Error != nil {
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
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.generateAndSendOTP(req.Email)

	s.logger.Info("new user registered (pending verification)",
		slog.Uint64("user_id", uint64(user.ID)),
		slog.String("email", user.Email),
	)

	return &dto.RegisterResponse{
		UserID: user.ID,
		Email:  user.Email,
	}, nil
}

func (s *authService) handleUnverifiedReRegistration(existingUser *domain.User, req *dto.RegisterRequest, hashedPassword string, dateOfBirth time.Time) (*dto.RegisterResponse, error) {
	existingUser.HashedPassword = hashedPassword
	profile := &domain.UserProfile{
		FullName:    req.FullName,
		BirthPlace:  req.BirthPlace,
		DateOfBirth: dateOfBirth,
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	if err := s.userRepo.UpdateWithProfile(tx, existingUser, profile); err != nil {
		tx.Rollback()
		s.logger.Error("failed to update unverified user, transaction rolled back",
			slog.String("email", req.Email),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.otpRepo.DeleteByEmail(req.Email)
	s.generateAndSendOTP(req.Email)

	s.logger.Info("unverified user re-registered (profile updated, new OTP sent)",
		slog.Uint64("user_id", uint64(existingUser.ID)),
		slog.String("email", existingUser.Email),
	)

	return &dto.RegisterResponse{
		UserID: existingUser.ID,
		Email:  existingUser.Email,
	}, nil
}

func (s *authService) VerifyOTP(req *dto.VerifyOTPRequest) error {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	entry, err := s.otpRepo.FindLatestByEmail(email)
	if err != nil {
		s.logger.Error("failed to find OTP", slog.String("email", email), slog.String("error", err.Error()))
		return fmt.Errorf("failed to find OTP: %w", err)
	}
	if entry == nil {
		return ErrOTPNotFound
	}

	if entry.FrozenUntil != nil && time.Now().Before(*entry.FrozenUntil) {
		remaining := time.Until(*entry.FrozenUntil).Round(time.Minute)
		s.logger.Warn("OTP verification frozen",
			slog.String("email", email),
			slog.String("remaining", remaining.String()),
		)
		return ErrOTPFrozen
	}

	if time.Now().After(entry.ExpiresAt) {
		return ErrOTPExpired
	}

	if entry.Code != req.Code {
		s.otpRepo.IncrementAttempts(entry.ID)
		newAttempts := entry.Attempts + 1

		if newAttempts >= 3 {
			freezeUntil := time.Now().Add(15 * time.Minute)
			s.otpRepo.FreezeByID(entry.ID, freezeUntil)
			s.logger.Warn("OTP verification frozen due to max attempts",
				slog.String("email", email),
				slog.Int("attempts", newAttempts),
			)
			return ErrOTPFrozen
		}

		s.logger.Warn("invalid OTP attempt",
			slog.String("email", email),
			slog.Int("attempts", newAttempts),
		)
		return ErrOTPInvalid
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil || user == nil {
		return fmt.Errorf("failed to find user for verification: %w", err)
	}

	if err := s.userRepo.SetVerified(user.ID); err != nil {
		return fmt.Errorf("failed to set user as verified: %w", err)
	}

	s.otpRepo.DeleteByEmail(email)

	s.logger.Info("user email verified successfully",
		slog.Uint64("user_id", uint64(user.ID)),
		slog.String("email", email),
	)

	return nil
}

func (s *authService) ResendOTP(req *dto.ResendOTPRequest) error {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}
	if user.IsVerified {
		return ErrAlreadyVerified
	}

	s.otpRepo.DeleteByEmail(email)
	s.generateAndSendOTP(email)

	s.logger.Info("OTP resent",
		slog.String("email", email),
	)

	return nil
}

func (s *authService) generateAndSendOTP(email string) {
	code, err := util.GenerateOTP()
	if err != nil {
		s.logger.Error("failed to generate OTP",
			slog.String("email", email),
			slog.String("error", err.Error()),
		)
		return
	}

	otpEntry := &domain.OTPEntry{
		Email:     email,
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := s.otpRepo.Create(otpEntry); err != nil {
		s.logger.Error("failed to save OTP",
			slog.String("email", email),
			slog.String("error", err.Error()),
		)
		return
	}

	worker.SendOTPAsync(s.emailSender, s.logger, email, code)
}

func (s *authService) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		s.logger.Error("failed to find user for login",
			slog.String("email", email),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password)); err != nil {
		s.logger.Warn("failed login attempt (wrong password)",
			slog.String("email", email),
		)
		return nil, ErrInvalidCredentials
	}

	if !user.IsVerified {
		s.logger.Warn("login attempt with unverified email",
			slog.String("email", email),
		)
		return nil, ErrEmailNotVerified
	}

	return s.generateTokenPair(user.ID, user.Email)
}

func (s *authService) RefreshToken(req *dto.RefreshTokenRequest) (*dto.LoginResponse, error) {
	existing, err := s.tokenRepo.FindByToken(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}
	if existing == nil {
		return nil, ErrInvalidRefreshToken
	}

	if time.Now().After(existing.ExpiresAt) {
		s.tokenRepo.DeleteByToken(req.RefreshToken)
		return nil, ErrInvalidRefreshToken
	}

	s.tokenRepo.DeleteByToken(req.RefreshToken)

	user, err := s.userRepo.FindByID(existing.UserID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("failed to find user for token refresh: %w", err)
	}

	s.logger.Info("token refreshed",
		slog.Uint64("user_id", uint64(user.ID)),
		slog.String("email", user.Email),
	)

	return s.generateTokenPair(user.ID, user.Email)
}

func (s *authService) GetUserProfile(userID uint) (*dto.UserProfileResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &dto.UserProfileResponse{
		UserID:      user.ID,
		Email:       user.Email,
		FullName:    user.Profile.FullName,
		BirthPlace:  user.Profile.BirthPlace,
		DateOfBirth: user.Profile.DateOfBirth.Format("2006-01-02"),
	}, nil
}

func (s *authService) generateTokenPair(userID uint, email string) (*dto.LoginResponse, error) {
	accessToken, expiresIn, err := s.tokenService.GenerateAccessToken(userID, email)
	if err != nil {
		s.logger.Error("failed to generate access token", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshTokenStr, err := s.tokenService.GenerateRefreshTokenString()
	if err != nil {
		s.logger.Error("failed to generate refresh token", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	rt := &domain.RefreshToken{
		UserID:    userID,
		Token:     refreshTokenStr,
		ExpiresAt: time.Now().Add(s.tokenService.GetRefreshTokenExpiry()),
	}
	if err := s.tokenRepo.Create(rt); err != nil {
		s.logger.Error("failed to save refresh token", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	s.logger.Info("user logged in successfully",
		slog.Uint64("user_id", uint64(userID)),
		slog.String("email", email),
	)

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int64(expiresIn.Seconds()),
	}, nil
}
