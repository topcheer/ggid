package webhooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateWebhook(t *testing.T) {
	h := NewHandler(NewMemoryStore(), nil)

	body := `{"url":"https://example.com/hook","events":["user.created","user.deleted"],"secret":"s3cr3t"}`
	req := httptest.NewRequest("POST", "/api/v1/webhooks", bodyReader(body))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var wh Webhook
	json.NewDecoder(w.Body).Decode(&wh)
	if wh.URL != "https://example.com/hook" {
		t.Errorf("expected URL, got %s", wh.URL)
	}
	if wh.Secret != "" {
		t.Error("secret should not be returned")
	}
	if !wh.Active {
		t.Error("expected active=true")
	}
}

func TestCreateWebhook_MissingFields(t *testing.T) {
	h := NewHandler(NewMemoryStore(), nil)
	req := httptest.NewRequest("POST", "/api/v1/webhooks", bodyReader(`{"url":""}`))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListWebhooks(t *testing.T) {
	store := NewMemoryStore()
	h := NewHandler(store, nil)

	// Create two webhooks
	ctx := context.Background()
	store.Create(ctx, &Webhook{ID: "w1", TenantID: "t1", URL: "http://a", Events: []string{"*"}, Active: true})
	store.Create(ctx, &Webhook{ID: "w2", TenantID: "t1", URL: "http://b", Events: []string{"user.created"}, Active: true})
	store.Create(ctx, &Webhook{ID: "w3", TenantID: "t2", URL: "http://c", Events: []string{"*"}, Active: true})

	req := httptest.NewRequest("GET", "/api/v1/webhooks", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Webhooks []*Webhook `json:"webhooks"`
		Total    int        `json:"total"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 2 {
		t.Errorf("expected 2 webhooks for tenant t1, got %d", resp.Total)
	}
}

func TestDeleteWebhook(t *testing.T) {
	store := NewMemoryStore()
	h := NewHandler(store, nil)

	ctx := context.Background()
	store.Create(ctx, &Webhook{ID: "w1", TenantID: "t1", URL: "http://a", Active: true})

	req := httptest.NewRequest("DELETE", "/api/v1/webhooks/w1", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify deleted
	wh, err := store.Get(ctx, "w1")
	if err == nil && wh != nil {
		t.Error("webhook should be deleted")
	}
}

func TestDeleteWebhook_WrongTenant(t *testing.T) {
	store := NewMemoryStore()
	h := NewHandler(store, nil)

	store.Create(context.Background(), &Webhook{ID: "w1", TenantID: "t1", URL: "http://a", Active: true})

	req := httptest.NewRequest("DELETE", "/api/v1/webhooks/w1", nil)
	req.Header.Set("X-Tenant-ID", "t2") // wrong tenant
	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404 for wrong tenant, got %d", w.Code)
	}
}

func TestTestWebhook_DeliverySuccess(t *testing.T) {
	delivered := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delivered = true
		w.WriteHeader(200)
	}))
	defer srv.Close()

	store := NewMemoryStore()
	store.Create(context.Background(), &Webhook{ID: "w1", TenantID: "t1", URL: srv.URL, Events: []string{"*"}, Active: true, Secret: "secret"})

	h := NewHandler(store, newTestDeliverer())
	req := httptest.NewRequest("POST", "/api/v1/webhooks/w1/test", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	w := httptest.NewRecorder()
	h.Test(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !delivered {
		t.Error("webhook should have been delivered")
	}
}

func TestListByEvent(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	store.Create(ctx, &Webhook{ID: "w1", TenantID: "t1", URL: "http://a", Events: []string{"user.created"}, Active: true})
	store.Create(ctx, &Webhook{ID: "w2", TenantID: "t1", URL: "http://b", Events: []string{"*"}, Active: true})
	store.Create(ctx, &Webhook{ID: "w3", TenantID: "t1", URL: "http://c", Events: []string{"user.deleted"}, Active: false})

	matches, err := store.ListByEvent(ctx, "user.created")
	if err != nil {
		t.Fatalf("ListByEvent error: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("expected 2 matching webhooks, got %d", len(matches))
	}
}

func TestDeliverEvent_Failed(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	// Register a webhook pointing to a non-existent server
	store.Create(ctx, &Webhook{ID: "w1", TenantID: "t1", URL: "http://127.0.0.1:1/hook", Events: []string{"*"}, Active: true})

	h := NewHandler(store, newTestDeliverer())
	// This should not block — failures are logged, not returned
	h.DeliverEvent(ctx, "user.created", []byte(`{"event":"user.created"}`))
}

func TestCreateNoTenantID(t *testing.T) {
	h := NewHandler(NewMemoryStore(), nil)
	req := httptest.NewRequest("POST", "/api/v1/webhooks", bodyReader(`{"url":"http://x","events":["*"]}`))
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 without tenant, got %d", w.Code)
	}
}

func bodyReader(s string) *stringReader {
	return &stringReader{s: s}
}

type stringReader struct {
	s   string
	pos int
}

func (r *stringReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.s) {
		return 0, errEOF
	}
	n = copy(p, r.s[r.pos:])
	r.pos += n
	return n, nil
}

var errEOF = errEOFType{}

type errEOFType struct{}

func (errEOFType) Error() string { return "EOF" }
