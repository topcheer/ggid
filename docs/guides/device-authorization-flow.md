# Device Authorization Flow (RFC 8628)

device_code lifecycle, polling strategy, user_code UX, QR code, TV/IoT/CLI scenarios, and security.

## Overview

The Device Authorization Grant enables devices without a browser (smart TVs, IoT, CLI tools) to obtain tokens by directing the user to authorize on a separate device (phone/laptop).

## Flow

```
┌─────────┐                     ┌──────────┐                    ┌──────────┐
│ Device  │                     │  GGID    │                    │ User's   │
│ (TV/CLI)│                     │  Auth    │                    │ Browser  │
└────┬────┘                     └────┬─────┘                    └────┬─────┘
     │                                │                               │
     │ 1. POST /device/code           │                               │
     │───────────────────────────────▶│                               │
     │  {device_code, user_code,      │                               │
     │   verification_uri}            │                               │
     │◀───────────────────────────────│                               │
     │                                │                               │
     │ 2. Display "Visit ggid.dev/    │                               │
     │    device and enter CODE"      │                               │
     │                                │                               │
     │                                │  3. User visits URL + enters  │
     │                                │◀──────────────────────────────│
     │                                │  4. User logs in + approves    │
     │                                │──────────────────────────────▶│
     │                                │                               │
     │ 5. POST /token (polling)       │                               │
     │───────────────────────────────▶│                               │
     │  {error: authorization_pending}│                               │
     │◀───────────────────────────────│                               │
     │                                │                               │
     │ 6. POST /token (polling)       │                               │
     │───────────────────────────────▶│                               │
     │  {access_token, refresh_token} │                               │
     │◀───────────────────────────────│                               │
```

## Step 1: Request Device Code

```bash
POST /api/v1/oauth/device/code
Content-Type: application/x-www-form-urlencoded

client_id=client-123
&scope=openid+profile
```

### Response

```json
{
  "device_code": "GmRh...long-opaque-code",
  "user_code": "WDJB-MJHQ",
  "verification_uri": "https://auth.ggid.dev/device",
  "verification_uri_complete": "https://auth.ggid.dev/device?user_code=WDJB-MJHQ",
  "expires_in": 1800,
  "interval": 5
}
```

| Field | Description |
|-------|-------------|
| `device_code` | Internal code for token polling (never shown to user) |
| `user_code` | Short code user enters (human-readable) |
| `verification_uri` | URL user visits to authorize |
| `verification_uri_complete` | URL with embedded code (QR code target) |
| `expires_in` | Device code TTL (seconds) |
| `interval` | Minimum seconds between poll requests |

## Step 2: Display to User

### TV Screen

```
┌────────────────────────────────────┐
│         Authorize Your Device       │
│                                    │
│   Visit:  ggid.dev/device          │
│   Enter:  WDJB-MJHQ                │
│                                    │
│   ┌─────────────────────┐          │
│   │   [QR Code Image]    │          │
│   └─────────────────────┘          │
│                                    │
│   Or scan QR code with your phone   │
└────────────────────────────────────┘
```

### CLI

```
$ ggid login
Open this URL in your browser: https://auth.ggid.dev/device
And enter the code: WDJB-MJHQ

Waiting for authorization...
```

### QR Code

```python
# Generate QR from verification_uri_complete
import qrcode
img = qrcode.make("https://auth.ggid.dev/device?user_code=WDJB-MJHQ")
img.save("device_qr.png")
```

## Step 3: User Authorization

User visits `verification_uri`, enters `user_code`, logs in, and approves:

```bash
GET /device?user_code=WDJB-MJHQ
# → Show consent screen:
#   "Device 'Smart TV' is requesting access to your account.
#    Scopes: openid, profile
#    [Deny]  [Allow]"
```

## Step 4: Polling (Token Request)

```bash
POST /api/v1/oauth/token
grant_type=urn:ietf:params:oauth:grant-type:device_code
&device_code=GmRh...
&client_id=client-123
```

### Polling Responses

| Response | Meaning | Action |
|----------|---------|--------|
| 200 `{access_token}` | Approved | Use tokens |
| `authorization_pending` | User hasn't approved yet | Wait `interval` seconds, retry |
| `slow_down` | Polling too fast | Increase interval by 5s |
| `expired_token` | Device code expired | Restart flow |
| `access_denied` | User denied | Stop, show error |
| `invalid_grant` | Bad device_code | Restart flow |

### Polling Implementation

```go
func pollForToken(deviceCode, clientID string, interval int) (*TokenResponse, error) {
    for {
        time.Sleep(time.Duration(interval) * time.Second)

        resp, err := requestToken(deviceCode, clientID)
        if err == nil {
            return resp, nil // Success
        }

        switch resp.Error {
        case "authorization_pending":
            continue // Wait and retry
        case "slow_down":
            interval += 5 // Back off
            continue
        case "expired_token":
            return nil, ErrDeviceCodeExpired
        case "access_denied":
            return nil, ErrAccessDenied
        }
    }
}
```

## User Code Format

| Format | Example |
|--------|---------|
| 8 chars (XXXX-XXXX) | `WDJB-MJHQ` |
| Charset | `BCDFGHJKLMNPQRSTVWXZ` (no vowels, no ambiguous chars) |
| Length | 8 characters |
| Case | Uppercase only |
| Hyphen | After 4th char for readability |

### Design Rules

- No vowels → prevents accidental profanity
- No `0/O`, `1/I/S/5` → prevents confusion
- Case-insensitive matching → user can type lowercase
- Auto-uppercase in UI
- Retry if collision with existing code

## Security

### Device Code Binding

```go
// Device code is bound to the client_id
// Only the original client can poll with it
func validateDeviceCode(deviceCode, clientID string) error {
    stored := getDeviceCode(deviceCode)
    if stored.ClientID != clientID {
        return ErrClientMismatch
    }
    return nil
}
```

### Rate Limiting

| Endpoint | Rate | Burst |
|----------|------|-------|
| `POST /device/code` | 10/hour per client | 20 |
| `POST /token` (device poll) | Per `interval` | Slow_down if faster |
| `GET /device` (user_code entry) | 5/min per IP | 10 |

### Phishing Prevention

- User code expires in 15-30 minutes
- Consent screen shows device name (if provided)
- User can deny authorization
- Audit log records device authorization

## Use Cases

### Smart TV

```
TV app → Request device code → Display on TV screen
User → Opens browser → Enters code → Approves
TV app → Polls → Gets tokens → Streams content
```

### CLI Tool

```bash
$ ggid device-login
# Opens browser automatically if possible
# Otherwise shows URL + code
# Polls until authorized
```

### IoT Device

```
Raspberry Pi → Requests device code → Displays on LED panel
User → Enters code in mobile app → Approves
Pi → Gets tokens → Accesses API
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Completion rate (approve/pending) | <50% → UX issue |
| Average time to authorize | >2min → too complex |
| Device code expiry rate | >20% → code too hard to enter |
| Polling rate violations | Spike → misconfigured client |

## See Also

- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
- [OAuth PKCE Deep Dive](oauth-pkce-deep-dive.md)
- [Token Introspection Design](token-introspection-design.md)
- [SDK Integration Guide](sdk-integration-guide.md)
