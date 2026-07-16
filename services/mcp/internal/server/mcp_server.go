// Package server implements the MCP protocol (JSON-RPC 2.0 over SSE).
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ggid/ggid/services/mcp/internal/client"
	"github.com/ggid/ggid/services/mcp/internal/tools"
)

// Server is the MCP protocol server.
type Server struct {
	cli      *client.Client
	registry *tools.Registry
	scopes   []string
}

// New creates an MCP server with the given Gateway client.
func New(cli *client.Client) *Server {
	return &Server{
		cli:      cli,
		registry: tools.NewRegistry(),
		scopes:   parseScopesFromEnv(),
	}
}

// ListenAndServe starts the HTTP server with SSE endpoint for MCP.
func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	return http.ListenAndServe(addr, mux)
}

// handleMCP processes JSON-RPC 2.0 requests over HTTP POST.
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"jsonrpc": "2.0", "error": map[string]any{"code": -32600, "message": "only POST supported"},
		})
		return
	}

	var req jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONRPCError(w, nil, -32700, "parse error")
		return
	}

	// Handle notification (no id) — just ignore
	if req.ID == nil {
		return
	}

	switch req.Method {
	case "initialize":
		writeJSON(w, http.StatusOK, jsonRPCResponse{
			JSONRPC: "2.0", ID: req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{"name": "ggid-mcp", "version": "1.0.0"},
			},
		})
	case "tools/list":
		available := s.registry.FilterByScopes(s.scopes)
		toolList := make([]map[string]any, len(available))
		for i, t := range available {
			toolList[i] = map[string]any{
				"name": t.Name, "description": t.Description, "inputSchema": t.InputSchema,
			}
		}
		writeJSON(w, http.StatusOK, jsonRPCResponse{
			JSONRPC: "2.0", ID: req.ID,
			Result: map[string]any{"tools": toolList},
		})
	case "tools/call":
		s.handleToolCall(w, &req)
	default:
		writeJSONRPCError(w, req.ID, -32601, "method not found: "+req.Method)
	}
}

func (s *Server) handleToolCall(w http.ResponseWriter, req *jsonRPCRequest) {
	var params struct {
		Name string         `json:"name"`
		Args map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeJSONRPCError(w, req.ID, -32602, "invalid params")
		return
	}

	tool, ok := s.registry.Find(params.Name)
	if !ok {
		writeJSONRPCError(w, req.ID, -32602, "unknown tool: "+params.Name)
		return
	}

	// Scope check
	available := s.registry.FilterByScopes(s.scopes)
	found := false
	for _, t := range available {
		if t.Name == params.Name {
			found = true
			break
		}
	}
	if !found {
		writeJSONRPCError(w, req.ID, -32603, "insufficient scopes for tool: "+params.Name)
		return
	}

	result, err := tool.Handler(context.Background(), s.cli, params.Args)
	if err != nil {
		writeJSON(w, http.StatusOK, jsonRPCResponse{
			JSONRPC: "2.0", ID: req.ID,
			Result: map[string]any{
				"isError": true,
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("tool error: %v", err)},
				},
			},
		})
		return
	}

	content, _ := json.MarshalIndent(result, "", "  ")
	writeJSON(w, http.StatusOK, jsonRPCResponse{
		JSONRPC: "2.0", ID: req.ID,
		Result: map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(content)},
			},
		},
	})
}

// parseScopesFromEnv reads MCP_SCOPES env var (comma-separated).
// Defaults to ["admin"] for local development.
func parseScopesFromEnv() []string {
	s := os.Getenv("MCP_SCOPES")
	if s == "" {
		return []string{"admin"}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{"admin"}
	}
	return result
}

// JSON-RPC types

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeJSONRPCError(w http.ResponseWriter, id any, code int, msg string) {
	log.Printf("MCP error [%d]: %s", code, msg)
	writeJSON(w, http.StatusOK, jsonRPCResponse{
		JSONRPC: "2.0", ID: id,
		Error: map[string]any{"code": code, "message": msg},
	})
}
