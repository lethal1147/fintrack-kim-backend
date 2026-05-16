package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/pkg/response"
)

type HealthChecker interface {
	Check() response.HealthResponse
}

type HealthHandler struct{ svc HealthChecker }

func NewHealthHandler(svc HealthChecker) *HealthHandler {
	return &HealthHandler{svc: svc}
}

// HealthCheck godoc
// @Summary      Health check
// @Description  Returns server version and database connectivity status
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	result := h.svc.Check()
	if result.DB == "error" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": true, "data": result})
		return
	}
	response.Success(c, result)
}
