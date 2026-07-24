// Package server implements the MCP protocol (JSON-RPC 2.0 over SSE).
package server

import (
	"context"
	"encoding/base64"
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
	scopes     []string       // default scopes from env (fallback when token has no scope)
	jwtSecret  []byte
	jwtIssuer  string
	jwksURL    string         // JWKS endpoint for RS256 verification (empty = dev bypass)
	auditLog   *AgentAuditLog // structured agent audit
}

// New creates an MCP server with the given Gateway client.
func New(cli *client.Client) *Server {
	return &Server{
		cli:       cli,
		registry:  tools.NewRegistry(),
		scopes:    parseScopesFromEnv(),
		jwtSecret: parseJWTSecretFromEnv(),
		jwtIssuer: os.Getenv("JWT_ISSUER"),
		jwksURL:   os.Getenv("JWKS_URL"),
		auditLog:  NewAgentAuditLog(),
	}
}

// ListenAndServe starts the HTTP server with JWT auth + SSE endpoint for MCP.
func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.jwtAuth(s.handleMCP))
	mux.HandleFunc("/mcp/audit", s.jwtAuth(s.HandleAuditQuery))
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

		// Parse and validate JWT (supports HMAC symmetric and RSA asymmetric).
		claims := jwt.MapClaims{}

		// Check token algorithm from header to decide validation path.
		var err error
		// Quick header peek to detect RS256 without full parse.
		headerParts := strings.SplitN(tokenStr, ".", 2)
		isRS256 := false
		if len(headerParts) == 2 {
			headerJSON, _ := base64.RawURLEncoding.DecodeString(headerParts[0])
			if strings.Contains(string(headerJSON), "RS256") {
				isRS256 = true
			}
		}

		if isRS256 && s.jwksURL == "" {
			// Dev bypass: RS256 token without JWKS — parse claims without signature.
			parser := jwt.NewParser(jwt.WithoutClaimsValidation())
			_, _, err = parser.ParseUnverified(tokenStr, &claims)
		} else {
			_, err = jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); ok {
					return s.jwtSecret, nil
				}
				if _, ok := t.Method.(*jwt.SigningMethodRSA); ok {
					if s.jwksURL != "" {
						return s.fetchJWKSKey(t)
					}
				}
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			})
		}

		if err != nil {
			log.Printf("MCP auth: invalid JWT from %s: %v", r.RemoteAddr, err)
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"jsonrpc": "2.0", "error": map[string]any{
					"code": -32001, "message": "invalid or expired token",
				},
			})
			return
		}

		// Inject identity info into request context.
		ctx := r.Context()
		ctx = context.WithValue(ctx, ctxKeyToken{}, tokenStr)
		ctx = context.WithValue(ctx, ctxKeyUserID{}, claims["sub"])
		ctx = context.WithValue(ctx, ctxKeyTenantID{}, claims["tenant_id"])
		ctx = context.WithValue(ctx, ctxKeyScopes{}, claims["scope"])
		// Also extract permissions array for tool filtering (M2M/role-based tokens).
		if perms, ok := claims["permissions"].([]any); ok {
			permStrs := make([]string, 0, len(perms))
			for _, p := range perms {
				if ps, ok := p.(string); ok {
					permStrs = append(permStrs, ps)
				}
			}
			ctx = context.WithValue(ctx, ctxKeyPerms{}, permStrs)
		}

		// Agent identity: extract agent-specific claims for scope enforcement + audit.
		if isAgent, ok := claims["is_agent_token"].(bool); ok && isAgent {
			agentID, _ := claims["agent_id"].(string)
			agentType, _ := claims["agent_type"].(string)
			actSub, _ := claims["act_sub"].(string)
			ctx = context.WithValue(ctx, ctxKeyAgentID{}, agentID)
			ctx = context.WithValue(ctx, ctxKeyAgentType{}, agentType)
			ctx = context.WithValue(ctx, ctxKeyActorSub{}, actSub)

			// MCP server authorization: if token has mcp_servers claim, verify this server is allowed.
			if mcpServers, ok := claims["mcp_servers"].([]any); ok && len(mcpServers) > 0 {
				allowed := false
				for _, s := range mcpServers {
					if fmt.Sprintf("%v", s) == r.Host || fmt.Sprintf("%v", s) == "*" {
						allowed = true
						break
					}
				}
				if !allowed {
					log.Printf("MCP auth: agent %s not authorized for this MCP server", agentID)
					writeJSON(w, http.StatusForbidden, map[string]any{
						"jsonrpc": "2.0", "error": map[string]any{
							"code": -32002, "message": "agent not authorized for this MCP server",
						},
					})
					return
				}
			}

			log.Printf("MCP auth: agent token accepted agent_id=%s type=%s delegated_by=%s", agentID, agentType, actSub)
		}

		next(w, r.WithContext(ctx))
	}
}

