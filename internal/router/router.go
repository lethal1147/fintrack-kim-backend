package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/joakim/fintrack-api/docs"
	"github.com/joakim/fintrack-api/internal/handler"
	"github.com/joakim/fintrack-api/internal/middleware"
)

type Handlers struct {
	Health      *handler.HealthHandler
	Auth        *handler.AuthHandler
	Transaction *handler.TransactionHandler
	Analytics   *handler.AnalyticsHandler
	Recurring   *handler.RecurringHandler
	Budget      *handler.BudgetHandler
	Profile     *handler.ProfileHandler
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

	auth := r.Group("/auth")
	{
		auth.POST("/register", h.Auth.Register)
		auth.POST("/login", h.Auth.Login)
		auth.POST("/refresh", h.Auth.Refresh)
		auth.POST("/logout", h.Auth.Logout) // public — cookie identifies the session
		protected := auth.Group("")
		protected.Use(middleware.Auth(cfg.JWTAccessSecret))
		{
			protected.POST("/logout-all", h.Auth.LogoutAll)
			protected.GET("/me", h.Auth.Me)
		}
	}

	tx := r.Group("/transactions")
	tx.Use(middleware.Auth(cfg.JWTAccessSecret))
	{
		tx.GET("", h.Transaction.List)
		tx.POST("", h.Transaction.Create)
		tx.GET("/summary", h.Transaction.Summary)     // must be before /:id
		tx.GET("/analytics", h.Analytics.Analytics)   // must be before /:id
		tx.GET("/:id", h.Transaction.Get)
		tx.PUT("/:id", h.Transaction.Update)
		tx.DELETE("/:id", h.Transaction.Delete)
	}

	rec := r.Group("/recurring")
	rec.Use(middleware.Auth(cfg.JWTAccessSecret))
	{
		rec.GET("", h.Recurring.List)
		rec.POST("", h.Recurring.Create)
		rec.POST("/process", h.Recurring.Process) // must be before /:id
		rec.PUT("/:id", h.Recurring.Update)
		rec.PATCH("/:id/status", h.Recurring.ToggleStatus)
		rec.DELETE("/:id", h.Recurring.Delete)
	}

	bud := r.Group("/budget")
	bud.Use(middleware.Auth(cfg.JWTAccessSecret))
	{
		bud.GET("", h.Budget.List)
		bud.POST("", h.Budget.Create)
		bud.PUT("/:id", h.Budget.Update)
		bud.DELETE("/:id", h.Budget.Delete)
	}

	prof := r.Group("/profile")
	prof.Use(middleware.Auth(cfg.JWTAccessSecret))
	{
		prof.PATCH("", h.Profile.Update)
		prof.POST("/avatar", h.Profile.UploadAvatar)
	}

	if cfg.SwaggerEnabled {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	return r
}

type RouterConfig struct {
	Env             string
	FrontendOrigin  string
	JWTAccessSecret string
	SwaggerEnabled  bool
}
