package service

import (
	"context"
	"testing"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
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
