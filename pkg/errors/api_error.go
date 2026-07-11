package errors

import (
	"encoding/json"
	"net/http"
)

// APIError is the structured error response format for all GGID services.
// All API endpoints should return errors in this format for consistency.
//
// Format: {"error": {"code": "...", "message": "...", "request_id": "..."}}
type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// apiErrorResponse wraps APIError in an "error" key for JSON output.
type apiErrorResponse struct {
	Error APIError `json:"error"`
}

// CodeToHTTPStatus maps GGIDError codes to HTTP status codes.
func CodeToHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrNotFound:
		return http.StatusNotFound
	case ErrAlreadyExists:
		return http.StatusConflict
	case ErrInvalidArgument:
		return http.StatusBadRequest
	case ErrUnauthenticated:
		return http.StatusUnauthorized
	case ErrPermissionDenied:
		return http.StatusForbidden
	case ErrResourceExhausted:
		return http.StatusTooManyRequests
	case ErrFailedPrecondition:
		return http.StatusPreconditionFailed
	case ErrInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// WriteAPIError writes a structured API error response.
// Uses GGIDError code → HTTP status mapping.
// requestID is optional (pass "" to omit).
func WriteAPIError(w http.ResponseWriter, err error, requestID string) {
	ge, ok := AsGGIDError(err)
	if !ok {
		// Unknown error → internal
		ge = New(ErrInternal, "internal server error")
	}

	status := CodeToHTTPStatus(ge.Code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := apiErrorResponse{
		Error: APIError{
			Code:      string(ge.Code),
			Message:   ge.Message,
			RequestID: requestID,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// WriteSimpleAPIError writes a structured error with explicit code and message.
// Useful for error paths that don't have a GGIDError (e.g., "method not allowed").
func WriteSimpleAPIError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
		},
	})
}
