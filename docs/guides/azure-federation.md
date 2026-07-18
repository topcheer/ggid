# Azure AD (Entra ID) Federation with GGID

Integrate GGID as an OIDC Provider with Azure AD B2B federation, enabling cross-tenant SSO for enterprise customers.

## Architecture

```
┌──────────┐   OIDC Authorization Code  ┌──────────────────┐
│  GGID    │ <────────────────────────> │  Azure AD        │
│ OIDC IdP │                            │  (Entra ID)      │
│          │   OIDC Discovery (JWKS)    │  B2B Federation  │
│ JWKS URL │ ─────────────────────────> │                  │
└──────────┘                            └──────────────────┘
```

**Flow**: Guest user in Azure AD clicks login -> redirected to GGID OIDC authorize -> user authenticates at GGID -> Azure AD receives ID token -> grants access.

## Prerequisites

- GGID OAuth service deployed with HTTPS
- Azure AD (Entra ID) tenant with Global Admin or Application Admin role
- GGID OIDC discovery endpoint accessible from Azure

## Step 1: Verify GGID OIDC Discovery

GGID exposes OIDC discovery at:

```
https://<your-ggid-domain>/.well-known/openid-configuration
```

```bash
curl -s https://auth.yourcompany.com/.well-known/openid-configuration | jq .
```

Key fields Azure AD requires:
- `issuer`: GGID issuer URL (must match `iss` claim in tokens)
- `authorization_endpoint`: `/api/v1/oauth/authorize`
- `token_endpoint`: `/api/v1/oauth/token`
- `jwks_uri`: `/.well-known/jwks.json`
- `id_token_signing_alg_values_supported`: `["RS256"]`
- `scopes_supported`: `["openid", "profile", "email"]`

## Step 2: Create GGID OAuth Client for Azure AD

Create an OAuth client in GGID for Azure AD:

```bash
curl -X POST "https://auth.yourcompany.com/api/v1/oauth/clients" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Azure AD B2B",
    "redirect_uris": ["https://login.microsoftonline.com/common/oauth2/redirect"],
    "grant_types": ["authorization_code"],
    "scopes": ["openid", "profile", "email"],
    "token_endpoint_auth_method": "client_secret_post"
  }'
```

Note the `client_id` and `client_secret` for Azure AD configuration.

## Step 3: Configure Azure AD B2B Federation

1. Log into **Azure Portal** -> **Entra ID** -> **External Identities**
2. Select **All identity providers** -> **Add** -> **OpenID Connect**
3. Fill in:
   - **Name**: GGID
   - **Client ID**: from Step 2
   - **Client secret**: from Step 2
   - **Metadata URL**: `https://auth.yourcompany.com/.well-known/openid-configuration`
4. Under **Claim mapping**:
   - User ID: `sub`
   - Email: `email`
   - Display name: `name`
5. Save and enable

## Step 4: Create Conditional Access Policy (Optional)

In Azure AD -> **Security** -> **Conditional Access**:
1. Create policy for guest users from GGID
2. Require MFA if sign-in risk is high
3. Block legacy authentication

## Step 5: Invite Guest Users

In Azure AD -> **Users** -> **Add** -> **Invite external user**:
1. Enter email addresses of users managed by GGID
2. Select **GGID** as the identity provider
3. Send invitation

## Step 6: Test B2B Login

1. Guest user receives invitation email
2. Clicks **Accept invitation**
3. Redirected to GGID login page
4. After authentication, redirected to Azure AD portal
5. Access Microsoft 365 / Azure resources as guest

## Verification

```bash
# Verify OIDC discovery is accessible
MWELLKNOWN=$(curl -s https://auth.yourcompany.com/.well-known/openid-configuration)
echo $WELLKNOWN | jq .issuer
echo $WELLKNOWN | jq .jwks_uri

# Verify JWKS has valid keys
curl -s https://auth.yourcompany.com/.well-known/jwks.json | jq .keys[0].kid

# Test authorization code flow
curl -s "https://auth.yourcompany.com/api/v1/oauth/authorize?response_type=code&client_id=<client-id>&redirect_uri=https://login.microsoftonline.com/common/oauth2/redirect&scope=openid+profile+email&state=test123"
```

## Troubleshooting

### "AADSTS700016: Application not found"

- **Cause**: Client ID in Azure doesn't match GGID OAuth client
- **Fix**: Verify client_id in Azure AD B2B federation matches GGID OAuth client

### JWKS fetch fails

- **Cause**: Azure can't reach GGID JWKS endpoint, or TLS certificate issue
- **Fix**: Ensure `/.well-known/jwks.json` is accessible from Azure IP ranges; verify TLS cert is valid

### ID token validation fails

- **Cause**: `iss` claim doesn't match Azure's expected issuer, or signing algorithm mismatch
- **Fix**: GGID must sign tokens with RS256; verify `iss` matches `issuer` in discovery document

### Claims not mapping

- **Cause**: Azure expects specific claim names that GGID doesn't send
- **Fix**: Configure GGID OIDC claim mapping at `/api/v1/oauth/claim-mapping` to include `email`, `name`, `sub` claims
