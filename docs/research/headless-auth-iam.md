# Headless and CLI Authentication for IAM Systems

> **Research Document** — GGID IAM Suite
> **Topic**: Authentication patterns for CLI tools, headless servers, CI/CD pipelines, and SDK consumers
> **Status**: Active Research
> **Last Updated**: 2025

---

## Table of Contents

1. [Device Authorization Flow (RFC 8628)](#1-device-authorization-flow-rfc-8628)
2. [PKCE for Native/CLI Apps](#2-pkce-for-nativecli-apps)
3. [Browser Launch Patterns](#3-browser-launch-patterns)
4. [Token Storage in OS Keychain](#4-token-storage-in-os-keychain)
5. [Token Refresh for CLI](#5-token-refresh-for-cli)
6. [SSH-Style Config](#6-ssh-style-config)
7. [Service Account Auth for CI/CD](#7-service-account-auth-for-cicd)
8. [SDK Auth Patterns](#8-sdk-auth-patterns)
9. [GGID CLI/SDK Auth Guide](#9-ggid-clisdk-auth-guide)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. Device Authorization Flow (RFC 8628)

### 1.1 Overview

The Device Authorization Grant (RFC 8628) is the gold standard for authenticating
devices that lack a rich browser input — CLI tools, IoT devices, smart TVs, and
headless servers. Instead of requiring the user to copy-paste credentials into a
terminal, the flow delegates authentication to a secondary device (typically a
phone or laptop with a browser).

**Flow diagram:**

```
  CLI Device                    GGID OAuth Server              User's Browser
      |                               |                              |
      |--- POST /device_authorization -->                           |
      |<-- device_code, user_code, ---|                             |
      |    verification_uri, interval  |                             |
      |                               |                              |
      |--- Display user_code & URL ---|----------------------------->|
      |                               |<---- User enters code -------|
      |                               |<---- User approves -----------|
      |                               |                              |
      |--- POST /token (poll) -------->|                             |
      |<-- authorization_pending ------|                             |
      |                               |                              |
      |--- POST /token (poll) -------->|                             |
      |<-- access_token + refresh -----|                             |
```

### 1.2 Why Device Flow Is Best for CLI/Headless

| Factor | Device Flow | Password Grant | PKCE Browser Flow |
|--------|-------------|----------------|-------------------|
| No password in terminal | Yes | No | Yes |
| Works without local browser | Yes | Yes | No |
| Works over SSH | Yes | Yes | No |
| User sees consent screen | Yes | No | Yes |
| MFA support | Yes (on approval device) | Requires secondary step | Yes |
| Phishing-resistant | Moderate | No | Moderate |

### 1.3 Polling Protocol

RFC 8628 §3.4 defines four error responses during polling:

- **`authorization_pending`** — User has not yet approved. Continue polling at `interval`.
- **`slow_down`** — Client is polling too fast. Increase interval by 5 seconds.
- **`expired_token`** — Device code has expired. Restart the flow.
- **`access_denied`** — User denied the request. Abort.

### 1.4 Server-Side Implementation (GGID Existing)

GGID already implements device flow server-side. The key components:

**`/api/v1/oauth/device_authorization`** — Creates device_code + user_code:

```go
// Source: services/oauth/internal/service/oauth_service.go (existing)

type DeviceAuthorizationResponse struct {
    DeviceCode      string `json:"device_code"`
    UserCode        string `json:"user_code"`
    VerificationURI string `json:"verification_uri"`
    ExpiresIn       int    `json:"expires_in"`
    Interval        int    `json:"interval"`
}

func (s *OAuthService) CreateDeviceAuthorization(req *DeviceAuthorizationRequest) (*DeviceAuthorizationResponse, error) {
    deviceCode := generateDeviceCode(40)
    userCode := generateUserCode()

    info := &DeviceCodeInfo{
        DeviceCode: deviceCode,
        UserCode:   userCode,
        ClientID:   req.ClientID,
        TenantID:   req.TenantID,
        Scope:      req.Scope,
        Status:     "pending",
        CreatedAt:  time.Now(),
        ExpiresAt:  time.Now().Add(15 * time.Minute),
    }

    deviceCodeMu.Lock()
    deviceCodeStore[deviceCode] = info
    userCodeIndex[userCode] = deviceCode
    deviceCodeMu.Unlock()

    return &DeviceAuthorizationResponse{
        DeviceCode:      deviceCode,
        UserCode:        userCode,
        VerificationURI: req.Issuer + "/device",
        ExpiresIn:       900, // 15 minutes
        Interval:        5,   // 5 seconds between polls
    }, nil
}
```

**Token endpoint polling** with proper RFC 8628 error codes:

```go
// Source: services/oauth/internal/server/server.go (existing)

case "urn:ietf:params:oauth:grant-type:device_code":
    resp, tokenErr = oauthSvc.PollDeviceToken(ctx, r.FormValue("device_code"), clientID)
    if tokenErr != nil {
        errMsg := tokenErr.Error()
        switch errMsg {
        case "authorization_pending":
            writeJSON(w, http.StatusBadRequest, map[string]string{"error": "authorization_pending"})
        case "slow_down":
            writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slow_down"})
        case "expired_token":
            writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expired_token"})
        case "access_denied":
            writeJSON(w, http.StatusBadRequest, map[string]string{"error": "access_denied"})
        default:
            writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": errMsg})
        }
        return
    }
```

### 1.5 CLI Client Implementation

```go
package cli

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"
)

// DeviceClient performs RFC 8628 device authorization from a CLI.
type DeviceClient struct {
    BaseURL  string
    ClientID string
    HTTP     *http.Client
}

type DeviceAuthResponse struct {
    DeviceCode      string `json:"device_code"`
    UserCode        string `json:"user_code"`
    VerificationURI string `json:"verification_uri"`
    ExpiresIn       int    `json:"expires_in"`
    Interval        int    `json:"interval"`
}

type TokenResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int    `json:"expires_in"`
    TokenType    string `json:"token_type"`
}

// StartDeviceFlow initiates device authorization and returns instructions for the user.
func (dc *DeviceClient) StartDeviceFlow(ctx context.Context, tenantID, scope string) (*DeviceAuthResponse, error) {
    form := url.Values{}
    form.Set("client_id", dc.ClientID)
    form.Set("scope", scope)

    req, err := http.NewRequestWithContext(ctx, http.MethodPost,
        dc.BaseURL+"/api/v1/oauth/device_authorization", strings.NewReader(form.Encode()))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("X-Tenant-ID", tenantID)

    resp, err := dc.HTTP.Do(req)
    if err != nil {
        return nil, fmt.Errorf("device authorization request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("device authorization failed (status %d): %s", resp.StatusCode, body)
    }

    var result DeviceAuthResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("decode device auth response: %w", err)
    }
    return &result, nil
}

// PollForToken polls the token endpoint until the user approves or the code expires.
// It handles slow_down by increasing the interval per RFC 8628 §3.5.
func (dc *DeviceClient) PollForToken(ctx context.Context, deviceCode, tenantID string, interval, expiresIn int) (*TokenResponse, error) {
    currentInterval := time.Duration(interval) * time.Second
    deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)

    for time.Now().Before(deadline) {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(currentInterval):
        }

        form := url.Values{}
        form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
        form.Set("device_code", deviceCode)
        form.Set("client_id", dc.ClientID)

        req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
            dc.BaseURL+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        req.Header.Set("X-Tenant-ID", tenantID)

        resp, err := dc.HTTP.Do(req)
        if err != nil {
            continue // transient error, retry
        }

        var tokenResp TokenResponse
        var errResp struct {
            Error string `json:"error"`
        }

        if resp.StatusCode == http.StatusOK {
            json.NewDecoder(resp.Body).Decode(&tokenResp)
            resp.Body.Close()
            return &tokenResp, nil
        }

        json.NewDecoder(resp.Body).Decode(&errResp)
        resp.Body.Close()

        switch errResp.Error {
        case "authorization_pending":
            // Continue polling at current interval
        case "slow_down":
            // RFC 8628 §3.5: increase interval by 5 seconds
            currentInterval += 5 * time.Second
        case "expired_token":
            return nil, fmt.Errorf("device code expired, please restart authentication")
        case "access_denied":
            return nil, fmt.Errorf("user denied the authorization request")
        }
    }

    return nil, fmt.Errorf("device authorization timed out")
}
```

### 1.6 Production Considerations

- **Redis-backed store**: GGID currently uses in-memory `deviceCodeStore`. Production
  should migrate to Redis with TTL for multi-instance deployments.
- **User code format**: RFC 8628 recommends 8-character codes in `XXXX-XXXX` format for
  readability. GGID's `generateUserCode()` implements this.
- **Verification URI complete**: Return `verification_uri_complete` with embedded user_code
  for QR code / deep link flows.
- **Rate limiting**: Device authorization endpoint should be rate-limited per client_id.

---

## 2. PKCE for Native/CLI Apps

### 2.1 Why PKCE Is Mandatory for Public Clients

RFC 7636 introduced PKCE (Proof Key for Code Exchange) to protect authorization codes
from interception. OAuth 2.1 makes PKCE mandatory for **all** clients, but it is
especially critical for public clients (CLI tools, mobile apps, SPAs) that cannot
securely store a client secret.

**Attack scenario without PKCE:**
1. CLI app opens browser to `https://ggid/oauth/authorize?client_id=...&redirect_uri=http://localhost:8080/callback`
2. Attacker on the same machine intercepts the authorization code via a malicious
   app registering the same custom scheme or by racing the localhost callback.
3. Attacker exchanges the stolen code for tokens.

**PKCE mitigates this** because the attacker cannot produce the correct `code_verifier`
that matches the `code_challenge` sent during the authorization request.

### 2.2 Code Verifier and Challenge Generation (S256)

```
code_verifier  = random 43-128 character string from [A-Z / a-z / 0-9 / - . _ ~]
code_challenge = BASE64URL-ENCODE(SHA256(ASCII(code_verifier)))
```

GGID already implements S256 verification server-side in `rfc7523.go`:

```go
// Source: services/oauth/internal/service/rfc7523.go (existing)

func VerifyCodeChallenge(challenge, verifier, method string) bool {
    switch method {
    case "S256", "":
        computed := hashTokenSHA256(verifier)
        return subtleConstantCompare(computed, challenge)
    case "plain":
        return subtleConstantCompare(verifier, challenge)
    default:
        return false
    }
}
```

### 2.3 CLI PKCE Client Implementation

```go
package cli

import (
    "context"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "net/http"
    "net/url"
)

// PKCEVerifier holds the PKCE code verifier and challenge.
type PKCEVerifier struct {
    Verifier  string
    Challenge string
    Method    string
}

// GeneratePKCE creates a random code_verifier and derives the S256 code_challenge.
func GeneratePKCE() (*PKCEVerifier, error) {
    // RFC 7636 §4.1: 43-128 characters from unreserved set
    const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
    buf := make([]byte, 64)
    if _, err := rand.Read(buf); err != nil {
        return nil, fmt.Errorf("generate PKCE verifier: %w", err)
    }
    for i := range buf {
        buf[i] = charset[int(buf[i])%len(charset)]
    }
    verifier := string(buf)

    h := sha256.Sum256([]byte(verifier))
    challenge := base64.RawURLEncoding.EncodeToString(h[:])

    return &PKCEVerifier{
        Verifier:  verifier,
        Challenge: challenge,
        Method:    "S256",
    }, nil
}

// AuthURLPKCE builds the authorization URL with PKCE challenge.
func (dc *DeviceClient) AuthURLPKCE(baseURL, clientID, redirectURI, scope string, pkce *PKCEVerifier) string {
    params := url.Values{
        "response_type":         {"code"},
        "client_id":             {clientID},
        "redirect_uri":          {redirectURI},
        "scope":                 {scope},
        "code_challenge":        {pkce.Challenge},
        "code_challenge_method": {pkce.Method},
    }
    return baseURL + "/api/v1/oauth/authorize?" + params.Encode()
}

// ExchangeCodeWithPKCE exchanges an authorization code for tokens using the PKCE verifier.
func (dc *DeviceClient) ExchangeCodeWithPKCE(ctx context.Context, code, redirectURI string, pkce *PKCEVerifier) (*TokenResponse, error) {
    form := url.Values{}
    form.Set("grant_type", "authorization_code")
    form.Set("code", code)
    form.Set("redirect_uri", redirectURI)
    form.Set("client_id", dc.ClientID)
    form.Set("code_verifier", pkce.Verifier)

    req, err := http.NewRequestWithContext(ctx, http.MethodPost,
        dc.BaseURL+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := dc.HTTP.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, body)
    }

    var tokenResp TokenResponse
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return nil, err
    }
    return &tokenResp, nil
}
```

---

## 3. Browser Launch Patterns

### 3.1 Launching the System Browser

When a CLI tool has access to a local browser (desktop terminal, not SSH), the PKCE
flow with a local callback server provides the best UX. The CLI:

1. Starts a local HTTP server on a random port.
2. Launches the system browser to the authorization URL.
3. Waits for the OAuth redirect to `http://localhost:PORT/callback`.
4. Extracts the authorization code from the callback.
5. Exchanges the code for tokens (with PKCE verifier).

### 3.2 Random Port Allocation

Using port 0 lets the OS assign an available port, avoiding conflicts:

```go
package cli

import (
    "fmt"
    "net"
    "net/http"
    "os/exec"
    "runtime"
)

// CallbackResult holds the authorization code or error from the browser redirect.
type CallbackResult struct {
    Code  string
    State string
    Error string
}

// LocalCallbackServer starts a temporary HTTP server to receive the OAuth redirect.
// It returns the callback result and shuts down after the first request.
func LocalCallbackServer() (string, <-chan CallbackResult, func(), error) {
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        return "", nil, nil, fmt.Errorf("start callback listener: %w", err)
    }
    port := listener.Addr().(*net.TCPAddr).Port
    redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

    resultCh := make(chan CallbackResult, 1)
    srv := &http.Server{}

    srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/callback" {
            http.NotFound(w, r)
            return
        }

        result := CallbackResult{
            Code:  r.URL.Query().Get("code"),
            State: r.URL.Query().Get("state"),
            Error: r.URL.Query().Get("error"),
        }

        // Show success page to the user
        if result.Error != "" {
            w.Header().Set("Content-Type", "text/html")
            w.WriteHeader(http.StatusBadRequest)
            fmt.Fprintf(w, "<h1>Authentication Failed</h1><p>%s</p><p>You can close this tab.</p>", result.Error)
        } else {
            w.Header().Set("Content-Type", "text/html")
            w.WriteHeader(http.StatusOK)
            w.Write([]byte(`<h1>Authentication Successful</h1><p>You can close this tab and return to your terminal.</p>`))
        }

        resultCh <- result
        // Shutdown in a goroutine to avoid blocking the response
        go srv.Shutdown(r.Context())
    })

    go srv.Serve(listener)

    cleanup := func() {
        srv.Close()
        close(resultCh)
    }

    return redirectURI, resultCh, cleanup, nil
}

// OpenBrowser opens the system default browser to the given URL.
// Returns an error if no browser can be launched.
func OpenBrowser(url string) error {
    switch runtime.GOOS {
    case "darwin":
        return exec.Command("open", url).Start()
    case "windows":
        return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
    case "linux":
        // Try xdg-open, fall back to common browsers
        if _, err := exec.LookPath("xdg-open"); err == nil {
            return exec.Command("xdg-open", url).Start()
        }
        return fmt.Errorf("no browser launcher found (xdg-open missing)")
    default:
        return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
    }
}
```

### 3.3 Handling No-Browser Environments

When running over SSH or in a headless container, the CLI should:

1. Print the authorization URL for the user to open manually.
2. Fall back to the device flow (RFC 8628) if browser launch fails.
3. Support `--no-browser` flag for explicit headless mode.

```go
// Authenticate tries PKCE browser flow first, falls back to device flow.
func Authenticate(ctx context.Context, cfg *Config) (*TokenResponse, error) {
    if cfg.NoBrowser {
        return authenticateDeviceFlow(ctx, cfg)
    }

    // Try browser flow
    redirectURI, resultCh, cleanup, err := LocalCallbackServer()
    if err == nil {
        defer cleanup()

        pkce, _ := GeneratePKCE()
        authURL := cfg.OAuthClient.AuthURLPKCE(cfg.IssuerURL, cfg.ClientID, redirectURI, cfg.Scopes, pkce)

        if openErr := OpenBrowser(authURL); openErr != nil {
            // Browser launch failed — print URL and wait
            fmt.Printf("Open this URL in your browser:\n  %s\n", authURL)
        } else {
            fmt.Println("Browser opened. Waiting for authentication...")
        }

        select {
        case result := <-resultCh:
            if result.Error != "" {
                return nil, fmt.Errorf("auth error: %s", result.Error)
            }
            return cfg.OAuthClient.ExchangeCodeWithPKCE(ctx, result.Code, redirectURI, pkce)
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }

    // Fall back to device flow
    return authenticateDeviceFlow(ctx, cfg)
}
```

---

## 4. Token Storage in OS Keychain

### 4.1 Why OS Keychain Beats Plaintext Files

Storing OAuth tokens in plaintext config files (e.g., `~/.ggid/token.json`) is a
significant security risk:

| Risk | Plaintext File | OS Keychain |
|------|---------------|-------------|
| Readable by other users | Yes (if permissions wrong) | No (keychain ACL) |
| Indexed by search/malware | Yes | No |
| Survives accidental commit to git | Common | N/A |
| Encrypted at rest | No | Yes |
| Requires user interaction to access | No | Configurable |

OS keychains provide hardware-backed encryption, access control, and audit trails.

### 4.2 Cross-Platform Keychain Access

- **macOS**: Keychain (via `security` CLI or CGo bindings)
- **Windows**: Credential Manager (via `cmdkey` or Win32 API)
- **Linux**: Secret Service / libsecret (via D-Bus) or fallback to encrypted file

```go
package keychain

import (
    "fmt"
    "os"
    "os/exec"
    "runtime"
    "strings"
)

// Keychain provides a cross-platform interface for secure token storage.
type Keychain interface {
    Set(service, account, data string) error
    Get(service, account string) (string, error)
    Delete(service, account string) error
}

// New returns the platform-appropriate keychain implementation.
func New() Keychain {
    switch runtime.GOOS {
    case "darwin":
        return &macOSKeychain{}
    case "windows":
        return &windowsKeychain{}
    case "linux":
        return &linuxKeychain{}
    default:
        return &fileKeychain{} // encrypted fallback
    }
}

// --- macOS Keychain ---

type macOSKeychain struct{}

func (k *macOSKeychain) Set(service, account, data string) error {
    cmd := exec.Command("security", "add-generic-password",
        "-a", account,
        "-s", service,
        "-w", data,
        "-U", // update if exists
    )
    return cmd.Run()
}

func (k *macOSKeychain) Get(service, account string) (string, error) {
    cmd := exec.Command("security", "find-generic-password",
        "-a", account,
        "-s", service,
        "-w",
    )
    out, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("keychain get failed: %w", err)
    }
    return strings.TrimSpace(string(out)), nil
}

func (k *macOSKeychain) Delete(service, account string) error {
    cmd := exec.Command("security", "delete-generic-password",
        "-a", account,
        "-s", service,
    )
    return cmd.Run()
}

// --- Windows Credential Manager ---

type windowsKeychain struct{}

func (k *windowsKeychain) Set(service, account, data string) error {
    // Uses PowerShell to store in Credential Manager
    cmd := exec.Command("powershell", "-Command",
        fmt.Sprintf(`cmdkey /generic:%s /user:%s /pass:%s`, service, account, data))
    return cmd.Run()
}

func (k *windowsKeychain) Get(service, account string) (string, error) {
    cmd := exec.Command("powershell", "-Command",
        fmt.Sprintf(`(cmdkey /list:%s /pass).Split(':')[1].Trim()`, service))
    out, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("credential manager get failed: %w", err)
    }
    return strings.TrimSpace(string(out)), nil
}

func (k *windowsKeychain) Delete(service, account string) error {
    cmd := exec.Command("cmdkey", "/delete:"+service)
    return cmd.Run()
}

// --- Linux Secret Service (libsecret) ---

type linuxKeychain struct{}

func (k *linuxKeychain) Set(service, account, data string) error {
    cmd := exec.Command("secret-tool", "store",
        "--label", service,
        "service", service,
        "account", account,
    )
    cmd.Stdin = strings.NewReader(data)
    return cmd.Run()
}

func (k *linuxKeychain) Get(service, account string) (string, error) {
    cmd := exec.Command("secret-tool", "lookup",
        "service", service,
        "account", account,
    )
    out, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("secret service get failed: %w", err)
    }
    return strings.TrimSpace(string(out)), nil
}

func (k *linuxKeychain) Delete(service, account string) error {
    cmd := exec.Command("secret-tool", "clear",
        "service", service,
        "account", account,
    )
    return cmd.Run()
}

// --- Fallback: Encrypted File ---

type fileKeychain struct{}

func (k *fileKeychain) path() string {
    home, _ := os.UserHomeDir()
    return home + "/.ggid/token.enc"
}

// In production, encrypt with a key derived from a passphrase or machine ID.
// This is a simplified placeholder.
func (k *fileKeychain) Set(service, account, data string) error {
    return os.WriteFile(k.path(), []byte(data), 0600)
}

func (k *fileKeychain) Get(service, account string) (string, error) {
    data, err := os.ReadFile(k.path())
    if err != nil {
        return "", err
    }
    return string(data), nil
}

func (k *fileKeychain) Delete(service, account string) error {
    return os.Remove(k.path())
}
```

### 4.3 Usage in GGID CLI

```go
const (
    keychainService = "ggid-cli"
    keychainAccount = "default"
)

func storeTokens(tokens *TokenResponse) error {
    kc := keychain.New()
    data, _ := json.Marshal(tokens)
    return kc.Set(keychainService, keychainAccount, string(data))
}

func loadTokens() (*TokenResponse, error) {
    kc := keychain.New()
    data, err := kc.Get(keychainService, keychainAccount)
    if err != nil {
        return nil, fmt.Errorf("not authenticated, run 'ggid auth login' first")
    }
    var tokens TokenResponse
    return &tokens, json.Unmarshal([]byte(data), &tokens)
}
```

---

## 5. Token Refresh for CLI

### 5.1 Long-Lived CLI Sessions

CLI tools need long-lived sessions that survive token expiry. The pattern:

1. Store both `access_token` (short-lived, ~1h) and `refresh_token` (long-lived, ~30d) in keychain.
2. Before each API call, check if the access token will expire soon.
3. If expiring, silently refresh using the refresh token.
4. If the refresh token is revoked, prompt the user to re-authenticate.

### 5.2 CLI Token Manager with Auto-Refresh

```go
package cli

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"
)

// TokenManager handles token lifecycle: caching, auto-refresh, revocation handling.
type TokenManager struct {
    kc          keychain.Keychain
    oauthClient *DeviceClient
    tenantID    string
    refreshBuf  time.Duration // refresh this far before expiry
    mu          sync.Mutex
    cached      *ManagedToken
}

type ManagedToken struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at"`
}

func NewTokenManager(kc keychain.Keychain, client *DeviceClient, tenantID string) *TokenManager {
    return &TokenManager{
        kc:          kc,
        oauthClient: client,
        tenantID:    tenantID,
        refreshBuf:  5 * time.Minute, // refresh 5 min before expiry
    }
}

// GetValidToken returns a non-expired access token, refreshing if necessary.
func (tm *TokenManager) GetValidToken(ctx context.Context) (string, error) {
    tm.mu.Lock()
    defer tm.mu.Unlock()

    // Load from keychain if not cached
    if tm.cached == nil {
        if err := tm.load(); err != nil {
            return "", fmt.Errorf("not logged in, run 'ggid auth login': %w", err)
        }
    }

    // Check if refresh is needed
    if time.Until(tm.cached.ExpiresAt) < tm.refreshBuf {
        if err := tm.refresh(ctx); err != nil {
            return "", fmt.Errorf("session expired, please re-authenticate: %w", err)
        }
    }

    return tm.cached.AccessToken, nil
}

func (tm *TokenManager) refresh(ctx context.Context) error {
    form := url.Values{}
    form.Set("grant_type", "refresh_token")
    form.Set("refresh_token", tm.cached.RefreshToken)
    form.Set("client_id", tm.oauthClient.ClientID)

    req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
        tm.oauthClient.BaseURL+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("X-Tenant-ID", tm.tenantID)

    resp, err := tm.oauthClient.HTTP.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
        // Refresh token revoked or expired
        tm.cached = nil
        tm.kc.Delete(keychainService, keychainAccount)
        return fmt.Errorf("refresh token revoked")
    }

    var tokenResp TokenResponse
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return err
    }

    tm.cached = &ManagedToken{
        AccessToken:  tokenResp.AccessToken,
        RefreshToken: tokenResp.RefreshToken,
        ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
    }

    return tm.save()
}

func (tm *TokenManager) load() error {
    data, err := tm.kc.Get(keychainService, keychainAccount)
    if err != nil {
        return err
    }
    return json.Unmarshal([]byte(data), tm.cached)
}

func (tm *TokenManager) save() error {
    data, _ := json.Marshal(tm.cached)
    return tm.kc.Set(keychainService, keychainAccount, string(data))
}

// Logout clears stored tokens and revokes the refresh token.
func (tm *TokenManager) Logout(ctx context.Context) error {
    tm.mu.Lock()
    defer tm.mu.Unlock()

    if tm.cached != nil {
        // Best-effort revoke
        form := url.Values{}
        form.Set("token", tm.cached.RefreshToken)
        form.Set("token_type_hint", "refresh_token")

        req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
            tm.oauthClient.BaseURL+"/api/v1/oauth/revoke", strings.NewReader(form.Encode()))
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        tm.oauthClient.HTTP.Do(req)
    }

    tm.cached = nil
    return tm.kc.Delete(keychainService, keychainAccount)
}
```

---

## 6. SSH-Style Config

### 6.1 Profile-Based Auth Configuration

Borrowing from the SSH config model (`~/.ssh/config`), GGID CLI should support
multiple named profiles in `~/.ggid/config`:

```ini
# ~/.ggid/config

[default]
issuer = https://iam.ggid.dev
tenant_id = 00000000-0000-0000-0000-000000000001
client_id = ggid-cli-public

[prod]
issuer = https://iam.ggid.example.com
tenant_id = 00000000-0000-0000-0000-000000000002
client_id = ggid-cli-public
scopes = users:read users:write roles:read

[staging]
issuer = https://staging.iam.ggid.example.com
tenant_id = 00000000-0000-0000-0000-000000000003
client_id = ggid-cli-staging
```

### 6.2 Profile-Based Config Management

```go
package config

import (
    "fmt"
    "os"
    "path/filepath"

    "gopkg.in/ini.v1"
)

type Profile struct {
    Issuer   string
    TenantID string
    ClientID string
    Scopes   string
}

type Config struct {
    DefaultProfile string
    Profiles       map[string]*Profile
    path           string
}

// Load reads the GGID config file from ~/.ggid/config.
func Load() (*Config, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }
    path := filepath.Join(home, ".ggid", "config")

    cfg, err := ini.Load(path)
    if err != nil {
        if os.IsNotExist(err) {
            return &Config{Profiles: make(map[string]*Profile), path: path}, nil
        }
        return nil, err
    }

    config := &Config{
        Profiles: make(map[string]*Profile),
        path:     path,
    }

    for _, section := range cfg.Sections() {
        name := section.Name()
        if name == "DEFAULT" {
            continue
        }
        config.Profiles[name] = &Profile{
            Issuer:   section.Key("issuer").String(),
            TenantID: section.Key("tenant_id").String(),
            ClientID: section.Key("client_id").String(),
            Scopes:   section.Key("scopes").String(),
        }
    }

    // Determine default profile
    if d := cfg.Section("DEFAULT").Key("default_profile").String(); d != "" {
        config.DefaultProfile = d
    } else if len(config.Profiles) > 0 {
        for name := range config.Profiles {
            config.DefaultProfile = name
            break
        }
    }

    return config, nil
}

// GetProfile returns the named profile or the default if name is empty.
func (c *Config) GetProfile(name string) (*Profile, error) {
    if name == "" {
        name = c.DefaultProfile
    }
    if name == "" {
        return nil, fmt.Errorf("no profile configured, run 'ggid auth login' first")
    }
    p, ok := c.Profiles[name]
    if !ok {
        return nil, fmt.Errorf("profile %q not found", name)
    }
    return p, nil
}

// Save writes the config file.
func (c *Config) Save() error {
    cfg := ini.Empty()
    for name, p := range c.Profiles {
        section := cfg.Section(name)
        section.Key("issuer").SetValue(p.Issuer)
        section.Key("tenant_id").SetValue(p.TenantID)
        section.Key("client_id").SetValue(p.ClientID)
        section.Key("scopes").SetValue(p.Scopes)
    }
    if err := os.MkdirAll(filepath.Dir(c.path), 0700); err != nil {
        return err
    }
    return cfg.SaveTo(c.path)
}

// KeychainService returns the keychain service name for a profile.
// This ensures tokens are cached per-profile.
func KeychainService(profileName string) string {
    return fmt.Sprintf("ggid-cli:%s", profileName)
}
```

### 6.3 Per-Profile Token Isolation

Each profile stores its tokens under a separate keychain entry
(`ggid-cli:prod`, `ggid-cli:staging`), preventing cross-environment
token leakage.

---

## 7. Service Account Auth for CI/CD

### 7.1 Machine-to-Machine Authentication

CI/CD pipelines and server-to-server integrations cannot use interactive browser
flows. They need non-interactive authentication:

1. **client_credentials grant** with client secret (simplest).
2. **mTLS client_credentials** (most secure, RFC 8705).
3. **JWT assertion auth** (RFC 7523) — key-based, no shared secret.

### 7.2 Client Credentials with mTLS

```go
package serviceaccount

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "os"
)

// ServiceAccountAuth handles non-interactive authentication for CI/CD.
type ServiceAccountAuth struct {
    TokenEndpoint string
    ClientID      string
    HTTP          *http.Client
}

// NewWithMTLS creates a service account client using mutual TLS.
func NewWithMTLS(tokenEndpoint, clientID, certFile, keyFile, caFile string) (*ServiceAccountAuth, error) {
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, fmt.Errorf("load client cert: %w", err)
    }

    caPool := x509.NewCertPool()
    caData, err := os.ReadFile(caFile)
    if err != nil {
        return nil, fmt.Errorf("load CA cert: %w", err)
    }
    caPool.AppendCertsFromPEM(caData)

    httpClient := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                Certificates: []tls.Certificate{cert},
                RootCAs:      caPool,
                MinVersion:   tls.VersionTLS12,
            },
        },
    }

    return &ServiceAccountAuth{
        TokenEndpoint: tokenEndpoint,
        ClientID:      clientID,
        HTTP:          httpClient,
    }, nil
}

