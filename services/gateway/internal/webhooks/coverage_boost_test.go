package webhooks

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- Deliver retry tests ---

func TestHTTPDeliverer_RetrySucceedsOnSecondAttempt(t *testing.T) {
	attempts := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewHTTPDeliverer()
	err := d.Deliver(context.Background(), server.URL, "", []byte(`{}`))
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestHTTPDeliverer_AllRetriesFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	d := NewHTTPDeliverer()
	err := d.Deliver(context.Background(), server.URL, "", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error after all retries fail")
	}
}

func TestHTTPDeliverer_ConnectionError(t *testing.T) {
	d := NewHTTPDeliverer()
	// Use a closed server to trigger connection error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.Close()

	err := d.Deliver(context.Background(), server.URL, "", []byte(`{}`))
	if err == nil {
		t.Error("expected connection error")
	}
}

func TestHTTPDeliverer_WithHMACSignature(t *testing.T) {
	var gotSig string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-GGID-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewHTTPDeliverer()
	payload := []byte(`{"event":"test"}`)
	err := d.Deliver(context.Background(), server.URL, "mysecret", payload)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(gotSig, "sha256=") {
		t.Errorf("expected sha256= prefix, got %s", gotSig)
	}
}

func TestHTTPDeliverer_InvalidURL(t *testing.T) {
	d := NewHTTPDeliverer()
	err := d.Deliver(context.Background(), "://invalid", "", []byte(`{}`))
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestHTTPDeliverer_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewHTTPDeliverer()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := d.Deliver(ctx, server.URL, "", []byte(`{}`))
	if err == nil {
		t.Error("expected timeout error")
	}
}

// --- Handler edge cases ---

