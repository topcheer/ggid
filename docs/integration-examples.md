# GGID Integration Examples

Real-world code samples for integrating GGID into your applications.

---

## Table of Contents

- [Protect a Go HTTP Service with JWT Middleware](#protect-a-go-http-service-with-jwt-middleware)
- [Add Google OAuth to an Express App via GGID](#add-google-oauth-to-an-express-app-via-ggid)
- [SCIM User Provisioning from Workday](#scim-user-provisioning-from-workday)
- [SAML SSO for Grafana](#saml-sso-for-grafana)
- [WebAuthn Registration in Vanilla JS](#webauthn-registration-in-vanilla-js)

---

## Protect a Go HTTP Service with JWT Middleware

Validate GGID JWTs in any Go HTTP service using the JWKS endpoint.

```go
// main.go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strings"
    "time"

    "github.com/go-jose/go-jose/v3/jwt"
)

type GGIDClaims struct {
    Sub      string   `json:"sub"`
    TenantID string   `json:"tenant_id"`
    Scope    string   `json:"scope"`
    Roles    []string `json:"roles"`
    jwt.Claims
}

type JWTMiddleware struct {
    jwksURL string
    issuer  string
}

func NewJWTMiddleware(jwksURL, issuer string) *JWTMiddleware {
    return &JWTMiddleware{jwksURL: jwksURL, issuer: issuer}
}

func (m *JWTMiddleware) Validate(tokenStr string) (*GGIDClaims, error) {
    // Fetch JWKS (cache in production with TTL)
    resp, err := http.Get(m.jwksURL)
    if err != nil {
        return nil, fmt.Errorf("fetching JWKS: %w", err)
    }
    defer resp.Body.Close()

    var jwks struct {
        Keys []json.RawMessage `json:"keys"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
        return nil, fmt.Errorf("decoding JWKS: %w", err)
    }

    // Parse token without verification to get the kid
    tok, err := jwt.ParseSigned(tokenStr)
    if err != nil {
        return nil, fmt.Errorf("parsing token: %w", err)
    }

    // Verify with the matching key
    claims := &GGIDClaims{}
    for _, key := range jwks.Keys {
        var pubKey interface{}
        if err := json.Unmarshal(key, &pubKey); err != nil {
            continue
        }
        if err := tok.Claims(pubKey, claims); err == nil {
            // Validate standard claims
            if err := claims.Validate(jwt.Expected{
                Issuer:    m.issuer,
                Time:      time.Now(),
            }); err != nil {
                return nil, fmt.Errorf("claim validation: %w", err)
            }
            return claims, nil
        }
    }

    return nil, fmt.Errorf("no matching key found")
}

func (m *JWTMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip public routes
        if r.URL.Path == "/healthz" {
            next.ServeHTTP(w, r)
            return
        }

        authHeader := r.Header.Get("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
            return
        }

        tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := m.Validate(tokenStr)
        if err != nil {
            http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
            return
        }

        // Inject claims into request context
        ctx := context.WithValue(r.Context(), "claims", claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// RequireRole checks if the user has the required role
func RequireRole(role string, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims, ok := r.Context().Value("claims").(*GGIDClaims)
        if !ok {
            http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
            return
        }

        for _, r := range claims.Roles {
            if r == role || r == "admin" {
                next.ServeHTTP(w, r)
                return
            }
        }

        http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
    })
}

func main() {
    mw := NewJWTMiddleware(
        "https://iam.example.com/.well-known/jwks.json",
        "https://iam.example.com",
    )

    mux := http.NewServeMux()

    // Public route
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    })

    // Protected route
    mux.Handle("/api/profile", mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := r.Context().Value("claims").(*GGIDClaims)
        json.NewEncoder(w).Encode(map[string]string{
            "user_id":   claims.Sub,
            "tenant_id": claims.TenantID,
        })
    })))

    // Admin-only route
    mux.Handle("/api/admin", mw.Middleware(
        RequireRole("admin", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            json.NewEncoder(w).Encode(map[string]string{"message": "admin access granted"})
        })),
    ))

    log.Println("Server starting on :3001")
    log.Fatal(http.ListenAndServe(":3001", mux))
}
```

---

## Add Google OAuth to an Express App via GGID

Use GGID as a social login broker — configure Google once in GGID, then your
Express app delegates to GGID for all social providers.

```javascript
// server.js
const express = require('express');
const session = require('express-session');
const axios = require('axios');

const app = express();
const GGID_URL = process.env.GGID_URL || 'https://iam.example.com';
const GGID_TENANT = process.env.GGID_TENANT_ID;
const CLIENT_ID = process.env.GGID_CLIENT_ID;
const CLIENT_SECRET = process.env.GGID_CLIENT_SECRET;
const REDIRECT_URI = process.env.REDIRECT_URI || 'http://localhost:3000/auth/callback';

