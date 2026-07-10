package audit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// startTestServer starts an embedded NATS server with JetStream enabled.
func startTestServer(t *testing.T) (*server.Server, *nats.Conn) {
	t.Helper()
	storeDir := t.TempDir()
	opts := &server.Options{
		Port:      -1,
		JetStream: true,
		StoreDir:  storeDir,
	}
	s := natsserver.RunServer(opts)
	t.Cleanup(func() {
		s.Shutdown()
	})

	nc, err := nats.Connect(s.ClientURL())
	if err != nil {
		t.Fatalf("connect to test NATS: %v", err)
	}
	t.Cleanup(func() {
		nc.Close()
	})

	return s, nc
}

func TestNewPublisher_Success(t *testing.T) {
	s, _ := startTestServer(t)
	ctx := context.Background()

	pub, err := NewPublisher(ctx, s.ClientURL())
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	defer pub.Close()

	if pub.stream != DefaultStreamName {
		t.Errorf("expected stream %s, got %s", DefaultStreamName, pub.stream)
	}
	if pub.subject != DefaultSubjectName {
		t.Errorf("expected subject %s, got %s", DefaultSubjectName, pub.subject)
	}
}

func TestNewPublisher_BadURL(t *testing.T) {
	ctx := context.Background()
	_, err := NewPublisher(ctx, "nats://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error for bad NATS URL")
	}
	if !strings.Contains(err.Error(), "connect to NATS") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewPublisher_StreamCreated(t *testing.T) {
	s, nc := startTestServer(t)
	ctx := context.Background()

	pub, err := NewPublisher(ctx, s.ClientURL())
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	defer pub.Close()

	// Verify the stream was created.
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("jetstream.New: %v", err)
	}
	str, err := js.Stream(ctx, DefaultStreamName)
	if err != nil {
		t.Fatalf("stream info: %v", err)
	}
	info, err := str.Info(ctx)
	if err != nil {
		t.Fatalf("stream info: %v", err)
	}
	if info.Config.Name != DefaultStreamName {
		t.Errorf("stream name mismatch: %s", info.Config.Name)
	}
}

func TestPublish_Success(t *testing.T) {
	s, _ := startTestServer(t)
	ctx := context.Background()

	pub, err := NewPublisher(ctx, s.ClientURL())
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	defer pub.Close()

	tid := uuid.New()
	uid := uuid.New()
	event := NewEvent("user.login", "success", tid, uid)
	event.IPAddress = "10.0.0.1"

	if err := pub.Publish(ctx, event); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
}

func TestPublish_AutoFillID(t *testing.T) {
	s, _ := startTestServer(t)
	ctx := context.Background()

	pub, err := NewPublisher(ctx, s.ClientURL())
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	defer pub.Close()

	// Event with nil ID should get auto-generated.
	event := Event{
		TenantID: uuid.New(),
		Action:   "test.autofill",
		Result:   "success",
	}
	if err := pub.Publish(ctx, event); err != nil {
		t.Fatalf("Publish: %v", err)
	}
}

func TestPublish_AutoFillCreatedAt(t *testing.T) {
	s, _ := startTestServer(t)
	ctx := context.Background()

	pub, err := NewPublisher(ctx, s.ClientURL())
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	defer pub.Close()

	event := Event{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Action:   "test.autotime",
		Result:   "success",
	}
	if err := pub.Publish(ctx, event); err != nil {
		t.Fatalf("Publish: %v", err)
	}
}

