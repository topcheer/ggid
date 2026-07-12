# OIDC Backchannel Logout

RFC draft implementation: logout token format, session ID propagation, per-client endpoints, error handling, race conditions, and testing.

## Overview

Backchannel logout enables single logout (SLO) across all relying parties (RPs) without relying on browser redirects (front-channel). The OpenID Provider (GGID) sends an HTTP POST directly to each RP's backchannel logout endpoint.

```
User logs out of GGID
    │
    ▼
GGID iterates all RPs with active sessions
    │
    ├── POST /backchannel-logout (RP 1) → session terminated
    ├── POST /backchannel-logout (RP 2) → session terminated
    └── POST /backchannel-logout (RP 3) → session terminated
```

## Logout Token Format

GGID sends a signed JWT (not an access token) to each RP:

```json
{
  "iss": "https://auth.ggid.dev",
  "sub": "user-uuid",
  "aud": "client-123",
  "iat": 1700000000,
  "jti": "logout-token-uuid",
  "events": {
    "http://schemas.openid.net/event/backchannel-logout": {}
  },
  "sid": "session-uuid-abc",
  "nonce": "optional-nonce"
}
```

| Claim | Required | Description |
|-------|----------|-------------|
| `iss` | Yes | GGID issuer URL |
| `aud` | Yes | Target client ID |
| `iat` | Yes | Issued at timestamp |
| `jti` | Yes | Unique token ID (prevent replay) |
| `events` | Yes | Must contain backchannel-logout event |
| `sub` | One of | User subject |
| `sid` | One of | Session identifier |
| `nonce` | No | Optional nonce |

Either `sub` or `sid` (or both) must be present. `sub` identifies the user; `sid` identifies the specific session.

## Configuration

### Register Backchannel Endpoint

```bash
POST /api/v1/oauth/register
{
  "client_name": "My App",
  "backchannel_logout_uri": "https://app.example.com/backchannel-logout",
  "backchannel_logout_session_required": true
}
```

### Discovery

```bash
GET /.well-known/openid-configuration
# → {
#   "backchannel_logout_supported": true,
#   "backchannel_logout_session_supported": true
# }
```

## Sending Logout Notifications

```go
func SendBackchannelLogout(userID, sessionID string, clients []Client) {
    for _, client := range clients {
        if client.BackchannelLogoutURI == "" { continue }
        
        token := createLogoutToken(client, userID, sessionID)
        
        go func(uri, token string) {
            resp, err := http.PostForm(uri, url.Values{
                "logout_token": {token},
            })
            
            if err != nil || resp.StatusCode != 200 {
                // Retry with backoff
                retryBackchannelLogout(uri, token)
            }
            
            audit.Log("backchannel_logout_sent", map[string]interface{}{
                "client_id": client.ID,
                "status":    resp.StatusCode,
            })
        }(client.BackchannelLogoutURI, token)
    }
}

func createLogoutToken(client Client, userID, sessionID string) string {
    claims := jwt.MapClaims{
        "iss":    issuerURL,
        "sub":    userID,
        "aud":    client.ID,
        "iat":    time.Now().Unix(),
        "jti":    uuid.New().String(),
        "sid":    sessionID,
        "events": map[string]interface{}{
            "http://schemas.openid.net/event/backchannel-logout": struct{}{},
        },
    }
    token, _ := jwt.SignWithKey(signingKey, claims)
    return token
}
```

## RP-Side Verification

```go
func HandleBackchannelLogout(w http.ResponseWriter, r *http.Request) {
    // 1. Extract logout token
    token := r.PostFormValue("logout_token")
    if token == "" { http.Error(w, "missing token", 400); return }
    
    // 2. Verify JWT signature
    claims, err := verifyLogoutToken(token)
    if err != nil { http.Error(w, "invalid token", 400); return }
    
    // 3. Verify required claims
    events, ok := claims["events"].(map[string]interface{})
    if !ok || events["http://schemas.openid.net/event/backchannel-logout"] == nil {
        http.Error(w, "missing event", 400); return
    }
    if claims["jti"] == nil { http.Error(w, "missing jti", 400); return }
    if claims["sub"] == nil && claims["sid"] == nil {
        http.Error(w, "missing sub and sid", 400); return
    }
    
    // 4. Prevent replay (jti tracking)
    if seen := redis.SetNX(ctx, "logout:"+claims["jti"].(string), 1, 5*time.Minute); !seen {
        http.Error(w, "duplicate", 400); return
    }
    
    // 5. Terminate session
    sid := claims["sid"].(string)
    sessionStore.Delete(sid)
    
    w.WriteHeader(200)
}
```

## Error Handling

| RP Response | GGID Action |
|-------------|-------------|
| 200 | Success, session terminated |
| 400 | Invalid token (GGID bug) — log error |
| 404 | Endpoint not found — disable backchannel for this client |
| 500/503 | RP unavailable — retry with backoff |
| Timeout | Retry up to 3 times, then give up |

### Retry Strategy

```go
delays := []time.Duration{0, 30*time.Second, 5*time.Minute, 1*time.Hour}
for attempt, delay := range delays {
    resp, err := sendLogout(uri, token)
    if err == nil && resp.StatusCode == 200 { return }
    time.Sleep(delay)
}
// After max retries: log failure, RP session remains until natural expiry
```

## Race Conditions

### Logout vs Active Request

```
User logs out → backchannel logout sent
  ↓
RP still processing a request from that session
  ↓
Session deleted → request fails with 401
```

Mitigation: RPs should handle mid-request session deletion gracefully (return 401, client re-authenticates).

### Concurrent Logouts

```
User clicks logout on two devices simultaneously
  → Two backchannel notifications for same session
  → RP receives duplicate → jti dedup prevents double-processing
```

## Session ID (sid) Propagation

GGID includes `sid` in:
1. ID token at login: `"sid": "session-uuid"`
2. Backchannel logout token: `"sid": "same-session-uuid"`

RPs must store the `sid` from the ID token to match it during logout.

## Testing

### Manual Test

```bash
# 1. User logs in to RP → RP stores sid
# 2. Trigger logout at GGID
POST /api/v1/auth/logout
Authorization: Bearer <token>

# 3. Verify RP received backchannel logout
GET https://app.example.com/session-status
# → {"session": "terminated"}
```

### Automated Test

```go
func TestBackchannelLogout(t *testing.T) {
    // Start mock RP with backchannel endpoint
    rp := startMockRP()
    
    // Login
    session := login(rp.ClientID)
    assert(rp.HasSession(session.SID))
    
    // Logout at GGID
    logout(session.Token)
    
    // Wait for backchannel notification
    time.Sleep(2 * time.Second)
    
    // Verify RP session terminated
    assert(!rp.HasSession(session.SID))
}
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Backchannel delivery success rate | <95% → RP issues |
| Retry rate | >5% → network or RP availability |
| Replay attempts (duplicate jti) | Any → investigate |
| Delivery latency | >5s → RP slow endpoint |

## See Also

- [Session Security](session-security.md)
- [Identity Federation Architecture](identity-federation-architecture.md)
- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
