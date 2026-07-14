# AI Agent Identity Delegation

Delegation chain construction, max depth enforcement, scope narrowing, token exchange between agents, audit trail, and revocation propagation.

## Overview

AI agents act on behalf of users or other agents. GGID's delegation chain ensures every action is traceable, scope-limited, and revocable.

## Delegation Chain

```
User (sub: user-uuid)
  → delegates to Agent A (act.sub: agent-a)
    → delegates to Agent B (act.sub: agent-b, act.act.sub: agent-a)
      → delegates to Agent C (max depth 3 — STOP)
```

### Token Structure

```json
// Original user token
{
  "sub": "user-uuid",
  "scope": "users:read users:write",
  "aud": "identity-svc"
}

// Agent A token (delegated by user)
{
  "sub": "user-uuid",
  "act": {
    "sub": "agent-a-uuid",
    "scope": "users:read"
  },
  "scope": "users:read",
  "aud": "identity-svc",
  "max_delegation_depth": 3
}

// Agent B token (delegated by Agent A)
{
  "sub": "user-uuid",
  "act": {
    "sub": "agent-b-uuid",
    "scope": "users:read",
    "act": {
      "sub": "agent-a-uuid",
      "scope": "users:read"
    }
  },
  "scope": "users:read",
  "max_delegation_depth": 3
}
```

## Depth Enforcement

```go
const MaxDelegationDepth = 3

func getDelegationDepth(claims jwt.MapClaims) int {
    depth := 0
    act := claims["act"]
    for act != nil {
        depth++
        if m, ok := act.(map[string]interface{}); ok {
            act = m["act"]
        } else {
            break
        }
    }
    return depth
}

func ValidateDelegation(claims jwt.MapClaims) error {
    depth := getDelegationDepth(claims)
    max := 3
    if d, ok := claims["max_delegation_depth"]; ok {
        max = int(d.(float64))
    }
    if depth >= max {
        return ErrMaxDelegationDepth
    }
    return nil
}
```

## Scope Narrowing

Each delegation can only narrow — never expand:

```go
func ExchangeAgentToken(ctx context.Context, subjectToken string, agentID string, requestedScope []string) (string, error) {
    claims, err := verifyToken(subjectToken)
    if err != nil { return "", err }

    subjectScope := strings.Fields(claims["scope"].(string))

    // Requested scope must be subset of subject scope
    for _, s := range requestedScope {
        if !containsScope(subjectScope, s) {
            return "", ErrScopeEscalation
        }
    }

    // Build delegated token
    newClaims := jwt.MapClaims{
        "sub":   claims["sub"],       // Original user
        "scope": strings.Join(requestedScope, " "),
        "act": map[string]interface{}{
            "sub":   agentID,
            "scope": strings.Join(requestedScope, " "),
        },
        "aud": claims["aud"],
        "exp": time.Now().Add(15 * time.Minute).Unix(),
        "max_delegation_depth": 3,
    }

    // Copy existing act chain
    if existingAct, ok := claims["act"]; ok {
        newClaims["act"].(map[string]interface{})["act"] = existingAct
    }

    return signToken(newClaims)
}
```

## Agent Registration

```bash
POST /api/v1/agents/register
{
  "agent_name": "Data Processor Bot",
  "owner_user_id": "user-uuid",
  "allowed_scopes": ["users:read", "data:read"],
  "mcp_servers": ["https://mcp.corp.com"],
  "max_delegation_depth": 2
}
# → {agent_id: "agent-uuid", agent_secret: "..."}
```

## Token Exchange

```bash
POST /api/v1/oauth/token
grant_type=urn:ietf:params:oauth:grant-type:token-exchange
&subject_token=<user-or-agent-token>
&subject_token_type=urn:ietf:params:oauth:token-type:access_token
&requested_token_type=urn:ietf:params:oauth:token-type:access_token
&audience=identity-svc
&scope=users:read
&may_delegate=true
# → Returns delegated agent token with act chain
```

## Audit Trail

Every agent action is fully traceable:

```json
{
  "event": "agent.action",
  "original_user": "user-uuid",
  "delegation_chain": ["agent-a", "agent-b"],
  "agent_id": "agent-b",
  "action": "users.read",
  "resource": "uuid-of-user-read",
  "scope_used": "users:read",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

## Revocation Propagation

When a user revokes an agent:

```bash
DELETE /api/v1/agents/{agent_id}/delegation
# → All tokens in delegation chain revoked
```

```go
func RevokeAgentDelegation(agentID string) error {
    // 1. Add agent to blacklist
    redis.SAdd("agent:blacklist", agentID)

    // 2. Find all tokens with this agent in act chain
    jtis := store.GetTokensByAgent(agentID)
    for _, jti := range jtis {
        redis.Set("jwt:blacklist:"+jti, "1", 15*time.Minute)
    }

    // 3. Prevent new token exchanges for this agent
    store.RevokeAgent(agentID)

    // 4. Audit
    audit.Log("agent.delegation_revoked", map[string]interface{}{
        "agent_id": agentID,
        "tokens_revoked": len(jtis),
    })

    return nil
}
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Max depth exceeded | Any → blocked (correct behavior) |
| Scope escalation attempt | Any → blocked + security alert |
| Agent token without delegation chain | Any → misconfigured agent |
| Revoked agent still active | Any → blacklist cache miss |
| Delegation chain >3 levels | Any → potential abuse |

## See Also

- [AI Agent Identity](ai-agent-identity.md)
- [Token Exchange Patterns](token-exchange-patterns.md)
- [JWT Security Best Practices](jwt-security-best-practices.md)
- [Delegated Administration](delegated-administration.md)
