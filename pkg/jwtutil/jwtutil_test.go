package jwtutil

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testAccessSecret  = "test-access-secret"
	testRefreshSecret = "test-refresh-secret"
)

func TestSignAndParseAccessToken(t *testing.T) {
	token, err := SignAccessToken("user-123", testAccessSecret, 15)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := ParseAccessToken(token, testAccessSecret)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("want UserID=user-123, got %s", claims.UserID)
	}
}

func TestSignAndParseRefreshToken(t *testing.T) {
	token, err := SignRefreshToken("session-abc", testRefreshSecret, 30)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := ParseRefreshToken(token, testRefreshSecret)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.SessionID != "session-abc" {
		t.Errorf("want SessionID=session-abc, got %s", claims.SessionID)
	}
}

func TestParseAccessToken_WrongSecret(t *testing.T) {
	token, _ := SignAccessToken("u1", testAccessSecret, 15)
	_, err := ParseAccessToken(token, "wrong-secret")
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestParseAccessToken_Expired(t *testing.T) {
	claims := AccessClaims{
		UserID: "u1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		},
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testAccessSecret))
	_, err := ParseAccessToken(token, testAccessSecret)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestParseAccessToken_Tampered(t *testing.T) {
	token, _ := SignAccessToken("u1", testAccessSecret, 15)
	_, err := ParseAccessToken(token+"x", testAccessSecret)
	if err == nil {
		t.Error("expected error for tampered token")
	}
}