app.use(session({
    secret: process.env.SESSION_SECRET,
    resave: false,
    saveUninitialized: false,
    cookie: { secure: process.env.NODE_ENV === 'production', httpOnly: true }
}));

// --- Login route: redirect to GGID for Google OAuth ---
app.get('/auth/google', (req, res) => {
    const state = require('crypto').randomBytes(32).toString('hex');
    req.session.oauthState = state;

    // Redirect to GGID's social login endpoint for Google
    const authUrl = new URL(`${GGID_URL}/api/v1/auth/social/google`);
    authUrl.searchParams.set('redirect_uri', REDIRECT_URI);
    authUrl.searchParams.set('state', state);
    authUrl.searchParams.set('tenant_id', GGID_TENANT);

    res.redirect(authUrl.toString());
});

// --- Callback: GGID redirects back with code ---
app.get('/auth/callback', async (req, res) => {
    const { code, state } = req.query;

    // Verify state to prevent CSRF
    if (state !== req.session.oauthState) {
        return res.status(403).send('Invalid state parameter');
    }

    try {
        // Exchange code for tokens via GGID
        const tokenResponse = await axios.post(`${GGID_URL}/api/v1/auth/social/google/callback`, {
            code,
            redirect_uri: REDIRECT_URI,
            client_id: CLIENT_ID,
        }, {
            headers: { 'X-Tenant-ID': GGID_TENANT }
        });

        const { access_token, refresh_token, user } = tokenResponse.data;

        // Store tokens in session
        req.session.accessToken = access_token;
        req.session.refreshToken = refresh_token;
        req.session.user = user;

        res.redirect('/dashboard');
    } catch (err) {
        console.error('OAuth callback error:', err.response?.data || err.message);
        res.redirect('/login?error=auth_failed');
    }
});

// --- Protected route ---
app.get('/dashboard', requireAuth, (req, res) => {
    res.json({
        message: `Welcome, ${req.session.user.email}`,
        user: req.session.user,
    });
});

// --- Logout ---
app.post('/auth/logout', async (req, res) => {
    try {
        // Revoke token at GGID
        await axios.post(`${GGID_URL}/api/v1/auth/logout`, {
            access_token: req.session.accessToken,
        }, {
            headers: { 'X-Tenant-ID': GGID_TENANT }
        });
    } catch (err) {
        // Ignore errors — token may already be expired
    }

    req.session.destroy();
    res.redirect('/login');
});

// --- Middleware ---
function requireAuth(req, res, next) {
    if (!req.session.accessToken) {
        return res.redirect('/auth/google');
    }
    next();
}

app.listen(3000, () => console.log('Express app on :3000'));
```

---

## SCIM User Provisioning from Workday

Provision users from Workday HCM to GGID via the SCIM 2.0 API.

```python
#!/usr/bin/env python3
"""
SCIM provisioning script: sync users from Workday to GGID.
Run as a daily cron job.

pip install requests python-dotenv
"""

import os
import requests
import json
from datetime import datetime
from dotenv import load_dotenv

load_dotenv()

GGID_URL = os.environ['GGID_URL']
GGID_TOKEN = os.environ['GGID_API_KEY']  # SCIM-provisioning API key
TENANT_ID = os.environ['GGID_TENANT_ID']

HEADERS = {
    'Authorization': f'Bearer {GGID_TOKEN}',
    'Content-Type': 'application/scim+json',
    'X-Tenant-ID': TENANT_ID,
}

def get_workday_users():
    """Fetch active employees from Workday (simplified)."""
    # In production: call Workday SOAP/REST API
    # Example response structure
    return [
        {"email": "alice@acme.com", "name": "Alice Chen", "dept": "Engineering", "active": True},
        {"email": "bob@acme.com", "name": "Bob Smith", "dept": "Sales", "active": True},
        {"email": "carol@acme.com", "name": "Carol Jones", "dept": "Engineering", "active": False},
    ]

def scim_user_exists(email):
    """Check if user already exists in GGID via SCIM."""
    resp = requests.get(
        f"{GGID_URL}/scim/v2/Users",
        headers=HEADERS,
        params={'filter': f'emails.value eq "{email}"'}
    )
    if resp.status_code == 200:
        resources = resp.json().get('Resources', [])
        return resources[0] if resources else None
    return None

