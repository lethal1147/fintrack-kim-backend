package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/pkg/response"
)

type mockHealthService struct{ dbStatus string }

func (m *mockHealthService) Check() response.HealthResponse {
	return response.HealthResponse{Status: "ok", Version: "test", DB: m.dbStatus}
}

func TestHealthHandler_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(&mockHealthService{dbStatus: "ok"})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/health", nil)

	h.HealthCheck(c)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}

	var body struct {
		Success bool `json:"success"`
		Data    response.HealthResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Success {
		t.Error("success must be true")
	}
	if body.Data.DB != "ok" {
		t.Errorf("want db=ok, got %s", body.Data.DB)
	}
}

func TestHealthHandler_DBDown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(&mockHealthService{dbStatus: "error"})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/health", nil)

	h.HealthCheck(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("want 503, got %d", w.Code)
	}
}
