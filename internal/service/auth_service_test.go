package service

import (
	"testing"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

// -- mock UserRepository --

type mockUserRepo struct {
	byEmail   map[string]*domain.User
	byID      map[string]*domain.User
	createErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		byEmail: make(map[string]*domain.User),
		byID:    make(map[string]*domain.User),
	}
}

func (m *mockUserRepo) FindByID(id string) (*domain.User, error) {
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return nil, apperror.NotFound("user not found")
}

func (m *mockUserRepo) FindByEmail(email string) (*domain.User, error) {
	if u, ok := m.byEmail[email]; ok {
		return u, nil
	}
	return nil, apperror.NotFound("user not found")
}

func (m *mockUserRepo) FindByProviderID(_ domain.AuthProvider, _ string) (*domain.User, error) {
	return nil, apperror.NotFound("not found")
}

func (m *mockUserRepo) Create(user *domain.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	user.ID = "user-" + user.Email
	user.CreatedAt = time.Now()
	m.byEmail[user.Email] = user
	m.byID[user.ID] = user
	return nil
}

func (m *mockUserRepo) Update(user *domain.User) error {
	m.byEmail[user.Email] = user
	m.byID[user.ID] = user
	return nil
}

// -- mock SessionRepository --

type mockSessionRepo struct {
	byToken         map[string]*domain.Session
	deleteAllCalled bool
	deleteAllUserID string
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{byToken: make(map[string]*domain.Session)}
}

func (m *mockSessionRepo) Create(s *domain.Session) error {
	if s.ID == "" {
		s.ID = "sess-mock"
	}
	m.byToken[s.RefreshToken] = s
	return nil
}

func (m *mockSessionRepo) FindByRefreshToken(token string) (*domain.Session, error) {
	if s, ok := m.byToken[token]; ok {
		return s, nil
	}
	return nil, apperror.NotFound("session not found")
}

func (m *mockSessionRepo) DeleteByID(id string) error {
	for k, s := range m.byToken {
		if s.ID == id {
			delete(m.byToken, k)
			return nil
		}
	}
	return nil // idempotent
}

func (m *mockSessionRepo) DeleteAllByUserID(userID string) error {
	m.deleteAllCalled = true
	m.deleteAllUserID = userID
	for k, s := range m.byToken {
		if s.UserID == userID {
			delete(m.byToken, k)
		}
	}
	return nil
}

// -- helpers --

func testCfg() AuthServiceConfig {
	return AuthServiceConfig{
		AccessSecret:        "test-access-secret-value-32chars",
		RefreshSecret:       "test-refresh-secret-value-32char",
		AccessExpiryMinutes: 15,
		RefreshExpiryDays:   30,
	}
}

// -- Register --

