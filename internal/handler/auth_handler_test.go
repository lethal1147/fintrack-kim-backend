package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/middleware"
	"github.com/joakim/fintrack-api/internal/service"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/jwtutil"
)

// -- mock AuthService --

type mockAuthSvc struct {
	registerResp   *service.AuthResponse
	registerErr    error
	loginResp      *service.LoginResult
	loginErr       error
	refreshResp    *service.RefreshResponse
	refreshErr     error
	logoutErr      error
	profileResp    *service.UserInfo
	profileErr     error
	verifyTOTPResp *service.AuthResponse
	verifyTOTPErr  error
}

func (m *mockAuthSvc) Register(_ service.AuthInput) (*service.AuthResponse, error) {
	return m.registerResp, m.registerErr
}
func (m *mockAuthSvc) Login(_ service.LoginInput) (*service.LoginResult, error) {
	return m.loginResp, m.loginErr
}
func (m *mockAuthSvc) Refresh(_ string) (*service.RefreshResponse, error) {
	return m.refreshResp, m.refreshErr
}
func (m *mockAuthSvc) Logout(_ string) error   { return m.logoutErr }
func (m *mockAuthSvc) LogoutAll(_ string) error { return m.logoutErr }
func (m *mockAuthSvc) GetProfile(_ string) (*service.UserInfo, error) {
	return m.profileResp, m.profileErr
}
func (m *mockAuthSvc) VerifyTOTP(_, _, _, _ string) (*service.AuthResponse, error) {
	return m.verifyTOTPResp, m.verifyTOTPErr
}

// -- helpers --

const testSecret = "test-access-secret-value-32chars"

func signTestToken(userID string) string {
	tok, err := jwtutil.SignAccessToken(userID, "sess-test", testSecret, 15)
	if err != nil {
		panic("signTestToken: " + err.Error())
	}
	return tok
}

func authRouter(svc service.AuthServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAuthHandler(svc, false) // cookieSecure=false in tests
	auth := r.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/totp-verify", h.VerifyTOTP)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/logout", h.Logout) // public — no auth required
		protected := auth.Group("")
		protected.Use(middleware.Auth(testSecret))
		{
			protected.POST("/logout-all", h.LogoutAll)
			protected.GET("/me", h.Me)
		}
	}
	return r
}

func doPost(r *gin.Engine, path, body string, token ...string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if len(token) > 0 {
		req.Header.Set("Authorization", "Bearer "+token[0])
	}
	r.ServeHTTP(w, req)
	return w
}

func doPostCookie(r *gin.Engine, path, body, cookieValue string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: cookieValue})
	r.ServeHTTP(w, req)
	return w
}

func doGet(r *gin.Engine, path string, token ...string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	if len(token) > 0 {
		req.Header.Set("Authorization", "Bearer "+token[0])
	}
	r.ServeHTTP(w, req)
	return w
}

// getRefreshCookie extracts the refresh_token Set-Cookie from a response.
func getRefreshCookie(w *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == "refresh_token" {
			return c
		}
	}
	return nil
}

// -- Register --

func TestRegisterHandler_Success(t *testing.T) {
	svc := &mockAuthSvc{registerResp: &service.AuthResponse{
		AccessToken:  "access-tok",
		RefreshToken: "refresh-tok",
		User:         service.UserInfo{ID: "u1", Email: "alice@example.com", Name: "Alice"},
	}}
	w := doPost(authRouter(svc), "/auth/register",
		`{"email":"alice@example.com","password":"password123","name":"Alice"}`)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d — %s", w.Code, w.Body.String())
	}
	var body struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Success {
		t.Error("success must be true")
	}
	if body.Data.AccessToken == "" {
		t.Error("access_token must be present in body")
	}
}

func TestRegisterHandler_SetsCookie(t *testing.T) {
	svc := &mockAuthSvc{registerResp: &service.AuthResponse{
		AccessToken: "access-tok", RefreshToken: "refresh-tok",
		User: service.UserInfo{ID: "u1", Email: "a@b.com", Name: "A"},
	}}
	w := doPost(authRouter(svc), "/auth/register",
		`{"email":"a@b.com","password":"password123","name":"A"}`)

	cookie := getRefreshCookie(w)
	if cookie == nil {
		t.Fatal("expected refresh_token cookie to be set")
	}
	if !cookie.HttpOnly {
		t.Error("refresh_token cookie must be HttpOnly")
	}
	if cookie.Value == "" {
		t.Error("refresh_token cookie must have a value")
	}
}