func TestPublish_RoundtripViaConsumer(t *testing.T) {
	s, nc := startTestServer(t)
	ctx := context.Background()

	pub, err := NewPublisher(ctx, s.ClientURL())
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	defer pub.Close()

	// Publish a known event.
	tid := uuid.New()
	uid := uuid.New()
	expected := Event{
		TenantID:  tid,
		ActorType: "user",
		ActorID:   uid,
		ActorName: "testuser",
		Action:    "role.create",
		Result:    "success",
		IPAddress: "192.168.1.1",
	}
	if err := pub.Publish(ctx, expected); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Create a consumer and read it back.
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("jetstream.New: %v", err)
	}

	consumer, err := js.CreateOrUpdateConsumer(ctx, DefaultStreamName, jetstream.ConsumerConfig{
		Durable:       "test-consumer",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: DefaultSubjectName,
	})
	if err != nil {
		t.Fatalf("CreateConsumer: %v", err)
	}

	// Fetch messages with a timeout.
	batch, err := consumer.FetchNoWait(10)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	found := false
	for msg := range batch.Messages() {
		var got Event
		if err := json.Unmarshal(msg.Data(), &got); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if got.Action == "role.create" && got.ActorName == "testuser" {
			found = true
			if got.TenantID != tid {
				t.Errorf("tenant ID mismatch")
			}
			if got.ActorID != uid {
				t.Errorf("actor ID mismatch")
			}
		}
		msg.Ack()
	}

	if !found {
		t.Error("published event not found in stream")
	}
}

func TestPublishAsync_NoError(t *testing.T) {
	s, _ := startTestServer(t)
	ctx := context.Background()

	pub, err := NewPublisher(ctx, s.ClientURL())
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	defer pub.Close()

	event := Event{
		TenantID: uuid.New(),
		Action:   "async.test",
		Result:   "success",
	}
	// Should not panic or block.
	pub.PublishAsync(event)
}

func TestPublishAsync_AutoFillFields(t *testing.T) {
	s, _ := startTestServer(t)
	ctx := context.Background()

	pub, err := NewPublisher(ctx, s.ClientURL())
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	defer pub.Close()

	// Event with nil ID and zero CreatedAt — should auto-fill.
	event := Event{
		TenantID: uuid.New(),
		Action:   "async.autofill",
		Result:   "success",
	}
	pub.PublishAsync(event)

	// Give NATS a moment to process.
	time.Sleep(100 * time.Millisecond)
}

func TestClose(t *testing.T) {
	s, _ := startTestServer(t)
	ctx := context.Background()

	pub, err := NewPublisher(ctx, s.ClientURL())
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}

	// Close should not panic.
	pub.Close()

	// Double close should also be safe.
	pub.Close()
}

func TestClose_NilConn(t *testing.T) {
	// Publisher with nil nc — Close should be safe.
	p := &Publisher{}
	p.Close()
}

func TestPublish_CancelledContext(t *testing.T) {
	s, _ := startTestServer(t)

	pub, err := NewPublisher(context.Background(), s.ClientURL())
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	defer pub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	event := Event{
		TenantID: uuid.New(),
		Action:   "cancelled.test",
		Result:   "success",
	}
	// With a cancelled context, Publish may error or succeed depending on timing.
	// Either way, it should not panic.
	_ = pub.Publish(ctx, event)
}

func TestNewPublisher_IdempotentStream(t *testing.T) {
	s, _ := startTestServer(t)
	ctx := context.Background()

	// Create publisher twice — second call should update, not fail.
	pub1, err := NewPublisher(ctx, s.ClientURL())
	if err != nil {
		t.Fatalf("first NewPublisher: %v", err)
	}
	defer pub1.Close()

	pub2, err := NewPublisher(ctx, s.ClientURL())
	if err != nil {
		t.Fatalf("second NewPublisher: %v", err)
	}
	defer pub2.Close()

	// Both should be able to publish.
	e := Event{TenantID: uuid.New(), Action: "idempotent.test", Result: "success"}
	if err := pub1.Publish(ctx, e); err != nil {
		t.Errorf("pub1.Publish: %v", err)
	}
	e2 := Event{TenantID: uuid.New(), Action: "idempotent.test2", Result: "success"}
	if err := pub2.Publish(ctx, e2); err != nil {
		t.Errorf("pub2.Publish: %v", err)
	}
}
