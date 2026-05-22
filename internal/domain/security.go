package domain

import (
	"context"
	"time"
)

// ─── OTP tokens (password change) ────────────────────────────────────────────

type OTPToken struct {
	ID        string
	UserID    string
	Purpose   string
	CodeHash  string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type OTPRepository interface {
	Create(token *OTPToken) error
	FindActive(userID, purpose string) (*OTPToken, error)
	MarkUsed(id string) error
	DeleteByUserAndPurpose(userID, purpose string) error
}

// ─── TOTP backup codes ────────────────────────────────────────────────────────

type TOTPBackupCode struct {
	ID        string
	UserID    string
	CodeHash  string
	UsedAt    *time.Time
	CreatedAt time.Time
}

type TOTPRepository interface {
	CreateBackupCodes(codes []*TOTPBackupCode) error
	FindUnusedBackupCodes(userID string) ([]*TOTPBackupCode, error)
	MarkBackupCodeUsed(id string) error
	DeleteBackupCodes(userID string) error
}

// ─── Security service types ───────────────────────────────────────────────────

type TOTPSetupResult struct {
	Secret      string
	QRCodeURI   string
	BackupCodes []string
}

type SessionInfo struct {
	ID           string
	Device       string
	LastActiveAt time.Time
	IsCurrent    bool
}

type SecurityServiceInterface interface {
	ListSessions(ctx context.Context, userID, currentSessionID string) ([]SessionInfo, error)
	RevokeSession(ctx context.Context, userID, sessionID string) error
	RequestPasswordChange(ctx context.Context, userID string) error
	ChangePassword(ctx context.Context, userID, otp, newPassword string) error
	SetupTOTP(ctx context.Context, userID string) (*TOTPSetupResult, error)
	ConfirmTOTP(ctx context.Context, userID, code string) ([]string, error)
	DisableTOTP(ctx context.Context, userID, code string) error
}
