package jwtutil

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type AccessClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

type ChallengeClaims struct {
	UserID  string `json:"user_id"`
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

func SignAccessToken(userID, secret string, expiryMinutes int) (string, error) {
	claims := AccessClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func SignRefreshToken(sessionID, secret string, expiryDays int) (string, error) {
	claims := RefreshClaims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryDays) * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func ParseAccessToken(tokenStr, secret string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, keyFunc(secret))
	if err != nil {
		return nil, apperror.Unauthorized("invalid or expired token")
	}
	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, apperror.Unauthorized("invalid token claims")
	}
	return claims, nil
}

func ParseRefreshToken(tokenStr, secret string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &RefreshClaims{}, keyFunc(secret))
	if err != nil {
		return nil, apperror.Unauthorized("invalid or expired refresh token")
	}
	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid {
		return nil, apperror.Unauthorized("invalid refresh token claims")
	}
	return claims, nil
}

func SignChallengeToken(userID, purpose, secret string, expiryMinutes int) (string, error) {
	claims := ChallengeClaims{
		UserID:  userID,
		Purpose: purpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func ParseChallengeToken(tokenStr, secret string) (*ChallengeClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &ChallengeClaims{}, keyFunc(secret))
	if err != nil {
		return nil, apperror.Unauthorized("invalid or expired challenge token")
	}
	claims, ok := token.Claims.(*ChallengeClaims)
	if !ok || !token.Valid {
		return nil, apperror.Unauthorized("invalid challenge token claims")
	}
	return claims, nil
}

func keyFunc(secret string) jwt.Keyfunc {
	return func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperror.Unauthorized("unexpected signing method")
		}
		return []byte(secret), nil
	}
}