// GetToken exchanges client credentials for an access token.
func (sa *ServiceAccountAuth) GetToken(ctx context.Context, scope string) (*TokenResponse, error) {
    form := url.Values{}
    form.Set("grant_type", "client_credentials")
    form.Set("client_id", sa.ClientID)
    form.Set("scope", scope)

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, sa.TokenEndpoint, strings.NewReader(form.Encode()))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := sa.HTTP.Do(req)
    if err != nil {
        return nil, fmt.Errorf("token request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("token request failed (status %d): %s", resp.StatusCode, body)
    }

    var tokenResp TokenResponse
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return nil, err
    }
    return &tokenResp, nil
}
```

### 7.3 JWT Assertion Auth (RFC 7523)

For environments where mTLS is not available, JWT assertion auth (RFC 7523) provides
key-based client authentication without sharing a secret. GGID already validates these
server-side in `rfc7523.go`.

```go
// CreateJWTAssertion builds an RFC 7523 client_assertion JWT.
// The JWT is signed with the client's RSA private key.
func CreateJWTAssertion(clientID, tokenEndpoint string, privateKey *rsa.PrivateKey) (string, error) {
    now := time.Now()
    claims := jwt.MapClaims{
        "iss": clientID,
        "sub": clientID,
        "aud": tokenEndpoint,
        "iat": now.Unix(),
        "exp": now.Add(5 * time.Minute).Unix(),
        "jti": uuid.New().String(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    return token.SignedString(privateKey)
}

// ExchangeWithAssertion uses RFC 7523 JWT assertion for token exchange.
func (sa *ServiceAccountAuth) ExchangeWithAssertion(ctx context.Context, assertion, scope string) (*TokenResponse, error) {
    form := url.Values{}
    form.Set("grant_type", "client_credentials")
    form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
    form.Set("client_assertion", assertion)
    form.Set("scope", scope)

    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, sa.TokenEndpoint, strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := sa.HTTP.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var tokenResp TokenResponse
    json.NewDecoder(resp.Body).Decode(&tokenResp)
    return &tokenResp, nil
}
```

### 7.4 CI/CD Token Caching

CI environments have no OS keychain. Cache tokens using:

- **GitHub Actions**: `$GITHUB_ENV` or cache action with 1-hour TTL.
- **GitLab CI**: Cache artifacts or CI/CD variables.
- **Generic**: Temp file with `0600` permissions, cleaned up after job.

```yaml
# GitHub Actions example
- name: Get GGID Token
  env:
    GGID_CLIENT_ID: ${{ secrets.GGID_CLIENT_ID }}
    GGID_CLIENT_SECRET: ${{ secrets.GGID_CLIENT_SECRET }}
  run: |
    # Check cache first
    if [ -f /tmp/ggid_token.json ] && [ $(($(date +%s) - $(cat /tmp/ggid_token.ts))) -lt 3000 ]; then
      echo "Using cached token"
    else
      TOKEN=$(curl -s -X POST https://iam.ggid.dev/api/v1/oauth/token \
        -d "grant_type=client_credentials" \
        -d "client_id=$GGID_CLIENT_ID" \
        -d "client_secret=$GGID_CLIENT_SECRET" \
        -d "scope=users:read roles:read")
      echo "$TOKEN" > /tmp/ggid_token.json
      date +%s > /tmp/ggid_token.ts
    fi
    echo "GGID_TOKEN=$(jq -r .access_token /tmp/ggid_token.json)" >> $GITHUB_ENV
```

### 7.5 Secret Management Best Practices

- Never hardcode client secrets in source code or Docker images.
- Use CI/CD secret management (GitHub Actions secrets, GitLab CI variables).
- Rotate service account credentials quarterly.
- Use workload identity (OIDC federation) where available to avoid long-lived secrets entirely.
- Scope service account tokens minimally — read-only for CI that only deploys.

---

## 8. SDK Auth Patterns

### 8.1 Automatic Token Injection

SDK consumers should not have to manually attach tokens to every request. The SDK
should provide an auth layer that:

1. Obtains a token (from cache, refresh, or interactive login).
2. Injects `Authorization: Bearer <token>` into every outbound request.
3. Handles 401 responses by refreshing and retrying.

### 8.2 Go SDK Auth Interceptor (HTTP)

GGID's existing Go SDK (`sdk/go/client.go`) uses an API key via `X-API-Key` header.
For OAuth-based auth, the SDK should add a `RoundTripper` that injects and refreshes tokens:

```go
package ggid

import (
    "context"
    "net/http"
    "sync"
    "time"
)

// TokenSource provides access tokens for SDK requests.
// Implementations include keychain-backed CLI token source,
// client_credentials service account source, etc.
type TokenSource interface {
    Token(ctx context.Context) (string, error)
}

// AuthTransport is an http.RoundTripper that injects Bearer tokens.
type AuthTransport struct {
    Base    http.RoundTripper
    Source  TokenSource
    mu      sync.Mutex
    cached  string
    exp     time.Time
}

func (t *AuthTransport) transport() http.RoundTripper {
    if t.Base != nil {
        return t.Base
    }
    return http.DefaultTransport
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    ctx := req.Context()

    token, err := t.Source.Token(ctx)
    if err != nil {
        return nil, fmt.Errorf("sdk auth: failed to get token: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := t.transport().RoundTrip(req)
    if err != nil {
        return nil, err
    }

    // On 401, force refresh and retry once
    if resp.StatusCode == http.StatusUnauthorized {
        resp.Body.Close()

        // Force token refresh
        if refresher, ok := t.Source.(Refresher); ok {
            refresher.Invalidate(ctx)
        }

        token, err = t.Source.Token(ctx)
        if err != nil {
            return nil, err
        }

        // Clone request with new token
        req2 := req.Clone(ctx)
        req2.Header.Set("Authorization", "Bearer "+token)
        return t.transport().RoundTrip(req2)
    }

    return resp, nil
}

// Refresher is an optional interface for token sources that support invalidation.
type Refresher interface {
    Invalidate(ctx context.Context)
}

// WithTokenSource configures the SDK client to use OAuth Bearer auth.
func WithTokenSource(source TokenSource) Option {
    return func(c *Client) {
        c.httpClient.Transport = &AuthTransport{
            Base:   c.httpClient.Transport,
            Source: source,
        }
    }
}
```

### 8.3 Usage Pattern

```go
// CLI usage with keychain-backed tokens
client := ggid.New("https://iam.ggid.dev",
    ggid.WithTokenSource(keychainSource),
)

// CI/CD usage with service account
client := ggid.New("https://iam.ggid.dev",
    ggid.WithTokenSource(serviceAccountSource),
)
```

### 8.4 gRPC Interceptor for Service-to-Service

For gRPC-based internal services, the same pattern applies as a `UnaryClientInterceptor`:

```go
func AuthInterceptor(tokenSource TokenSource) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        token, err := tokenSource.Token(ctx)
        if err != nil {
            return fmt.Errorf("grpc auth: %w", err)
        }

        ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
        return invoker(ctx, method, req, reply, cc, opts...)
    }
}
```

---

## 9. GGID CLI/SDK Auth Guide

### 9.1 Current State Assessment

After reviewing the GGID codebase, here is what exists and what is missing:

#### What Exists (Server-Side)

| Feature | Location | Status |
|---------|----------|--------|
| Device Authorization Flow (RFC 8628) | `services/oauth/internal/service/oauth_service.go:1155-1354` | Fully implemented |
| Device auth endpoint | `services/oauth/internal/server/server.go:855-924` | `/api/v1/oauth/device_authorization` |
| Device approval endpoint | `services/oauth/internal/server/server.go:896-924` | `/api/v1/oauth/device/approve` |
| Token polling with RFC 8628 errors | `services/oauth/internal/server/server.go:351-368` | authorization_pending, slow_down, expired_token, access_denied |
| PKCE verification (S256 + plain) | `services/oauth/internal/service/rfc7523.go:93-115` | `VerifyCodeChallenge` |
| RFC 7523 JWT client assertion | `services/oauth/internal/service/rfc7523.go:30-91` | `ValidateClientAssertion` |
| Client credentials grant | `services/oauth/internal/server/server.go:344-350` | Implemented |
| Refresh token grant | `services/oauth/internal/server/server.go` | Implemented |
| Token revocation endpoint | `services/oauth/internal/server/server.go` | Implemented |
| mTLS support (RFC 8705) | `services/oauth/internal/service/jar_mtls.go` | Implemented |

#### What Exists (SDK-Side)

| Feature | Location | Status |
|---------|----------|--------|
| Go SDK client (CRUD + auth) | `sdk/go/client.go` | API key auth, Login/Logout/Refresh |
| Go SDK middleware | `sdk/go/middleware.go` | JWT verification, RequireRole/Scope |
| Go SDK JWKS caching | `sdk/go/client.go:421-457` | RSA key cache with TTL |
| Node SDK | `sdk/node/src/client.ts` | Basic client + JWT verify |
| Java SDK | `sdk/java/src/` | Basic skeleton |

#### What Is Missing

| Gap | Impact | Priority |
|-----|--------|----------|
| **CLI tool** (`ggid` binary) | No headless/CLI auth UX at all | P1 |
| **Go SDK device flow client** | SDK cannot initiate device auth | P1 |
| **Go SDK PKCE client** | SDK cannot do browser-based PKCE auth | P1 |
| **Go SDK OAuth transport** | No automatic token injection/refresh | P1 |
| **Keychain integration** | No secure token storage | P1 |
| **Profile config** | No multi-environment support | P2 |
| **Node SDK device flow** | Node SDK lacks device flow | P2 |
| **Service account helpers** | No CI/CD auth convenience | P2 |
| **Device flow: Redis store** | In-memory store won't scale | P2 |
| **`verification_uri_complete`** | No embedded user_code for QR/links | P3 |

### 9.2 Device Flow Server Review

The GGID device flow implementation is solid:

- Proper RFC 8628 error codes on the token endpoint.
- `slow_down` enforcement via `LastPoll` timestamp with 5-second minimum interval.
- 15-minute expiry with cleanup on poll.
- `XXXX-XXXX` format user codes for readability.
- In-memory store (needs Redis for production).

**Gap**: The `verification_uri` is always `{issuer}/device`. There is no
`verification_uri_complete` field that includes the user_code for deep-link
or QR code flows. This is an RFC 8628 SHOULD.

---

## 10. Gap Analysis & Recommendations

### 10.1 Action Items

#### Action 1: Build `ggid` CLI Binary (Effort: 2-3 weeks)

Create a standalone CLI tool (`cmd/ggid-cli/main.go`) that provides:

- `ggid auth login` — Interactive login (browser PKCE flow with device flow fallback)
- `ggid auth login --device` — Force device flow
- `ggid auth logout` — Revoke tokens and clear keychain
- `ggid auth status` — Show current profile and token expiry
- `ggid auth whoami` — Show authenticated user info
- `ggid config set-profile` — Manage profiles
- `ggid users list/create/delete` — CRUD through authenticated client
- `ggid roles list/create` — Role management

**Dependencies**: keychain package, profile config, device flow client, PKCE client.

#### Action 2: Add OAuth Transport to Go SDK (Effort: 3-5 days)

Add `WithTokenSource()` option and `AuthTransport` RoundTripper to `sdk/go/client.go`.
This enables any Go application consuming the SDK to authenticate via OAuth tokens
with automatic refresh, without managing tokens manually.

**Files to modify**: `sdk/go/client.go` (add `TokenSource` interface + transport).
**New file**: `sdk/go/auth.go` (transport + keychain-backed token source).

#### Action 3: Add Device Flow + PKCE Client to Go SDK (Effort: 1 week)

Add `sdk/go/device_flow.go` and `sdk/go/pkce.go` to the Go SDK, enabling SDK consumers
to implement CLI authentication without reimplementing RFC 8628 or RFC 7636.

**New files**:
- `sdk/go/device_flow.go` — DeviceAuthClient with StartDeviceFlow + PollForToken
- `sdk/go/pkce.go` — GeneratePKCE + ExchangeCodeWithPKCE
- `sdk/go/browser.go` — LocalCallbackServer + OpenBrowser

#### Action 4: Migrate Device Code Store to Redis (Effort: 2-3 days)

Replace the in-memory `deviceCodeStore` in `services/oauth/internal/service/oauth_service.go`
with a Redis-backed store using TTL keys. This is required for multi-instance OAuth
service deployments.

**Files to modify**: `services/oauth/internal/service/oauth_service.go` (add Redis
client, replace map operations with Redis GET/SET with TTL).

#### Action 5: Add `verification_uri_complete` (Effort: 0.5 day)

Update `DeviceAuthorizationResponse` to include `verification_uri_complete` field
containing the user_code embedded in the URL, per RFC 8628 §3.3.1. This enables
QR code and deep-link flows.

**Files to modify**: `services/oauth/internal/service/oauth_service.go`
(add field to struct + populate in `CreateDeviceAuthorization`).

### 10.2 Priority Matrix

| Action | Impact | Effort | ROI |
|--------|--------|--------|-----|
| CLI binary | High | High | Medium-term |
| SDK OAuth transport | High | Low | Immediate |
| SDK device flow + PKCE | High | Medium | Immediate |
| Redis device store | Medium | Low | Short-term |
| verification_uri_complete | Low | Low | Quick win |

### 10.3 Security Checklist for CLI Implementation

- [ ] PKCE mandatory for all public client auth (S256 only, reject plain)
- [ ] Tokens stored in OS keychain, never plaintext
- [ ] Config file permissions `0600`
- [ ] Refresh token rotation on each refresh (detect theft)
- [ ] `--no-browser` flag for headless/SSH environments
- [ ] Token expiry checked before every API call
- [ ] Graceful re-authentication prompt when refresh token revoked
- [ ] No tokens in command-line arguments (visible in process list / shell history)
- [ ] No tokens in log output
- [ ] Service account secrets from environment variables or files, never hardcoded

---

## Conclusion

GGID has a strong server-side foundation for headless authentication — the device
flow (RFC 8628), PKCE verification (RFC 7636), and JWT client assertion (RFC 7523)
are all implemented and tested. The primary gap is on the **client side**: there is
no CLI tool, no SDK-level OAuth transport, and no secure token storage. Building
these client-side capabilities will unlock headless/CI/CD authentication scenarios
that are essential for any production IAM system.

The recommended priority is to add the SDK OAuth transport first (lowest effort,
highest immediate value), then the device flow/PKCE client, then the full CLI binary.
The Redis migration and `verification_uri_complete` are quick wins that improve
production readiness.