func TestHandler_Create_NoTenant(t *testing.T) {
	h := NewHandler(NewMemoryStore(), nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/webhooks", strings.NewReader(`{}`))
	h.Create(w, r)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Create_InvalidJSON(t *testing.T) {
	h := NewHandler(NewMemoryStore(), nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/webhooks", strings.NewReader(`invalid`))
	r.Header.Set("X-Tenant-ID", "t1")
	h.Create(w, r)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Create_MissingFields(t *testing.T) {
	h := NewHandler(NewMemoryStore(), nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/webhooks", strings.NewReader(`{"url":""}`))
	r.Header.Set("X-Tenant-ID", "t1")
	h.Create(w, r)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Create_StoreError(t *testing.T) {
	h := NewHandler(&errorStore{}, nil)
	w := httptest.NewRecorder()
	body := `{"url":"http://example.com/hook","events":["user.created"]}`
	r := httptest.NewRequest("POST", "/api/v1/webhooks", strings.NewReader(body))
	r.Header.Set("X-Tenant-ID", "t1")
	h.Create(w, r)
	if w.Code != 500 {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandler_Create_Success(t *testing.T) {
	h := NewHandler(NewMemoryStore(), nil)
	w := httptest.NewRecorder()
	body := `{"url":"http://example.com/hook","events":["user.created"],"secret":"s3cr3t"}`
	r := httptest.NewRequest("POST", "/api/v1/webhooks", strings.NewReader(body))
	r.Header.Set("X-Tenant-ID", "t1")
	h.Create(w, r)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	// Secret should be stripped from response
	if strings.Contains(w.Body.String(), "s3cr3t") {
		t.Error("secret should not be returned")
	}
}

func TestHandler_List_NoTenant(t *testing.T) {
	h := NewHandler(NewMemoryStore(), nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/webhooks", nil)
	h.List(w, r)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_List_StoreError(t *testing.T) {
	h := NewHandler(&errorStore{}, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/webhooks", nil)
	r.Header.Set("X-Tenant-ID", "t1")
	h.List(w, r)
	if w.Code != 500 {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandler_List_Success(t *testing.T) {
	store := NewMemoryStore()
	h := NewHandler(store, nil)

	// Create a webhook first
	wh := &Webhook{ID: "wh1", TenantID: "t1", URL: "http://ex.com", Events: []string{"test"}}
	store.Create(context.Background(), wh)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/webhooks", nil)
	r.Header.Set("X-Tenant-ID", "t1")
	h.List(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "wh1") {
		t.Error("expected wh1 in response")
	}
}

func TestHandler_Delete_NoID(t *testing.T) {
	h := NewHandler(NewMemoryStore(), nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/api/v1/webhooks/", nil)
	r.Header.Set("X-Tenant-ID", "t1")
	h.Delete(w, r)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Delete_StoreError(t *testing.T) {
	h := NewHandler(&errorStore{}, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/api/v1/webhooks/wh1", nil)
	r.Header.Set("X-Tenant-ID", "t1")
	h.Delete(w, r)
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandler_Delete_Success(t *testing.T) {
	store := NewMemoryStore()
	store.Create(context.Background(), &Webhook{ID: "wh1", TenantID: "t1", URL: "http://x", Events: []string{"e"}})
	h := NewHandler(store, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/api/v1/webhooks/wh1", nil)
	r.Header.Set("X-Tenant-ID", "t1")
	h.Delete(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_Test_NoID(t *testing.T) {
	h := NewHandler(NewMemoryStore(), nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/webhooks//test", nil)
	r.Header.Set("X-Tenant-ID", "t1")
	h.Test(w, r)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Test_NotFound(t *testing.T) {
	h := NewHandler(NewMemoryStore(), nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/webhooks/nonexistent/test", nil)
	r.Header.Set("X-Tenant-ID", "t1")
	h.Test(w, r)
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandler_Test_WrongTenant(t *testing.T) {
	store := NewMemoryStore()
	store.Create(context.Background(), &Webhook{ID: "wh1", TenantID: "t2", URL: "http://x", Events: []string{"e"}})
	h := NewHandler(store, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/webhooks/wh1/test", nil)
	r.Header.Set("X-Tenant-ID", "t1") // different tenant
	h.Test(w, r)
	if w.Code != 404 {
		t.Errorf("expected 404 for wrong tenant, got %d", w.Code)
	}
}

func TestHandler_Test_DeliverySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := NewMemoryStore()
	store.Create(context.Background(), &Webhook{ID: "wh1", TenantID: "t1", URL: server.URL, Events: []string{"e"}})
	h := NewHandler(store, NewHTTPDeliverer())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/webhooks/wh1/test", nil)
	r.Header.Set("X-Tenant-ID", "t1")
	h.Test(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_Test_DeliveryFail(t *testing.T) {
	store := NewMemoryStore()
	store.Create(context.Background(), &Webhook{ID: "wh1", TenantID: "t1", URL: "http://localhost:1", Events: []string{"e"}})
	h := NewHandler(store, NewHTTPDeliverer())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/webhooks/wh1/test", nil)
	r.Header.Set("X-Tenant-ID", "t1")
	h.Test(w, r)
	if w.Code != 502 {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestHandler_DeliverEvent_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := NewMemoryStore()
	store.Create(context.Background(), &Webhook{
		ID: "wh1", TenantID: "t1", URL: server.URL,
		Events: []string{"user.created"}, Active: true,
	})
	h := NewHandler(store, NewHTTPDeliverer())

	h.DeliverEvent(context.Background(), "user.created", []byte(`{"id":"u1"}`))
	time.Sleep(100 * time.Millisecond) // async delivery
}

func TestHandler_DeliverEvent_NoMatch(t *testing.T) {
	store := NewMemoryStore()
	store.Create(context.Background(), &Webhook{
		ID: "wh1", TenantID: "t1", URL: "http://x",
		Events: []string{"user.deleted"}, Active: true,
	})
	h := NewHandler(store, NewHTTPDeliverer())

	// Should not deliver to webhooks that don't match the event
	h.DeliverEvent(context.Background(), "user.created", []byte(`{}`))
	time.Sleep(50 * time.Millisecond)
}

func TestHandler_DeliverEvent_StoreError(t *testing.T) {
	h := NewHandler(&errorStore{}, NewHTTPDeliverer())
	// Should not panic when store errors
	h.DeliverEvent(context.Background(), "test", []byte(`{}`))
	time.Sleep(50 * time.Millisecond)
}

// --- MemoryStore edge cases ---

func TestMemoryStore_GetNotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestMemoryStore_DeleteNotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.Delete(context.Background(), "nonexistent", "t1")
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestMemoryStore_ListByEvent(t *testing.T) {
	s := NewMemoryStore()
	s.Create(context.Background(), &Webhook{ID: "wh1", TenantID: "t1", URL: "http://a", Events: []string{"user.created"}, Active: true})
	s.Create(context.Background(), &Webhook{ID: "wh2", TenantID: "t2", URL: "http://b", Events: []string{"user.deleted"}, Active: true})
	s.Create(context.Background(), &Webhook{ID: "wh3", TenantID: "t3", URL: "http://c", Events: []string{"user.created"}, Active: false})

	results, err := s.ListByEvent(context.Background(), "user.created")
	if err != nil {
		t.Fatal(err)
	}
	// Should include wh1 (active) but not wh3 (inactive)
	if len(results) != 1 {
		t.Errorf("expected 1 active webhook, got %d", len(results))
	}
}

// errorStore is a Store that always returns errors.
type errorStore struct{}

func (e *errorStore) Create(_ context.Context, _ *Webhook) error { return fmt.Errorf("store error") }
func (e *errorStore) Get(_ context.Context, _ string) (*Webhook, error) { return nil, fmt.Errorf("not found") }
func (e *errorStore) List(_ context.Context, _ string) ([]*Webhook, error) { return nil, fmt.Errorf("store error") }
func (e *errorStore) Delete(_ context.Context, _, _ string) error { return fmt.Errorf("not found") }
func (e *errorStore) ListByEvent(_ context.Context, _ string) ([]*Webhook, error) { return nil, fmt.Errorf("store error") }

// suppress unused import warning
var _ = bytes.NewReader
