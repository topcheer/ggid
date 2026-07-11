# SCIM Ecosystem Analysis

> SCIM 2.0 compatibility analysis across IdPs and how GGID compares.

---

## SCIM Implementations

### Okta
- Full SCIM 2.0 (User + Group CRUD)
- Custom SCIM app integration via Okta Workflows
- Automatic deprovisioning on user deactivation
- Supports PATCH (partial update)

### Microsoft Entra ID (Azure AD)
- SCIM 2.0 via enterprise applications
- User provisioning + group sync
- Attribute mapping UI
- Automatic deprovisioning
- Supports bulk operations

### Slack
- SCIM 2.0 API for user management
- Group-based channel access
- Deprovisioning via SCIM DELETE

### Google Workspace
- SCIM 2.0 via Cloud Identity
- User provisioning, limited group sync

---

## GGID SCIM 2.0 Support

| Feature | Status |
|---------|--------|
| GET /Users (list + filter) | Done |
| POST /Users (create) | Done |
| GET /Users/{id} | Done |
| PUT /Users/{id} (replace) | Done |
| PATCH /Users/{id} | Done |
| DELETE /Users/{id} | Done |
| GET/POST /Groups | Done |
| PUT/PATCH/DELETE /Groups | Done |
| POST /Bulk | Done |
| Filter expressions (eq, ne, co, sw, pr, and, or) | Done |
| Enterprise extension (URN) | Done |
| URN colon notation | Done (fixed 513548b) |

---

## Integration Examples

### Okta → GGID

1. Okta Admin → Applications → Browse App Catalog → Create SCIM App
2. SCIM Base URL: `https://ggid.example.com/scim/v2`
3. Auth: Bearer Token (admin JWT)
4. Test provisioning: assign user to app → Okta pushes to GGID

### Azure AD → GGID

1. Azure Portal → Enterprise Applications → New → Non-gallery
2. Provisioning Mode: Automatic
3. Tenant URL: `https://ggid.example.com/scim/v2`
4. Secret Token: admin JWT
5. Mappings: map `userPrincipalName` → `userName`, `mail` → `emails[0].value`

---

## Competitive Advantage

GGID has **full SCIM 2.0** while Keycloak has **no native SCIM support**. This is a key differentiator for enterprise customers using Okta/Entra ID for HR-driven provisioning.

---

*See: [SCIM API Reference](../api/scim-api.md) | [Identity Service](../architecture/microservices.md) | [Gap Closure Report](gap-closure-report.md)*

*Last updated: 2025-07-11*
