package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type errorBody struct {
	Success bool      `json:"success"`
	Error   errorData `json:"error"`
}

type errorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HealthResponse is the shape returned by GET /health.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	DB      string `json:"db"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, envelope{Success: true, Data: data})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, envelope{Success: true, Data: data})
}

func Error(c *gin.Context, err error) {
	status := apperror.HTTPStatus(err)
	code := "INTERNAL_ERROR"
	message := "an unexpected error occurred"

	if e, ok := err.(*apperror.AppError); ok {
		code = e.Code
		message = e.Message
	}

	c.JSON(status, errorBody{
		Success: false,
		Error:   errorData{Code: code, Message: message},
	})
}
