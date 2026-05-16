package router

import (
	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/handler"
)

// newTestRouter wires a stub HealthHandler into the real router.New
// so integration tests exercise the full routing stack without a real DB.
func newTestRouter(cfg RouterConfig) *gin.Engine {
	healthHandler := handler.NewHealthHandler(&stubHealthSvc{})
	return New(cfg, Handlers{Health: healthHandler})
}