def create_scim_user(wd_user):
    """Create a new user in GGID via SCIM."""
    scim_user = {
        'schemas': ['urn:ietf:params:scim:schemas:core:2.0:User'],
        'userName': wd_user['email'],
        'name': {
            'formatted': wd_user['name'],
        },
        'emails': [{
            'value': wd_user['email'],
            'type': 'work',
            'primary': True,
        }],
        'active': wd_user['active'],
        'department': wd_user['dept'],
    }

    resp = requests.post(
        f"{GGID_URL}/scim/v2/Users",
        headers=HEADERS,
        json=scim_user
    )

    if resp.status_code == 201:
        print(f"  [CREATED] {wd_user['email']}")
        return resp.json()
    else:
        print(f"  [ERROR] Failed to create {wd_user['email']}: {resp.status_code}")
        return None

def update_scim_user(user_id, wd_user):
    """Update an existing user in GGID via SCIM PUT."""
    scim_user = {
        'schemas': ['urn:ietf:params:scim:schemas:core:2.0:User'],
        'id': user_id,
        'userName': wd_user['email'],
        'name': {'formatted': wd_user['name']},
        'emails': [{'value': wd_user['email'], 'type': 'work', 'primary': True}],
        'active': wd_user['active'],
        'department': wd_user['dept'],
    }

    resp = requests.put(
        f"{GGID_URL}/scim/v2/Users/{user_id}",
        headers=HEADERS,
        json=scim_user
    )

    if resp.status_code == 200:
        print(f"  [UPDATED] {wd_user['email']}")
    else:
        print(f"  [ERROR] Failed to update {wd_user['email']}: {resp.status_code}")

def main():
    print(f"=== SCIM Sync: Workday → GGID ({datetime.now().isoformat()}) ===")

    wd_users = get_workday_users()
    print(f"Workday users: {len(wd_users)}")

    for wd_user in wd_users:
        existing = scim_user_exists(wd_user['email'])

        if existing:
            # Update if changed
            update_scim_user(existing['id'], wd_user)
        else:
            # Create new
            create_scim_user(wd_user)

    print("=== Sync complete ===")

if __name__ == '__main__':
    main()
```

Run as a cron job:

```bash
# crontab -e
# Run daily at 2 AM
0 2 * * * /usr/bin/python3 /opt/ggid/scim-sync.py >> /var/log/ggid-scim-sync.log 2>&1
```

---

## SAML SSO for Grafana

Configure GGID as a SAML identity provider for Grafana.

### Step 1: Register Grafana as a SAML SP in GGID

```bash
curl -X POST $API/api/v1/saml/sp \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "Grafana",
    "entity_id": "https://grafana.example.com",
    "assertion_consumer_service_url": "https://grafana.example.com/saml/acs",
    "name_id_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
  }'
```

### Step 2: Download GGID IdP Metadata

```bash
# Get the GGID IdP metadata XML
curl $API/saml/metadata -o /etc/grafana/ggid-idp-metadata.xml
```

### Step 3: Configure Grafana

```ini
# /etc/grafana/grafana.ini

[auth.saml]
enabled = true
private_key_path = /etc/grafana/sp-private.key
certificate_path = /etc/grafana/sp-cert.pem
idp_metadata_path = /etc/grafana/ggid-idp-metadata.xml

# Map GGID roles to Grafana roles
role_values_editor = "editor,developer"
role_values_admin = "admin"
role_values_grafana_admin = "superadmin"

# Auto-create users on first login
auto_sign_up = true

