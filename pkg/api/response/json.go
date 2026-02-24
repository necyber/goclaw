// Package response provides HTTP response utilities.
package response

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response with the given status code and data.
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// If encoding fails, we can't do much since headers are already sent
			// Log the error in production
			http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
		}
	}
}

// Error writes an error response with the given status code and error details.
func Error(w http.ResponseWriter, statusCode int, code, message string, requestID string) {
	errResp := ErrorResponse{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: requestID,
		},
	}
	JSON(w, statusCode, errResp)
}

// ErrorWithDetails writes an error response with additional details.
func ErrorWithDetails(w http.ResponseWriter, statusCode int, code, message string, details map[string]interface{}, requestID string) {
	errResp := ErrorResponse{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: requestID,
		},
	}
	JSON(w, statusCode, errResp)
}
