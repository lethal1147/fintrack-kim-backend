package router

import (
	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/handler"
	"github.com/joakim/fintrack-api/internal/middleware"
)

type Handlers struct {
	Health *handler.HealthHandler
}

func New(cfg RouterConfig, h Handlers) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(cfg.FrontendOrigin))
	r.Use(middleware.RateLimiter())
	r.Use(gin.Recovery())

	r.GET("/health", h.Health.HealthCheck)

	return r
}

type RouterConfig struct {
	Env            string
	FrontendOrigin string
	SwaggerEnabled bool
}
