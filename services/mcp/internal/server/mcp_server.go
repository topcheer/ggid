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
	"time"

	"github.com/ggid/ggid/services/mcp/internal/client"
	"github.com/ggid/ggid/services/mcp/internal/tools"
	"github.com/golang-jwt/jwt/v5"
)

// Server is the MCP protocol server.
type Server struct {
	cli        *client.Client
	registry   *tools.Registry
	scopes     []string
	jwtSecret  []byte
	jwtIssuer  string
}

// New creates an MCP server with the given Gateway client.
func New(cli *client.Client) *Server {
	return &Server{
		cli:       cli,
		registry:  tools.NewRegistry(),
		scopes:    parseScopesFromEnv(),
		jwtSecret: parseJWTSecretFromEnv(),
		jwtIssuer: os.Getenv("JWT_ISSUER"),
	}
}

// ListenAndServe starts the HTTP server with JWT auth + SSE endpoint for MCP.
func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.jwtAuth(s.handleMCP))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return srv.ListenAndServe()
}

// jwtAuth middleware validates a Bearer JWT token before allowing MCP requests.
// In dev mode (no JWT_SECRET set), auth is optional but logged.
func (s *Server) jwtAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for healthz (handled separately).
		if r.URL.Path == "/healthz" {
			next(w, r)
			return
		}

		// Extract Bearer token.
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			// In dev mode without secret configured, allow without JWT (with warning).
			if len(s.jwtSecret) == 0 {
				log.Printf("MCP WARNING: no JWT_SECRET configured — allowing unauthenticated request from %s", r.RemoteAddr)
				next(w, r)
				return
			}
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"jsonrpc": "2.0", "error": map[string]any{
					"code": -32001, "message": "authorization required: Bearer token expected",
				},
			})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// If no secret configured, skip validation (dev mode).
		if len(s.jwtSecret) == 0 {
			log.Printf("MCP WARNING: no JWT_SECRET — token accepted without validation (dev mode)")
			next(w, r)
			return
		}

		// Parse and validate JWT.
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return s.jwtSecret, nil
		})

		if err != nil {
			log.Printf("MCP auth: invalid JWT from %s: %v", r.RemoteAddr, err)
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"jsonrpc": "2.0", "error": map[string]any{
					"code": -32001, "message": "invalid or expired token",
				},
			})
			return
		}

		// Inject user info into request context for audit logging.
		ctx := r.Context()
		ctx = context.WithValue(ctx, ctxKeyUserID{}, claims["sub"])
		ctx = context.WithValue(ctx, ctxKeyTenantID{}, claims["tenant_id"])
		ctx = context.WithValue(ctx, ctxKeyScopes{}, claims["scope"])

		next(w, r.WithContext(ctx))
	}
}

// Context keys for MCP auth.
type ctxKeyUserID struct{}
type ctxKeyTenantID struct{}
type ctxKeyScopes struct{}

// getUserFromContext extracts authenticated user info from request context.
func getUserFromContext(ctx context.Context) (userID, tenantID string) {
	if v, ok := ctx.Value(ctxKeyUserID{}).(string); ok {
		userID = v
	}
	if v, ok := ctx.Value(ctxKeyTenantID{}).(string); ok {
		tenantID = v
	}
	return
}

// auditToolCall logs an MCP tool invocation for security audit.
func (s *Server) auditToolCall(ctx context.Context, toolName string, args map[string]any, result any, err error) {
	userID, tenantID := getUserFromContext(ctx)
	status := "success"
	if err != nil {
		status = "error"
	}
	log.Printf("MCP audit: tool=%s user=%s tenant=%s status=%s args=%v result_len=%d",
		toolName, userID, tenantID, status, args, len(fmt.Sprintf("%v", result)))
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
		s.handleToolCall(w, r, &req)
	default:
		writeJSONRPCError(w, req.ID, -32601, "method not found: "+req.Method)
	}
}

func (s *Server) handleToolCall(w http.ResponseWriter, r *http.Request, req *jsonRPCRequest) {
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

	result, err := tool.Handler(r.Context(), s.cli, params.Args)

	// Audit: log every tool call with user identity.
	s.auditToolCall(r.Context(), params.Name, params.Args, result, err)

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

// parseJWTSecretFromEnv reads JWT_SECRET or GGID_INTERNAL_SECRET for token validation.
func parseJWTSecretFromEnv() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = os.Getenv("GGID_INTERNAL_SECRET")
	}
	if secret == "" {
		if os.Getenv("GGID_ENV") == "production" {
			log.Fatal("MCP: JWT_SECRET or GGID_INTERNAL_SECRET must be set in production")
		}
		return nil // dev mode — no auth enforcement
	}
	return []byte(secret)
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

// suppress unused import in case time isn't used in future refactor
var _ time.Duration
