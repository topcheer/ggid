# SCIM 2.0 Provisioning Guide

> Configure SCIM user provisioning with Slack, Google, Microsoft Entra ID, and Okta.

---

## GGID SCIM Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/scim/v2/Users` | List/filter users |
| POST | `/scim/v2/Users` | Create user |
| PUT | `/scim/v2/Users/{id}` | Replace user |
| PATCH | `/scim/v2/Users/{id}` | Partial update |
| DELETE | `/scim/v2/Users/{id}` | Deactivate |
| GET/POST | `/scim/v2/Groups` | Group CRUD |
| POST | `/scim/v2/Bulk` | Batch operations |

---

## Sync Modes

| Mode | Trigger | Use Case |
|------|---------|----------|
| **Inbound** | External IdP → GGID | HR-driven provisioning (Okta/Entra pushes users) |
| **Outbound** | GGID → External App | GGID pushes to downstream apps (Slack/Google) |

---

## Slack Integration

1. Slack Admin → Settings → SCIM Provisioning
2. Enable SCIM, note the SCIM URL
3. In GGID, configure outbound SCIM:
```bash
curl -X POST http://localhost:8080/api/v1/scim-providers \
  -d '{
    "name": "Slack",
    "type": "outbound",
    "base_url": "https://api.slack.com/scim/v2",
    "token": "xoxb-slack-token"
  }'
```

---

## Microsoft Entra ID

1. Azure Portal → Enterprise Applications → New
2. Provisioning → Automatic
3. Tenant URL: `https://ggid.example.com/scim/v2`
4. Secret Token: Admin JWT
5. Attribute mappings:
   - `userPrincipalName` → `userName`
   - `mail` → `emails[0].value`
   - `department` → `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department`

---

## Okta Integration

1. Okta Admin → Applications → Browse App Catalog → Create SCIM App
2. SCIM Base URL: `https://ggid.example.com/scim/v2`
3. Auth: Bearer Token
4. Test: Assign user → Okta pushes to GGID via POST `/scim/v2/Users`
5. Deprovisioning: Unassign user → Okta sends PATCH `active: false`

---

## Verification

```bash
# Check SCIM connectivity
curl https://ggid.example.com/scim/v2/Users?count=1 \
  -H "Authorization: Bearer $TOKEN"
# Should return 200 + ListResponse
```

---

*See: [SCIM API Reference](../api/scim-api.md) | [SCIM Ecosystem Analysis](scim-ecosystem.md) | [Tenant Onboarding](tenant-onboarding.md)*

*Last updated: 2025-07-11*