func TestRegister_Success(t *testing.T) {
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	svc := NewAuthService(userRepo, sessionRepo, testCfg())

	resp, err := svc.Register(AuthInput{
		Email: "alice@example.com", Password: "password123", Name: "Alice",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("access_token must not be empty")
	}
	if resp.RefreshToken == "" {
		t.Error("refresh_token must not be empty")
	}
	if resp.User.Email != "alice@example.com" {
		t.Errorf("want email=alice@example.com, got %s", resp.User.Email)
	}
	if len(sessionRepo.byToken) != 1 {
		t.Errorf("want 1 session created, got %d", len(sessionRepo.byToken))
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	userRepo := newMockUserRepo()
	svc := NewAuthService(userRepo, newMockSessionRepo(), testCfg())

	userRepo.byEmail["alice@example.com"] = &domain.User{
		ID: "existing", Email: "alice@example.com",
	}

	_, err := svc.Register(AuthInput{
		Email: "alice@example.com", Password: "password123", Name: "Alice",
	})
	if err == nil {
		t.Fatal("expected Conflict error for duplicate email")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != "CONFLICT" {
		t.Errorf("want CONFLICT error, got %v", err)
	}
}

// -- Login --

func TestLogin_Success(t *testing.T) {
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	svc := NewAuthService(userRepo, sessionRepo, testCfg())

	if _, err := svc.Register(AuthInput{
		Email: "bob@example.com", Password: "password123", Name: "Bob",
	}); err != nil {
		t.Fatalf("setup Register: %v", err)
	}

	resp, err := svc.Login(LoginInput{
		Email: "bob@example.com", Password: "password123",
		UserAgent: "Test/1.0", IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Error("token pair must be non-empty")
	}
	if resp.User.Email != "bob@example.com" {
		t.Errorf("want email=bob@example.com, got %s", resp.User.Email)
	}
}

func TestLogin_WrongEmail(t *testing.T) {
	svc := NewAuthService(newMockUserRepo(), newMockSessionRepo(), testCfg())

	_, err := svc.Login(LoginInput{Email: "nobody@example.com", Password: "pw"})
	if err == nil {
		t.Fatal("expected error for unknown email")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != "UNAUTHORIZED" {
		t.Errorf("want UNAUTHORIZED, got %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	userRepo := newMockUserRepo()
	svc := NewAuthService(userRepo, newMockSessionRepo(), testCfg())

	svc.Register(AuthInput{Email: "carol@example.com", Password: "correct-pw!", Name: "Carol"})

	_, err := svc.Login(LoginInput{Email: "carol@example.com", Password: "wrong-pw!"})
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != "UNAUTHORIZED" {
		t.Errorf("want UNAUTHORIZED, got %v", err)
	}
}

// -- Refresh --

func TestRefresh_Success(t *testing.T) {
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	svc := NewAuthService(userRepo, sessionRepo, testCfg())

	resp, err := svc.Register(AuthInput{
		Email: "dave@example.com", Password: "password123", Name: "Dave",
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	refreshResp, err := svc.Refresh(resp.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if refreshResp.AccessToken == "" {
		t.Error("access_token must not be empty")
	}
}

func TestRefresh_NotInDB(t *testing.T) {
	svc := NewAuthService(newMockUserRepo(), newMockSessionRepo(), testCfg())

	_, err := svc.Refresh("totally-fake-token")
	if err == nil {
		t.Fatal("expected error for token not in DB")
	}
}

func TestRefresh_ExpiredToken(t *testing.T) {
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	svc := NewAuthService(userRepo, sessionRepo, testCfg())

	svc.Register(AuthInput{Email: "eve@example.com", Password: "password123", Name: "Eve"})
	user := userRepo.byEmail["eve@example.com"]

	// Store a malformed token string (not a valid JWT) — ParseRefreshToken will fail
	fakeExpiredToken := "not.a.valid.jwt"
	sessionRepo.byToken[fakeExpiredToken] = &domain.Session{
		ID:           "sess-expired",
		UserID:       user.ID,
		RefreshToken: fakeExpiredToken,
		ExpiresAt:    time.Now().Add(-time.Hour),
	}

	_, err := svc.Refresh(fakeExpiredToken)
	if err == nil {
		t.Fatal("expected error for invalid token in DB")
	}
}

// -- Logout --

func TestLogout_Success(t *testing.T) {
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	svc := NewAuthService(userRepo, sessionRepo, testCfg())

	resp, err := svc.Register(AuthInput{
		Email: "frank@example.com", Password: "password123", Name: "Frank",
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := svc.Logout(resp.RefreshToken); err != nil {
		t.Errorf("Logout: %v", err)
	}
	if len(sessionRepo.byToken) != 0 {
		t.Error("expected session to be deleted after Logout")
	}
}

func TestLogout_NotFound_IsIdempotent(t *testing.T) {
	svc := NewAuthService(newMockUserRepo(), newMockSessionRepo(), testCfg())

	if err := svc.Logout("nonexistent-token"); err != nil {
		t.Errorf("Logout of missing token should be idempotent, got: %v", err)
	}
}

// -- LogoutAll --

func TestLogoutAll_Success(t *testing.T) {
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	svc := NewAuthService(userRepo, sessionRepo, testCfg())

	svc.Register(AuthInput{Email: "grace@example.com", Password: "pw12345678", Name: "Grace"})
	user := userRepo.byEmail["grace@example.com"]

	if err := svc.LogoutAll(user.ID); err != nil {
		t.Errorf("LogoutAll: %v", err)
	}
	if !sessionRepo.deleteAllCalled {
		t.Error("expected DeleteAllByUserID to be called")
	}
	if sessionRepo.deleteAllUserID != user.ID {
		t.Errorf("want userID=%s, got %s", user.ID, sessionRepo.deleteAllUserID)
	}
}

// -- GetProfile --

func TestGetProfile_Success(t *testing.T) {
	userRepo := newMockUserRepo()
	svc := NewAuthService(userRepo, newMockSessionRepo(), testCfg())

	svc.Register(AuthInput{Email: "henry@example.com", Password: "password123", Name: "Henry"})
	user := userRepo.byEmail["henry@example.com"]

	profile, err := svc.GetProfile(user.ID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if profile.Email != "henry@example.com" {
		t.Errorf("want email=henry@example.com, got %s", profile.Email)
	}
	if profile.Name != "Henry" {
		t.Errorf("want name=Henry, got %s", profile.Name)
	}
}

func TestGetProfile_NotFound(t *testing.T) {
	svc := NewAuthService(newMockUserRepo(), newMockSessionRepo(), testCfg())

	_, err := svc.GetProfile("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for unknown user ID")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != "NOT_FOUND" {
		t.Errorf("want NOT_FOUND, got %v", err)
	}
}
