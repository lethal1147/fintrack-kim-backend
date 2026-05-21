package service

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/emailclient"
	"github.com/joakim/fintrack-api/pkg/hashutil"
)

// ─── constructor ──────────────────────────────────────────────────────────────

type SecurityService struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
	otpRepo     domain.OTPRepository   // nil until Task 4
	totpRepo    domain.TOTPRepository  // nil until Task 5
	emailSender emailclient.Sender
	jwtSecret   string // JWTRefreshSecret — for identifying current session
}

func NewSecurityService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	otpRepo domain.OTPRepository,
	totpRepo domain.TOTPRepository,
	emailSender emailclient.Sender,
	jwtSecret string,
) *SecurityService {
	return &SecurityService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		otpRepo:     otpRepo,
		totpRepo:    totpRepo,
		emailSender: emailSender,
		jwtSecret:   jwtSecret,
	}
}

// ─── Sessions ─────────────────────────────────────────────────────────────────

func (s *SecurityService) ListSessions(_ context.Context, userID, currentSessionID string) ([]domain.SessionInfo, error) {
	sessions, err := s.sessionRepo.ListByUserID(userID)
	if err != nil {
		return nil, err
	}

	infos := make([]domain.SessionInfo, len(sessions))
	for i, sess := range sessions {
		infos[i] = domain.SessionInfo{
			ID:           sess.ID,
			Device:       parseUserAgent(sess.UserAgent),
			LastActiveAt: sess.LastActiveAt,
			IsCurrent:    sess.ID == currentSessionID,
		}
	}

	sort.Slice(infos, func(i, j int) bool {
		if infos[i].IsCurrent != infos[j].IsCurrent {
			return infos[i].IsCurrent
		}
		return infos[i].LastActiveAt.After(infos[j].LastActiveAt)
	})

	return infos, nil
}

func (s *SecurityService) RevokeSession(_ context.Context, _, sessionID string) error {
	return s.sessionRepo.DeleteByID(sessionID)
}

// ─── Password change ──────────────────────────────────────────────────────────

func (s *SecurityService) RequestPasswordChange(ctx context.Context, userID string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	// Clear any stale tokens before issuing a new one
	if err := s.otpRepo.DeleteByUserAndPurpose(userID, "password_change"); err != nil {
		return err
	}

	code := fmt.Sprintf("%06d", rand.Intn(1_000_000))
	codeHash, err := hashutil.Hash(code)
	if err != nil {
		return apperror.Internal("failed to hash OTP")
	}

	if err := s.otpRepo.Create(&domain.OTPToken{
		UserID:    userID,
		Purpose:   "password_change",
		CodeHash:  codeHash,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}); err != nil {
		return err
	}

	html := fmt.Sprintf(
		"<p>Your FinTrack verification code is: <strong>%s</strong></p><p>This code expires in 15 minutes.</p>",
		code,
	)
	return s.emailSender.Send(ctx, user.Email, "Your FinTrack verification code", html)
}

func (s *SecurityService) ChangePassword(ctx context.Context, userID, otp, newPassword string) error {
	if len(newPassword) < 8 || len(newPassword) > 100 {
		return apperror.BadRequest("password must be between 8 and 100 characters")
	}

	token, err := s.otpRepo.FindActive(userID, "password_change")
	if err != nil {
		return err
	}
	if token == nil {
		return apperror.BadRequest("invalid or expired code")
	}

	if err := hashutil.Verify(otp, token.CodeHash); err != nil {
		return apperror.BadRequest("invalid or expired code")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	newHash, err := hashutil.Hash(newPassword)
	if err != nil {
		return apperror.Internal("failed to hash password")
	}
	user.PasswordHash = newHash
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(user); err != nil {
		return err
	}

	if err := s.otpRepo.MarkUsed(token.ID); err != nil {
		return err
	}

	return s.sessionRepo.DeleteAllByUserID(userID)
}

// ─── TOTP stubs (implemented in Task 5) ──────────────────────────────────────

func (s *SecurityService) SetupTOTP(_ context.Context, _ string) (*domain.TOTPSetupResult, error) {
	return nil, nil
}

func (s *SecurityService) ConfirmTOTP(_ context.Context, _, _ string) ([]string, error) {
	return nil, nil
}

func (s *SecurityService) DisableTOTP(_ context.Context, _, _ string) error {
	return nil
}

// ─── User-agent parsing ───────────────────────────────────────────────────────

var (
	reChrome  = regexp.MustCompile(`Chrome/[\d.]+`)
	reFirefox = regexp.MustCompile(`Firefox/[\d.]+`)
	reSafari  = regexp.MustCompile(`Version/[\d.]+ Safari`)
	reEdge    = regexp.MustCompile(`Edg/[\d.]+`)

	reWindows = regexp.MustCompile(`Windows NT`)
	reMacOS   = regexp.MustCompile(`Mac OS X`)
	reIOS     = regexp.MustCompile(`iPhone|iPad`)
	reAndroid = regexp.MustCompile(`Android`)
	reLinux   = regexp.MustCompile(`Linux`)
)

func parseUserAgent(ua string) string {
	if ua == "" {
		return "Unknown device"
	}

	var browser string
	switch {
	case reEdge.MatchString(ua):
		browser = "Edge"
	case reChrome.MatchString(ua) && !strings.Contains(ua, "Chromium"):
		browser = "Chrome"
	case reFirefox.MatchString(ua):
		browser = "Firefox"
	case reSafari.MatchString(ua):
		browser = "Safari"
	default:
		browser = "Browser"
	}

	var os string
	switch {
	case reIOS.MatchString(ua):
		os = "iOS"
	case reAndroid.MatchString(ua):
		os = "Android"
	case reWindows.MatchString(ua):
		os = "Windows"
	case reMacOS.MatchString(ua):
		os = "macOS"
	case reLinux.MatchString(ua):
		os = "Linux"
	default:
		os = "Unknown OS"
	}

	return browser + " on " + os
}
