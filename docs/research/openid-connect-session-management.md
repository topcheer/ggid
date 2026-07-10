# OIDC Session Management 1.0

> Real-time session status monitoring between RPs and the OP via check_session_iframe.
> Date: 2025-07-20 · Status: Research
>
> **Scope**: Focuses on the OIDC Session Management spec — `check_session_iframe`,
> `session_state`, cookie coordination, and SameSite implications. For logout
> flows, see [oidc-logout-spec-analysis.md](./oidc-logout-spec-analysis.md).
> For internal token lifecycle, see [session-management-design.md](./session-management-design.md).

---

## 1. Overview

OIDC Session Management 1.0 defines a mechanism for Relying Parties (RPs) to
detect in real time whether the user's session at the OpenID Provider (OP) is
still active — without a token refresh or API call.

**The problem**: An RP holds a valid token pair, but the user may have logged out
at the OP (or another RP triggered back-channel logout). The RP discovers this
only when the access token expires or an API call fails with 401. Until then, the
RP's local session remains active and the user sees stale state.

```
Without session management:
  User logs out at OP ──────────────────────▶ RP session still active
                      (no notification)        Access token expires → 401
                                               RP finally discovers logout

With session management:
  User logs out at OP ──▶ iframe detects change (<5s) ──▶ RP clears local session
```

**Spec status**: Implementer's draft — never reached final status, yet widely
deployed by Google, Microsoft, Auth0, Keycloak. Long-term viability is uncertain
given third-party cookie deprecation (section 5); fallback strategies are essential.

---

## 2. Check Session iframe

### How It Works

The OP advertises a `check_session_iframe` URL in discovery metadata. The RP
embeds this URL in a hidden iframe. The iframe's JavaScript listens for
`postMessage` events from the RP, compares the received session state with the
OP's actual browser state, and responds `"changed"` or `"unchanged"`.

### postMessage Protocol

**RP → iframe** (space-delimited string): `{client_id} {session_state}`

**iframe → RP** (single word): `"unchanged"` or `"changed"`

**Origin checking**: The RP must verify `event.origin` matches the OP's origin
before acting. Without this, any page element could inject a fake response.

### session_state Calculation

```
session_state = base64url(SHA256(client_id + " " + origin + " " + opbs)) + "." + opbs

  client_id = the RP's client identifier
  origin    = the RP's origin (scheme://host:port)
  opbs      = OP Browser State — random value in a cookie on the OP's domain
```

When the user logs out (or a new session starts) at the OP, the `opbs` cookie
changes. The iframe recomputes `session_state` with the new opbs and compares
it with the value the RP sent. A mismatch means `"changed"`.

---

## 3. RP Session Monitoring Flow

### Setup and Polling

```html
<!-- RP embeds the hidden iframe after receiving the auth response -->
<iframe id="op-check" src="https://op.example.com/oauth/check_session" hidden></iframe>

<script>
const OP_ORIGIN = "https://op.example.com";
const POLL_MS = 5000; // 5s recommended
let sessionState = new URLSearchParams(location.hash).get("session_state");
let clientId = "my-client-id";
let frame = document.getElementById("op-check");

function checkSession() {
  if (!frame || !sessionState) return;
  frame.contentWindow.postMessage(clientId + " " + sessionState, OP_ORIGIN);
}

window.addEventListener("message", function (e) {
  if (e.origin !== OP_ORIGIN) return; // origin verification!
  if (e.data === "changed") {
    // Silent re-auth or log out locally
    window.location.href = "/oauth/authorize?prompt=none&client_id=" + clientId
      + "&redirect_uri=" + encodeURIComponent(location.origin + "/callback")
      + "&response_type=code&scope=openid";
  }
});

setInterval(checkSession, POLL_MS);
checkSession(); // initial check on load
</script>
```

### Polling Loop Sequence Diagram

```
  RP Page              Hidden Iframe           OP (op.example.com)
  (rp.example.com)     (op.example.com)
     │                      │                         │
     │ postMessage:         │                         │
     │ "client_id ss_val"   │                         │
     ├─────────────────────>│ Read opbs cookie        │
     │                      │ Recompute ss_local       │
     │     "unchanged"      │ (opbs unchanged)        │
     │<─────────────────────┤                         │
     │                      │                         │ User logs out at OP
     │                      │                         │ opbs cookie changes
     │ postMessage:         │                         │
     │ "client_id ss_val"   │                         │
     ├─────────────────────>│ Read opbs (new value)   │
     │                      │ ss_local !== ss_val     │
     │     "changed"        │                         │
     │<─────────────────────┤                         │
     │ RP redirects to OP   │                         │
     │ /authorize?prompt=none                        │ │
     ├──────────────────────────────────────────────>│ User has no session
     │                  302 redirect error=login_required
     │<──────────────────────────────────────────────┤
     │ RP clears local session                        │
```

