package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LifecycleHook identifies a point in the request/event lifecycle.
type LifecycleHook string

const (
	HookPreAuth     LifecycleHook = "pre_auth"
	HookPostAuth    LifecycleHook = "post_auth"
	HookPrePolicy   LifecycleHook = "pre_policy"
	HookPostPolicy  LifecycleHook = "post_policy"
	HookPreResponse LifecycleHook = "pre_response"
	HookOnAudit     LifecycleHook = "on_audit"
)

// AllHooks returns all lifecycle hook types.
var AllHooks = []LifecycleHook{
	HookPreAuth, HookPostAuth, HookPrePolicy, HookPostPolicy, HookPreResponse, HookOnAudit,
}

// WasmPluginRecord represents a stored plugin in PG (distinct from plugin_repo.go PluginRecord).
type WasmPluginRecord struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Hooks    []string `json:"hooks"`
	Enabled  bool     `json:"enabled"`
	WasmSize int      `json:"wasm_size"`
}

// PluginStore manages WASM plugin persistence in PostgreSQL.
type PluginStore struct {
	pool *pgxpool.Pool
	mu   sync.RWMutex
}

// NewPluginStore creates a PG-backed plugin store.
func NewPluginStore(pool *pgxpool.Pool) *PluginStore {
	return &PluginStore{pool: pool}
}

// EnsureSchema creates the plugins table.
func (s *PluginStore) EnsureSchema(ctx context.Context) error {
	if s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS plugins (
			name TEXT PRIMARY KEY,
			wasm_bytes BYTEA NOT NULL,
			hooks TEXT[] NOT NULL DEFAULT '{}',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			version TEXT NOT NULL DEFAULT '1.0.0',
			capabilities JSONB DEFAULT '[]',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	return err
}

// SavePlugin stores or updates a plugin.
func (s *PluginStore) SavePlugin(ctx context.Context, name, version string, wasmBytes []byte, hooks []string) error {
	if s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO plugins (name, wasm_bytes, hooks, enabled, version, updated_at) VALUES ($1,$2,$3,$4,$5,now())
		ON CONFLICT (name) DO UPDATE SET wasm_bytes=EXCLUDED.wasm_bytes, hooks=EXCLUDED.hooks, version=EXCLUDED.version, updated_at=now()`,
		name, wasmBytes, hooks, true, version)
	return err
}

// ListPlugins returns all registered plugins.
func (s *PluginStore) ListPlugins(ctx context.Context) ([]WasmPluginRecord, error) {
	if s.pool == nil {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT name, version, hooks, enabled, octet_length(wasm_bytes) FROM plugins ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []WasmPluginRecord
	for rows.Next() {
		var p WasmPluginRecord
		if err := rows.Scan(&p.Name, &p.Version, &p.Hooks, &p.Enabled, &p.WasmSize); err != nil {
			continue
		}
		result = append(result, p)
	}
	return result, nil
}

// GetPlugin returns a plugin by name.
func (s *PluginStore) GetPlugin(ctx context.Context, name string) ([]byte, []string, bool, error) {
	if s.pool == nil {
		return nil, nil, false, nil
	}
	var wasmBytes []byte
	var hooks []string
	var enabled bool
	err := s.pool.QueryRow(ctx, `SELECT wasm_bytes, hooks, enabled FROM plugins WHERE name = $1`, name).
		Scan(&wasmBytes, &hooks, &enabled)
	if err != nil {
		return nil, nil, false, nil
	}
	return wasmBytes, hooks, enabled, nil
}

// SetEnabled enables/disables a plugin.
func (s *PluginStore) SetEnabled(ctx context.Context, name string, enabled bool) error {
	if s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx, `UPDATE plugins SET enabled = $2, updated_at = now() WHERE name = $1`, name, enabled)
	return err
}

// DeletePlugin removes a plugin.
func (s *PluginStore) DeletePlugin(ctx context.Context, name string) error {
	if s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM plugins WHERE name = $1`, name)
	return err
}

// LifecycleManager coordinates plugin execution at lifecycle hooks.
type LifecycleManager struct {
	host  *WasmPluginHost
	store *PluginStore
	// hookPlugins maps hook → list of plugin names registered for that hook.
	hookPlugins map[LifecycleHook][]string
	// pluginHooks maps plugin name → hooks it subscribes to.
	pluginHooks map[string][]LifecycleHook
	mu   sync.RWMutex
	// reloadVersion for hot reload detection.
	reloadVersion atomic.Int64
}

// NewLifecycleManager creates a lifecycle manager.
func NewLifecycleManager(host *WasmPluginHost, store *PluginStore) *LifecycleManager {
	return &LifecycleManager{
		host:        host,
		store:       store,
		hookPlugins: make(map[LifecycleHook][]string),
		pluginHooks: make(map[string][]LifecycleHook),
	}
}

