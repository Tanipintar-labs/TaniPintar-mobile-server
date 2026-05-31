package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type TokenService interface {
	GenerateAccessToken(userID uint, email string) (string, time.Duration, error)
	ValidateAccessToken(tokenString string) (*TokenClaims, error)
	GenerateRefreshTokenString() (string, error)
	GetRefreshTokenExpiry() time.Duration
}

type tokenService struct {
	secret       []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewTokenService(cfg config.JwtConfig) TokenService {
	return &tokenService{
		secret:        []byte(cfg.Secret),
		accessExpiry:  time.Duration(cfg.AccessExpiryMinutes) * time.Minute,
		refreshExpiry: time.Duration(cfg.RefreshExpiryDays) * 24 * time.Hour,
	}
}

func (s *tokenService) GenerateAccessToken(userID uint, email string) (string, time.Duration, error) {
	now := time.Now()
	claims := TokenClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiry)),
			Issuer:    "tanipintar-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign access token: %w", err)
	}

	return signed, s.accessExpiry, nil
}

func (s *tokenService) GetRefreshTokenExpiry() time.Duration {
	return s.refreshExpiry
}

func (s *tokenService) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (s *tokenService) GenerateRefreshTokenString() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
