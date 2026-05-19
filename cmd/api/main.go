package main

import (
	"log"

	"github.com/joakim/fintrack-api/internal/config"
	"github.com/joakim/fintrack-api/internal/database"
	"github.com/joakim/fintrack-api/internal/handler"
	"github.com/joakim/fintrack-api/internal/repository/postgres"
	"github.com/joakim/fintrack-api/internal/router"
	"github.com/joakim/fintrack-api/internal/service"
)

// @title           FinTrack API
// @version         0.1.0
// @description     Personal finance tracker API
// @host            localhost:8080
// @BasePath        /
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DSN(), cfg.AppEnv == "production")
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	sqlDB, _ := db.DB()

	userRepo := postgres.NewUserRepo(db)
	sessionRepo := postgres.NewSessionRepo(db)
	txRepo := postgres.NewTransactionRepo(db)

	healthSvc := service.NewHealthService("0.1.0", sqlDB)
	authSvc := service.NewAuthService(userRepo, sessionRepo, service.AuthServiceConfig{
		AccessSecret:        cfg.JWTAccessSecret,
		RefreshSecret:       cfg.JWTRefreshSecret,
		AccessExpiryMinutes: cfg.JWTAccessExpiryMinutes,
		RefreshExpiryDays:   cfg.JWTRefreshExpiryDays,
	})
	txSvc := service.NewTransactionService(txRepo)

	healthHandler := handler.NewHealthHandler(healthSvc)
	authHandler := handler.NewAuthHandler(authSvc, cfg.AppCookieSecure)
	txHandler := handler.NewTransactionHandler(txSvc)

	r := router.New(router.RouterConfig{
		Env:             cfg.AppEnv,
		FrontendOrigin:  cfg.AppFrontendOrigin,
		JWTAccessSecret: cfg.JWTAccessSecret,
		SwaggerEnabled:  cfg.SwaggerEnabled,
	}, router.Handlers{
		Health:      healthHandler,
		Auth:        authHandler,
		Transaction: txHandler,
	})

	log.Printf("starting server on :%s (env=%s)", cfg.AppPort, cfg.AppEnv)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("server: %v", err)
	}
}
