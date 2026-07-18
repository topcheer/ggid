# Integration Guide

Complete guide for integrating your application with GGID. Choose the method that matches your use case.

## Integration Methods at a Glance

| Method | Use Case | User-facing? | Complexity |
|--------|----------|-------------|------------|
| OAuth 2.1 Authorization Code | Web/Mobile apps with user login | Yes | Medium |
| Client Credentials | Machine-to-machine, services, CI/CD | No | Low |
| SAML 2.0 SP | Enterprise/legacy apps | Yes | Medium |
| WebAuthn/Passkey | Passwordless, high-security | Yes | Medium |

## Prerequisites

- GGID running at `http://localhost:8080` (or your deployment URL)
- Admin account: username=`admin`, password=`Admin@123456`
- Tenant ID: `00000000-0000-0000-0000-000000000001`

---

## Method 1: OAuth 2.1 Authorization Code (Web Apps)

Best for: SPAs, server-rendered web apps, mobile apps where users log in interactively.

### Architecture

```
User → Browser → Your App
                   ↓ redirect to GGID
              GGID Login Page
                   ↓ callback with code
              Your App → exchanges code for token
                   ↓ API calls with token
              GGID Gateway → your backend
```

### Step 1: Register an OAuth Client

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"Admin@123456"}' | jq -r '.access_token')

curl -s -X POST http://localhost:8080/api/v1/oauth/clients \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "client_name": "My Web App",
    "redirect_uris": ["http://localhost:3001/callback"],
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"],
    "token_endpoint_auth_method": "client_secret_post",
    "scope": "openid profile email"
  }' | jq .
```

Save the `client_id` and `client_secret` from the response.

### Step 2: Redirect User to Authorization

```
https://ggid.example.com/api/v1/oauth/authorize?
  client_id=YOUR_CLIENT_ID&
  redirect_uri=http://localhost:3001/callback&
  response_type=code&
  scope=openid%20profile%20email&
  state=RANDOM_STATE_STRING
```

### Step 3: Exchange Authorization Code for Tokens

```bash
curl -s -X POST http://localhost:8080/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=AUTHORIZATION_CODE_FROM_CALLBACK" \
  -d "client_id=YOUR_CLIENT_ID" \
  -d "client_secret=YOUR_CLIENT_SECRET" \
  -d "redirect_uri=http://localhost:3001/callback" | jq .
```

### Step 4: Use Access Token

```bash
curl -s http://localhost:8080/api/v1/auth/profile \
  -H "Authorization: Bearer ACCESS_TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq .
```

### Verify (curl)

```bash
# Full flow test
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/auth/profile \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
# Expected: 200
```

---

## Method 2: Client Credentials (Machine-to-Machine)

Best for: API services, background jobs, CI/CD pipelines — no user interaction.

### Architecture

```
Service → GGID Token Endpoint → receives JWT → calls API with JWT
                                                      ↓
                                              GGID Gateway → backend
```

### Step 1: Register a Service Client

```bash
curl -s -X POST http://localhost:8080/api/v1/oauth/clients \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "client_name": "My Backend Service",
    "grant_types": ["client_credentials"],
    "token_endpoint_auth_method": "client_secret_post",
    "scope": "users:read roles:read"
  }' | jq .
```

### Step 2: Request Token

```bash
curl -s -X POST http://localhost:8080/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=YOUR_CLIENT_ID" \
  -d "client_secret=YOUR_CLIENT_SECRET" \
  -d "scope=users:read" | jq .
```

### Step 3: Use Token for API Access

```bash
curl -s http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer MACHINE_TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq '.users | length'
```

### Code Example (Go)

```go
// Simple token refresh loop
func getGGIDToken(clientID, clientSecret, ggidURL string) (string, error) {
    resp, err := http.PostForm(ggidURL+"/api/v1/oauth/token", url.Values{
        "grant_type":    {"client_credentials"},
        "client_id":     {clientID},
        "client_secret": {clientSecret},
        "scope":         {"users:read"},
    })
    if err != nil { return "", err }
    defer resp.Body.Close()
    var result struct{ AccessToken string `json:"access_token"` }
    json.NewDecoder(resp.Body).Decode(&result)
    return result.AccessToken, nil
}
```

---

## Method 3: SAML 2.0 Service Provider (Enterprise)

Best for: Legacy enterprise apps, government systems requiring SAML federation.

### Architecture

```
User → Your App (SP)
         ↓ SAML request via redirect
       GGID (IdP) — user authenticates
         ↓ SAML response posted back
       Your App — validates SAML, creates session
