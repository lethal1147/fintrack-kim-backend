package service

import (
	"fmt"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/hashutil"
	"github.com/joakim/fintrack-api/pkg/jwtutil"
	"github.com/joakim/fintrack-api/pkg/totputil"
)

// AuthServiceConfig holds the JWT secrets and expiry durations for auth operations.
type AuthServiceConfig struct {
	AccessSecret        string
	RefreshSecret       string
	AccessExpiryMinutes int
	RefreshExpiryDays   int
}

// AuthServiceInterface is the contract that the handler layer depends on.
type AuthServiceInterface interface {
	Register(input AuthInput) (*AuthResponse, error)
	Login(input LoginInput) (*LoginResult, error)
	Refresh(refreshToken string) (*RefreshResponse, error)
	Logout(refreshToken string) error
	LogoutAll(userID string) error
	GetProfile(userID string) (*UserInfo, error)
	VerifyTOTP(challengeToken, code, userAgent, ipAddress string) (*AuthResponse, error)
}

// AuthInput is the payload for Register.
type AuthInput struct {
	Email    string
	Password string
	Name     string
}

// LoginInput is the payload for Login.
type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
	IPAddress string
}

// UserInfo is the safe public representation of a user.
type UserInfo struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	AvatarURL   string    `json:"avatar_url"`
	Provider    string    `json:"provider"`
	TOTPEnabled bool      `json:"totp_enabled"`
	Locale      string    `json:"locale"`
	CreatedAt   time.Time `json:"created_at"`
}

// AuthResponse is returned by Register and successful Login.
type AuthResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	User         UserInfo `json:"user"`
}

// TOTPChallengeResponse is returned by Login when the user has TOTP enabled.
type TOTPChallengeResponse struct {
	TOTPRequired   bool   `json:"totp_required"`
	ChallengeToken string `json:"challenge_token"`
}

// LoginResult wraps either a full auth response or a TOTP challenge.
// Exactly one field is non-nil.
type LoginResult struct {
	Auth      *AuthResponse
	Challenge *TOTPChallengeResponse
}

// RefreshResponse is returned by Refresh.
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

// AuthService implements AuthServiceInterface.
type AuthService struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
	totpRepo    domain.TOTPRepository
	cfg         AuthServiceConfig
}

// NewAuthService creates an AuthService with the given dependencies.
func NewAuthService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	totpRepo domain.TOTPRepository,
	cfg AuthServiceConfig,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		totpRepo:    totpRepo,
		cfg:         cfg,
	}
}

// Register creates a user account and returns a token pair.
func (s *AuthService) Register(input AuthInput) (*AuthResponse, error) {
	existing, _ := s.userRepo.FindByEmail(input.Email)
	if existing != nil {
		return nil, apperror.Conflict("email already registered")
	}

	hash, err := hashutil.Hash(input.Password)
	if err != nil {
		return nil, apperror.Internal("failed to hash password")
	}

	user := &domain.User{
		Email:        input.Email,
		Name:         input.Name,
		PasswordHash: hash,
		Provider:     domain.ProviderLocal,
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return s.issueTokenPair(user, "", "")
}

// Login verifies credentials. If TOTP is enabled, returns a short-lived challenge
// token instead of issuing a session. Otherwise returns a full token pair.
func (s *AuthService) Login(input LoginInput) (*LoginResult, error) {
	user, err := s.userRepo.FindByEmail(input.Email)
	if err != nil {
		return nil, apperror.Unauthorized("Invalid email or password")
	}

	if err := hashutil.Verify(input.Password, user.PasswordHash); err != nil {
		return nil, apperror.Unauthorized("Invalid email or password")
	}

	if user.TOTPEnabled {
		challengeToken, err := jwtutil.SignChallengeToken(user.ID, "totp_challenge", s.cfg.AccessSecret, 5)
		if err != nil {
			return nil, apperror.Internal("failed to sign challenge token")
		}
		return &LoginResult{Challenge: &TOTPChallengeResponse{
			TOTPRequired:   true,
			ChallengeToken: challengeToken,
		}}, nil
	}

	auth, err := s.issueTokenPair(user, input.UserAgent, input.IPAddress)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Auth: auth}, nil
}

