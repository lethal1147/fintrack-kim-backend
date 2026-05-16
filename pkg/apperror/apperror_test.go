package apperror

import (
	"errors"
	"net/http"
	"testing"
)

func TestConstructors(t *testing.T) {
	cases := []struct {
		name       string
		err        *AppError
		wantCode   string
		wantStatus int
	}{
		{"bad request", BadRequest("bad"), "BAD_REQUEST", http.StatusBadRequest},
		{"unauthorized", Unauthorized("unauth"), "UNAUTHORIZED", http.StatusUnauthorized},
		{"forbidden", Forbidden("forbidden"), "FORBIDDEN", http.StatusForbidden},
		{"not found", NotFound("nf"), "NOT_FOUND", http.StatusNotFound},
		{"conflict", Conflict("conflict"), "CONFLICT", http.StatusConflict},
		{"internal", Internal("oops"), "INTERNAL_ERROR", http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Code != tc.wantCode {
				t.Errorf("code: want %s, got %s", tc.wantCode, tc.err.Code)
			}
			if tc.err.HTTPStatus != tc.wantStatus {
				t.Errorf("status: want %d, got %d", tc.wantStatus, tc.err.HTTPStatus)
			}
			if tc.err.Error() == "" {
				t.Error("Error() must not be empty")
			}
		})
	}
}

func TestHTTPStatus(t *testing.T) {
	if HTTPStatus(nil) != http.StatusOK {
		t.Error("nil error should return 200")
	}
	if HTTPStatus(NotFound("x")) != http.StatusNotFound {
		t.Error("AppError should return its status")
	}
	if HTTPStatus(errors.New("unknown")) != http.StatusInternalServerError {
		t.Error("unknown error should return 500")
	}
}
