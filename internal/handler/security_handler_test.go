package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/internal/middleware"
)

// ─── mock security service ────────────────────────────────────────────────────

type mockSecuritySvc struct {
	sessionsResp       []domain.SessionInfo
	sessionsErr        error
	revokeErr          error
	reqPwChangeErr     error
	changePwErr        error
	setupTOTPResp      *domain.TOTPSetupResult
	setupTOTPErr       error
	confirmTOTPCodes   []string
	confirmTOTPErr     error
	disableTOTPErr     error
	reqPwResetErr      error
	resetPasswordErr   error
}

func (m *mockSecuritySvc) ListSessions(_ context.Context, _, _ string) ([]domain.SessionInfo, error) {
	return m.sessionsResp, m.sessionsErr
}
func (m *mockSecuritySvc) RevokeSession(_ context.Context, _, _ string) error { return m.revokeErr }
func (m *mockSecuritySvc) RequestPasswordChange(_ context.Context, _ string) error {
	return m.reqPwChangeErr
}
func (m *mockSecuritySvc) ChangePassword(_ context.Context, _, _, _ string) error {
	return m.changePwErr
}
func (m *mockSecuritySvc) SetupTOTP(_ context.Context, _ string) (*domain.TOTPSetupResult, error) {
	return m.setupTOTPResp, m.setupTOTPErr
}
func (m *mockSecuritySvc) ConfirmTOTP(_ context.Context, _, _ string) ([]string, error) {
	return m.confirmTOTPCodes, m.confirmTOTPErr
}
func (m *mockSecuritySvc) DisableTOTP(_ context.Context, _, _ string) error { return m.disableTOTPErr }
func (m *mockSecuritySvc) RequestPasswordReset(_ context.Context, _ string) error {
	return m.reqPwResetErr
}
func (m *mockSecuritySvc) ResetPassword(_ context.Context, _, _, _ string) error {
	return m.resetPasswordErr
}

// ─── test helpers ─────────────────────────────────────────────────────────────

func securityRouter(svc domain.SecurityServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSecurityHandler(svc, testSecret, false)
	sec := r.Group("/security")
	sec.Use(middleware.Auth(testSecret))
	{
		sec.GET("/sessions", h.ListSessions)
		sec.DELETE("/sessions/:id", h.RevokeSession)
		sec.POST("/password/request", h.RequestPasswordChange)
		sec.POST("/password/change", h.ChangePassword)
		sec.POST("/totp/setup", h.SetupTOTP)
		sec.POST("/totp/confirm", h.ConfirmTOTP)
		sec.DELETE("/totp", h.DisableTOTP)
	}
	return r
}

func doSecGet(r *gin.Engine, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func doSecPost(r *gin.Engine, path, token string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func doSecDelete(r *gin.Engine, path, token string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(http.MethodDelete, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ─── Sessions tests ───────────────────────────────────────────────────────────

func TestSecurity_ListSessions_OK(t *testing.T) {
	svc := &mockSecuritySvc{sessionsResp: []domain.SessionInfo{
		{ID: "s1", Device: "Chrome on macOS", LastActiveAt: time.Now(), IsCurrent: true},
	}}
	r := securityRouter(svc)
	tok := signTestToken("user1")

	w := doSecGet(r, "/security/sessions", tok)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurity_ListSessions_Unauthorized(t *testing.T) {
	r := securityRouter(&mockSecuritySvc{})
	w := doSecGet(r, "/security/sessions", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSecurity_RevokeSession_OK(t *testing.T) {
	r := securityRouter(&mockSecuritySvc{})
	tok := signTestToken("user1")
	w := doSecDelete(r, "/security/sessions/sess-abc", tok, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurity_RevokeSession_Unauthorized(t *testing.T) {
	r := securityRouter(&mockSecuritySvc{})
	w := doSecDelete(r, "/security/sessions/sess-abc", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ─── Password change tests ────────────────────────────────────────────────────

func TestSecurity_RequestPasswordChange_OK(t *testing.T) {
	r := securityRouter(&mockSecuritySvc{})
	w := doSecPost(r, "/security/password/request", signTestToken("user1"), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurity_RequestPasswordChange_Unauthorized(t *testing.T) {
	r := securityRouter(&mockSecuritySvc{})
	w := doSecPost(r, "/security/password/request", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSecurity_ChangePassword_OK(t *testing.T) {
	r := securityRouter(&mockSecuritySvc{})
	tok := signTestToken("user1")
	w := doSecPost(r, "/security/password/change", tok, map[string]string{
		"otp":          "123456",
		"new_password": "newpassword123",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecurity_ChangePassword_MissingBody(t *testing.T) {
	r := securityRouter(&mockSecuritySvc{})
	w := doSecPost(r, "/security/password/change", signTestToken("user1"), nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ─── forgot password router ───────────────────────────────────────────────────

func forgotPasswordRouter(svc domain.SecurityServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSecurityHandler(svc, testSecret, false)
	auth := r.Group("/auth")
	{
		auth.POST("/forgot-password/request", h.ForgotPasswordRequest)
		auth.POST("/forgot-password/reset", h.ForgotPasswordReset)
	}
	return r
}

// ─── TestSecurity_ForgotPasswordRequest_OK ───────────────────────────────────

func TestSecurity_ForgotPasswordRequest_OK(t *testing.T) {
	r := forgotPasswordRouter(&mockSecuritySvc{})
	w := doSecPost(r, "/auth/forgot-password/request", "", map[string]string{
		"email": "user@example.com",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── TestSecurity_ForgotPasswordRequest_InvalidEmail ─────────────────────────

func TestSecurity_ForgotPasswordRequest_InvalidEmail(t *testing.T) {
	r := forgotPasswordRouter(&mockSecuritySvc{})
	w := doSecPost(r, "/auth/forgot-password/request", "", map[string]string{
		"email": "not-an-email",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── TestSecurity_ForgotPasswordReset_OK ─────────────────────────────────────

func TestSecurity_ForgotPasswordReset_OK(t *testing.T) {
	r := forgotPasswordRouter(&mockSecuritySvc{})
	w := doSecPost(r, "/auth/forgot-password/reset", "", map[string]interface{}{
		"email":        "user@example.com",
		"otp":          "123456",
		"new_password": "newpassword123",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── TestSecurity_ForgotPasswordReset_BadRequest ──────────────────────────────

func TestSecurity_ForgotPasswordReset_BadRequest(t *testing.T) {
	r := forgotPasswordRouter(&mockSecuritySvc{})
	// Missing new_password field
	w := doSecPost(r, "/auth/forgot-password/reset", "", map[string]string{
		"email": "user@example.com",
		"otp":   "123456",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
