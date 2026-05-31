package middleware

import (
	"net/http"
	"strings"

	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/dto"
	"github.com/Tanipintar-labs/TaniPintar-mobile-server/internal/service"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware(tokenService service.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			dto.ErrorResponse(c, http.StatusUnauthorized,
				"Authentication required",
				[]string{"Authorization header is missing"},
			)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			dto.ErrorResponse(c, http.StatusUnauthorized,
				"Authentication required",
				[]string{"Authorization header must be in 'Bearer <token>' format"},
			)
			c.Abort()
			return
		}

		claims, err := tokenService.ValidateAccessToken(parts[1])
		if err != nil {
			dto.ErrorResponse(c, http.StatusUnauthorized,
				"Authentication failed",
				[]string{"Invalid or expired access token"},
			)
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		c.Next()
	}
}