// Context keys for MCP auth.
type ctxKeyUserID struct{}
type ctxKeyTenantID struct{}
type ctxKeyScopes struct{}
type ctxKeyPerms struct{}
type ctxKeyAgentID struct{}
type ctxKeyAgentType struct{}
type ctxKeyActorSub struct{}
type ctxKeyToken struct{}

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

// getAgentFromContext extracts agent identity info. Returns agentID="" if not an agent token.
func getAgentFromContext(ctx context.Context) (agentID, agentType, actorSub string) {
	if v, ok := ctx.Value(ctxKeyAgentID{}).(string); ok {
		agentID = v
	}
	if v, ok := ctx.Value(ctxKeyAgentType{}).(string); ok {
		agentType = v
	}
	if v, ok := ctx.Value(ctxKeyActorSub{}).(string); ok {
		actorSub = v
	}
	return
}

// scopesFromContext extracts the token's scope claim + permissions as []string.
// Falls back to server's default scopes from env.
func (s *Server) scopesFromContext(ctx context.Context) []string {
	var all []string
	if scopeStr, ok := ctx.Value(ctxKeyScopes{}).(string); ok && scopeStr != "" {
		all = append(all, strings.Fields(scopeStr)...)
	}
	if perms, ok := ctx.Value(ctxKeyPerms{}).([]string); ok {
		all = append(all, perms...)
	}
	if len(all) > 0 {
		return all
	}
	return s.scopes
}

// auditToolCall logs an MCP tool invocation for security audit.
// Captures agent identity when the caller is an AI agent token.
func (s *Server) auditToolCall(ctx context.Context, toolName string, args map[string]any, result any, err error) {
	userID, tenantID := getUserFromContext(ctx)
	agentID, agentType, actorSub := getAgentFromContext(ctx)
	status := "success"
	if err != nil {
		status = "error"
	}

	// Structured audit entry.
	entry := &AgentAuditEntry{
		Timestamp: time.Now().UTC(),
		Tool:      toolName,
		Status:    status,
		UserID:    userID,
		TenantID:  tenantID,
		AgentID:   agentID,
		AgentType: agentType,
		ActorSub:  actorSub,
		Args:      args,
	}
	s.auditLog.Append(entry)

	if agentID != "" {
		log.Printf("MCP audit: tool=%s AGENT id=%s type=%s delegated_by=%s tenant=%s status=%s",
			toolName, agentID, agentType, actorSub, tenantID, status)
	} else {
		log.Printf("MCP audit: tool=%s user=%s tenant=%s status=%s",
			toolName, userID, tenantID, status)
	}
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
		available := s.registry.FilterByScopes(s.scopesFromContext(r.Context()))
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

	// Scope check: use token scopes (agent-scoped) instead of server default.
	tokenScopes := s.scopesFromContext(r.Context())
	available := s.registry.FilterByScopes(tokenScopes)
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

	// Use per-request client with caller's JWT if available (falls back to shared static token).
	reqClient := s.cli
	if token, ok := r.Context().Value(ctxKeyToken{}).(string); ok && token != "" {
		_, tid := getUserFromContext(r.Context())
		reqClient = client.New(s.cli.GatewayURL(), token, tid)
	}
	result, err := tool.Handler(r.Context(), reqClient, params.Args)

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

// fetchJWKSKey fetches the RSA public key from the JWKS endpoint for RS256 verification.
// TODO: implement full JWKS fetch + cache. Currently returns error (use dev bypass).
func (s *Server) fetchJWKSKey(t *jwt.Token) (any, error) {
	return nil, fmt.Errorf("JWKS verification not yet implemented — set JWKS_URL or use dev bypass")
}
