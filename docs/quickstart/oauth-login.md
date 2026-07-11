# OAuth Login Quickstart

> Complete OAuth 2.1 authorization code flow with PKCE end-to-end.

---

## 1. Register OAuth Client

```bash
JWT="your-admin-jwt"
TENANT="00000000-0000-0000-0000-000000000001"

curl -s -X POST http://localhost:8080/api/v1/oauth/clients \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "demo-app",
    "client_secret": "demo-secret",
    "redirect_uris": ["http://localhost:3000/callback"],
    "grant_types": ["authorization_code"],
    "response_types": ["code"],
    "scope": "openid profile email read:users"
  }' | jq .
```

## 2. Generate PKCE Pair

```bash
VERIFIER=$(openssl rand -base64 32 | tr -d '=+/' | cut -c1-43)
CHALLENGE=$(echo -n "$VERIFIER" | openssl dgst -sha256 -binary | base64 | tr -d '=+/' | cut -c1-43)
echo "verifier:  $VERIFIER"
echo "challenge: $CHALLENGE"
```

## 3. Open Authorization URL

```
http://localhost:8080/api/v1/oauth/authorize?
  response_type=code
  &client_id=demo-app
  &redirect_uri=http://localhost:3000/callback
  &scope=openid profile read:users
  &code_challenge=CHALLENGE
  &code_challenge_method=S256
  &state=random-state-123
```

User logs in → redirected to `redirect_uri?code=AUTH_CODE&state=random-state-123`

## 4. Exchange Code for Token

```bash
curl -s -X POST http://localhost:8080/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=AUTH_CODE" \
  -d "redirect_uri=http://localhost:3000/callback" \
  -d "client_id=demo-app" \
  -d "client_secret=demo-secret" \
  -d "code_verifier=$VERIFIER" | jq .
```

Response:
```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "openid profile read:users"
}
```

## 5. Use the Token

```bash
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer eyJ..."
```

---

*See: [OAuth Flows Guide](../oauth-flows-guide.md) | [API Reference](../api-reference.md)*