# Attribute mapping
name_id_format = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
email_attribute_name = "email"
name_attribute_name = "displayName"
login_attribute_name = "username"
```

### Step 4: Verify

```bash
# Test SP-initiated SSO
# Navigate to: https://grafana.example.com/login
# Click "Sign in with SAML"
# Should redirect to GGID login page
# After login, redirect back to Grafana authenticated
```

---

## WebAuthn Registration in Vanilla JS

A complete browser-side WebAuthn registration flow using the GGID API. No
libraries needed — uses the native Web Credentials API.

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Register Passkey</title>
</head>
<body>
    <h1>Register a Passkey</h1>
    <button id="register-btn">Register Passkey</button>
    <p id="status"></p>
    <p id="error"></p>

<script>
const API = 'https://iam.example.com';
const TENANT_ID = '00000000-0000-0000-0000-000000000001';
let jwt = localStorage.getItem('access_token'); // User's existing JWT

// --- Base64URL helpers ---
function base64urlToBuffer(base64url) {
    const pad = '='.repeat((4 - base64url.length % 4) % 4);
    const base64 = (base64url + pad).replace(/-/g, '+').replace(/_/g, '/');
    const raw = atob(base64);
    const buffer = new Uint8Array(raw.length);
    for (let i = 0; i < raw.length; i++) {
        buffer[i] = raw.charCodeAt(i);
    }
    return buffer.buffer;
}

function bufferToBase64url(buffer) {
    const bytes = new Uint8Array(buffer);
    let str = '';
    for (const byte of bytes) {
        str += String.fromCharCode(byte);
    }
    return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

// --- Registration Flow ---
async function registerPasskey() {
    const statusEl = document.getElementById('status');
    const errorEl = document.getElementById('error');
    errorEl.textContent = '';
    statusEl.textContent = 'Starting registration...';

    try {
        // Step 1: Get challenge from GGID
        const beginResp = await fetch(`${API}/api/v1/auth/webauthn/register/begin`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${jwt}`,
                'Content-Type': 'application/json',
                'X-Tenant-ID': TENANT_ID,
            },
            body: JSON.stringify({}),
        });

        if (!beginResp.ok) throw new Error(`Begin failed: ${beginResp.status}`);
        const options = await beginResp.json();

        statusEl.textContent = 'Please interact with your authenticator...';

        // Step 2: Create credential via browser WebAuthn API
        const publicKey = {
            challenge: base64urlToBuffer(options.challenge),
            rp: options.rp,
            user: {
                id: base64urlToBuffer(options.user.id),
                name: options.user.name,
                displayName: options.user.displayName,
            },
            pubKeyCredParams: options.pubKeyCredParams,
            timeout: options.timeout || 60000,
            attestation: options.attestation || 'none',
        };

        if (options.excludeCredentials) {
            publicKey.excludeCredentials = options.excludeCredentials.map(c => ({
                type: c.type,
                id: base64urlToBuffer(c.id),
            }));
        }

        const credential = await navigator.credentials.create({ publicKey });

        statusEl.textContent = 'Verifying with server...';

        // Step 3: Send credential to GGID for verification
        const finishResp = await fetch(`${API}/api/v1/auth/webauthn/register/finish`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${jwt}`,
                'Content-Type': 'application/json',
                'X-Tenant-ID': TENANT_ID,
            },
            body: JSON.stringify({
                id: credential.id,
                rawId: bufferToBase64url(credential.rawId),
                type: credential.type,
                response: {
                    attestationObject: bufferToBase64url(
                        credential.response.attestationObject
                    ),
                    clientDataJSON: bufferToBase64url(
                        credential.response.clientDataJSON
                    ),
                    transports: credential.response.getTransports ?
                        credential.response.getTransports() : [],
                },
            }),
        });

        if (!finishResp.ok) {
            const err = await finishResp.json();
            throw new Error(err.error || `Finish failed: ${finishResp.status}`);
        }

        const result = await finishResp.json();
        statusEl.textContent = `Success! Registered "${result.name}". Credential ID: ${result.credential_id}`;
    } catch (err) {
        errorEl.textContent = `Error: ${err.message}`;
        statusEl.textContent = '';
    }
}

document.getElementById('register-btn').addEventListener('click', registerPasskey);
</script>
</body>
</html>
```

### Authentication Flow (Login with Passkey)

```javascript
async function loginWithPasskey(username) {
    // Step 1: Get challenge
    const beginResp = await fetch(`${API}/api/v1/auth/webauthn/login/begin`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': TENANT_ID,
        },
        body: JSON.stringify({ username }),
    });
    const options = await beginResp.json();

    // Step 2: Get assertion from authenticator
    const publicKey = {
        challenge: base64urlToBuffer(options.challenge),
        rpId: options.rpId,
        timeout: options.timeout || 60000,
        userVerification: options.userVerification || 'preferred',
    };

    if (options.allowCredentials && options.allowCredentials.length > 0) {
        publicKey.allowCredentials = options.allowCredentials.map(c => ({
            type: c.type,
            id: base64urlToBuffer(c.id),
            transports: c.transports || [],
        }));
    }

    const assertion = await navigator.credentials.get({ publicKey });

    // Step 3: Verify and get JWT
    const finishResp = await fetch(`${API}/api/v1/auth/webauthn/login/finish`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': TENANT_ID,
        },
        body: JSON.stringify({
            id: assertion.id,
            rawId: bufferToBase64url(assertion.rawId),
            type: assertion.type,
            response: {
                authenticatorData: bufferToBase64url(
                    assertion.response.authenticatorData
                ),
                clientDataJSON: bufferToBase64url(
                    assertion.response.clientDataJSON
                ),
                signature: bufferToBase64url(assertion.response.signature),
                userHandle: assertion.response.userHandle ?
                    bufferToBase64url(assertion.response.userHandle) : null,
            },
        }),
    });

    const tokens = await finishResp.json();
    localStorage.setItem('access_token', tokens.access_token);
    localStorage.setItem('refresh_token', tokens.refresh_token);
    return tokens;
}
```

---

## References

- [API Reference](./api-reference.md) — REST endpoints
- [SDK Cookbook](./sdk-cookbook.md) — More integration recipes
- [WebAuthn Guide](./webauthn-guide.md) — Passkey implementation
- [SAML Guide](./saml-guide.md) — SSO configuration
- [SCIM Guide](./scim-guide.md) — Provisioning API
