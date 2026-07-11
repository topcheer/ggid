# Quick Start: AI Agent Identity

> Get an AI agent authenticated and making API calls in 3 steps.

---

## Prerequisites

- GGID Gateway running at `http://localhost:8080`
- A user JWT with admin or agent-owner scope
- Default tenant: `00000000-0000-0000-0000-000000000001`

---

## 3-Step Agent Flow

### Step 1: Register the Agent

```bash
curl -X POST http://localhost:8080/api/v1/agents/register \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My AI Agent",
    "type": "custom",
    "owner_user_id": "'"$USER_ID"'",
    "allowed_scopes": ["read:users"],
    "max_delegation_depth": 1
  }'
```

Save the returned `id` — that's your agent ID.

### Step 2: Exchange User Token for Agent Token

```bash
AGENT_TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/agents/token \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"'"$AGENT_ID"'","requested_scope":["read:users"]}' \
  | jq -r .access_token)
```

### Step 3: Verify and Use

```bash
# Verify the agent token
curl -X POST http://localhost:8080/api/v1/agents/verify \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"token":"'"$AGENT_TOKEN"'"}'

# Use it to call APIs as the agent
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $AGENT_TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

---

**Full guide:** [AI Agent Identity & MCP Auth](ai-agent-identity.md)
