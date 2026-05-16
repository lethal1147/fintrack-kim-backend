package config

import (
	"os"
	"testing"
)

func setSecrets(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_ACCESS_SECRET", "test-access-secret-value")
	t.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret-value")
}

func TestLoad_Defaults(t *testing.T) {
	setSecrets(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppPort != "8080" {
		t.Errorf("want AppPort=8080, got %s", cfg.AppPort)
	}
	if cfg.DBSSLMode != "disable" {
		t.Errorf("want DBSSLMode=disable, got %s", cfg.DBSSLMode)
	}
	if cfg.JWTAccessExpiryMinutes != 15 {
		t.Errorf("want JWTAccessExpiryMinutes=15, got %d", cfg.JWTAccessExpiryMinutes)
	}
}

func TestLoad_EnvOverridesDefault(t *testing.T) {
	setSecrets(t)
	t.Setenv("APP_PORT", "9090")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppPort != "9090" {
		t.Errorf("want AppPort=9090, got %s", cfg.AppPort)
	}
}

func TestLoad_MissingAccessSecret(t *testing.T) {
	os.Unsetenv("JWT_ACCESS_SECRET")
	os.Unsetenv("JWT_REFRESH_SECRET")
	_, err := Load()
	if err == nil {
		t.Error("expected error for missing JWT_ACCESS_SECRET")
	}
}

func TestLoad_ProductionRequiresSSL(t *testing.T) {
	setSecrets(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_SSLMODE", "disable")
	_, err := Load()
	if err == nil {
		t.Error("expected error for production + sslmode=disable")
	}
}

func TestConfig_DSN(t *testing.T) {
	cfg := &Config{
		DBHost: "localhost", DBPort: "5433", DBUser: "u",
		DBPassword: "p", DBName: "db", DBSSLMode: "disable",
	}
	dsn := cfg.DSN()
	want := "host=localhost port=5433 user=u password=p dbname=db sslmode=disable TimeZone=UTC"
	if dsn != want {
		t.Errorf("want DSN %q, got %q", want, dsn)
	}
}
