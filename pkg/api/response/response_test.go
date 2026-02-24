package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		data       interface{}
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success with data",
			statusCode: http.StatusOK,
			data:       map[string]string{"message": "success"},
			wantStatus: http.StatusOK,
			wantBody:   `{"message":"success"}`,
		},
		{
			name:       "created with data",
			statusCode: http.StatusCreated,
			data:       map[string]int{"id": 123},
			wantStatus: http.StatusCreated,
			wantBody:   `{"id":123}`,
		},
		{
			name:       "no content",
			statusCode: http.StatusNoContent,
			data:       nil,
			wantStatus: http.StatusNoContent,
			wantBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			JSON(w, tt.statusCode, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("JSON() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if tt.data != nil {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("JSON() Content-Type = %v, want application/json", contentType)
				}

				var got, want interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if err := json.Unmarshal([]byte(tt.wantBody), &want); err != nil {
					t.Fatalf("failed to unmarshal expected: %v", err)
				}

				gotJSON, _ := json.Marshal(got)
				wantJSON, _ := json.Marshal(want)
				if string(gotJSON) != string(wantJSON) {
					t.Errorf("JSON() body = %s, want %s", gotJSON, wantJSON)
				}
			}
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		code       string
		message    string
		requestID  string
		wantStatus int
	}{
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			code:       ErrCodeBadRequest,
			message:    "invalid input",
			requestID:  "req-123",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			code:       ErrCodeNotFound,
			message:    "resource not found",
			requestID:  "req-456",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Error(w, tt.statusCode, tt.code, tt.message, tt.requestID)

			if w.Code != tt.wantStatus {
				t.Errorf("Error() status = %v, want %v", w.Code, tt.wantStatus)
			}

			var resp ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if resp.Error.Code != tt.code {
				t.Errorf("Error() code = %v, want %v", resp.Error.Code, tt.code)
			}
			if resp.Error.Message != tt.message {
				t.Errorf("Error() message = %v, want %v", resp.Error.Message, tt.message)
			}
			if resp.Error.RequestID != tt.requestID {
				t.Errorf("Error() requestID = %v, want %v", resp.Error.RequestID, tt.requestID)
			}
		})
	}
}

func TestHTTPStatusFromError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "not found",
			err:  ErrNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "invalid input",
			err:  ErrInvalidInput,
			want: http.StatusBadRequest,
		},
		{
			name: "validation failed",
			err:  ErrValidationFailed,
			want: http.StatusBadRequest,
		},
		{
			name: "conflict",
			err:  ErrConflict,
			want: http.StatusConflict,
		},
		{
			name: "service unavailable",
			err:  ErrServiceUnavailable,
			want: http.StatusServiceUnavailable,
		},
		{
			name: "timeout",
			err:  ErrTimeout,
			want: http.StatusGatewayTimeout,
		},
		{
			name: "unknown error",
			err:  ErrInternalServer,
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTTPStatusFromError(tt.err); got != tt.want {
				t.Errorf("HTTPStatusFromError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorCodeFromStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   string
	}{
		{
			name:   "bad request",
			status: http.StatusBadRequest,
			want:   ErrCodeBadRequest,
		},
		{
			name:   "not found",
			status: http.StatusNotFound,
			want:   ErrCodeNotFound,
		},
		{
			name:   "conflict",
			status: http.StatusConflict,
			want:   ErrCodeConflict,
		},
		{
			name:   "service unavailable",
			status: http.StatusServiceUnavailable,
			want:   ErrCodeServiceUnavailable,
		},
		{
			name:   "unknown status",
			status: 999,
			want:   ErrCodeInternalServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrorCodeFromStatus(tt.status); got != tt.want {
				t.Errorf("ErrorCodeFromStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