### Silent Re-authentication

When the iframe returns `"changed"`, the RP does **not** immediately log out.
Instead it performs a silent re-authentication with `prompt=none`:

- User still authenticated at OP → new code issued → session continues.
- User logged out → OP returns `error=login_required` → RP logs out locally.

---

## 4. Cookie Coordination

### OP Browser State Cookie (opbs)

| Attribute | Value | Why |
|-----------|-------|-----|
| **Domain** | `op.example.com` (OP domain) | Must be readable inside the iframe |
| **HttpOnly** | `false` | iframe JavaScript reads via `document.cookie` |
| **SameSite** | `None` | Must be sent in third-party iframe context |
| **Secure** | `true` | Required when `SameSite=None` |

```http
Set-Cookie: opbs=a3F5bG9...; Domain=op.example.com; SameSite=None; Secure; Path=/
```

### Third-Party Cookie Problem

From the RP's perspective, the iframe at the OP is a **third-party** context.
The opbs cookie is a third-party cookie. Browsers are blocking these:

| Browser | Policy | Impact |
|---------|--------|--------|
| **Safari (ITP)** | Blocks 3rd-party cookies (Safari 13.1+) | opbs not sent → monitoring breaks |
| **Chrome** | 3rd-party cookie deprecation in progress | Same impact once fully enforced |
| **Firefox** | ETP strict mode blocks them | Strict-mode users affected |

**Workarounds**:

1. **Storage Access API** (Safari/Firefox): iframe calls
   `document.requestStorageAccess()` to prompt user for first-party storage
   permission. Requires user click on first use.
2. **CHIPS (Partitioned cookies)**: Chrome's partitioned cookies via the
   `Partitioned` attribute — each top-level site gets a separate cookie jar.
3. **Token introspection fallback**: RP periodically calls `/oauth/introspect`
   server-side (every 60s). No browser cookies needed. See Phase 4 in roadmap.

---

## 5. SameSite Cookie Implications

### SameSite=Strict

Cookie only sent on same-site requests. **Breaks OIDC**: the top-level redirect
from RP to OP is cross-site → OP session cookie not sent → user not recognized →
forced re-login. **NOT recommended** for the OP session cookie.

### SameSite=Lax

Cookie sent on top-level GET navigations (including cross-site redirects). Works
for OIDC auth code flow: redirect to `/oauth/authorize` is top-level navigation.
Default in modern browsers. **Does NOT work for iframes** — iframe requests are
not top-level navigations. **RECOMMENDED** for the OP's main session cookie.

### SameSite=None

Cookie sent on all cross-site requests including iframe embeds. **Required** for
opbs to work in check_session_iframe. Must pair with `Secure=true`. **At risk**
from third-party cookie deprecation.

### Recommended Cookie Strategy

```
OP sets two cookies on login:

1. op_session (main auth session)  → SameSite=Lax; Secure; HttpOnly
   Used for top-level OIDC redirects

2. opbs (browser state for monitoring) → SameSite=None; Secure
   Used by check_session_iframe JS. Add Partitioned for CHIPS.
```

---

## 6. Logout Integration

Session Management **detects** that the session changed; **logout mechanisms**
act on it. The two work together:

```
  Session Management (detection)         Logout Mechanism (action)
       │                                      │
  User logs out at OP ───┤                      │
  opbs cookie changes ───┤                      │
  iframe: "changed" ─────┤                      │
  RP re-auth (prompt=none)                      │
  OP returns login_required                     │
  RP triggers back-channel ────────────────────>│ OP confirms
  logout token                                  │ session destroyed
  RP clears local session                       │
```

**Key distinction**: Session Management is the **detection** layer (browser-based,
iframe polling). Back/Front-channel logout is the **notification** layer
(server-to-server or browser iframe fan-out). In production, deploy both: iframe
for real-time UX, back-channel for reliable server-side session cleanup.

> For RP-initiated, back-channel, and front-channel logout details, see
> [oidc-logout-spec-analysis.md](./oidc-logout-spec-analysis.md).

---

## 7. GGID Implementation Analysis

