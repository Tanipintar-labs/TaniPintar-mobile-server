package middleware

import (
	"net/http"
	"sync"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/dto"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

func newIPRateLimiter(r rate.Limit, burst int) *ipRateLimiter {
	return &ipRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    burst,
	}
}

func (rl *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[ip]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists = rl.limiters[ip]
	if exists {
		return limiter
	}

	limiter = rate.NewLimiter(rl.rate, rl.burst)
	rl.limiters[ip] = limiter
	return limiter
}

func RateLimiter(r rate.Limit, burst int) gin.HandlerFunc {
	limiter := newIPRateLimiter(r, burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := limiter.getLimiter(ip)

		if !l.Allow() {
			dto.ErrorResponse(c, http.StatusTooManyRequests,
				"Too many requests, please try again later",
				[]string{"rate limit exceeded"},
			)
			c.Abort()
			return
		}

		c.Next()
	}
}
