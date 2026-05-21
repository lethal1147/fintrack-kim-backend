package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/hashutil"
)

// ─── mock session repo ────────────────────────────────────────────────────────

type mockSecSessionRepo struct {
	sessions     []*domain.Session
	deleteByID   string
	deleteAllUID string
}

func (m *mockSecSessionRepo) Create(s *domain.Session) error { return nil }
func (m *mockSecSessionRepo) FindByRefreshToken(token string) (*domain.Session, error) {
	return nil, nil
}
func (m *mockSecSessionRepo) DeleteByID(id string) error {
	m.deleteByID = id
	return nil
}
func (m *mockSecSessionRepo) DeleteAllByUserID(uid string) error {
	m.deleteAllUID = uid
	return nil
}
func (m *mockSecSessionRepo) ListByUserID(_ string) ([]*domain.Session, error) {
	return m.sessions, nil
}
func (m *mockSecSessionRepo) UpdateLastActive(_ string, _ time.Time) error { return nil }

// ─── TestSecurity_ListSessions_MarksCurrent ───────────────────────────────────

func TestSecurity_ListSessions_MarksCurrent(t *testing.T) {
	now := time.Now()
	repo := &mockSecSessionRepo{sessions: []*domain.Session{
		{ID: "s1", UserAgent: "Mozilla/5.0 (Macintosh) Chrome/120", LastActiveAt: now.Add(-1 * time.Hour)},
		{ID: "s2", UserAgent: "Mozilla/5.0 (iPhone) Safari/604", LastActiveAt: now.Add(-2 * time.Hour)},
		{ID: "s3", UserAgent: "Mozilla/5.0 (Windows NT) Firefox/119", LastActiveAt: now},
	}}
	svc := NewSecurityService(nil, repo, nil, nil, nil, "secret")

	infos, err := svc.ListSessions(context.Background(), "user1", "s3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(infos))
	}
	// current session (s3) must be first
	if !infos[0].IsCurrent || infos[0].ID != "s3" {
		t.Errorf("expected s3 to be first and IsCurrent=true, got id=%s IsCurrent=%v", infos[0].ID, infos[0].IsCurrent)
	}
	for _, info := range infos[1:] {
		if info.IsCurrent {
			t.Errorf("session %s should not be current", info.ID)
		}
	}
}

// ─── TestSecurity_ListSessions_ParsesUserAgent ────────────────────────────────

func TestSecurity_ListSessions_ParsesUserAgent(t *testing.T) {
	cases := []struct {
		ua   string
		want string
	}{
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0", "Chrome on macOS"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/119.0", "Firefox on Windows"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0) AppleWebKit/605.1.15 Version/17.0 Safari/604.1", "Safari on iOS"},
		{"Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 Chrome/120 Safari/537.36 Edg/120.0", "Edge on Windows"},
		{"", "Unknown device"},
	}
	for _, tc := range cases {
		got := parseUserAgent(tc.ua)
		if got != tc.want {
			t.Errorf("parseUserAgent(%q) = %q, want %q", tc.ua, got, tc.want)
		}
	}
}

// ─── TestSecurity_RevokeSession_CallsDelete ───────────────────────────────────

func TestSecurity_RevokeSession_CallsDelete(t *testing.T) {
	repo := &mockSecSessionRepo{}
	svc := NewSecurityService(nil, repo, nil, nil, nil, "secret")

	if err := svc.RevokeSession(context.Background(), "user1", "sess-abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.deleteByID != "sess-abc" {
		t.Errorf("expected DeleteByID(sess-abc), got %q", repo.deleteByID)
	}
}

// ─── mock user repo ───────────────────────────────────────────────────────────

type mockSecUserRepo struct {
	user      *domain.User
	updateErr error
	updated   *domain.User
}

func (m *mockSecUserRepo) FindByID(_ string) (*domain.User, error)             { return m.user, nil }
func (m *mockSecUserRepo) FindByEmail(_ string) (*domain.User, error)          { return nil, nil }
func (m *mockSecUserRepo) FindByProviderID(_ domain.AuthProvider, _ string) (*domain.User, error) {
	return nil, nil
}
func (m *mockSecUserRepo) Create(_ *domain.User) error { return nil }
func (m *mockSecUserRepo) Update(u *domain.User) error {
	m.updated = u
	return m.updateErr
}

// ─── mock OTP repo ────────────────────────────────────────────────────────────

type mockOTPRepo struct {
	activeToken      *domain.OTPToken
	created          *domain.OTPToken
	markedUsed       string
	deletedPurpose   string
}