| Feature | Current State | Gap |
|---------|---------------|-----|
| `check_session_iframe` endpoint | Not implemented — no route in `server.go` | Add `GET /oauth/check_session` |
| `check_session_iframe` in discovery | Missing — `OIDCDiscoveryConfig` has no field | Add `json:"check_session_iframe"` |
| `session_state` in auth response | Not implemented — handler returns `{redirect_url, code, state}` | Compute and append `session_state` |
| opbs cookie | Not set — no cookie middleware in OAuth service | Add cookie on login/authorize |
| OP session cookie SameSite | N/A — GGID uses JWT + X-Session-ID header, not cookies | Architectural decision needed |
| RP iframe embedding | Console-side concern — Console uses API-based session | No iframe monitoring |
| Third-party cookie handling | Not addressed | Implement introspection fallback |
| Gateway session validation | Redis `SessionManager` implemented | Server-side only, not browser iframe |

**Current architecture**: GGID is API-first — JWT access tokens (stateless) +
opaque refresh tokens (Redis-backed). The gateway validates JWT signatures and
checks Redis for revocation. There is no browser-side session cookie on the OP
domain. Implementing Session Management requires adding a cookie-based session
layer to the OAuth service.

---

## 8. Implementation Design

### New Endpoint: `GET /oauth/check_session`

```go
func CheckSessionHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.Header().Set("X-Frame-Options", "ALLOWALL")
        w.Write([]byte(checkSessionHTML))
    }
}

const checkSessionHTML = `<!DOCTYPE html><html><body><script>
(function () {
  function getCookie(name) {
    var m = document.cookie.match(new RegExp('(^| )' + name + '=([^;]+)'));
    return m ? decodeURIComponent(m[2]) : '';
  }
  window.addEventListener('message', function (e) {
    var parts = e.data.split(' ');
    if (parts.length !== 2) return;
    var clientId = parts[0], receivedState = parts[1];
    var opbs = getCookie('opbs');
    if (!opbs) { e.source.postMessage('changed', e.origin); return; }
    crypto.subtle.digest('SHA-256',
      new TextEncoder().encode(clientId + ' ' + e.origin + ' ' + opbs)
    ).then(function (buf) {
      var b64 = btoa(String.fromCharCode.apply(null, new Uint8Array(buf)))
        .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
      e.source.postMessage(b64 === receivedState ? 'unchanged' : 'changed', e.origin);
    });
  });
})();
</script></body></html>`
```

### Auth Response: session_state Computation

```go
// In /oauth/authorize handler, after generating the auth code:

opbs, err := r.Cookie("opbs")
if err != nil || opbs.Value == "" {
    opbs = &http.Cookie{
        Name: "opbs", Value: generateRandomString(32),
        Path: "/", MaxAge: 86400, Secure: true,
        SameSite: http.SameSiteNoneMode,
    }
    http.SetCookie(w, opbs)
}

sessionState := computeSessionState(clientID, redirectURIOrigin, opbs.Value)
redirectURL += "&session_state=" + sessionState
```

```go
func computeSessionState(clientID, origin, opbs string) string {
    h := sha256.Sum256([]byte(clientID + " " + origin + " " + opbs))
    return base64.RawURLEncoding.EncodeToString(h[:]) + "." + opbs
}
```

### Discovery Config Update

```go
type OIDCDiscoveryConfig struct {
    // ... existing fields ...
    CheckSessionIframe string `json:"check_session_iframe"`
    EndSessionEndpoint string `json:"end_session_endpoint"`
}
```

---

## 9. Roadmap

| Phase | Scope | Effort |
|-------|-------|--------|
| **1.** opbs cookie + `session_state` computation | Cookie middleware, `computeSessionState()`, add to auth response | 2-3 days |
| **2.** `check_session_iframe` endpoint | `CheckSessionHandler` HTML/JS, register route, discovery metadata | 2-3 days |
| **3.** Console integration | Embed hidden iframe, `postMessage` polling, handle `"changed"` | 2-3 days |
| **4.** Third-party cookie fallback | Detect blocked cookies, introspection polling (60s), Storage Access API | 2-3 days |
| **5.** Front-channel logout integration | On `"changed"` + re-auth fail, trigger logout fan-out via NATS | 1-2 days |

**Total**: ~1.5-2 weeks for Phase 1-3 (core functionality).

> **Warning**: Third-party cookie deprecation may significantly reduce the value
> of iframe-based monitoring. Chrome's full deprecation would break the opbs
> cookie. Monitor browser changes and prioritize the introspection fallback
> (Phase 4). Consider WebSocket or SSE-based session change notifications as a
> longer-term alternative that does not depend on third-party cookies.
