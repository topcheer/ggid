package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDelegation_ValidateDelegation(t *testing.T) {
	tests := []struct {
		name    string
		d       UserDelegation
		wantErr bool
	}{
		{
			name: "valid",
			d: UserDelegation{
				DelegatorID: uuid.New().String(), DelegateeID: uuid.New().String(),
				Scopes: []string{"read", "write"}, ExpiresAt: time.Now().Add(time.Hour),
			},
			wantErr: false,
		},
		{
			name: "self-delegation",
			d: UserDelegation{
				DelegatorID: "same", DelegateeID: "same",
				Scopes: []string{"read"}, ExpiresAt: time.Now().Add(time.Hour),
			},
			wantErr: true,
		},
		{
			name: "admin scope forbidden",
			d: UserDelegation{
				DelegatorID: uuid.New().String(), DelegateeID: uuid.New().String(),
				Scopes: []string{"admin"}, ExpiresAt: time.Now().Add(time.Hour),
			},
			wantErr: true,
		},
		{
			name: "no scopes",
			d: UserDelegation{
				DelegatorID: uuid.New().String(), DelegateeID: uuid.New().String(),
				Scopes: []string{}, ExpiresAt: time.Now().Add(time.Hour),
			},
			wantErr: true,
		},
		{
			name: "expired",
			d: UserDelegation{
				DelegatorID: uuid.New().String(), DelegateeID: uuid.New().String(),
				Scopes: []string{"read"}, ExpiresAt: time.Now().Add(-time.Hour),
			},
			wantErr: true,
		},
		{
			name: "missing delegator",
			d: UserDelegation{
				DelegateeID: uuid.New().String(),
				Scopes: []string{"read"}, ExpiresAt: time.Now().Add(time.Hour),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDelegation(&tt.d)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDelegation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDelegation_ListNotConfigured(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("GET", "/api/v1/auth/delegations", nil)
	w := httptest.NewRecorder()
	h.handleDelegations(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", w.Code)
	}
}

func TestDelegation_CreateValidation(t *testing.T) {
	h := &Handler{}
	body := `{"delegatee_id":"` + uuid.New().String() + `","scopes":["read"],"expires_in_hours":24}`
	req := httptest.NewRequest("POST", "/api/v1/auth/delegations", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.handleDelegations(w, req)
	// 503 because no repo configured — confirms routing works.
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", w.Code)
	}
}

func TestDelegation_RevokeWrongMethod(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("POST", "/api/v1/auth/delegations/dlg-123", nil)
	w := httptest.NewRecorder()
	h.handleDelegations(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestDelegation_CheckRoute(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("GET", "/api/v1/auth/delegations/check?delegator_id=x&delegatee_id=y&scope=read", nil)
	w := httptest.NewRecorder()
	h.handleDelegations(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", w.Code)
	}
}

func TestDelegation_WrongMethod(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("PATCH", "/api/v1/auth/delegations", nil)
	w := httptest.NewRecorder()
	h.handleDelegations(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for PATCH, got %d", w.Code)
	}
}
