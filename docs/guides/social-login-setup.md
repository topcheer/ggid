# Social Login Setup

> Configure GitHub, Google, Microsoft, LDAP, and other identity providers for social login.

---

## Supported Providers

| Provider | Config Prefix | Scopes |
|----------|--------------|--------|
| GitHub | `GITHUB_` | `user:email`, `read:user` |
| Google | `GOOGLE_` | `openid`, `email`, `profile` |
| Microsoft | `MICROSOFT_` | `User.Read`, `openid` |
| GitLab | `GITLAB_` | `read_user`, `email` |
| Discord | `DISCORD_` | `identify`, `email` |
| Slack | `SLACK_` | `identity.basic`, `identity.email` |
| LinkedIn | `LINKEDIN_` | `r_liteprofile`, `r_emailaddress` |
| Apple | `APPLE_` | `email`, `name` |
| OIDC (Generic) | `OIDC_` | `openid`, `email`, `profile` |
| LDAP | `LDAP_` | User filter based |

---

## GitHub

### 1. Create OAuth App

Go to [GitHub Settings → Developer settings → OAuth Apps → New OAuth App](https://github.com/settings/developers).

- **Application name**: Your App
- **Homepage URL**: `https://your-app.com`
- **Authorization callback URL**: `http://localhost:8080/api/v1/auth/callback/github`

### 2. Configure GGID

```bash
GITHUB_CLIENT_ID=Ov23liXXXXXXXXXX
GITHUB_CLIENT_SECRET=your-client-secret
GITHUB_REDIRECT_URL=http://localhost:8080/api/v1/auth/callback/github
```

### 3. Test Login

```
GET http://localhost:8080/api/v1/auth/github
→ Redirects to GitHub consent → callback → JWT
```

---

## Google

### 1. Create OAuth Credentials

Go to [Google Cloud Console → APIs & Services → Credentials](https://console.cloud.google.com/apis/credentials).

- Create OAuth 2.0 Client ID (Web application)
- **Authorized redirect URI**: `http://localhost:8080/api/v1/auth/callback/google`

### 2. Configure GGID

```bash
GOOGLE_CLIENT_ID=xxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/callback/google
```

---

## Microsoft (Azure AD)

### 1. Register Application

Go to [Azure Portal → App registrations → New registration](https://portal.azure.com).

- **Redirect URI**: `http://localhost:8080/api/v1/auth/callback/microsoft`

### 2. Configure GGID

```bash
MICROSOFT_CLIENT_ID=your-tenant-app-id
MICROSOFT_CLIENT_SECRET=your-client-secret
MICROSOFT_REDIRECT_URL=http://localhost:8080/api/v1/auth/callback/microsoft
MICROSOFT_TENANT_ID=common  # or specific tenant UUID
```

---

## LDAP / Active Directory

### Configuration

```bash
LDAP_URL=ldap://ldap.example.com:389
LDAP_BIND_DN=cn=admin,dc=example,dc=com
LDAP_BIND_PASSWORD=admin-password
LDAP_BASE_DN=ou=users,dc=example,dc=com
LDAP_USER_FILTER=(uid=%s)
LDAP_START_TLS=true
LDAP_AUTO_PROVISION=true  # Create GGID user on first LDAP login
```

### How It Works

1. User submits username + password via `/api/v1/auth/login`
2. Auth Service tries LocalProvider first (internal DB)
3. If local fails, tries LDAPProvider:
   - Binds with service account
   - Searches for user by filter
   - Attempts bind with user's DN + password
4. If LDAP bind succeeds and `LDAP_AUTO_PROVISION=true`, creates local user

---

## Generic OIDC Provider

For any OIDC-compliant provider (Okta, Auth0, Keycloak, etc.):

```bash
OIDC_ISSUER=https://your-provider.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=http://localhost:8080/api/v1/auth/callback/oidc
```

---

## Login Flow (All Providers)

```
User → GET /api/v1/auth/{provider}
     → Redirect to provider consent screen
     → User authorizes
     → Provider redirects to /api/v1/auth/callback/{provider}?code=...
     → GGID exchanges code for access token
     → GGID fetches user profile
     → GGID creates/issues JWT
     → Redirect to app with JWT
```

---

## Multi-Provider Setup

Enable multiple providers simultaneously:

```bash
# All providers can be active at once
GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
LDAP_URL=ldap://...
```

The Auth Provider Chain tries each provider in order: Local → LDAP → OAuth.

---

## Production Checklist

- [ ] Use HTTPS for all callback URLs
- [ ] Set `SESSION_SECRET` for OAuth state validation
- [ ] Store client secrets in secrets manager (not env files in git)
- [ ] Configure `LDAP_START_TLS=true` for LDAP security
- [ ] Set `LDAP_AUTO_PROVISION=false` if you want manual user provisioning

---

*See: [Authentication Guide](../authentication-guide.md) | [3-Line Integration](../quickstart/3-line-integration.md) | [External DB Setup](../quickstart/external-db-setup.md)*

*Last updated: 2025-07-11*
