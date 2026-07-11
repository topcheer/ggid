package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// ErrorResponse is the standard gateway error envelope.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains structured error details.
type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// WriteError writes a standardized JSON error response.
// It extracts the request ID from context if available, generating one otherwise.
func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	requestID := ""
	if r != nil {
		if id, ok := r.Context().Value(RequestIDContextKey{}).(string); ok && id != "" {
			requestID = id
		}
	}
	if requestID == "" {
		requestID = uuid.New().String()
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorBody{
			Code:      code,
			Message:   message,
			RequestID: requestID,
		},
	})
}

// WriteErrorNoRequest writes a standardized error without a *http.Request.
// Use when the request is not available (e.g., in health check handlers).
func WriteErrorNoRequest(w http.ResponseWriter, status int, code, message string) {
	requestID := uuid.New().String()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorBody{
			Code:      code,
			Message:   message,
			RequestID: requestID,
		},
	})
}
