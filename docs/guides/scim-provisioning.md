# SCIM Provisioning — Technical Guide

> Feature: SCIM 2.0 Outbound Provisioning
> Location: `services/identity/internal/scim/`
> Console: `/settings/scim-provisioning`

## What It Does

GGID supports both inbound SCIM (receiving provisioning from IdPs like Okta/Azure AD) and outbound SCIM (pushing user/group changes to downstream applications). This guide covers outbound provisioning — automatically syncing GGID users to external systems.

## Components

### SCIM Targets

Each target is a downstream application that receives provisioning updates:

| Field | Description |
|-------|-------------|
| **Name** | Display name (e.g., "Slack", "Salesforce") |
| **Base URL** | SCIM endpoint (e.g., `https://api.slack.com/scim/v2/`) |
| **Auth Token** | Bearer token for the target system |
| **Enabled** | Active/inactive toggle |
| **Entity Types** | Users, Groups, or both |

### Attribute Mapping

Map GGID attributes to target SCIM attributes:

| GGID Attribute | SCIM Attribute | Example |
|----------------|----------------|---------|
| `username` | `userName` | `john.doe` |
| `email` | `emails[0].value` | `john@acme.com` |
| `first_name` | `name.givenName` | `John` |
| `last_name` | `name.familyName` | `Doe` |
| `display_name` | `displayName` | `John Doe` |
| `groups` | `groups` | `[...]` |
| `active` | `active` | `true/false` |

Custom mappings supported via JSON path expressions.

### Sync Log

Every provisioning operation is logged:

- **Operation**: Create, Update, Delete, Group Sync
- **Target**: Which SCIM target received the change
- **Entity**: User/group ID
- **Status**: Success, Failed, Retried
- **Timestamp**: When the sync occurred
- **Response**: HTTP status code from target

### Circuit Breaker

Protects against cascading failures when a target is down:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `max_failures` | 5 | Consecutive failures before tripping |
| `reset_timeout` | 60s | Time before retry attempt |
| `half_open_max` | 1 | Probes in half-open state |

When tripped, provisioning to that target pauses, queues changes, and retries after reset timeout.

## Provisioning Workflow

```
GGID User Change (create/update/delete)
         ↓
   Evaluate SCIM Targets
         ↓
   For each enabled target:
   ┌──────────────────────┐
   │ Map attributes       │
   │ ↓                    │
   │ Build SCIM request   │
   │ ↓                    │
   │ Send to target       │
   │ ↓                    │
   │ Log result           │
   │ ↓                    │
   │ Circuit breaker check│
   └──────────────────────┘
```

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/scim/Groups` | GET/POST | Inbound SCIM group management |
| `/api/v1/scim/Groups/:id` | GET/PATCH/DELETE | Inbound SCIM group operations |
| `/api/v1/settings/scim/targets` | GET/POST | Outbound target management |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# List SCIM targets (outbound)
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/settings/scim/targets" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Create outbound SCIM target
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/settings/scim/targets" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"name":"Slack","base_url":"https://api.slack.com/scim/v2/","auth_token":"xoxb-...","entity_types":["users","groups"],"enabled":true}'

# Inbound: Create group via SCIM
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/scim/Groups" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/scim+json" \
  -d '{"schemas":["urn:ietf:params:scim:schemas:core:2.0:Group"],"displayName":"Engineering","members":[]}'
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Target not receiving updates | Target disabled or circuit breaker tripped | Check target status; reset circuit breaker |
| Attribute mapping wrong | Field names don't match target schema | Review target's SCIM schema; adjust mappings |
| Sync fails with 401 | Auth token expired | Rotate target auth token in target config |
| Duplicate users | Matching attribute not unique | Ensure `userName` or `email` is unique |

## Best Practices

- **Test with one target first**: Validate attribute mappings before enabling multiple targets.
- **Monitor circuit breaker**: Watch for repeated trips — indicates target instability.
- **Use email as match key**: Email is more stable than username across systems.
- **Enable group sync**: Sync group memberships to avoid manual group management in each app.
- **Audit sync log weekly**: Catch silent failures before they cascade.
