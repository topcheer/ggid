# Agentic AI Governance

Agent identity lifecycle, privilege drift detection, shadow agent discovery, delegation chain trust, MCP tool scoping, runtime access review, OWASP Agentic alignment, CSA framework, consent for autonomous agents, and audit requirements.

## Agent Identity Lifecycle

```
Register → Provision → Monitor → Review → Decommission
    │          │          │          │          │
    ▼          ▼          ▼          ▼          ▼
 Issue ID  Assign scopes Track actions Periodic   Revoke all
 + secret  + MCP tools  + decisions   review      tokens+scopes
```

### Registration

```bash
POST /api/v1/agents/register
{
  "agent_name": "Data Processor Bot",
  "owner_user_id": "user-uuid",
  "allowed_scopes": ["users:read", "data:read"],
  "mcp_servers": ["https://mcp.corp.com"],
  "max_delegation_depth": 2,
  "auto_decommission_days": 90
}
# → {agent_id, agent_secret}
```

## Privilege Drift Detection

```go
func detectPrivilegeDrift(agentID string) []Drift {
    baseline := getAgentBaseline(agentID)   // Registered scopes
    current := getAgentActiveScopes(agentID) // Scopes in active tokens
    
    drifts := []Drift{}
    
    // Scope expansion
    for _, s := range current {
        if !contains(baseline.Scopes, s) {
            drifts = append(drifts, Drift{Type: "scope_added", Scope: s})
        }
    }
    
    // MCP tool access beyond registered
    for _, tool := range getAgentMCPUsage(agentID) {
        if !contains(baseline.MCPTools, tool) {
            drifts = append(drifts, Drift{Type: "unauthorized_tool", Tool: tool})
        }
    }
    
    return drifts
}
```

## Shadow Agent Discovery

Detect agents not registered in GGID but active in the environment:

```bash
GET /api/v1/admin/agents/shadow
# → Scans for:
#   - API keys not in inventory
#   - Service accounts with no owner
#   - Tokens with non-human patterns but no agent registration
#   - MCP connections from unknown clients
```

## Delegation Chain Trust

```
User (trust: 1.0)
  → Agent A (trust: 0.8, scope narrowed)
    → Agent B (trust: 0.6, scope further narrowed)
      → Max depth 3 → STOP
```

Trust degrades with each delegation level. Below threshold 0.3 → require human approval.

## MCP Tool Access Scoping

```yaml
agent_mcp_scoping:
  agent: "data-processor"
  allowed_tools:
    - "read_user_profile"     # Read-only
    - "read_audit_events"     # Read-only
  denied_tools:
    - "delete_user"           # Never
    - "assign_role"           # Never
    - "modify_policy"         # Never
  rate_limit:
    tool_calls_per_minute: 30
    tool_calls_per_hour: 500
```

## Runtime Access Review

| Check | Frequency | Action on Fail |
|-------|-----------|----------------|
| Agent still has owner | Daily | Quarantine if orphaned |
| Scopes still needed | Weekly | Auto-revoke unused >30d |
| MCP tool usage within bounds | Real-time | Block + alert on violation |
| Delegation depth | Per request | Reject if > max |
| Token age | Per request | Reject if > 15 min |

## OWASP Top 10 for Agentic Apps (Alignment)

| Risk | GGID Mitigation |
|------|----------------|
| Agent impersonation | mTLS + DPoP binding |
| Unauthorized tool access | MCP scoping per agent |
| Privilege escalation | Scope narrowing at delegation |
| Prompt injection | Input sanitization + output validation |
| Data exfiltration | DLP egress filtering per agent |
| Audit gap | Every agent action logged with chain |
| Unbounded autonomy | Rate limits + human approval for destructive |
| Agent persistence after decommission | Token revocation + blacklist |
| Supply chain (MCP server) | MCP server allowlist |
| Model hallucination → action | Human-in-the-loop for destructive ops |

## CSA Framework (DIDs for Agents)

```json
{
  "did": "did:web:ggid.dev:agents:data-processor",
  "verificationMethod": "did:web:...#key-1",
  "service": [{"type": "MCP", "serviceEndpoint": "https://mcp.corp.com"}]
}
```

Each agent gets a DID for verifiable identity in federated environments.

## Consent for Autonomous Agents

```
Agent wants to perform destructive action:
1. Agent requests: DELETE /api/v1/users/{id}
2. GGID checks: agent has "users:write" but NOT "users:delete"
3. GGID sends consent request to owner:
   "Your agent 'Data Processor' wants to delete user Jane Doe. Approve?"
4. Owner approves/denies via mobile push
5. If approved → time-boxed token (5 min) with users:delete scope
6. Agent performs action → audit logged
```

## Audit Requirements

Every agent action must record:

```json
{
  "event": "agent.action",
  "agent_id": "agent-uuid",
  "owner_user_id": "user-uuid",
  "delegation_chain": ["user-uuid", "agent-a", "agent-b"],
  "action": "users.read",
  "resource": "uuid",
  "scope_used": "users:read",
  "mcp_tool": "read_user_profile",
  "approved_by": null,
  "timestamp": "2025-01-15T10:30:00Z"
}
```

Retention: 7 years (same as human audit).

## Monitoring

| Metric | Alert |
|--------|-------|
| Privilege drift detected | Any → review |
| Shadow agents found | Any → register or revoke |
| MCP tool violation | Any → block + alert |
| Destructive action without approval | Any → block |
| Orphaned agents | Any → quarantine |

## See Also

- [AI Agent Identity](ai-agent-identity.md)
- [Agent Identity Delegation](agent-identity-delegation.md)
- [NHI Lifecycle Management](nhi-lifecycle-management.md)
- [Token Binding Strategies](token-binding-strategies.md)