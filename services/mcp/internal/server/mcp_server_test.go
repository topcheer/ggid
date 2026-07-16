package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/services/mcp/internal/client"
)

// newTestServer creates an MCP server with a mock gateway backend.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	// Use a dummy base URL — we won't make real calls in unit tests.
	cli := client.New("http://localhost:0", "test-token")
	return New(cli)
}

func TestHandleMCP_MethodNotAllowed(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rr := httptest.NewRecorder()
	s.handleMCP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleMCP_ParseError(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte("invalid json")))
	rr := httptest.NewRecorder()
	s.handleMCP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 (JSON-RPC returns 200 with error), got %d", rr.Code)
	}
	var resp jsonRPCResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errObj, ok := resp.Error.(map[string]any)
	if !ok {
		t.Fatal("expected error in response")
	}
	if int(errObj["code"].(float64)) != -32700 {
		t.Errorf("expected error code -32700, got %v", errObj["code"])
	}
}

func TestHandleMCP_Initialize(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0", ID: 1, Method: "initialize",
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	s.handleMCP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp jsonRPCResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("expected result map")
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("unexpected protocolVersion: %v", result["protocolVersion"])
	}
	info, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("expected serverInfo map")
	}
	if info["name"] != "ggid-mcp" {
		t.Errorf("unexpected server name: %v", info["name"])
	}
}

func TestHandleMCP_ToolsList(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0", ID: 2, Method: "tools/list",
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	s.handleMCP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp jsonRPCResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("expected result map")
	}
	toolsList, ok := result["tools"].([]map[string]any)
	if !ok {
		// JSON unmarshal may produce []any
		if rawList, ok2 := result["tools"].([]any); ok2 {
			if len(rawList) < 5 {
				t.Errorf("expected at least 5 tools, got %d", len(rawList))
			}
			return
		}
		t.Fatal("expected tools list in result")
	}
	if len(toolsList) < 5 {
		t.Errorf("expected at least 5 tools, got %d", len(toolsList))
	}
}

func TestHandleMCP_MethodNotFound(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0", ID: 3, Method: "resources/list",
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	s.handleMCP(rr, req)
	var resp jsonRPCResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	errObj, ok := resp.Error.(map[string]any)
	if !ok {
		t.Fatal("expected error in response")
	}
	if int(errObj["code"].(float64)) != -32601 {
		t.Errorf("expected -32601, got %v", errObj["code"])
	}
}

func TestHandleMCP_Notification(t *testing.T) {
	s := newTestServer(t)
	// Notification = no ID field (nil), should be silently ignored
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "method": "initialized",
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	s.handleMCP(rr, req)
	// Notification should produce no response body
	if rr.Body.Len() > 0 {
		// Some implementations write nothing for notifications
		var resp jsonRPCResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err == nil {
			if resp.Result != nil || resp.Error != nil {
				t.Error("notification should not produce result or error")
			}
		}
	}
}

func TestHandleToolCall_UnknownTool(t *testing.T) {
	s := newTestServer(t)
	params, _ := json.Marshal(map[string]any{
		"name":      "nonexistent_tool",
		"arguments": map[string]any{},
	})
	body, _ := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0", ID: 4, Method: "tools/call", Params: params,
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	s.handleMCP(rr, req)
	var resp jsonRPCResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	errObj, ok := resp.Error.(map[string]any)
	if !ok {
		t.Fatal("expected error for unknown tool")
	}
	if int(errObj["code"].(float64)) != -32602 {
		t.Errorf("expected -32602, got %v", errObj["code"])
	}
}

func TestHandleToolCall_InvalidParams(t *testing.T) {
	s := newTestServer(t)
	// Send valid outer JSON but params as a string (not object) — json.Unmarshal will fail
	body, _ := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0", ID: 5, Method: "tools/call",
		Params: json.RawMessage(`"not-an-object"`),
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	s.handleMCP(rr, req)
	var resp jsonRPCResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	errObj, ok := resp.Error.(map[string]any)
	if !ok {
		t.Fatal("expected error for invalid params")
	}
	if int(errObj["code"].(float64)) != -32602 {
		t.Errorf("expected -32602, got %v", errObj["code"])
	}
}

func TestParseScopesFromEnv(t *testing.T) {
	t.Run("default admin scope", func(t *testing.T) {
		scopes := parseScopesFromEnv()
		if len(scopes) != 1 || scopes[0] != "admin" {
			t.Errorf("expected [admin], got %v", scopes)
		}
	})

	t.Run("custom scopes", func(t *testing.T) {
		t.Setenv("MCP_SCOPES", "users:read, roles:read , audit:read")
		scopes := parseScopesFromEnv()
		if len(scopes) != 3 {
			t.Fatalf("expected 3 scopes, got %d: %v", len(scopes), scopes)
		}
		if scopes[0] != "users:read" || scopes[1] != "roles:read" || scopes[2] != "audit:read" {
			t.Errorf("unexpected scopes: %v", scopes)
		}
	})

	t.Run("empty string after trim defaults to admin", func(t *testing.T) {
		t.Setenv("MCP_SCOPES", " , , ")
		scopes := parseScopesFromEnv()
		if len(scopes) != 1 || scopes[0] != "admin" {
			t.Errorf("expected [admin] fallback, got %v", scopes)
		}
	})
}

func TestHealthz(t *testing.T) {
	rr := httptest.NewRecorder()
	// healthz is registered as a closure in ListenAndServe, so test writeJSON directly
	writeJSON(rr, http.StatusOK, map[string]any{"status": "ok"})
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}
}
