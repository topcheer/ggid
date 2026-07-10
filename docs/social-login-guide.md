# Social Login Configuration Guide

Complete guide for configuring social login providers in GGID: Google,
GitHub, Microsoft, Apple, GitLab, Discord, Slack, LinkedIn, and generic OIDC.

> **See also**: [OAuth Flows Guide](oauth-flows-guide.md) for the underlying
> OAuth 2.1 flows, [Configuration Reference](configuration-reference.md)
> for environment variables.

---

## Table of Contents

- [Supported Providers](#supported-providers)
- [Configuration Overview](#configuration-overview)
- [Provider Setup](#provider-setup)
- [Account Linking](#account-linking)
- [Attribute Mapping](#attribute-mapping)
- [Troubleshooting](#troubleshooting)

---

## Supported Providers

| Provider | Protocol | Key Fields | Docs |
|----------|----------|------------|------|
| Google | OAuth 2.0 / OIDC | email, name, picture | [Setup](https://developers.google.com/identity) |
| GitHub | OAuth 2.0 | login, email, name | [Setup](https://docs.github.com/apps) |
| Microsoft | OAuth 2.0 / OIDC | email, displayName | [Setup](https://learn.microsoft.com/entra) |
| Apple | OAuth 2.0 / OIDC | email (first time only), name | [Setup](https://developer.apple.com) |
| GitLab | OAuth 2.0 | email, username, name | [Setup](https://docs.gitlab.com/ee/integration/oauth_provider.html) |
| Discord | OAuth 2.0 | email, username, avatar | [Setup](https://discord.com/developers) |
| Slack | OAuth 2.0 | email, name, team | [Setup](https://api.slack.com/authentication) |
| LinkedIn | OAuth 2.0 | email, name | [Setup](https://learn.linkedin.com/developers) |
| Generic OIDC | OIDC | configurable | Any OIDC-compliant provider |

---

## Configuration Overview

### Environment Variables

```bash
# Enable social login
SOCIAL_LOGIN_ENABLED=true

# Redirect URL pattern: https://iam.example.com/oauth/callback/{provider}
SOCIAL_REDIRECT_BASE=https://iam.example.com/oauth/callback

# Per-provider config (see below)
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...
```

### API Configuration

```bash
curl -X POST https://iam.example.com/api/v1/admin/social/providers \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "provider": "google",
    "client_id": "your-client-id.apps.googleusercontent.com",
    "client_secret": "your-client-secret",
    "scopes": ["openid", "email", "profile"],
    "enabled": true
  }'
```

### List Configured Providers

```bash
curl https://iam.example.com/api/v1/admin/social/providers \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

---

## Provider Setup

### Google

1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Create OAuth 2.0 Client ID (Web application)
3. Add authorized redirect URI: `https://iam.example.com/oauth/callback/google`

```yaml
social:
  google:
    client_id: "xxx.apps.googleusercontent.com"
    client_secret: "${GOOGLE_CLIENT_SECRET}"
    scopes: ["openid", "email", "profile"]
    redirect_url: "https://iam.example.com/oauth/callback/google"
```

**Returned attributes**: `email`, `email_verified`, `given_name`, `family_name`, `picture`, `locale`

### GitHub

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Create OAuth App
3. Authorization callback URL: `https://iam.example.com/oauth/callback/github`

```yaml
social:
  github:
    client_id: "${GITHUB_CLIENT_ID}"
    client_secret: "${GITHUB_CLIENT_SECRET}"
    scopes: ["user:email"]
    redirect_url: "https://iam.example.com/oauth/callback/github"
```

**Returned attributes**: `login` (username), `email`, `name`, `avatar_url`, `company`, `location`

> GitHub email may be `null` if the user has set email privacy. The `user:email` scope allows fetching the primary email.

### Microsoft (Azure AD / Entra ID)

1. Go to [Azure Portal](https://portal.azure.com) → App Registrations
2. Register application, add redirect URI
3. Grant Microsoft Graph permissions: `openid`, `email`, `profile`

```yaml
social:
  microsoft:
    client_id: "${MS_CLIENT_ID}"
    client_secret: "${MS_CLIENT_SECRET}"
    tenant_id: "common"    # or specific tenant UUID
    scopes: ["openid", "email", "profile", "User.Read"]
    redirect_url: "https://iam.example.com/oauth/callback/microsoft"
```

**Returned attributes**: `email`, `displayName`, `givenName`, `surname`, `jobTitle`, `department`

### Apple

1. Go to [Apple Developer](https://developer.apple.com) → Certificates, Identifiers & Profiles
2. Create Services ID (not App ID) for Sign in with Apple
3. Configure Return URL: `https://iam.example.com/oauth/callback/apple`

```yaml
social:
  apple:
    client_id: "com.example.signin"        # Services ID
    team_id: "TEAMXXXXXX"
    key_id: "KEYXXXXXXX"
    private_key: "${APPLE_PRIVATE_KEY}"     # .p8 file contents
    scopes: ["email", "name"]
    redirect_url: "https://iam.example.com/oauth/callback/apple"
```

**Apple-specific notes**:
- Apple provides `email` and `name` **only on the first login**. Store them immediately.
- Apple may provide a relay email (`xxx@privaterelay.appleid.com`).
- Private key is used for JWT client authentication (not client_secret).

### GitLab

```yaml
social:
  gitlab:
    client_id: "${GITLAB_CLIENT_ID}"
    client_secret: "${GITLAB_CLIENT_SECRET}"
    scopes: ["read_user"]
    redirect_url: "https://iam.example.com/oauth/callback/gitlab"
    base_url: "https://gitlab.com"    # or self-hosted URL
```

### Discord

```yaml
social:
  discord:
    client_id: "${DISCORD_CLIENT_ID}"
    client_secret: "${DISCORD_CLIENT_SECRET}"
    scopes: ["identify", "email"]
    redirect_url: "https://iam.example.com/oauth/callback/discord"
```

### Slack

```yaml
social:
  slack:
    client_id: "${SLACK_CLIENT_ID}"
    client_secret: "${SLACK_CLIENT_SECRET}"
    scopes: ["openid", "email", "profile"]
    redirect_url: "https://iam.example.com/oauth/callback/slack"
```

### LinkedIn

```yaml
social:
  linkedin:
    client_id: "${LINKEDIN_CLIENT_ID}"
    client_secret: "${LINKEDIN_CLIENT_SECRET}"
    scopes: ["openid", "profile", "email"]
    redirect_url: "https://iam.example.com/oauth/callback/linkedin"
```

### Generic OIDC

For any OIDC-compliant provider (Okta, Auth0, Keycloak, etc.):

```yaml
social:
  oidc:
    client_id: "${OIDC_CLIENT_ID}"
    client_secret: "${OIDC_CLIENT_SECRET}"
    issuer: "https://your-provider.com"
    scopes: ["openid", "email", "profile"]
    redirect_url: "https://iam.example.com/oauth/callback/oidc"
    attribute_mapping:
      email: "email"
      name: "name"
      username: "preferred_username"
```

---

## Account Linking

### How Linking Works

When a user logs in via a social provider:

1. GGID checks if a user with that `email` exists
2. If yes → link the social account to the existing user
3. If no → auto-provision a new user (configurable)

### Configuration

```yaml
social:
  account_linking:
    enabled: true
    auto_provision: true            # Create user if not exists
    link_by_email: true             # Match existing users by email
    require_verified_email: true    # Only link if email is verified
    conflict_resolution: "link"     # link, deny, or merge
```

### Manual Linking

```bash
# Link a social account to an existing user
curl -X POST .../users/{user_id}/social-accounts \
  -d '{ "provider": "google", "provider_user_id": "google-12345", "email": "user@gmail.com" }'
```

### List Linked Accounts

```bash
curl .../users/{user_id}/social-accounts \
  -H "Authorization: Bearer $TOKEN"
```

```json
{
  "accounts": [
    { "provider": "google", "provider_user_id": "google-12345", "email": "user@gmail.com", "linked_at": "2024-01-15T10:00:00Z" },
    { "provider": "github", "provider_user_id": "gh-67890", "email": "user@example.com", "linked_at": "2024-01-10T15:00:00Z" }
  ]
}
```

### Unlink Account

```bash
curl -X DELETE .../users/{user_id}/social-accounts/{provider}
```

---

## Attribute Mapping

Map provider-specific attributes to GGID user fields:

```yaml
social:
  attribute_mapping:
    google:
      email: "email"
      first_name: "given_name"
      last_name: "family_name"
      display_name: "name"
      avatar_url: "picture"
    github:
      email: "email"
      username: "login"
      display_name: "name"
      avatar_url: "avatar_url"
    microsoft:
      email: "email"
      first_name: "givenName"
      last_name: "surname"
      display_name: "displayName"
```

---

## Troubleshooting

### Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| `redirect_uri_mismatch` | Redirect URI not registered in provider | Add exact URI to provider config |
| `invalid_client` | Wrong client_id or secret | Verify credentials |
| `email_not_verified` | Provider email not verified | Require verified_email: false, or verify email |
| `state_mismatch` | State parameter mismatch (CSRF) | Check session/cookie configuration |
| `no_email_returned` | Provider didn't return email | Check scopes (need `email` scope) |
| `account_exists` | Email matches but linking disabled | Enable account_linking or use different email |

### Debug Mode

```bash
# Enable social login debug logging
curl -X PATCH .../admin/tenant/settings/social \
  -d '{ "debug": true }'

# Check logs
docker logs ggid-auth 2>&1 | grep -i "social\|oauth\|callback"
```

### Testing Callbacks

Use the built-in test endpoint:

```bash
# Simulate Google callback
curl .../oauth/callback/google?code=test_code&state=test_state
```

### Provider Connection Test

```bash
# Test provider connectivity
curl .../admin/social/providers/google/test \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```json
{
  "provider": "google",
  "status": "connected",
  "discovery_url": "https://accounts.google.com/.well-known/openid-configuration",
  "response_time_ms": 120
}
```
