package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCodeToHTTPStatus(t *testing.T) {
	cases := []struct {
		code   ErrorCode
		expect int
	}{
		{ErrNotFound, http.StatusNotFound},
		{ErrAlreadyExists, http.StatusConflict},
		{ErrInvalidArgument, http.StatusBadRequest},
		{ErrUnauthenticated, http.StatusUnauthorized},
		{ErrPermissionDenied, http.StatusForbidden},
		{ErrResourceExhausted, http.StatusTooManyRequests},
		{ErrFailedPrecondition, http.StatusPreconditionFailed},
		{ErrInternal, http.StatusInternalServerError},
		{ErrorCode("unknown"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		got := CodeToHTTPStatus(tc.code)
		if got != tc.expect {
			t.Errorf("CodeToHTTPStatus(%s) = %d, want %d", tc.code, got, tc.expect)
		}
	}
}

func TestWriteAPIError_WithGGIDError(t *testing.T) {
	w := httptest.NewRecorder()
	err := New(ErrNotFound, "user not found")
	WriteAPIError(w, err, "req-123")

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var body apiErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("code = %s, want not_found", body.Error.Code)
	}
	if body.Error.Message != "user not found" {
		t.Errorf("message = %s", body.Error.Message)
	}
	if body.Error.RequestID != "req-123" {
		t.Errorf("request_id = %s", body.Error.RequestID)
	}
}

func TestWriteAPIError_WithGenericError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteAPIError(w, errors.New("something broke"), "")

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}

	var body apiErrorResponse
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Error.Code != "internal" {
		t.Errorf("code = %s, want internal", body.Error.Code)
	}
}

func TestWriteSimpleAPIError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteSimpleAPIError(w, http.StatusBadRequest, "invalid_argument", "bad input")

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var body apiErrorResponse
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Error.Code != "invalid_argument" {
		t.Errorf("code = %s", body.Error.Code)
	}
	if body.Error.Message != "bad input" {
		t.Errorf("message = %s", body.Error.Message)
	}
}
