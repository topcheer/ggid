# Federation Setup Wizard — User Guide

> Feature: F-52 Federation Setup Wizard
> Location: **Settings > Federation** (`/settings/federation`)

## What It Does

The Federation Setup Wizard provides a guided interface for configuring federated identity relationships with external Identity Providers (IdPs) and Service Providers (SPs). It supports SAML 2.0, OpenID Connect (OIDC), and WS-Federation protocols, with trust level management and metadata auto-import.

## How to Access

1. Log in to the GGID Admin Console.
2. Navigate to **Settings** in the sidebar.
3. Click **Federation**.

Alternatively, go to `/settings/federation` directly.

## Tabs and Sections

### 1. Entities

Lists all configured federation entities:

- **Entity Name**: Display name for the federation partner.
- **Protocol**: SAML 2.0, OpenID Connect, or WS-Federation.
- **Trust Level**: Trusted, Verified, or Pending.
- **Trust Direction**: Inbound (they login to us), Outbound (we login to them), Bidirectional.
- **Status**: Enabled or Disabled.
- **Last Checked**: Last metadata refresh timestamp.

**Workflow — Delete an entity:**
1. Find the entity in the list.
2. Click the trash icon.
3. Confirm deletion.

### 2. Setup Wizard (3 Steps)

A guided 3-step wizard to quickly add a new federation partner.

#### Step 1: Choose Protocol
Select the federation protocol:
- **SAML 2.0**: For enterprise SSO with SAML-based IdPs (Okta, Azure AD, AD FS).
- **OpenID Connect**: For modern OAuth 2.0/OIDC-based providers (Google, Auth0, Keycloak).

#### Step 2: Enter Entity Details
Provide the partner's information:
- **Entity Name**: Human-readable name (e.g., "Corporate Okta").
- **Entity ID**: Unique identifier (e.g., `https://okta.corp.com/saml`).
- **Metadata URL**: URL to the partner's federation metadata (auto-imports configuration).
- **Issuer**: For OIDC, the issuer URL.
- **JWKS URL**: For OIDC, the JSON Web Key Set endpoint.
- **Trust Direction**: Inbound, Outbound, or Bidirectional.
- **Auto-Import**: Automatically import attributes/claims from metadata.

#### Step 3: Review and Confirm
Review all settings and click **Create Entity**. The entity is created with trust level "Pending".

**Workflow — Add a new SAML IdP:**
1. Go to the Wizard tab.
2. Step 1: Select **SAML 2.0**.
3. Step 2: Enter "Corporate Okta" as name, paste the Metadata URL.
4. Set Trust Direction to **Inbound** (users from Okta login to GGID).
5. Enable **Auto-Import**.
6. Step 3: Review and click **Create**.
7. The entity appears in the Entities tab with status "Pending".
8. Verify the metadata was imported correctly.
9. Change trust level to "Trusted" after verification.

### 3. Trust Management

View and manage trust relationships:

- **Trust Matrix**: Shows all entities and their trust levels in a grid.
- **Trust Level**: Upgrade from Pending → Verified → Trusted.
- **Trust Direction**: View inbound/outbound/bidirectional relationships.

**Workflow — Promote an entity to Trusted:**
1. Find the entity in the Trust tab.
2. Verify the metadata and certificate are correct.
3. Change trust level from "Pending" to "Verified".
4. After successful test authentication, change to "Trusted".

### 4. Monitoring

Monitor federation health:

- **Metadata Freshness**: When each entity's metadata was last refreshed.
- **Certificate Expiry**: Days until signing certificates expire.
- **Active Sessions**: Current federated login sessions per entity.
- **Error Rate**: Recent authentication failures per entity.

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/identity/federation/entities` | GET | List all federation entities |
| `/api/v1/identity/federation/entities` | POST | Create a new entity |
| `/api/v1/identity/federation/entities?id=:id` | DELETE | Delete an entity |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# List all federation entities
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/identity/federation/entities" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"

# Create a new SAML entity
NEW_TOKEN="your-jwt-token"
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/identity/federation/entities" \
  -H "Authorization: Bearer $NEW_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"entity_id":"https://okta.corp.com/saml","entity_name":"Corporate Okta","entity_type":"idp","protocol":"saml","metadata_url":"https://okta.corp.com/app/sso/saml/metadata","trust_level":"pending","trust_direction":"inbound"}'

# Delete an entity
curl -k -H 'Accept-Encoding: identity' \
  -X DELETE "https://ggid.iot2.win/api/v1/identity/federation/entities?id=ent-123" \
  -H "Authorization: Bearer $NEW_TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Metadata import fails | URL unreachable or invalid XML | Verify metadata URL is accessible from the identity pod; check XML format |
| Entity stays "Pending" | Trust level not manually upgraded | Use Trust tab to upgrade to Verified, then Trusted |
| SAML login fails | Certificate mismatch or entity ID mismatch | Verify entity ID matches IdP metadata; re-import certificate |
| OIDC discovery fails | Issuer URL incorrect or JWKS unreachable | Verify issuer URL returns valid `.well-known/openid-configuration` |
| Certificate expiring | Signing cert nearing expiration | Contact federation partner to rotate certificates; update metadata |

## Best Practices

- **Start with Pending trust**: Always create entities with "Pending" trust level.
- **Test before trusting**: Perform a test authentication before upgrading to "Trusted".
- **Enable auto-import**: Let the wizard import metadata attributes automatically.
- **Monitor certificates**: Set up alerts for certificates expiring within 30 days.
- **Document each entity**: Use descriptive names that identify the partner and protocol.
- **Regular metadata refresh**: Ensure metadata is refreshed at least every 24 hours.
- **Principle of least trust**: Only upgrade trust level when business requirements demand it.