func TestRegisterHandler_MissingFields(t *testing.T) {
	w := doPost(authRouter(&mockAuthSvc{}), "/auth/register", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestRegisterHandler_ShortPassword(t *testing.T) {
	w := doPost(authRouter(&mockAuthSvc{}), "/auth/register",
		`{"email":"a@b.com","password":"short","name":"A"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for password < 8 chars, got %d", w.Code)
	}
}

func TestRegisterHandler_Conflict(t *testing.T) {
	svc := &mockAuthSvc{registerErr: apperror.Conflict("email already registered")}
	w := doPost(authRouter(svc), "/auth/register",
		`{"email":"a@b.com","password":"password123","name":"A"}`)
	if w.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", w.Code)
	}
}

// -- Login --

func TestLoginHandler_Success(t *testing.T) {
	svc := &mockAuthSvc{loginResp: &service.LoginResult{Auth: &service.AuthResponse{
		AccessToken: "access-tok", RefreshToken: "refresh-tok",
		User: service.UserInfo{ID: "u1", Email: "bob@example.com", Name: "Bob"},
	}}}
	w := doPost(authRouter(svc), "/auth/login",
		`{"email":"bob@example.com","password":"password123"}`)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d — %s", w.Code, w.Body.String())
	}
	var body struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &body)
	if !body.Success || body.Data.AccessToken == "" {
		t.Error("expected success=true and access_token present")
	}
}

func TestLoginHandler_SetsCookie(t *testing.T) {
	svc := &mockAuthSvc{loginResp: &service.LoginResult{Auth: &service.AuthResponse{
		AccessToken: "access-tok", RefreshToken: "refresh-tok",
		User: service.UserInfo{ID: "u1", Email: "b@c.com", Name: "B"},
	}}}
	w := doPost(authRouter(svc), "/auth/login",
		`{"email":"b@c.com","password":"password123"}`)

	cookie := getRefreshCookie(w)
	if cookie == nil {
		t.Fatal("expected refresh_token cookie to be set on login")
	}
	if !cookie.HttpOnly {
		t.Error("refresh_token cookie must be HttpOnly")
	}
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	svc := &mockAuthSvc{loginErr: apperror.Unauthorized("invalid credentials")}
	w := doPost(authRouter(svc), "/auth/login",
		`{"email":"a@b.com","password":"wrongpassword"}`)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestLoginHandler_MissingFields(t *testing.T) {
	w := doPost(authRouter(&mockAuthSvc{}), "/auth/login", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing login body, got %d", w.Code)
	}
}

// -- Refresh --

func TestRefreshHandler_Success(t *testing.T) {
	svc := &mockAuthSvc{refreshResp: &service.RefreshResponse{AccessToken: "new-access-tok"}}
	// Refresh reads from cookie, no request body needed
	w := doPostCookie(authRouter(svc), "/auth/refresh", "", "some-valid-refresh-token")

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d — %s", w.Code, w.Body.String())
	}
	var body struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body.Data.AccessToken == "" {
		t.Error("access_token must be present")
	}
}

func TestRefreshHandler_NoCookie(t *testing.T) {
	// No cookie → 401
	w := doPost(authRouter(&mockAuthSvc{}), "/auth/refresh", "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401 when no refresh cookie, got %d", w.Code)
	}
}

func TestRefreshHandler_InvalidCookie(t *testing.T) {
	svc := &mockAuthSvc{refreshErr: apperror.Unauthorized("invalid refresh token")}
	w := doPostCookie(authRouter(svc), "/auth/refresh", "", "bad-cookie-value")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401 for invalid cookie, got %d", w.Code)
	}
}

// -- Logout --

func TestLogoutHandler_Success(t *testing.T) {
	// Logout is public — no bearer token needed, just the cookie
	w := doPostCookie(authRouter(&mockAuthSvc{}), "/auth/logout", "", "some-refresh-tok")
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d — %s", w.Code, w.Body.String())
	}
}

func TestLogoutHandler_ClearsCookie(t *testing.T) {
	w := doPostCookie(authRouter(&mockAuthSvc{}), "/auth/logout", "", "some-refresh-tok")

	cookie := getRefreshCookie(w)
	if cookie == nil {
		t.Fatal("expected Set-Cookie for refresh_token (to clear it)")
	}
	if cookie.MaxAge >= 0 {
		t.Errorf("refresh_token cookie must be cleared (MaxAge < 0), got MaxAge=%d", cookie.MaxAge)
	}
}

func TestLogoutHandler_NoCookieIsIdempotent(t *testing.T) {
	// Logout without a cookie is still OK (session already gone)
	w := doPost(authRouter(&mockAuthSvc{}), "/auth/logout", "")
	if w.Code != http.StatusOK {
		t.Errorf("want 200 (idempotent), got %d", w.Code)
	}
}

func TestLogoutHandler_ServiceError(t *testing.T) {
	svc := &mockAuthSvc{logoutErr: apperror.Internal("db down")}
	w := doPostCookie(authRouter(svc), "/auth/logout", "", "some-tok")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", w.Code)
	}
}

// -- LogoutAll --

func TestLogoutAllHandler_Success(t *testing.T) {
	tok := signTestToken("user-1")
	w := doPost(authRouter(&mockAuthSvc{}), "/auth/logout-all", ``, tok)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d — %s", w.Code, w.Body.String())
	}
}

func TestLogoutAllHandler_ServiceError(t *testing.T) {
	tok := signTestToken("user-1")
	svc := &mockAuthSvc{logoutErr: apperror.Internal("db down")}
	w := doPost(authRouter(svc), "/auth/logout-all", ``, tok)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", w.Code)
	}
}

// -- Me --

func TestMeHandler_Success(t *testing.T) {
	tok := signTestToken("user-1")
	svc := &mockAuthSvc{profileResp: &service.UserInfo{
		ID: "user-1", Email: "henry@example.com", Name: "Henry",
		Provider: "local", CreatedAt: time.Now(),
	}}
	w := doGet(authRouter(svc), "/auth/me", tok)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d — %s", w.Code, w.Body.String())
	}
	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Email string `json:"email"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &body)
	if !body.Success || body.Data.Email == "" {
		t.Error("expected success=true and email present")
	}
}

func TestMeHandler_Unauthenticated(t *testing.T) {
	w := doGet(authRouter(&mockAuthSvc{}), "/auth/me")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestMeHandler_ProfileNotFound(t *testing.T) {
	tok := signTestToken("user-1")
	svc := &mockAuthSvc{profileErr: apperror.NotFound("user not found")}
	w := doGet(authRouter(svc), "/auth/me", tok)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

// ─── Login TOTP challenge ─────────────────────────────────────────────────────

func TestLoginHandler_ReturnsTOTPChallenge(t *testing.T) {
	svc := &mockAuthSvc{loginResp: &service.LoginResult{Challenge: &service.TOTPChallengeResponse{
		TOTPRequired:   true,
		ChallengeToken: "challenge-tok",
	}}}
	w := doPost(authRouter(svc), "/auth/login", `{"email":"a@b.com","password":"pw"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	// Must NOT set a refresh cookie when TOTP is required.
	if cookie := getRefreshCookie(w); cookie != nil && cookie.MaxAge >= 0 {
		t.Error("must not set refresh cookie when TOTP challenge is returned")
	}
	var body struct {
		Data struct {
			TOTPRequired   bool   `json:"totp_required"`
			ChallengeToken string `json:"challenge_token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &body)
	if !body.Data.TOTPRequired || body.Data.ChallengeToken == "" {
		t.Error("expected totp_required=true and non-empty challenge_token")
	}
}

// ─── VerifyTOTP handler ───────────────────────────────────────────────────────

func TestVerifyTOTPHandler_OK(t *testing.T) {
	svc := &mockAuthSvc{verifyTOTPResp: &service.AuthResponse{
		AccessToken: "access-tok", RefreshToken: "refresh-tok",
		User: service.UserInfo{ID: "u1", Email: "a@b.com"},
	}}
	w := doPost(authRouter(svc), "/auth/totp-verify",
		`{"challenge_token":"ch-tok","code":"123456"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if cookie := getRefreshCookie(w); cookie == nil || cookie.Value == "" {
		t.Error("expected refresh cookie to be set on successful TOTP verify")
	}
}

func TestVerifyTOTPHandler_InvalidCode(t *testing.T) {
	svc := &mockAuthSvc{verifyTOTPErr: apperror.Unauthorized("invalid code")}
	w := doPost(authRouter(svc), "/auth/totp-verify",
		`{"challenge_token":"ch-tok","code":"000000"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestVerifyTOTPHandler_MissingBody(t *testing.T) {
	w := doPost(authRouter(&mockAuthSvc{}), "/auth/totp-verify", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}
