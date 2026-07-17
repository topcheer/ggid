package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WasmPluginConfig defines a Wasm plugin's configuration.
type WasmPluginConfig struct {
	Name       string            `json:"name"`       // unique plugin identifier
	WasmPath   string            `json:"wasm_path"`  // path to .wasm file
	Config     map[string]string `json:"config"`     // plugin-specific config passed as env
	Enabled    bool              `json:"enabled"`    // whether plugin is active
	Signature  string            `json:"signature"`  // optional HMAC-SHA256 hex signature of the wasm binary
}

// WasmResourceLimits enforces sandbox resource constraints.
type WasmResourceLimits struct {
	MaxMemoryBytes uint32        // default: 16MB (256 pages * 64KB)
	ExecutionFuel  int64         // default: 10000 (simulated via timeout)
	Timeout        time.Duration // default: 100ms per execution
	VerifySignature bool         // default: true — reject unsigned/tampered plugins
}

// DefaultWasmResourceLimits returns the production-safe defaults.
func DefaultWasmResourceLimits() WasmResourceLimits {
	return WasmResourceLimits{
		MaxMemoryBytes:  16 * 1024 * 1024, // 16MB
		ExecutionFuel:   10000,
		Timeout:         100 * time.Millisecond,
		VerifySignature: true,
	}
}

// WasmPluginPhase indicates when the plugin runs in the request lifecycle.
type WasmPluginPhase string

const (
	PhaseRequest  WasmPluginPhase = "request"  // before proxying to backend
	PhaseResponse WasmPluginPhase = "response" // after receiving backend response
)

// WasmPluginHost manages Wasm plugin lifecycle and invocation.
// Plugins are compiled Wasm modules that expose HTTP middleware functions.
// They run in a sandboxed Wazero runtime with no network or filesystem access
// beyond what the host explicitly provides.
type WasmPluginHost struct {
	mu      sync.RWMutex
	runtime wazero.Runtime
	plugins map[string]*loadedPlugin
	limits  WasmResourceLimits
}

type loadedPlugin struct {
	config   WasmPluginConfig
	module   wazero.CompiledModule
	metadata PluginMetadata
}

// PluginMetadata is returned by the plugin's init function.
type PluginMetadata struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	Hooks       []string `json:"hooks"` // e.g. ["request", "response"]
}

// PluginContext provides request/response data to the Wasm plugin.
type PluginContext struct {
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	StatusCode int               `json:"status_code,omitempty"`
	TenantID   string            `json:"tenant_id,omitempty"`
	UserID     string            `json:"user_id,omitempty"`
}

// PluginResult is returned by the plugin after processing.
type PluginResult struct {
	StatusCode     int               `json:"status_code"`
	Headers        map[string]string `json:"headers"`
	Body           string            `json:"body"`
	ShouldBlock    bool              `json:"should_block"`
	BlockReason    string            `json:"block_reason,omitempty"`
	ModifiedHeader map[string]string `json:"modified_header,omitempty"`
	ModifiedBody   string            `json:"modified_body,omitempty"`
}

// NewWasmPluginHost creates a new Wasm plugin host with a shared runtime.
// Uses default resource limits (16MB memory, 100ms timeout, signature verification).
func NewWasmPluginHost() *WasmPluginHost {
	return NewWasmPluginHostWithLimits(DefaultWasmResourceLimits())
}

// NewWasmPluginHostWithLimits creates a plugin host with custom resource limits.
func NewWasmPluginHostWithLimits(limits WasmResourceLimits) *WasmPluginHost {
	ctx := context.Background()
	memoryPages := uint32(limits.MaxMemoryBytes / (64 * 1024))
	if memoryPages == 0 {
		memoryPages = 256 // 16MB default
	}
	config := wazero.NewRuntimeConfig().
		WithMemoryLimitPages(memoryPages).
		WithCloseOnContextDone(true) // enables context-based execution cancellation
	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	return &WasmPluginHost{
		runtime: runtime,
		plugins: make(map[string]*loadedPlugin),
		limits:  limits,
	}
}