func (m *mockOTPRepo) Create(t *domain.OTPToken) error {
	m.created = t
	return nil
}
func (m *mockOTPRepo) FindActive(_, _ string) (*domain.OTPToken, error) {
	return m.activeToken, nil
}
func (m *mockOTPRepo) MarkUsed(id string) error {
	m.markedUsed = id
	return nil
}
func (m *mockOTPRepo) DeleteByUserAndPurpose(_, purpose string) error {
	m.deletedPurpose = purpose
	return nil
}

// ─── mock email sender ────────────────────────────────────────────────────────

type mockEmailSender struct {
	lastTo      string
	lastSubject string
	lastHTML    string
}

func (m *mockEmailSender) Send(_ context.Context, to, subject, html string) error {
	m.lastTo = to
	m.lastSubject = subject
	m.lastHTML = html
	return nil
}

// ─── TestSecurity_RequestPasswordChange_SendsOTP ─────────────────────────────

func TestSecurity_RequestPasswordChange_SendsOTP(t *testing.T) {
	userRepo := &mockSecUserRepo{user: &domain.User{ID: "u1", Email: "user@example.com"}}
	otpRepo := &mockOTPRepo{}
	email := &mockEmailSender{}
	svc := NewSecurityService(userRepo, &mockSecSessionRepo{}, otpRepo, nil, email, "secret")

	if err := svc.RequestPasswordChange(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if email.lastTo != "user@example.com" {
		t.Errorf("email sent to %q, want user@example.com", email.lastTo)
	}
	if otpRepo.created == nil {
		t.Fatal("expected OTP token to be created")
	}
	if otpRepo.created.Purpose != "password_change" {
		t.Errorf("OTP purpose = %q, want password_change", otpRepo.created.Purpose)
	}
}

// ─── TestSecurity_RequestPasswordChange_ClearsStaleTokens ────────────────────

func TestSecurity_RequestPasswordChange_ClearsStaleTokens(t *testing.T) {
	userRepo := &mockSecUserRepo{user: &domain.User{ID: "u1", Email: "user@example.com"}}
	otpRepo := &mockOTPRepo{}
	svc := NewSecurityService(userRepo, &mockSecSessionRepo{}, otpRepo, nil, &mockEmailSender{}, "secret")

	if err := svc.RequestPasswordChange(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if otpRepo.deletedPurpose != "password_change" {
		t.Errorf("expected stale tokens deleted, got deletedPurpose=%q", otpRepo.deletedPurpose)
	}
}

// ─── TestSecurity_ChangePassword_OK ──────────────────────────────────────────

func TestSecurity_ChangePassword_OK(t *testing.T) {
	code := "123456"
	hash, _ := hashutil.Hash(code)
	otpRepo := &mockOTPRepo{activeToken: &domain.OTPToken{
		ID:        "tok1",
		CodeHash:  hash,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}}
	sessRepo := &mockSecSessionRepo{}
	userRepo := &mockSecUserRepo{user: &domain.User{ID: "u1", Email: "user@example.com"}}
	svc := NewSecurityService(userRepo, sessRepo, otpRepo, nil, nil, "secret")

	if err := svc.ChangePassword(context.Background(), "u1", code, "newpassword123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if otpRepo.markedUsed != "tok1" {
		t.Errorf("expected token tok1 marked used, got %q", otpRepo.markedUsed)
	}
	if sessRepo.deleteAllUID != "u1" {
		t.Errorf("expected all sessions deleted for u1, got %q", sessRepo.deleteAllUID)
	}
	if userRepo.updated == nil {
		t.Fatal("expected user to be updated")
	}
}

// ─── TestSecurity_ChangePassword_InvalidOTP ──────────────────────────────────

func TestSecurity_ChangePassword_InvalidOTP(t *testing.T) {
	hash, _ := hashutil.Hash("correct")
	otpRepo := &mockOTPRepo{activeToken: &domain.OTPToken{
		ID:        "tok1",
		CodeHash:  hash,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}}
	svc := NewSecurityService(&mockSecUserRepo{user: &domain.User{}}, &mockSecSessionRepo{}, otpRepo, nil, nil, "secret")

	err := svc.ChangePassword(context.Background(), "u1", "wrong", "newpassword123")
	if err == nil {
		t.Fatal("expected error for invalid OTP, got nil")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected 'invalid' in error, got %q", err.Error())
	}
}

// ─── TestSecurity_ChangePassword_ShortPassword ───────────────────────────────

func TestSecurity_ChangePassword_ShortPassword(t *testing.T) {
	svc := NewSecurityService(nil, nil, nil, nil, nil, "secret")
	err := svc.ChangePassword(context.Background(), "u1", "123456", "short")
	if err == nil {
		t.Fatal("expected error for short password, got nil")
	}
	if !strings.Contains(err.Error(), "8") {
		t.Errorf("expected '8' in error message, got %q", err.Error())
	}
}
