package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// usageRecord captures one request's metering data for batch insert.
type usageRecord struct {
	tenantID   string
	method     string
	path       string
	statusCode int
	latencyMs  int
}

// MeteringConfig controls the async batch writer behaviour.
type MeteringConfig struct {
	// BatchSize is the maximum records buffered before a flush.
	BatchSize int
	// FlushInterval is the maximum time between flushes.
	FlushInterval time.Duration
}

// DefaultMeteringConfig returns production-sensible defaults.
func DefaultMeteringConfig() MeteringConfig {
	return MeteringConfig{
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
	}
}

// apiUsageCollector buffers usage records and flushes them to the
// api_usage_log table via batch INSERT. It is safe for concurrent use.
type apiUsageCollector struct {
	pool   *pgxpool.Pool
	dbURL  string
	cfg    MeteringConfig
	ch     chan usageRecord
	done   chan struct{}
	once   sync.Once
}

var (
	meteringCollector *apiUsageCollector
	meteringMu        sync.Mutex
)

// getMeteringCollector lazily creates the singleton collector and its
// background flush goroutine.
func getMeteringCollector(dbURL string, cfg MeteringConfig) *apiUsageCollector {
	meteringMu.Lock()
	defer meteringMu.Unlock()
	if meteringCollector != nil {
		return meteringCollector
	}
	c := &apiUsageCollector{
		dbURL: dbURL,
		cfg:   cfg,
		ch:    make(chan usageRecord, cfg.BatchSize*4), // 4x buffer headroom
		done:  make(chan struct{}),
	}
	go c.flushLoop()
	meteringCollector = c
	return c
}

func (c *apiUsageCollector) ensurePool(ctx context.Context) error {
	if c.pool != nil {
		return nil
	}
	pool, err := pgxpool.New(ctx, c.dbURL)
	if err != nil {
		return err
	}
	c.pool = pool
	return nil
}

// flushLoop periodically drains the buffer and batch-inserts records.
func (c *apiUsageCollector) flushLoop() {
	ticker := time.NewTicker(c.cfg.FlushInterval)
	defer ticker.Stop()
	buf := make([]usageRecord, 0, c.cfg.BatchSize)
	for {
		select {
		case rec := <-c.ch:
			buf = append(buf, rec)
			if len(buf) >= c.cfg.BatchSize {
				c.insertBatch(buf)
				buf = buf[:0]
			}
		case <-ticker.C:
			// Drain all pending records.
			drained := false
			for {
				select {
				case rec := <-c.ch:
					buf = append(buf, rec)
					drained = true
				default:
					goto flush
				}
			}
		flush:
			if drained && len(buf) > 0 {
				c.insertBatch(buf)
				buf = buf[:0]
			}
		case <-c.done:
			// Final flush on shutdown.
			for {
				select {
				case rec := <-c.ch:
					buf = append(buf, rec)
				default:
					if len(buf) > 0 {
						c.insertBatch(buf)
					}
					return
				}
			}
		}
	}
}

// insertBatch performs a single COPY FROM INSERT for all buffered records.
func (c *apiUsageCollector) insertBatch(records []usageRecord) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.ensurePool(ctx); err != nil {
		return // pool init failed — records are lost (best-effort metering)
	}

	rows := make([][]any, len(records))
	for i, r := range records {
		rows[i] = []any{r.tenantID, r.method, r.path, r.statusCode, r.latencyMs}
	}

	_, _ = c.pool.CopyFrom(ctx,
		pgx.Identifier{"api_usage_log"},
		[]string{"tenant_id", "method", "path", "status_code", "latency_ms"},
		pgx.CopyFromRows(rows),
	)
}

// StopMetering gracefully flushes and stops the background collector.
func StopMetering() {
	meteringMu.Lock()
	defer meteringMu.Unlock()
	if meteringCollector == nil {
		return
	}
	close(meteringCollector.done)
	meteringCollector = nil
}

// meteringResponseWriter captures status code and measures latency.
type meteringResponseWriter struct {
	http.ResponseWriter
	status     int
	wroteHeader bool
}

func (w *meteringResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *meteringResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.status = http.StatusOK
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(b)
}

// APIMetering returns middleware that records per-tenant API count and
// latency into the api_usage_log table via an async batch writer.
//
// The middleware is non-blocking: each request enqueues a record onto a
// buffered channel; a background goroutine flushes batches periodically.
// Health-check paths (/healthz, /readyz) are excluded.
func APIMetering(dbURL string, cfg MeteringConfig) func(http.Handler) http.Handler {
	collector := getMeteringCollector(dbURL, cfg)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip health checks.
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			mw := &meteringResponseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(mw, r)

			latencyMs := int(time.Since(start).Milliseconds())

			// Resolve tenant ID (set by TenantResolver / InjectTenantContext).
			tenantID := ""
			if tid, ok := TenantIDFromRequest(r); ok {
				tenantID = tid
			}

			// Non-blocking enqueue — drop if buffer full (best-effort).
			select {
			case collector.ch <- usageRecord{
				tenantID:   tenantID,
				method:     r.Method,
				path:       r.URL.Path,
				statusCode: mw.status,
				latencyMs:  latencyMs,
			}:
			default:
				// Channel full — silently drop to avoid blocking request path.
			}
		})
	}
}
