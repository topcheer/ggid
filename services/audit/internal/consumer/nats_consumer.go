// Package consumer provides the NATS JetStream consumer for audit events.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/ggid/ggid/services/audit/internal/repository"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Config holds NATS connection parameters.
type Config struct {
	URL        string
	StreamName string
	Subject    string
	Consumer   string
	MaxDeliver int
	BatchSize  int
}

// EventConsumer subscribes to audit events from NATS JetStream and persists them.
type EventConsumer struct {
	nc    *nats.Conn
	js    jetstream.JetStream
	repo  *repository.AuditRepository
	cfg   Config
	ctx   context.Context
	cancel context.CancelFunc
}

// New creates a new NATS consumer.
func New(parentCtx context.Context, cfg Config, repo *repository.AuditRepository) (*EventConsumer, error) {
	nc, err := nats.Connect(cfg.URL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	ctx, cancel := context.WithCancel(parentCtx)

	return &EventConsumer{
		nc:    nc,
		js:    js,
		repo:  repo,
		cfg:   cfg,
		ctx:   ctx,
		cancel: cancel,
	}, nil
}

// ensureStream creates or updates the audit events stream.
func (c *EventConsumer) ensureStream(ctx context.Context) error {
	_, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      c.cfg.StreamName,
		Subjects:  []string{c.cfg.Subject},
	Retention:  jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
		MaxAge:    72 * time.Hour,
		MaxBytes:  1 << 30, // 1 GB
		Replicas:  1,
	})
	return err
}

// Start begins consuming audit events from JetStream.
func (c *EventConsumer) Start() error {
	ctx := c.ctx

	if err := c.ensureStream(ctx); err != nil {
		return fmt.Errorf("ensure stream: %w", err)
	}

	cons, err := c.js.CreateOrUpdateConsumer(ctx, c.cfg.StreamName, jetstream.ConsumerConfig{
		Name:          c.cfg.Consumer,
		Durable:       c.cfg.Consumer,
		FilterSubject: c.cfg.Subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    c.cfg.MaxDeliver,
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	batchSize := c.cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}

	// Start consuming in a goroutine.
	go func() {
		log.Printf("Audit Consumer: consuming from %s (consumer=%s)", c.cfg.Subject, c.cfg.Consumer)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			batch, err := cons.FetchNoWait(batchSize)
			if err != nil {
				if err == jetstream.ErrNoMessages {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				log.Printf("Audit Consumer: fetch error: %v", err)
				time.Sleep(time.Second)
				continue
			}

			for msg := range batch.Messages() {
				if err := c.processMessage(ctx, msg); err != nil {
					log.Printf("Audit Consumer: process error: %v", err)
					msg.Nak()
				} else {
					msg.Ack()
				}
			}
		}
	}()

	return nil
}

// processMessage decodes a NATS message into an AuditEvent and persists it.
func (c *EventConsumer) processMessage(ctx context.Context, msg jetstream.Msg) error {
	var event domain.AuditEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		// If we can't decode, ack and drop — don't retry forever.
		log.Printf("Audit Consumer: failed to decode event: %v", err)
		return nil // returning nil so msg.Ack() is called
	}

	// Defensive: default ActorType to "user" if empty — DB enum rejects "".
	if event.ActorType == "" {
		event.ActorType = domain.ActorUser
	}

	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	if err := c.repo.Insert(ctx, &event); err != nil {
		return fmt.Errorf("persist event: %w", err)
	}

	return nil
}

// Close shuts down the consumer.
func (c *EventConsumer) Close() {
	c.cancel()
	c.nc.Close()
}
