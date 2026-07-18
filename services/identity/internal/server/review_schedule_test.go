package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateSchedule(t *testing.T) {
	tests := []struct {
		name    string
		s       ReviewSchedule
		wantErr bool
	}{
		{"valid quarterly", ReviewSchedule{ScopeType: "user", ScopeID: "u1", FrequencyDays: 90}, false},
		{"valid monthly", ReviewSchedule{ScopeType: "role", ScopeID: "r1", FrequencyDays: 30}, false},
		{"valid annual", ReviewSchedule{ScopeType: "group", ScopeID: "g1", FrequencyDays: 365}, false},
		{"invalid freq", ReviewSchedule{ScopeType: "user", ScopeID: "u1", FrequencyDays: 7}, true},
		{"missing scope_id", ReviewSchedule{ScopeType: "user", FrequencyDays: 90}, true},
		{"missing scope_type", ReviewSchedule{ScopeID: "u1", FrequencyDays: 90}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSchedule(&tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSchedule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReviewSched_NotConfigured(t *testing.T) {
	h := &HTTPHandler{}
	req := httptest.NewRequest("GET", "/api/v1/identity/review-schedules", nil)
	w := httptest.NewRecorder()
	h.handleReviewSchedules(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestReviewSched_RunNotConfigured(t *testing.T) {
	h := &HTTPHandler{}
	req := httptest.NewRequest("POST", "/api/v1/identity/review-schedules/run", nil)
	w := httptest.NewRecorder()
	h.handleReviewSchedules(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestReviewSched_WrongMethod(t *testing.T) {
	h := &HTTPHandler{}
	req := httptest.NewRequest("PATCH", "/api/v1/identity/review-schedules", nil)
	w := httptest.NewRecorder()
	h.handleReviewSchedules(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestReviewSchedule_NextRunCalc(t *testing.T) {
	// Verify the frequency → next_run_at calculation.
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	quarterly := now.Add(90 * 24 * time.Hour)
	if quarterly.Day() != 16 || quarterly.Month() != 10 {
		t.Errorf("90 days from July 18 = Oct 16, got %v", quarterly)
	}
	annual := now.Add(365 * 24 * time.Hour)
	if annual.Year() != 2027 {
		t.Errorf("365 days = 2027, got %v", annual)
	}
}
