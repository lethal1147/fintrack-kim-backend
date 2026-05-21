package service

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/emailclient"
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

// ─── Password change stubs (implemented in Task 4) ───────────────────────────

func (s *SecurityService) RequestPasswordChange(_ context.Context, _ string) error {
	return nil
}

func (s *SecurityService) ChangePassword(_ context.Context, _, _, _ string) error {
	return nil
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
