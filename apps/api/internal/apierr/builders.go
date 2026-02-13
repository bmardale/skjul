package apierr

import "net/http"

var (
	ErrUnauthorized = New(http.StatusUnauthorized, CodeUnauthorized, "missing or invalid session")
	ErrForbidden    = New(http.StatusForbidden, CodeForbidden, "forbidden")
	ErrNotFound     = New(http.StatusNotFound, CodeNotFound, "not found")
	ErrInternal     = New(http.StatusInternalServerError, CodeInternalError, "internal error")
)

func BadRequest(msg string) *APIError {
	return New(http.StatusBadRequest, CodeInvalidRequest, msg)
}

func InternalError(msg string) *APIError {
	return New(http.StatusInternalServerError, CodeInternalError, msg)
}

func Unauthorized(msg string) *APIError {
	return New(http.StatusUnauthorized, CodeUnauthorized, msg)
}

func Forbidden(msg string) *APIError {
	return New(http.StatusForbidden, CodeForbidden, msg)
}

func NotFound(msg string) *APIError {
	return New(http.StatusNotFound, CodeNotFound, msg)
}
