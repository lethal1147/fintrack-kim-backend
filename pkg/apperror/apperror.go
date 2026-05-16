package apperror

import "net/http"

type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *AppError) Error() string { return e.Message }

func New(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: status}
}

func BadRequest(message string) *AppError {
	return New("BAD_REQUEST", message, http.StatusBadRequest)
}

func Unauthorized(message string) *AppError {
	return New("UNAUTHORIZED", message, http.StatusUnauthorized)
}

func Forbidden(message string) *AppError {
	return New("FORBIDDEN", message, http.StatusForbidden)
}

func NotFound(message string) *AppError {
	return New("NOT_FOUND", message, http.StatusNotFound)
}

func Conflict(message string) *AppError {
	return New("CONFLICT", message, http.StatusConflict)
}

func Internal(message string) *AppError {
	return New("INTERNAL_ERROR", message, http.StatusInternalServerError)
}

// HTTPStatus returns the HTTP status code for the error.
// Returns 500 for unknown error types.
func HTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if e, ok := err.(*AppError); ok {
		return e.HTTPStatus
	}
	return http.StatusInternalServerError
}