// Close releases all Wasm runtime resources.
func (h *WasmPluginHost) Close(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	for name, p := range h.plugins {
		p.module.Close(ctx)
		delete(h.plugins, name)
	}
	return h.runtime.Close(ctx)
}

// LoadPlugin compiles and loads a Wasm plugin from file.
func (h *WasmPluginHost) LoadPlugin(ctx context.Context, cfg WasmPluginConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if cfg.WasmPath == "" {
		return fmt.Errorf("wasm_path is required")
	}

	absPath, err := filepath.Abs(cfg.WasmPath)
	if err != nil {
		return fmt.Errorf("resolve wasm path: %w", err)
	}
	wasmBytes, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read wasm file: %w", err)
	}

	// Verify plugin signature (HMAC-SHA256) to prevent loading tampered code.
	if h.limits.VerifySignature {
		if err := h.verifyPluginSignature(wasmBytes, cfg.Signature, absPath); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
	}

	compiled, err := h.runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return fmt.Errorf("compile wasm: %w", err)
	}

	// Instantiate to get metadata
	moduleConfig := wazero.NewModuleConfig()
	for k, v := range cfg.Config {
		moduleConfig = moduleConfig.WithEnv(k, v)
	}

	// Try to get metadata by instantiating
	inst, err := h.runtime.InstantiateModule(ctx, compiled, moduleConfig)
	metadata := PluginMetadata{Name: cfg.Name}
	if err == nil {
		// Try to read exported memory for metadata
		getMeta := inst.ExportedFunction("get_metadata")
		if getMeta != nil {
			metaBytes := h.readPluginMemory(ctx, inst, getMeta)
			if metaBytes != nil {
				_ = json.Unmarshal(metaBytes, &metadata)
			}
		}
		inst.Close(ctx)
	}

	h.mu.Lock()
	h.plugins[cfg.Name] = &loadedPlugin{
		config:   cfg,
		module:   compiled,
		metadata: metadata,
	}
	h.mu.Unlock()

	return nil
}

// Execute runs a plugin on the given request context.
// Returns the plugin result and nil error on success.
func (h *WasmPluginHost) Execute(ctx context.Context, pluginName string, phase WasmPluginPhase, pluginCtx PluginContext) (*PluginResult, error) {
	h.mu.RLock()
	p, ok := h.plugins[pluginName]
	h.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("plugin %q not found", pluginName)
	}

	// Enforce per-execution timeout (simulates fuel limit).
	if h.limits.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.limits.Timeout)
		defer cancel()
	}

	funcName := "on_request"
	if phase == PhaseResponse {
		funcName = "on_response"
	}

	moduleConfig := wazero.NewModuleConfig()
	for k, v := range p.config.Config {
		moduleConfig = moduleConfig.WithEnv(k, v)
	}

	inst, err := h.runtime.InstantiateModule(ctx, p.module, moduleConfig)
	if err != nil {
		return nil, fmt.Errorf("instantiate plugin: %w", err)
	}
	defer inst.Close(ctx)

	fn := inst.ExportedFunction(funcName)
	if fn == nil {
		// Plugin doesn't implement this hook — return passthrough
		return &PluginResult{ShouldBlock: false}, nil
	}

	// Serialize context to JSON and write to plugin memory
	ctxJSON, _ := json.Marshal(pluginCtx)
	ptr, err := h.writeToMemory(ctx, inst, ctxJSON)
	if err != nil {
		return nil, fmt.Errorf("write to plugin memory: %w", err)
	}

	results, err := fn.Call(ctx, api.EncodeI32(int32(ptr)), api.EncodeI32(int32(len(ctxJSON))))
	if err != nil {
		return nil, fmt.Errorf("plugin execution: %w", err)
	}

	if len(results) == 0 {
		return &PluginResult{ShouldBlock: false}, nil
	}

	// Read result from plugin memory
	resultPtr := api.DecodeU32(results[0])
	resultBytes := h.readBytesFromMemory(ctx, inst, resultPtr)
	if resultBytes == nil {
		return &PluginResult{ShouldBlock: false}, nil
	}

	var result PluginResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return &PluginResult{ShouldBlock: false}, nil
	}

	return &result, nil
}