```

### Step 1: Register SAML Application

```bash
curl -s -X POST http://localhost:8080/api/v1/oauth/clients \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "client_name": "Enterprise SAML App",
    "grant_types": [],
    "protocol": "saml",
    "redirect_uris": ["https://app.example.com/saml/acs"],
    "response_types": ["saml"]
  }' | jq .
```

### Step 2: Get IdP Metadata

```bash
curl -s http://localhost:8080/api/v1/oauth/saml/metadata | jq .
```

Register GGID's IdP metadata in your Service Provider configuration:
- **Entity ID**: from GGID metadata
- **SSO URL**: GGID's `/api/v1/oauth/saml/sso`
- **X.509 Certificate**: from metadata for response verification

### Step 3: Initiate SSO

Redirect user to:
```
http://localhost:8080/api/v1/oauth/saml/sso?SAMLRequest=BASE64_ENCODED_REQUEST
```

GGID authenticates the user and POSTs the SAML response to your ACS endpoint.

### Step 4: Validate SAML Response

Your SP library validates:
1. Response signature using GGID's public certificate
2. Response destination matches your ACS URL
3. Conditions (NotBefore/NotOnOrAfter)
4. Audience restriction matches your Entity ID

---

## Method 4: WebAuthn/Passkey (Passwordless)

Best for: High-security apps, modern UX, phishing-resistant authentication.

### Architecture

```
Registration:
  Browser → Your App → begin registration → GGID returns challenge
         → authenticator creates credential → post to GGID → stored

Login:
  Browser → Your App → begin login → GGID returns challenge
         → authenticator signs → post to GGID → returns JWT
```

### Step 1: Begin Registration

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/webauthn/register/begin \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"display_name":"My Passkey"}' | jq .
```

Pass the returned challenge to the browser's WebAuthn API:
```javascript
const credential = await navigator.credentials.create({ publicKey: challenge });
```

### Step 2: Finish Registration

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/webauthn/register/finish \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d "{\"credential\": \"BASE64_ENCODED_CREDENTIAL\"}" | jq .
```

### Step 3: Login with Passkey

```bash
# Begin login
curl -s -X POST http://localhost:8080/api/v1/auth/webauthn/login/begin \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin"}' | jq .

# Finish login (after authenticator signs)
curl -s -X POST http://localhost:8080/api/v1/auth/webauthn/login/finish \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"credential":"BASE64_SIGNED_ASSERTION"}' | jq .access_token
```

### Verify (curl)

```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/auth/webauthn/register/begin \
  -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"display_name":"Test"}'
# Expected: 200
```

---

## SDK Integration

GGID provides 11 SDKs. Quick start for the most popular:

### React SDK

```bash
npm install @ggid/sdk-react
```

```tsx
import { GGIDProvider, useAuth } from '@ggid/sdk-react';

function App() {
  return (
    <GGIDProvider
      domain="http://localhost:8080"
      tenantId="00000000-0000-0000-0000-000000000001"
      clientId="YOUR_CLIENT_ID"
      redirectUri="http://localhost:3001/callback"
    >
      <Dashboard />
    </GGIDProvider>
  );
}
```

### Go SDK

```go
import "github.com/ggid/ggid/sdk/go"

client := ggid.New("http://localhost:8080",
    ggid.WithJWKS(15*time.Minute),
    ggid.WithTenantID("00000000-0000-0000-0000-000000000001"),
)

users, err := client.Users.List(ctx, ggid.WithToken(accessToken))
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| 401 on API calls | Ensure `X-Tenant-ID` header is set |
| 404 on routes | All API calls go through Gateway (:8080), not backend services |
| Login returns invalid credentials | Use `username` field, not `email` |
| OAuth callback fails | Verify `redirect_uri` matches client registration |
| CORS errors | Gateway allows all origins by default; check your proxy config |
