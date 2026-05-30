package worker

import (
	"fmt"
	"log/slog"
	"net/smtp"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/config"
)

type EmailSender interface {
	SendOTP(to string, code string) error
}

type smtpEmailSender struct {
	cfg    config.SmtpConfig
	logger *slog.Logger
}

func NewSmtpEmailSender(cfg config.SmtpConfig, logger *slog.Logger) EmailSender {
	return &smtpEmailSender{cfg: cfg, logger: logger}
}

func (s *smtpEmailSender) SendOTP(to string, code string) error {
	subject := "TaniPintar - Email Verification Code"
	body := fmt.Sprintf(
		"Your verification code is: %s\n\nThis code will expire in 5 minutes.\nIf you did not request this, please ignore this email.",
		code,
	)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s",
		s.cfg.From, to, subject, body,
	)

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	addr := s.cfg.Host + ":" + s.cfg.Port

	if err := smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send email to %s: %w", to, err)
	}

	return nil
}

type logEmailSender struct {
	logger *slog.Logger
}

func NewLogEmailSender(logger *slog.Logger) EmailSender {
	return &logEmailSender{logger: logger}
}

func (s *logEmailSender) SendOTP(to string, code string) error {
	s.logger.Info("========== OTP EMAIL (dev mode) ==========",
		slog.String("to", to),
		slog.String("code", code),
	)
	return nil
}

func SendOTPAsync(sender EmailSender, logger *slog.Logger, to string, code string) {
	go func() {
		if err := sender.SendOTP(to, code); err != nil {
			logger.Error("failed to send OTP email",
				slog.String("to", to),
				slog.String("error", err.Error()),
			)
		} else {
			logger.Info("OTP email sent successfully",
				slog.String("to", to),
			)
		}
	}()
}
