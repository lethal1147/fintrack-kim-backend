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
	registerResp *service.AuthResponse
	registerErr  error
	loginResp    *service.AuthResponse
	loginErr     error
	refreshResp  *service.RefreshResponse
	refreshErr   error
	logoutErr    error
	profileResp  *service.UserInfo
	profileErr   error
}

func (m *mockAuthSvc) Register(_ service.AuthInput) (*service.AuthResponse, error) {
	return m.registerResp, m.registerErr
}
func (m *mockAuthSvc) Login(_ service.LoginInput) (*service.AuthResponse, error) {
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

// -- helpers --

const testSecret = "test-access-secret-value-32chars"

func signTestToken(userID string) string {
	tok, err := jwtutil.SignAccessToken(userID, testSecret, 15)
	if err != nil {
		panic("signTestToken: " + err.Error())
	}
	return tok
}

func authRouter(svc service.AuthServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAuthHandler(svc)
	auth := r.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		protected := auth.Group("")
		protected.Use(middleware.Auth(testSecret))
		{
			protected.POST("/logout", h.Logout)
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

func doGet(r *gin.Engine, path string, token ...string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	if len(token) > 0 {
		req.Header.Set("Authorization", "Bearer "+token[0])
	}
	r.ServeHTTP(w, req)
	return w
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
		t.Error("access_token must be present")
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
	svc := &mockAuthSvc{loginResp: &service.AuthResponse{
		AccessToken:  "access-tok",
		RefreshToken: "refresh-tok",
		User:         service.UserInfo{ID: "u1", Email: "bob@example.com", Name: "Bob"},
	}}
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

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	svc := &mockAuthSvc{loginErr: apperror.Unauthorized("invalid credentials")}
	w := doPost(authRouter(svc), "/auth/login",
		`{"email":"a@b.com","password":"wrongpassword"}`)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

// -- Refresh --

func TestRefreshHandler_Success(t *testing.T) {
	svc := &mockAuthSvc{refreshResp: &service.RefreshResponse{AccessToken: "new-access-tok"}}
	w := doPost(authRouter(svc), "/auth/refresh", `{"refresh_token":"some-refresh-token"}`)

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

func TestRefreshHandler_InvalidToken(t *testing.T) {
	svc := &mockAuthSvc{refreshErr: apperror.Unauthorized("invalid refresh token")}
	w := doPost(authRouter(svc), "/auth/refresh", `{"refresh_token":"bad-token"}`)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

// -- Logout --

func TestLogoutHandler_Success(t *testing.T) {
	tok := signTestToken("user-1")
	w := doPost(authRouter(&mockAuthSvc{}), "/auth/logout",
		`{"refresh_token":"some-refresh-tok"}`, tok)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d — %s", w.Code, w.Body.String())
	}
}

func TestLogoutHandler_Unauthenticated(t *testing.T) {
	w := doPost(authRouter(&mockAuthSvc{}), "/auth/logout", `{"refresh_token":"tok"}`)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
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
	if !body.Success {
		t.Error("success must be true")
	}
	if body.Data.Email == "" {
		t.Error("email must be present in profile response")
	}
}

func TestMeHandler_Unauthenticated(t *testing.T) {
	w := doGet(authRouter(&mockAuthSvc{}), "/auth/me")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}
