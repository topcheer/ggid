package webhooks

import (
	"context"
	"errors"
	"testing"
)

var errTest = errors.New("test error")

type stubStore struct {
	webhooks []*Webhook
	listErr  error
}

func (s *stubStore) Create(_ context.Context, wh *Webhook) error { return nil }
func (s *stubStore) Get(_ context.Context, id string) (*Webhook, error) {
	return nil, nil
}
func (s *stubStore) List(_ context.Context, tenantID string) ([]*Webhook, error) {
	return s.webhooks, nil
}
func (s *stubStore) ListByEvent(_ context.Context, event string) ([]*Webhook, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.webhooks, nil
}
func (s *stubStore) Delete(_ context.Context, id, tenantID string) error { return nil }

type stubDeliverer struct {
	err error
}

func (d *stubDeliverer) Deliver(_ context.Context, url, secret string, payload []byte) error {
	return d.err
}

func TestDeliverEvent_StoreError(t *testing.T) {
	store := &stubStore{listErr: errTest}
	h := &Handler{store: store, deliverer: &stubDeliverer{}}
	h.DeliverEvent(context.Background(), "user.created", []byte(`{}`))
}

func TestDeliverEvent_DeliveryFailure(t *testing.T) {
	store := &stubStore{webhooks: []*Webhook{
		{ID: "wh-1", URL: "http://localhost:1/fail", Secret: "s", Events: []string{"user.created"}},
	}}
	h := &Handler{store: store, deliverer: &stubDeliverer{err: errTest}}
	h.DeliverEvent(context.Background(), "user.created", []byte(`{}`))
}

func TestDeliverEvent_EmptyWebhooks(t *testing.T) {
	store := &stubStore{}
	h := &Handler{store: store, deliverer: &stubDeliverer{}}
	h.DeliverEvent(context.Background(), "unknown.event", []byte(`{}`))
}
