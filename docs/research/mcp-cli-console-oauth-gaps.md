# Major Productization Gaps — MCP, CLI, Console OAuth

## Date: 2026-07-16

## Gap 1: MCP Server (P0 — Highest Value)
**Problem:** No runnable MCP server. OAuth service has AI Agent Identity claims skeleton (server.go:1603-1718) but no actual MCP tool implementations.

**Required:**
- MCP server exposing GGID management operations as tools (create user, assign role, list policies, check permissions, query audit events)
- OAuth 2.1 PKCE flow for MCP client authentication
- Scope-based fine-grained permissions (users:read, users:write, roles:manage, etc.)
- Integration with existing Gateway JWT verification

**Priority: 1** — LLM-managed IAM is a key differentiator.

## Gap 2: Console OAuth Self-Registration (P1)
**Problem:** Console uses username/password → JWT directly. Should register as GGID OAuth application using authorization code flow.

**Required:**
- Console registers as OAuth client on first startup (client_id stored in env/config)
- Login flow: redirect to GGID authorize → callback → token exchange
- Refresh token rotation
- Works in all deployments (k3s, all-in-one, docker-compose)

**Priority: 2** — Security best practice.

## Gap 3: CLI Tool (P2)
**Problem:** No command-line tool for GGID management.

**Required:**
- `ggid-cli` in cmd/ggid-cli/
- Commands: users, roles, policies, audit, tenants, tokens
- Uses SDK (sdk/go) for API calls
- Config file for credentials (~/.ggid/config.yaml)
- OAuth client credentials flow for machine-to-machine auth

**Priority: 3** — DevOps efficiency.
