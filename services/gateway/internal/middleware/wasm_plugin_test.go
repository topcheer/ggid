package middleware

import (
	"context"
	"net/http"
	"testing"
)

func TestWasmPluginHost_Empty(t *testing.T) {
	host := NewWasmPluginHost()
	defer host.Close(context.Background())

	plugins := host.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestWasmPluginHost_Execute_NotFound(t *testing.T) {
	host := NewWasmPluginHost()
	defer host.Close(context.Background())

	_, err := host.Execute(context.Background(), "nonexistent", PhaseRequest, PluginContext{
		Method: "GET",
		Path:   "/test",
	})
	if err == nil {
		t.Error("expected error for non-existent plugin")
	}
}

func TestWasmPluginHost_UnloadPlugin_NotFound(t *testing.T) {
	host := NewWasmPluginHost()
	defer host.Close(context.Background())

	err := host.UnloadPlugin(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for unloading non-existent plugin")
	}
}

func TestWasmPluginHost_LoadPlugin_MissingName(t *testing.T) {
	host := NewWasmPluginHost()
	defer host.Close(context.Background())

	err := host.LoadPlugin(context.Background(), WasmPluginConfig{
		WasmPath: "/tmp/test.wasm",
	})
	if err == nil {
		t.Error("expected error for missing plugin name")
	}
}

func TestWasmPluginHost_LoadPlugin_MissingPath(t *testing.T) {
	host := NewWasmPluginHost()
	defer host.Close(context.Background())

	err := host.LoadPlugin(context.Background(), WasmPluginConfig{
		Name: "test",
	})
	if err == nil {
		t.Error("expected error for missing wasm path")
	}
}

func TestWasmPluginHost_LoadPlugin_NonExistentFile(t *testing.T) {
	host := NewWasmPluginHost()
	defer host.Close(context.Background())

	err := host.LoadPlugin(context.Background(), WasmPluginConfig{
		Name:     "test",
		WasmPath: "/nonexistent/path/to/plugin.wasm",
	})
	if err == nil {
		t.Error("expected error for non-existent wasm file")
	}
}

func TestWasmMiddleware_NilHost(t *testing.T) {
	mw := WasmMiddleware(nil, nil)
	if mw == nil {
		t.Fatal("expected non-nil middleware")
	}
	// Should be a pass-through middleware
	called := false
	mw(testHandler(&called)).ServeHTTP(nil, nil)
	if !called {
		t.Error("expected handler to be called")
	}
}

func TestPluginPhase_Constants(t *testing.T) {
	if PhaseRequest != "request" {
		t.Errorf("expected 'request', got %q", PhaseRequest)
	}
	if PhaseResponse != "response" {
		t.Errorf("expected 'response', got %q", PhaseResponse)
	}
}

// testHandler creates a simple handler that sets a flag when called
func testHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
	})
}