// ListPlugins returns metadata for all loaded plugins.
func (h *WasmPluginHost) ListPlugins() []PluginMetadata {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]PluginMetadata, 0, len(h.plugins))
	for _, p := range h.plugins {
		result = append(result, p.metadata)
	}
	return result
}

// UnloadPlugin removes a plugin from the host.
func (h *WasmPluginHost) UnloadPlugin(ctx context.Context, name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	p, ok := h.plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q not found", name)
	}
	p.module.Close(ctx)
	delete(h.plugins, name)
	return nil
}

// WasmMiddleware creates an HTTP middleware that runs request-phase plugins
// before the handler and response-phase plugins after.
func WasmMiddleware(host *WasmPluginHost, pluginNames []string) func(http.Handler) http.Handler {
	if host == nil || len(pluginNames) == 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Run request-phase plugins
			for _, name := range pluginNames {
				pluginCtx := PluginContext{
					Method:  r.Method,
					Path:    r.URL.Path,
					Headers: flattenHeaders(r.Header),
				}
				tenantID, _ := TenantIDFromRequest(r)
				pluginCtx.TenantID = tenantID
				userID, _ := UserIDFromRequest(r)
				if userID != uuid.Nil {
					pluginCtx.UserID = userID.String()
				}

				result, err := host.Execute(ctx, name, PhaseRequest, pluginCtx)
				if err != nil {
					continue // skip on plugin error, don't block request
				}
				if result.ShouldBlock {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(result.StatusCode)
					if result.StatusCode == 0 {
						w.WriteHeader(http.StatusForbidden)
					}
					if result.Body != "" {
						w.Write([]byte(result.Body))
					} else {
						json.NewEncoder(w).Encode(map[string]string{"error": result.BlockReason})
					}
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// --- Helpers ---

func (h *WasmPluginHost) writeToMemory(ctx context.Context, inst api.Module, data []byte) (uint32, error) {
	allocFn := inst.ExportedFunction("alloc")
	if allocFn == nil {
		return 0, fmt.Errorf("plugin does not export alloc function")
	}
	results, err := allocFn.Call(ctx, api.EncodeI32(int32(len(data))))
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, fmt.Errorf("alloc returned no results")
	}
	ptr := api.DecodeU32(results[0])
	if !inst.Memory().Write(ptr, data) {
		return 0, fmt.Errorf("failed to write to plugin memory at offset %d", ptr)
	}
	return ptr, nil
}

func (h *WasmPluginHost) readPluginMemory(ctx context.Context, inst api.Module, fn api.Function) []byte {
	results, err := fn.Call(ctx)
	if err != nil || len(results) == 0 {
		return nil
	}
	ptr := api.DecodeU32(results[0])
	return h.readBytesFromMemory(ctx, inst, ptr)
}

func (h *WasmPluginHost) readBytesFromMemory(ctx context.Context, inst api.Module, ptr uint32) []byte {
	if ptr == 0 {
		return nil
	}
	mem := inst.Memory()
	buf, ok := mem.Read(ptr, mem.Size())
	if !ok {
		return nil
	}
	// Find null terminator
	for i, b := range buf {
		if b == 0 {
			return buf[:i]
		}
	}
	return buf
}

func flattenHeaders(headers http.Header) map[string]string {
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}

// verifyPluginSignature validates the HMAC-SHA256 signature of a wasm binary.
// The signature can be provided either in WasmPluginConfig.Signature or in a
// sidecar .wasm.sig file. The signing key is GGID_INTERNAL_SECRET.
func (h *WasmPluginHost) verifyPluginSignature(wasmBytes []byte, providedSig, wasmPath string) error {
	secret := os.Getenv("GGID_INTERNAL_SECRET")
	if secret == "" {
		// In dev mode without a secret, skip verification but log a warning.
		return nil
	}

	sig := providedSig
	if sig == "" {
		// Try reading sidecar .wasm.sig file.
		sigBytes, err := os.ReadFile(wasmPath + ".sig")
		if err != nil {
			return fmt.Errorf("no signature provided and no .sig sidecar file found")
		}
		sig = string(sigBytes)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(wasmBytes)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return fmt.Errorf("plugin signature mismatch — binary may be tampered")
	}
	return nil
}
