package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/dto"
	"github.com/gin-gonic/gin"
)

type emailRateLimiter struct {
	mu       sync.RWMutex
	lastSent map[string]time.Time
	cooldown time.Duration
}

func newEmailRateLimiter(cooldown time.Duration) *emailRateLimiter {
	return &emailRateLimiter{
		lastSent: make(map[string]time.Time),
		cooldown: cooldown,
	}
}

func (rl *emailRateLimiter) isAllowed(email string) (bool, time.Duration) {
	rl.mu.RLock()
	lastTime, exists := rl.lastSent[email]
	rl.mu.RUnlock()

	if !exists {
		return true, 0
	}

	elapsed := time.Since(lastTime)
	if elapsed >= rl.cooldown {
		return true, 0
	}

	return false, rl.cooldown - elapsed
}

func (rl *emailRateLimiter) record(email string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.lastSent[email] = time.Now()
}

func OTPEmailRateLimiter(cooldown time.Duration) gin.HandlerFunc {
	limiter := newEmailRateLimiter(cooldown)

	return func(c *gin.Context) {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			dto.ErrorResponse(c, http.StatusBadRequest, "Invalid request body", []string{"unable to read request body"})
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var payload struct {
			Email string `json:"email"`
		}
		if err := json.Unmarshal(bodyBytes, &payload); err != nil || payload.Email == "" {
			c.Next()
			return
		}

		email := strings.ToLower(strings.TrimSpace(payload.Email))

		allowed, remaining := limiter.isAllowed(email)
		if !allowed {
			dto.ErrorResponse(c, http.StatusTooManyRequests,
				"Please wait before requesting another OTP",
				[]string{
					"You can request a new OTP in " + remaining.Round(time.Second).String(),
				},
			)
			c.Abort()
			return
		}

		c.Next()

		if c.Writer.Status() < 400 {
			limiter.record(email)
		}
	}
}
