package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/dto"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	service service.AuthService
	logger  *slog.Logger
}

func NewAuthHandler(svc service.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		service: svc,
		logger:  logger,
	}
}

func (h *AuthHandler) Register() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.RegisterRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			errs := formatValidationErrors(err)
			h.logger.Warn("registration validation failed",
				slog.String("ip", c.ClientIP()),
				slog.Any("errors", errs),
			)
			dto.ValidationErrorResponse(c, errs)
			return
		}

		h.logger.Info("registration attempt",
			slog.String("email", req.Email),
			slog.String("ip", c.ClientIP()),
		)

		result, err := h.service.Register(&req)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEmailTaken):
				dto.ErrorResponse(c, http.StatusConflict,
					"Registration failed",
					[]string{"Email is already registered"},
				)
			case errors.Is(err, service.ErrInvalidDateFormat):
				dto.ValidationErrorResponse(c, []string{
					"date_of_birth must be in YYYY-MM-DD format",
				})
			default:
				h.logger.Error("registration failed",
					slog.String("email", req.Email),
					slog.String("error", err.Error()),
				)
				dto.ErrorResponse(c, http.StatusInternalServerError,
					"Registration failed",
					[]string{"An internal error occurred, please try again later"},
				)
			}
			return
		}

		dto.SuccessResponse(c, http.StatusCreated, "Registration successful, please verify your email", result)
	}
}

func (h *AuthHandler) VerifyOTP() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.VerifyOTPRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			errs := formatValidationErrors(err)
			dto.ValidationErrorResponse(c, errs)
			return
		}

		h.logger.Info("OTP verification attempt",
			slog.String("email", req.Email),
			slog.String("ip", c.ClientIP()),
		)

		err := h.service.VerifyOTP(&req)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrOTPNotFound):
				dto.ErrorResponse(c, http.StatusNotFound,
					"Verification failed",
					[]string{"No OTP found for this email, please register first"},
				)
			case errors.Is(err, service.ErrOTPExpired):
				dto.ErrorResponse(c, http.StatusGone,
					"Verification failed",
					[]string{"OTP has expired, please request a new one"},
				)
			case errors.Is(err, service.ErrOTPInvalid):
				dto.ErrorResponse(c, http.StatusUnprocessableEntity,
					"Verification failed",
					[]string{"Invalid OTP code"},
				)
			case errors.Is(err, service.ErrOTPFrozen):
				dto.ErrorResponse(c, http.StatusTooManyRequests,
					"Verification failed",
					[]string{"Too many failed attempts, please try again in 15 minutes"},
				)
			default:
				h.logger.Error("OTP verification failed",
					slog.String("email", req.Email),
					slog.String("error", err.Error()),
				)
				dto.ErrorResponse(c, http.StatusInternalServerError,
					"Verification failed",
					[]string{"An internal error occurred, please try again later"},
				)
			}
			return
		}

		dto.SuccessResponse(c, http.StatusOK, "Email verified successfully", nil)
	}
}

func (h *AuthHandler) ResendOTP() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ResendOTPRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			errs := formatValidationErrors(err)
			dto.ValidationErrorResponse(c, errs)
			return
		}

		h.logger.Info("OTP resend request",
			slog.String("email", req.Email),
			slog.String("ip", c.ClientIP()),
		)

		err := h.service.ResendOTP(&req)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrUserNotFound):
				dto.ErrorResponse(c, http.StatusNotFound,
					"Resend failed",
					[]string{"No unverified account found for this email"},
				)
			case errors.Is(err, service.ErrAlreadyVerified):
				dto.ErrorResponse(c, http.StatusConflict,
					"Resend failed",
					[]string{"Email is already verified"},
				)
			default:
				h.logger.Error("OTP resend failed",
					slog.String("email", req.Email),
					slog.String("error", err.Error()),
				)
				dto.ErrorResponse(c, http.StatusInternalServerError,
					"Resend failed",
					[]string{"An internal error occurred, please try again later"},
				)
			}
			return
		}

		dto.SuccessResponse(c, http.StatusOK, "OTP sent successfully", nil)
	}
}

func formatValidationErrors(err error) []string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		errs := make([]string, 0, len(ve))
		for _, fe := range ve {
			errs = append(errs, formatFieldError(fe))
		}
		return errs
	}
	return []string{err.Error()}
}

func formatFieldError(fe validator.FieldError) string {
	field := camelToSnake(fe.Field())

	switch fe.Tag() {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "min":
		return field + " must be at least " + fe.Param() + " characters"
	case "max":
		return field + " must not exceed " + fe.Param() + " characters"
	case "len":
		return field + " must be exactly " + fe.Param() + " characters"
	case "eqfield":
		return "password and password_confirmation must match"
	default:
		return field + " is invalid"
	}
}

func camelToSnake(s string) string {
	var result []byte
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c+32))
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}
