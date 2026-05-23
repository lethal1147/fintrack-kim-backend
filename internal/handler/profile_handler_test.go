package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/middleware"
	"github.com/joakim/fintrack-api/internal/service"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

// ── mock profile service ───────────────────────────────────────────────────────

type mockProfileSvc struct {
	userInfo   *service.UserInfo
	avatarURL  string
	updateErr  error
	uploadErr  error
	deleteErr  error
}

func (m *mockProfileSvc) UpdateProfile(_ string, _ service.UpdateProfileRequest) (*service.UserInfo, error) {
	return m.userInfo, m.updateErr
}

func (m *mockProfileSvc) UploadAvatar(_, _, _ string, _ int64, _ io.Reader) (string, error) {
	return m.avatarURL, m.uploadErr
}

func (m *mockProfileSvc) DeleteAccount(_, _ string) error {
	return m.deleteErr
}

// ── router helper ──────────────────────────────────────────────────────────────

func profileRouter(svc service.ProfileServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewProfileHandler(svc)
	g := r.Group("/profile", middleware.Auth(testSecret))
	{
		g.PATCH("", h.Update)
		g.POST("/avatar", h.UploadAvatar)
		g.DELETE("", h.DeleteAccount)
	}
	return r
}

func sampleUserInfo() *service.UserInfo {
	return &service.UserInfo{
		ID:        "user-1",
		Email:     "kim@example.com",
		Name:      "Kim Johnson",
		AvatarURL: "",
		Provider:  "local",
		CreatedAt: time.Now(),
	}
}

// ── tests ──────────────────────────────────────────────────────────────────────

func TestProfile_Update_OK(t *testing.T) {
	svc := &mockProfileSvc{userInfo: sampleUserInfo()}
	r := profileRouter(svc)

	body, _ := json.Marshal(map[string]string{"name": "New Name", "email": "new@example.com"})
	req, _ := http.NewRequest(http.MethodPatch, "/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200 — body: %s", w.Code, w.Body.String())
	}
}

func TestProfile_Update_Unauthorized(t *testing.T) {
	svc := &mockProfileSvc{userInfo: sampleUserInfo()}
	r := profileRouter(svc)

	body, _ := json.Marshal(map[string]string{"name": "X", "email": "x@x.com"})
	req, _ := http.NewRequest(http.MethodPatch, "/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", w.Code)
	}
}

func TestProfile_Avatar_OK(t *testing.T) {
	svc := &mockProfileSvc{
		userInfo:  sampleUserInfo(),
		avatarURL: "https://r2.example.com/avatars/user-1/abc.jpg",
	}
	r := profileRouter(svc)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("avatar", "photo.jpg")
	fw.Write([]byte("fake-image-data"))
	mw.Close()

	req, _ := http.NewRequest(http.MethodPost, "/profile/avatar", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200 — body: %s", w.Code, w.Body.String())
	}
}

func TestProfile_Avatar_Unauthorized(t *testing.T) {
	svc := &mockProfileSvc{avatarURL: "https://r2.example.com/x"}
	r := profileRouter(svc)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("avatar", "photo.jpg")
	fw.Write([]byte("data"))
	mw.Close()

	req, _ := http.NewRequest(http.MethodPost, "/profile/avatar", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", w.Code)
	}
}

func TestProfile_DeleteAccount_OK(t *testing.T) {
	svc := &mockProfileSvc{}
	r := profileRouter(svc)

	body, _ := json.Marshal(map[string]string{"password": "correct-pw"})
	req, _ := http.NewRequest(http.MethodDelete, "/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200 — body: %s", w.Code, w.Body.String())
	}
}

func TestProfile_DeleteAccount_WrongPassword(t *testing.T) {
	svc := &mockProfileSvc{deleteErr: apperror.BadRequest("incorrect password")}
	r := profileRouter(svc)

	body, _ := json.Marshal(map[string]string{"password": "wrong-pw"})
	req, _ := http.NewRequest(http.MethodDelete, "/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400 — body: %s", w.Code, w.Body.String())
	}
}

func TestProfile_DeleteAccount_MissingPassword(t *testing.T) {
	svc := &mockProfileSvc{}
	r := profileRouter(svc)

	body, _ := json.Marshal(map[string]string{})
	req, _ := http.NewRequest(http.MethodDelete, "/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400 — body: %s", w.Code, w.Body.String())
	}
}
