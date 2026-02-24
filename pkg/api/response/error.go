package response

import (
	"errors"
	"net/http"
)

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information.
type ErrorDetail struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	RequestID string                 `json:"request_id"`
}

// Common error codes
const (
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeValidationFailed    = "VALIDATION_FAILED"
	ErrCodeInternalServer      = "INTERNAL_SERVER_ERROR"
	ErrCodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	ErrCodeGatewayTimeout      = "GATEWAY_TIMEOUT"
)

// Common errors
var (
	ErrNotFound            = errors.New("resource not found")
	ErrInvalidInput        = errors.New("invalid input")
	ErrValidationFailed    = errors.New("validation failed")
	ErrConflict            = errors.New("resource conflict")
	ErrServiceUnavailable  = errors.New("service unavailable")
	ErrTimeout             = errors.New("request timeout")
	ErrInternalServer      = errors.New("internal server error")
)

// HTTPStatusFromError maps common errors to HTTP status codes.
func HTTPStatusFromError(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrValidationFailed):
		return http.StatusBadRequest
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrServiceUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, ErrTimeout):
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// ErrorCodeFromStatus returns an error code for the given HTTP status.
func ErrorCodeFromStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return ErrCodeBadRequest
	case http.StatusUnauthorized:
		return ErrCodeUnauthorized
	case http.StatusForbidden:
		return ErrCodeForbidden
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusMethodNotAllowed:
		return ErrCodeMethodNotAllowed
	case http.StatusConflict:
		return ErrCodeConflict
	case http.StatusServiceUnavailable:
		return ErrCodeServiceUnavailable
	case http.StatusGatewayTimeout:
		return ErrCodeGatewayTimeout
	default:
		return ErrCodeInternalServer
	}
}

// HandleError is a convenience function to handle errors and write appropriate responses.
func HandleError(w http.ResponseWriter, err error, requestID string) {
	status := HTTPStatusFromError(err)
	code := ErrorCodeFromStatus(status)
	Error(w, status, code, err.Error(), requestID)
}
