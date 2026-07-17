package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/audit/internal/repository"
)

// IntelCollector periodically polls external threat intel sources and cleans expired indicators.
type IntelCollector struct {
	repo   *repository.ThreatIntelRepository
	client *http.Client
	stop   chan struct{}
}

// NewIntelCollector creates a collector with the given repo.
func NewIntelCollector(repo *repository.ThreatIntelRepository) *IntelCollector {
	return &IntelCollector{
		repo:   repo,
		client: &http.Client{Timeout: 15 * time.Second},
		stop:   make(chan struct{}),
	}
}

// Run starts the collection loop. Call in a goroutine.
func (c *IntelCollector) Run(ctx context.Context, pollInterval time.Duration) {
	if c.repo == nil {
		return
	}
	if pollInterval <= 0 {
		pollInterval = 10 * time.Minute
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	c.collect(ctx)
	c.cleanup(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stop:
			return
		case <-ticker.C:
			c.collect(ctx)
			c.cleanup(ctx)
		}
	}
}

func (c *IntelCollector) collect(ctx context.Context) {
	sources, err := c.repo.ListEnabledSources(ctx)
	if err != nil {
		slog.Error("threat intel collector: list sources", "error", err)
		return
	}

	for _, src := range sources {
		if err := ctx.Err(); err != nil {
			return
		}

		adapter := getAdapter(src.SourceType)
		switch a := adapter.(type) {
		case *AbuseIPDBAdapter:
			a.APIKey = src.APIKeyRef
			a.SourceID = src.ID
			a.TenantID = src.TenantID
			a.Endpoint = src.APIEndpoint
		case *OTXAdapter:
			a.APIKey = src.APIKeyRef
			a.SourceID = src.ID
			a.TenantID = src.TenantID
			a.Endpoint = src.APIEndpoint
		}

		fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		indicators, err := adapter.Fetch(fetchCtx, c.client)
		cancel()

		if err != nil {
			slog.Warn("threat intel fetch failed", "source", src.Name, "error", err)
			continue
		}

		for _, ind := range indicators {
			c.repo.UpsertIndicator(ctx, &ind)
		}
		c.repo.UpdateLastPoll(ctx, src.ID)
		slog.Info("threat intel collected", "source", src.Name, "count", len(indicators))
	}
}

func (c *IntelCollector) cleanup(ctx context.Context) {
	deleted, err := c.repo.DeleteExpired(ctx)
	if err != nil {
		slog.Warn("threat intel cleanup failed", "error", err)
		return
	}
	if deleted > 0 {
		slog.Info("threat intel expired indicators cleaned", "count", deleted)
	}
}

func (c *IntelCollector) Stop() {
	close(c.stop)
}
