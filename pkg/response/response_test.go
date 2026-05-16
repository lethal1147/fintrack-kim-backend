package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

func newContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestSuccess(t *testing.T) {
	c, w := newContext()
	Success(c, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var body envelope
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Success {
		t.Error("success must be true")
	}
	if body.Data == nil {
		t.Error("data must be present")
	}
}

func TestError_AppError(t *testing.T) {
	c, w := newContext()
	Error(c, apperror.NotFound("item not found"))

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
	var body errorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Success {
		t.Error("success must be false")
	}
	if body.Error.Code != "NOT_FOUND" {
		t.Errorf("want NOT_FOUND, got %s", body.Error.Code)
	}
	if body.Error.Message == "" {
		t.Error("message must be present")
	}
}

func TestError_UnknownError(t *testing.T) {
	c, w := newContext()
	Error(c, apperror.Internal("boom"))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", w.Code)
	}
}

func TestCreated(t *testing.T) {
	c, w := newContext()
	Created(c, map[string]string{"id": "new-123"})

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", w.Code)
	}
	var body envelope
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Success {
		t.Error("success must be true")
	}
}