// VerifyTOTP validates a TOTP or backup code against the challenge token and
// issues a full session if valid.
func (s *AuthService) VerifyTOTP(challengeToken, code, userAgent, ipAddress string) (*AuthResponse, error) {
	claims, err := jwtutil.ParseChallengeToken(challengeToken, s.cfg.AccessSecret)
	if err != nil {
		return nil, apperror.Unauthorized("invalid or expired challenge token")
	}
	if claims.Purpose != "totp_challenge" {
		return nil, apperror.Unauthorized("invalid challenge token purpose")
	}

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, apperror.Unauthorized("user not found")
	}

	if totputil.Validate(user.TOTPSecret, code) {
		return s.issueTokenPair(user, userAgent, ipAddress)
	}

	// Try backup codes.
	backupCodes, err := s.totpRepo.FindUnusedBackupCodes(user.ID)
	if err != nil {
		return nil, err
	}
	for _, bc := range backupCodes {
		if hashutil.Verify(code, bc.CodeHash) == nil {
			if err := s.totpRepo.MarkBackupCodeUsed(bc.ID); err != nil {
				return nil, err
			}
			return s.issueTokenPair(user, userAgent, ipAddress)
		}
	}

	return nil, apperror.Unauthorized("invalid code")
}

// Refresh issues a new access token for a valid refresh token.
func (s *AuthService) Refresh(refreshToken string) (*RefreshResponse, error) {
	session, err := s.sessionRepo.FindByRefreshToken(refreshToken)
	if err != nil {
		return nil, apperror.Unauthorized("invalid refresh token")
	}

	if _, err := jwtutil.ParseRefreshToken(refreshToken, s.cfg.RefreshSecret); err != nil {
		return nil, apperror.Unauthorized("invalid or expired refresh token")
	}

	accessToken, err := jwtutil.SignAccessToken(session.UserID, session.ID, s.cfg.AccessSecret, s.cfg.AccessExpiryMinutes)
	if err != nil {
		return nil, apperror.Internal("failed to sign access token")
	}

	return &RefreshResponse{AccessToken: accessToken}, nil
}

// Logout deletes the session identified by the refresh token. Idempotent.
func (s *AuthService) Logout(refreshToken string) error {
	session, err := s.sessionRepo.FindByRefreshToken(refreshToken)
	if err != nil {
		return nil // already gone — idempotent
	}
	return s.sessionRepo.DeleteByID(session.ID)
}

// LogoutAll deletes all sessions for the given user.
func (s *AuthService) LogoutAll(userID string) error {
	return s.sessionRepo.DeleteAllByUserID(userID)
}

// GetProfile returns the user profile for the given userID.
func (s *AuthService) GetProfile(userID string) (*UserInfo, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	return toUserInfo(user), nil
}

// issueTokenPair signs both tokens and creates a session row.
func (s *AuthService) issueTokenPair(user *domain.User, userAgent, ipAddress string) (*AuthResponse, error) {
	sessionLabel := fmt.Sprintf("%s-%d", user.ID, time.Now().UnixNano())

	refreshToken, err := jwtutil.SignRefreshToken(sessionLabel, s.cfg.RefreshSecret, s.cfg.RefreshExpiryDays)
	if err != nil {
		return nil, apperror.Internal("failed to sign refresh token")
	}

	session := &domain.Session{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
		ExpiresAt:    time.Now().Add(time.Duration(s.cfg.RefreshExpiryDays) * 24 * time.Hour),
	}
	if err := s.sessionRepo.Create(session); err != nil {
		return nil, err
	}

	accessToken, err := jwtutil.SignAccessToken(user.ID, session.ID, s.cfg.AccessSecret, s.cfg.AccessExpiryMinutes)
	if err != nil {
		return nil, apperror.Internal("failed to sign access token")
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *toUserInfo(user),
	}, nil
}

func toUserInfo(u *domain.User) *UserInfo {
	return &UserInfo{
		ID:          u.ID,
		Email:       u.Email,
		Name:        u.Name,
		AvatarURL:   u.AvatarURL,
		Provider:    string(u.Provider),
		TOTPEnabled: u.TOTPEnabled,
		Locale:      u.Locale,
		CreatedAt:   u.CreatedAt,
	}
}
