package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestGGIDError_Error(t *testing.T) {
	e := New(ErrNotFound, "user not found")
	msg := e.Error()
	if !strings.Contains(msg, "not_found") {
		t.Fatalf("error should contain code: %s", msg)
	}
	if !strings.Contains(msg, "user not found") {
		t.Fatalf("error should contain message: %s", msg)
	}
}

func TestGGIDError_Wrapped(t *testing.T) {
	inner := fmt.Errorf("db connection refused")
	e := Wrap(ErrInternal, "failed to query users", inner)

	msg := e.Error()
	if !strings.Contains(msg, "db connection refused") {
		t.Fatalf("wrapped error should contain cause: %s", msg)
	}
	if !strings.Contains(msg, "failed to query users") {
		t.Fatalf("error should contain message: %s", msg)
	}
	if !errors.Is(e, inner) {
		t.Fatal("errors.Is should find the wrapped cause")
	}
}

func TestAsGGIDError(t *testing.T) {
	original := New(ErrPermissionDenied, "access denied")

	var extracted *GGIDError
	ok := false

	// Test with a GGIDError directly
	extracted, ok = AsGGIDError(original)
	if !ok {
		t.Fatal("should extract GGIDError")
	}
	if extracted.Code != ErrPermissionDenied {
		t.Fatalf("code mismatch: got %s", extracted.Code)
	}

	// Test with a non-GGIDError
	plainErr := fmt.Errorf("plain error")
	_, ok = AsGGIDError(plainErr)
	if ok {
		t.Fatal("should not extract from non-GGIDError")
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name string
		err  *GGIDError
		code ErrorCode
	}{
		{"NotFound", NotFound("User", "123"), ErrNotFound},
		{"AlreadyExists", AlreadyExists("User", "123"), ErrAlreadyExists},
		{"InvalidArgument", InvalidArgument("bad input"), ErrInvalidArgument},
		{"Unauthenticated", Unauthenticated("no token"), ErrUnauthenticated},
		{"PermissionDenied", PermissionDenied("forbidden"), ErrPermissionDenied},
		{"Internal", Internal("boom", fmt.Errorf("err")), ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Fatalf("%s: code mismatch: got %s, want %s", tt.name, tt.err.Code, tt.code)
			}
			if tt.err.Message == "" {
				t.Fatalf("%s: message should not be empty", tt.name)
			}
		})
	}
}

func TestGGIDError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	e := Wrap(ErrInternal, "wrapper", cause)

	unwrapped := e.Unwrap()
	if unwrapped == nil || unwrapped.Error() != "root cause" {
		t.Fatalf("Unwrap should return the cause, got: %v", unwrapped)
	}
}
