package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func corsRouter(origin string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(origin))
	r.GET("/data", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestCORS_AllowedOrigin(t *testing.T) {
	r := corsRouter("http://localhost:3000")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	got := w.Header().Get("Access-Control-Allow-Origin")
	if got != "http://localhost:3000" {
		t.Errorf("want allowed origin header, got %q", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	r := corsRouter("http://localhost:3000")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Origin", "http://evil.com")
	r.ServeHTTP(w, req)

	got := w.Header().Get("Access-Control-Allow-Origin")
	if got == "http://evil.com" {
		t.Error("must not echo back a disallowed origin")
	}
}

func TestCORS_Preflight(t *testing.T) {
	r := corsRouter("http://localhost:3000")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/data", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("preflight: want 204, got %d", w.Code)
	}
}

func TestCORS_AllowHeaders(t *testing.T) {
	r := corsRouter("http://localhost:3000")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	methods := w.Header().Get("Access-Control-Allow-Methods")
	headers := w.Header().Get("Access-Control-Allow-Headers")
	if methods == "" || headers == "" {
		t.Errorf("expected Allow-Methods and Allow-Headers to be set, got methods=%q headers=%q", methods, headers)
	}
}
