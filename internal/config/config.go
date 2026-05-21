package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv           string `mapstructure:"APP_ENV"`
	AppPort          string `mapstructure:"APP_PORT"`
	AppFrontendOrigin string `mapstructure:"APP_FRONTEND_ORIGIN"`

	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	DBSSLMode  string `mapstructure:"DB_SSLMODE"`

	JWTAccessSecret         string `mapstructure:"JWT_ACCESS_SECRET"`
	JWTRefreshSecret        string `mapstructure:"JWT_REFRESH_SECRET"`
	JWTAccessExpiryMinutes  int    `mapstructure:"JWT_ACCESS_EXPIRY_MINUTES"`
	JWTRefreshExpiryDays    int    `mapstructure:"JWT_REFRESH_EXPIRY_DAYS"`

	GoogleClientID       string `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret   string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	FacebookClientID     string `mapstructure:"FACEBOOK_CLIENT_ID"`
	FacebookClientSecret string `mapstructure:"FACEBOOK_CLIENT_SECRET"`
	OAuthRedirectBaseURL string `mapstructure:"OAUTH_REDIRECT_BASE_URL"`

	SwaggerEnabled   bool `mapstructure:"SWAGGER_ENABLED"`
	AppCookieSecure  bool `mapstructure:"APP_COOKIE_SECURE"`

	R2AccountID       string `mapstructure:"R2_ACCOUNT_ID"`
	R2AccessKeyID     string `mapstructure:"R2_ACCESS_KEY_ID"`
	R2SecretAccessKey string `mapstructure:"R2_SECRET_ACCESS_KEY"`
	R2Bucket          string `mapstructure:"R2_BUCKET"`
	R2PublicURL       string `mapstructure:"R2_PUBLIC_URL"`
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

// Load reads .env (if present) then OS env vars, returning a validated Config.
func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_PORT", "8080")
	v.SetDefault("APP_FRONTEND_ORIGIN", "http://localhost:3000")
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", "5433")
	v.SetDefault("DB_USER", "fintrack")
	v.SetDefault("DB_NAME", "fintrack")
	v.SetDefault("DB_SSLMODE", "disable")
	v.SetDefault("JWT_ACCESS_EXPIRY_MINUTES", 15)
	v.SetDefault("JWT_REFRESH_EXPIRY_DAYS", 30)
	v.SetDefault("SWAGGER_ENABLED", true)
	v.SetDefault("APP_COOKIE_SECURE", false)
	// Empty-string defaults so Viper tracks these keys and AutomaticEnv can override them.
	v.SetDefault("JWT_ACCESS_SECRET", "")
	v.SetDefault("JWT_REFRESH_SECRET", "")
	v.SetDefault("DB_PASSWORD", "")
	v.SetDefault("GOOGLE_CLIENT_ID", "")
	v.SetDefault("GOOGLE_CLIENT_SECRET", "")
	v.SetDefault("FACEBOOK_CLIENT_ID", "")
	v.SetDefault("FACEBOOK_CLIENT_SECRET", "")
	v.SetDefault("OAUTH_REDIRECT_BASE_URL", "http://localhost:3000")
	v.SetDefault("R2_ACCOUNT_ID", "")
	v.SetDefault("R2_ACCESS_KEY_ID", "")
	v.SetDefault("R2_SECRET_ACCESS_KEY", "")
	v.SetDefault("R2_BUCKET", "")
	v.SetDefault("R2_PUBLIC_URL", "")

	// .env file (optional — ignored if not present)
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	_ = v.ReadInConfig()

	// OS env vars take precedence
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("config unmarshal: %w", err)
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if cfg.JWTAccessSecret == "" || cfg.JWTAccessSecret == "change-me-access-secret" {
		return fmt.Errorf("JWT_ACCESS_SECRET must be set to a secure value")
	}
	if cfg.JWTRefreshSecret == "" || cfg.JWTRefreshSecret == "change-me-refresh-secret" {
		return fmt.Errorf("JWT_REFRESH_SECRET must be set to a secure value")
	}
	if cfg.AppEnv == "production" && cfg.DBSSLMode == "disable" {
		return fmt.Errorf("DB_SSLMODE must not be 'disable' in production")
	}
	return nil
}
