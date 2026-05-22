package main

import (
	"log"

	"github.com/joakim/fintrack-api/internal/config"
	"github.com/joakim/fintrack-api/internal/database"
	"github.com/joakim/fintrack-api/internal/handler"
	"github.com/joakim/fintrack-api/internal/repository/postgres"
	"github.com/joakim/fintrack-api/internal/router"
	"github.com/joakim/fintrack-api/internal/service"
	"github.com/joakim/fintrack-api/pkg/emailclient"
	"github.com/joakim/fintrack-api/pkg/r2client"
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

	userRepo    := postgres.NewUserRepo(db)
	sessionRepo := postgres.NewSessionRepo(db)
	txRepo      := postgres.NewTransactionRepo(db)
	recurringRepo := postgres.NewRecurringRepo(db)
	budgetRepo  := postgres.NewBudgetRepo(db)
	otpRepo     := postgres.NewOTPRepo(db)
	totpRepo    := postgres.NewTOTPRepo(db)
	emailSender := emailclient.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)

	healthSvc := service.NewHealthService("0.1.0", sqlDB)
	authSvc := service.NewAuthService(userRepo, sessionRepo, totpRepo, service.AuthServiceConfig{
		AccessSecret:        cfg.JWTAccessSecret,
		RefreshSecret:       cfg.JWTRefreshSecret,
		AccessExpiryMinutes: cfg.JWTAccessExpiryMinutes,
		RefreshExpiryDays:   cfg.JWTRefreshExpiryDays,
	})
	txSvc := service.NewTransactionService(txRepo)
	analyticsSvc := service.NewAnalyticsService(txRepo)
	recurringSvc := service.NewRecurringService(recurringRepo, txRepo)
	budgetSvc := service.NewBudgetService(budgetRepo, txRepo)

	r2 := r2client.New(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket, cfg.R2PublicURL)
	profileSvc := service.NewProfileService(userRepo, r2)
	securitySvc     := service.NewSecurityService(userRepo, sessionRepo, otpRepo, totpRepo, emailSender, cfg.JWTRefreshSecret)
	securityHandler := handler.NewSecurityHandler(securitySvc, cfg.JWTRefreshSecret, cfg.AppCookieSecure)

	healthHandler := handler.NewHealthHandler(healthSvc)
	authHandler := handler.NewAuthHandler(authSvc, cfg.AppCookieSecure)
	txHandler := handler.NewTransactionHandler(txSvc)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsSvc)
	recurringHandler := handler.NewRecurringHandler(recurringSvc)
	budgetHandler := handler.NewBudgetHandler(budgetSvc)
	profileHandler := handler.NewProfileHandler(profileSvc)

	r := router.New(router.RouterConfig{
		Env:              cfg.AppEnv,
		FrontendOrigin:   cfg.AppFrontendOrigin,
		JWTAccessSecret:  cfg.JWTAccessSecret,
		JWTRefreshSecret: cfg.JWTRefreshSecret,
		SwaggerEnabled:   cfg.SwaggerEnabled,
	}, router.Handlers{
		Health:      healthHandler,
		Auth:        authHandler,
		Transaction: txHandler,
		Analytics:   analyticsHandler,
		Recurring:   recurringHandler,
		Budget:      budgetHandler,
		Profile:     profileHandler,
		Security:    securityHandler,
	})

	log.Printf("starting server on :%s (env=%s)", cfg.AppPort, cfg.AppEnv)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("server: %v", err)
	}
}
