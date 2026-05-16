package service

import (
	"fmt"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/hashutil"
	"github.com/joakim/fintrack-api/pkg/jwtutil"
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
	Login(input LoginInput) (*AuthResponse, error)
	Refresh(refreshToken string) (*RefreshResponse, error)
	Logout(refreshToken string) error
	LogoutAll(userID string) error
	GetProfile(userID string) (*UserInfo, error)
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
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthResponse is returned by Register and Login.
type AuthResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	User         UserInfo `json:"user"`
}

// RefreshResponse is returned by Refresh.
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

// AuthService implements AuthServiceInterface.
type AuthService struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
	cfg         AuthServiceConfig
}

// NewAuthService creates an AuthService with the given dependencies.
func NewAuthService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	cfg AuthServiceConfig,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
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

// Login verifies credentials and returns a token pair.
func (s *AuthService) Login(input LoginInput) (*AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(input.Email)
	if err != nil {
		return nil, apperror.Unauthorized("invalid credentials")
	}

	if err := hashutil.Verify(input.Password, user.PasswordHash); err != nil {
		return nil, apperror.Unauthorized("invalid credentials")
	}

	return s.issueTokenPair(user, input.UserAgent, input.IPAddress)
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

	accessToken, err := jwtutil.SignAccessToken(session.UserID, s.cfg.AccessSecret, s.cfg.AccessExpiryMinutes)
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

	accessToken, err := jwtutil.SignAccessToken(user.ID, s.cfg.AccessSecret, s.cfg.AccessExpiryMinutes)
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
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
		Provider:  string(u.Provider),
		CreatedAt: u.CreatedAt,
	}
}
