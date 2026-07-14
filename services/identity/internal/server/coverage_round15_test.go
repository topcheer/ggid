package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestHandleCertificationStatus(t *testing.T) {
	h := NewHTTPHandler(nil)
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	t.Run("GET returns certification status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/certification-status", nil)
		req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if body["status"] != "current" {
			t.Fatalf("expected status current, got %v", body["status"])
		}
	})

	t.Run("POST returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/certification-status", nil)
		req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})
}

func TestHandleManagementChain(t *testing.T) {
	h := NewHTTPHandler(nil)
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	t.Run("GET returns chain", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/management-chain", nil)
		req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if depth, ok := body["depth"].(float64); !ok || depth != 4 {
			t.Fatalf("expected depth 4, got %v", body["depth"])
		}
	})

	t.Run("POST returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/management-chain", nil)
		req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})
}

func TestReassignUser(t *testing.T) {
	h := NewHTTPHandler(nil)
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	ctx := context.Background()

	t.Run("POST reassigns user", func(t *testing.T) {
		payload := map[string]string{
			"new_org":      "org-2",
			"new_role":     "analyst",
			"new_manager":  "mgr-99",
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/reassign", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		h.reassignUser(ctx, userID, rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if body["new_org"] != "org-2" {
			t.Fatalf("expected org-2, got %v", body["new_org"])
		}
	})

	t.Run("GET method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/reassign", nil)
		rec := httptest.NewRecorder()
		h.reassignUser(ctx, userID, rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/reassign", bytes.NewReader([]byte("not-json")))
		rec := httptest.NewRecorder()
		h.reassignUser(ctx, userID, rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandleGDPRExport(t *testing.T) {
	h := NewHTTPHandler(nil)

	t.Run("POST returns accepted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/identity/gdpr/export", nil)
		req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if body["status"] != "processing" {
			t.Fatalf("expected processing, got %v", body["status"])
		}
	})

	t.Run("GET returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/identity/gdpr/export", nil)
		req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})
}
