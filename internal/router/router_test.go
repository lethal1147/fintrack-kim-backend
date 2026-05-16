package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/pkg/response"
)

type stubHealthSvc struct{}

func (s *stubHealthSvc) Check() response.HealthResponse {
	return response.HealthResponse{Status: "ok", Version: "test", DB: "ok"}
}

func buildRouter(swaggerEnabled bool) *gin.Engine {
	gin.SetMode(gin.TestMode)

	// Import the handler package inline via the HealthChecker interface
	// (avoids circular imports — handler imports response, router imports handler)
	return newTestRouter(RouterConfig{
		Env:            "development",
		FrontendOrigin: "http://localhost:3000",
		SwaggerEnabled: swaggerEnabled,
	})
}

func TestRouter_HealthRoute(t *testing.T) {
	r := buildRouter(false)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d — %s", w.Code, w.Body.String())
	}
	var body struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.Success {
		t.Error("health response success must be true")
	}
}

func TestRouter_SwaggerDisabled(t *testing.T) {
	r := buildRouter(false)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("swagger must return 404 when disabled, got %d", w.Code)
	}
}

func TestRouter_SwaggerEnabled(t *testing.T) {
	r := buildRouter(true)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	// gin-swagger matches on Request.RequestURI; http.NewRequest leaves it empty in tests.
	req.RequestURI = req.URL.RequestURI()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Errorf("swagger route must be registered when enabled, got %d", w.Code)
	}
}

func TestRouter_CORS_Header(t *testing.T) {
	r := buildRouter(false)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Error("CORS header must be set for allowed origin")
	}
}

func TestRouter_UnknownRoute(t *testing.T) {
	r := buildRouter(false)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("unknown route: want 404, got %d", w.Code)
	}
}