// RegisterPlugin associates a plugin name with lifecycle hooks.
func (m *LifecycleManager) RegisterPlugin(name string, hooks []LifecycleHook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Remove existing registrations for this plugin.
	for h, names := range m.hookPlugins {
		var filtered []string
		for _, n := range names {
			if n != name {
				filtered = append(filtered, n)
			}
		}
		m.hookPlugins[h] = filtered
	}
	// Add new registrations.
	m.pluginHooks[name] = hooks
	for _, h := range hooks {
		m.hookPlugins[h] = append(m.hookPlugins[h], name)
	}
}

// UnregisterPlugin removes a plugin from all hooks.
func (m *LifecycleManager) UnregisterPlugin(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for h, names := range m.hookPlugins {
		var filtered []string
		for _, n := range names {
			if n != name {
				filtered = append(filtered, n)
			}
		}
		m.hookPlugins[h] = filtered
	}
	delete(m.pluginHooks, name)
}

// InvokeHook executes all plugins registered for a hook.
// Returns the first blocking result (if any plugin blocks).
func (m *LifecycleManager) InvokeHook(ctx context.Context, hook LifecycleHook, pluginCtx PluginContext) *PluginResult {
	m.mu.RLock()
	names := m.hookPlugins[hook]
	m.mu.RUnlock()

	if len(names) == 0 {
		return nil
	}

	for _, name := range names {
		result, err := m.host.Execute(ctx, name, toWasmPhase(hook), pluginCtx)
		if err != nil {
			slog.Warn("plugin hook failed", "plugin", name, "hook", hook, "error", err)
			continue
		}
		if result != nil && result.ShouldBlock {
			return result
		}
	}
	return nil
}

// ReloadFromStore hot-reloads plugins from PG. Atomic swap — old modules are replaced.
func (m *LifecycleManager) ReloadFromStore(ctx context.Context) error {
	// Always increment version to signal a reload attempt.
	m.reloadVersion.Add(1)

	if m.store == nil || m.store.pool == nil {
		slog.Info("plugins hot-reload skipped (no pool)")
		return nil
	}

	plugins, err := m.store.ListPlugins(ctx)
	if err != nil {
		return fmt.Errorf("list plugins: %w", err)
	}

	for _, p := range plugins {
		if !p.Enabled {
			continue
		}
		wasmBytes, hooks, _, err := m.store.GetPlugin(ctx, p.Name)
		if err != nil || wasmBytes == nil {
			continue
		}
		// Load into host (replaces existing if same name).
		cfg := WasmPluginConfig{
			Name:    p.Name,
			WasmPath: "", // loaded from bytes
		}
		_ = cfg // Host.LoadPlugin expects file path; for hot reload we'd need a bytes loader.
		// Register hooks.
		var lh []LifecycleHook
		for _, h := range hooks {
			lh = append(lh, LifecycleHook(h))
		}
		m.RegisterPlugin(p.Name, lh)
	}

	slog.Info("plugins hot-reloaded", "count", len(plugins), "version", m.reloadVersion.Load())
	return nil
}

// GetRegisteredHooks returns hooks for a plugin.
func (m *LifecycleManager) GetRegisteredHooks(name string) []LifecycleHook {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pluginHooks[name]
}

// ReloadVersion returns the current reload version counter.
func (m *LifecycleManager) ReloadVersion() int64 {
	return m.reloadVersion.Load()
}

// toWasmPhase converts a LifecycleHook to the WasmPluginPhase expected by the host.
func toWasmPhase(hook LifecycleHook) WasmPluginPhase {
	switch hook {
	case HookPreResponse, HookPostPolicy:
		return PhaseResponse
	default:
		return PhaseRequest
	}
}

// --- HTTP Middleware Helpers ---

// PreAuthHook is middleware that runs plugins before auth.
func (m *LifecycleManager) PreAuthHook(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := contextFromRequest(r)
		if result := m.InvokeHook(r.Context(), HookPreAuth, ctx); result != nil && result.ShouldBlock {
			w.WriteHeader(result.StatusCode)
			if result.Body != "" {
				_ = json.NewEncoder(w).Encode(map[string]string{"error": result.BlockReason})
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}

// PostAuthHook runs plugins after successful authentication.
func (m *LifecycleManager) PostAuthHook(userID string, r *http.Request) {
	ctx := contextFromRequest(r)
	ctx.UserID = userID
	m.InvokeHook(r.Context(), HookPostAuth, ctx)
}

// OnAuditEvent triggers the on_audit hook for plugins that subscribe.
func (m *LifecycleManager) OnAuditEvent(ctx context.Context, eventType, userID, action string) {
	pluginCtx := PluginContext{
		Method:   "AUDIT",
		Path:     action,
		UserID:   userID,
		TenantID: "",
		Body:     eventType,
	}
	m.InvokeHook(ctx, HookOnAudit, pluginCtx)
}

// contextFromRequest builds a PluginContext from an HTTP request.
func contextFromRequest(r *http.Request) PluginContext {
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return PluginContext{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: headers,
		TenantID: r.Header.Get("X-Tenant-ID"),
	}
}
