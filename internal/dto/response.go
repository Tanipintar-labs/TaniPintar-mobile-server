package dto

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Errors  []string    `json:"errors"`
}

type RegisterResponse struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
}

func SuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
		Errors:  nil,
	})
}

func ErrorResponse(c *gin.Context, statusCode int, message string, errs []string) {
	c.JSON(statusCode, Response{
		Success: false,
		Message: message,
		Data:    nil,
		Errors:  errs,
	})
}

func ValidationErrorResponse(c *gin.Context, errs []string) {
	ErrorResponse(c, http.StatusUnprocessableEntity, "Validation failed", errs)
}
