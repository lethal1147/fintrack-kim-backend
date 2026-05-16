package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/pkg/jwtutil"
)

const testSecret = "test-access-secret"

func setupRouter(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth(secret))
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get(ContextUserID)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	return r
}

func TestAuth_ValidToken(t *testing.T) {
	token, err := jwtutil.SignAccessToken("user-42", testSecret, 15)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	r := setupRouter(testSecret)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d — body: %s", w.Code, w.Body.String())
	}
	if !containsUserID(w.Body.String(), "user-42") {
		t.Errorf("want user_id=user-42 in body, got: %s", w.Body.String())
	}
}

func TestAuth_MissingHeader(t *testing.T) {
	r := setupRouter(testSecret)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestAuth_WrongScheme(t *testing.T) {
	r := setupRouter(testSecret)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	r := setupRouter(testSecret)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestAuth_WrongSecret(t *testing.T) {
	token, _ := jwtutil.SignAccessToken("u1", "other-secret", 15)
	r := setupRouter(testSecret)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestAuth_AbortsPropagation(t *testing.T) {
	reached := false
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth(testSecret))
	r.GET("/protected", func(c *gin.Context) { reached = true })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	if reached {
		t.Error("handler must not be called when auth fails")
	}
}

func containsUserID(body, id string) bool {
	return len(body) > 0 && (body[0] == '{') &&
		(func() bool {
			for i := 0; i < len(body)-len(id); i++ {
				if body[i:i+len(id)] == id {
					return true
				}
			}
			return false
		})()
}
