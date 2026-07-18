package shutdown

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// shuttingDown is set to true when shutdown begins.
// Health check endpoints should return 503 when this is true.
var shuttingDown atomic.Bool

// IsShuttingDown returns true during graceful shutdown.
func IsShuttingDown() bool {
	return shuttingDown.Load()
}

// Resources holds connections that need orderly cleanup.
type Resources struct {
	HTTPServer *http.Server
	Pool       *pgxpool.Pool
	Logger     *slog.Logger
	// Optional: NATS, Redis cleanup functions
	OnShutdown []func(ctx context.Context) error
}

// Manager handles ordered graceful shutdown on SIGTERM/SIGINT.
type Manager struct {
	resources *Resources
	timeout   time.Duration
}

// New creates a shutdown manager with 30s default timeout.
func New(res *Resources) *Manager {
	if res.Logger == nil {
		res.Logger = slog.Default()
	}
	return &Manager{
		resources: res,
		timeout:   30 * time.Second,
	}
}

// WithTimeout sets a custom shutdown timeout.
func (m *Manager) WithTimeout(d time.Duration) *Manager {
	m.timeout = d
	return m
}

// Wait blocks until SIGTERM or SIGINT is received, then executes shutdown.
func (m *Manager) Wait() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	sig := <-sigCh

	m.resources.Logger.Info("received shutdown signal, starting graceful shutdown",
		"signal", sig.String(), "timeout", m.timeout.String())

	m.execute()
}

// execute runs the ordered shutdown sequence.
func (m *Manager) execute() {
	shuttingDown.Store(true)

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	log := m.resources.Logger
	var step int

	// Step 1: Stop accepting new HTTP requests.
	if m.resources.HTTPServer != nil {
		step++
		log.Info("shutting down HTTP server", "step", step)
		if err := m.resources.HTTPServer.Shutdown(ctx); err != nil {
			log.Error("HTTP server shutdown error", "error", err)
		} else {
			log.Info("HTTP server stopped cleanly", "step", step)
		}
	}

	// Step 2: Run custom cleanup functions (NATS drain, Redis close, etc).
	for i, fn := range m.resources.OnShutdown {
		step++
		log.Info("running custom shutdown handler", "step", step, "index", i)
		if err := fn(ctx); err != nil {
			log.Error("custom shutdown handler error", "error", err)
		}
	}

	// Step 3: Close PG connection pool.
	if m.resources.Pool != nil {
		step++
		log.Info("closing database pool", "step", step)
		m.resources.Pool.Close()
		log.Info("database pool closed", "step", step)
	}

	log.Info("graceful shutdown complete", "total_steps", step)
}

// HealthCheckMiddleware returns 503 during shutdown, otherwise passes through.
func HealthCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsShuttingDown() {
			w.Header().Set("Connection", "close")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"shutting_down"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Register is a convenience function that creates a Manager and calls Wait().
func Register(srv *http.Server, pool *pgxpool.Pool, log *slog.Logger, extraCleanup ...func(ctx context.Context) error) {
	New(&Resources{
		HTTPServer: srv,
		Pool:       pool,
		Logger:     log,
		OnShutdown: extraCleanup,
	}).Wait()
}
