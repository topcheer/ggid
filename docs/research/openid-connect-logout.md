# OIDC Logout Implementation for IAM Systems

> Implementation-focused companion to `oidc-logout-spec-analysis.md`.
> Covers Go code patterns, session propagation architecture, and GGID-specific gaps.
> Date: 2025-01-20 · Status: Research

---

## Table of Contents

1. [Session Propagation Architecture](#1-session-propagation-architecture)
2. [RP-Initiated Logout Implementation](#2-rp-initiated-logout-implementation)
3. [Back-Channel Logout Implementation](#3-back-channel-logout-implementation)
4. [Front-Channel Logout: Iframe Timing Issues](#4-front-channel-logout-iframe-timing-issues)
5. [Logout Token JWT Verification](#5-logout-token-jwt-verification)
6. [GGID Logout Gap Analysis](#6-ggid-logout-gap-analysis)
7. [Implementation Roadmap](#7-implementation-roadmap)

---

## 1. Session Propagation Architecture

In a federated IAM system, a user holds sessions at the Identity Provider (OP)
and at every Relying Party (RP) they visited during their authenticated
session. Logging out of one does not guarantee the others are destroyed.
The fundamental challenge is **propagation**: how does a logout event at the
OP reach every downstream RP?

### 1.1 Distributed Session Stores

A production OP cannot rely on in-memory session state. Sessions must be
stored in a shared, distributed cache (Redis) so that any OP instance can
read, validate, and revoke them.

**Redis session key structure:**

```
ggid:session:{session_id}      → JSON: {user_id, tenant_id, client_ids[], created_at, expires_at}
ggid:session:user:{user_id}    → SET of session_ids for this user
ggid:session:client:{client_id} → SET of session_ids where this RP was involved
```

The `client_ids` array (or the per-client session set) is critical: when a
user logs out, the OP needs to know **which RPs to notify**. Without this
mapping, back-channel logout has no targets.

**Go: Redis-backed session store with RP mapping:**

```go
// SessionStore manages distributed sessions in Redis.
type SessionStore struct {
	rdb *redis.Client
	ttl time.Duration
}

// RecordRPLogin records that a client (RP) established a session for this user.
// Called during the authorization code exchange.
func (s *SessionStore) RecordRPLogin(ctx context.Context, userID, clientID string) error {
	pipe := s.rdb.TxPipeline()

	// Add this RP to the user's active-RP set.
	rpKey := fmt.Sprintf("ggid:session:user_rps:%s", userID)
	pipe.SAdd(ctx, rpKey, clientID)
	pipe.Expire(ctx, rpKey, s.ttl)

	return pipe.Exec(ctx)
}

// GetActiveRPs returns the set of client_ids that have active sessions for a user.
// Used during logout to determine back-channel notification targets.
func (s *SessionStore) GetActiveRPs(ctx context.Context, userID string) ([]string, error) {
	rpKey := fmt.Sprintf("ggid:session:user_rps:%s", userID)
	return s.rdb.SMembers(ctx, rpKey).Result()
}

// DestroySession removes the session and clears all RP associations.
func (s *SessionStore) DestroySession(ctx context.Context, userID string) error {
	pipe := s.rdb.TxPipeline()

	// Remove the user's RP mapping.
	rpKey := fmt.Sprintf("ggid:session:user_rps:%s", userID)
	pipe.Del(ctx, rpKey)

	// Remove the session itself.
	sessKey := fmt.Sprintf("ggid:session:user:%s", userID)
	pipe.Del(ctx, sessKey)

	return pipe.Exec(ctx)
}
```

### 1.2 Event-Driven Logout via NATS JetStream

GGID already uses NATS JetStream for audit events. The same infrastructure
can decouple logout propagation: the OP publishes a `logout.requested` event,
and a dedicated consumer delivers back-channel logout tokens to RPs with
retry semantics — surviving OP restarts and transient RP failures.

**Go: NATS-based logout event publisher:**

```go
// LogoutEvent is published when a user session is destroyed.
type LogoutEvent struct {
	UserID    string   `json:"user_id"`
	TenantID  string   `json:"tenant_id"`
	SessionID string   `json:"session_id"`
	Subject   string   `json:"subject"`
	ClientIDs []string `json:"client_ids"` // RPs to notify
	Reason    string   `json:"reason"`     // "rp_initiated", "timeout", "admin"
	Timestamp int64    `json:"timestamp"`
}

// PublishLogout publishes a logout event to NATS JetStream.
// A consumer picks this up and delivers back-channel logout tokens to each RP.
func (p *LogoutPublisher) PublishLogout(ctx context.Context, evt *LogoutEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal logout event: %w", err)
	}

	_, err = p.js.Publish(ctx, "GGID.logout.requested", data,
		nats.MsgId(evt.SessionID), // idempotent: NATS deduplicates by MsgId
	)
	return err
}
```

The consumer then iterates `ClientIDs`, generates a logout token per RP,
and POSTs it to each RP's `backchannel_logout_uri` with exponential backoff
retry.

### 1.3 GGID Multi-Service Logout Flow

```
                         ┌──────────┐
                         │  User    │
                         │ Browser  │
                         └────┬─────┘
                              │ 1. Click "Logout" on RP-A
                              ▼
                    ┌─────────────────┐
                    │   RP-A          │
                    │ (Client App)    │
                    └────────┬────────┘
                             │ 2. 302 → Gateway /oauth/logout
                             │    ?id_token_hint=...&client_id=rp-a
                             │    &post_logout_redirect_uri=https://rp-a/cb
                             ▼
              ┌──────────────────────────────┐
              │        API Gateway           │
              │  (proxy /oauth/ → OAuth)     │
              └──────────────┬───────────────┘
                             │ 3. Proxy to OAuth Service
                             ▼
              ┌──────────────────────────────┐
              │      OAuth Service           │
              │  end_session_endpoint        │
              │                              │
              │  4. Validate id_token_hint   │
              │  5. Revoke access/refresh    │
              │     tokens (Redis)           │
              │  6. Destroy OP session       │
              │     (Redis DEL)              │
              │  7. Publish logout event     │
              │     → NATS JetStream         │
              └──────┬───────────┬───────────┘
                     │           │
          ┌──────────▼──┐  ┌────▼──────────────┐
          │   NATS      │  │  Redis            │
          │ JetStream   │  │  - session:{id}   │
          │             │  │  - revoked:{jti}  │
          │ logout.     │  │  - user_rps:{sub} │
          │ requested   │  └───────────────────┘
          └──────┬──────┘
                 │ 8. Consumer delivers
                 │    logout_token JWT
                 ▼
    ┌────────────────────────────────────────┐
    │  Back-Channel Logout Consumer          │
    │                                        │
    │  for each client_id in evt.ClientIDs:  │
    │    POST logout_token → RP endpoint     │
    │    retry w/ backoff on 5xx             │
    └────────┬──────────────┬────────────────┘
             │              │
      ┌──────▼───┐   ┌──────▼───┐
      │  RP-A    │   │  RP-B    │
      │ POST 200 │   │ POST 200 │
      │ session  │   │ session  │
      │ cleared  │   │ cleared  │
      └──────────┘   └──────────┘

                 │ 9. Redirect to post_logout_redirect_uri
                 ▼
           ┌──────────┐
           │  User    │
           │ Logged   │
           │ Out      │
           └──────────┘
```

**Key design points:**

- Steps 4-7 happen **synchronously** before the user is redirected.
- Step 8 (back-channel notification) happens **asynchronously** via NATS.
  The user does not wait for all RPs to acknowledge.
- If a user closes the browser before step 9, the OP session is already
  destroyed (step 6). Tokens are revoked (step 5). Back-channel
  notification (step 8) proceeds independently.

> See `oidc-logout-spec-analysis.md` for the spec-level sequence diagrams
> and parameter tables.

---

## 2. RP-Initiated Logout Implementation

RP-initiated logout is the most user-facing mechanism. The RP redirects
the user's browser to the OP's `end_session_endpoint`. The OP validates
the request, destroys the session, and redirects back.

### 2.1 Handler Overview

The handler must:

1. Extract `id_token_hint`, `client_id`, `post_logout_redirect_uri`, and
   `state` from query parameters.
2. Validate `id_token_hint` to identify the user and session.
3. Optionally show a confirmation page if `id_token_hint` is absent.
4. Validate `post_logout_redirect_uri` against the client's registered
   allow-list.
5. Destroy the OP session, revoke tokens, and trigger back-channel
   notification.
6. Redirect the user-agent to `post_logout_redirect_uri`.

### 2.2 Go: Complete Logout Handler

```go
// EndSessionHandler implements OIDC RP-Initiated Logout 1.0.
// Route: GET/POST /oauth/end_session
func (h *Handler) EndSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	// Extract parameters (works for both GET query and POST form).
	idTokenHint := r.FormValue("id_token_hint")
	clientID := r.FormValue("client_id")
	postLogoutRedirectURI := r.FormValue("post_logout_redirect_uri")
	state := r.FormValue("state")

	var subject string
	var sessionID string

	// 1. Parse id_token_hint to extract subject and session ID.
	if idTokenHint != "" {
		claims, err := h.oauthSvc.VerifyIDToken(idTokenHint)
		if err == nil {
			subject = claims["sub"].(string)
			if sid, ok := claims["sid"].(string); ok {
				sessionID = sid
			}
		}
		// If verification fails, we still proceed — the hint is advisory.
		// If no hint is provided, we should show a confirmation page.
	}

	// 2. If no id_token_hint, require client_id and show confirmation.
	if idTokenHint == "" && clientID == "" {
		// Without id_token_hint, the OP cannot identify the session.
		// Per spec, the OP SHOULD show a confirmation page.
		// For API-first systems, return an error.
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":             "invalid_request",
			"error_description": "id_token_hint or client_id is required",
		})
		return
	}

	// 3. Validate post_logout_redirect_uri against registered URIs.
	if postLogoutRedirectURI != "" {
		if clientID == "" && subject != "" {
			// Try to infer client_id from the session.
			clientID = h.sessionStore.GetClientForSubject(subject)
		}

		if !h.isPostLogoutRedirectURIAllowed(clientID, postLogoutRedirectURI) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":             "invalid_request",
				"error_description": "post_logout_redirect_uri not registered",
			})
			return
		}
	}

	// 4. Destroy the OP session.
	ctx := r.Context()
	if subject != "" {
		// Revoke all access/refresh tokens for the user.
		_ = h.tokenService.RevokeAllForUser(ctx, h.tenantFromContext(ctx), subject)

		// Destroy the session in Redis.
		_ = h.sessionStore.DestroySession(ctx, subject)

		// Publish logout event for back-channel notification.
		activeRPs, _ := h.sessionStore.GetActiveRPs(ctx, subject)
		_ = h.logoutPublisher.PublishLogout(ctx, &LogoutEvent{
			UserID:    subject,
			Subject:   subject,
			SessionID: sessionID,
			ClientIDs: activeRPs,
			Reason:    "rp_initiated",
			Timestamp: time.Now().Unix(),
		})
	}

	// 5. Clear the OP session cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "ggid_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // delete immediately
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// 6. Redirect to post_logout_redirect_uri (or show a generic logout page).
	if postLogoutRedirectURI != "" {
		redirectURL := postLogoutRedirectURI
		if state != "" {
			u, _ := url.Parse(redirectURL)
			q := u.Query()
			q.Set("state", state)
			u.RawQuery = q.Encode()
			redirectURL = u.String()
		}
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	// No redirect URI — show a simple "logged out" page.
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "logged_out",
		"message": "You have been logged out successfully.",
	})
}

// isPostLogoutRedirectURIAllowed checks the client's registered URIs.
func (h *Handler) isPostLogoutRedirectURIAllowed(clientID, redirectURI string) bool {
	client, err := h.clientRepo.GetByClientID(clientID)
	if err != nil || client == nil {
		return false
	}
	for _, registered := range client.PostLogoutRedirectURIs {
		if registered == redirectURI {
			return true
		}
	}
	return false
}
```

### 2.3 Client Model Extension

The `OAuthClient` domain model must include logout-related fields. The
current GGID model (see `services/oauth/internal/domain/models.go`) has
`RedirectURIs` but lacks all logout fields:

```go
// Required additions to OAuthClient:
type OAuthClient struct {
	// ... existing fields ...

	// RP-Initiated Logout (OIDC RP-Initiated Logout 1.0)
	PostLogoutRedirectURIs []string `json:"post_logout_redirect_uris,omitempty"`

	// Back-Channel Logout (OIDC Back-Channel Logout 1.0)
	BackchannelLogoutURI  string `json:"backchannel_logout_uri,omitempty"`
	BackchannelLogoutTokenAlg string `json:"backchannel_logout_token_alg,omitempty"`

	// Front-Channel Logout (OIDC Front-Channel Logout 1.0)
	FrontchannelLogoutURI string `json:"frontchannel_logout_uri,omitempty"`
}
```

### 2.4 Discovery Endpoint

The `GetDiscoveryConfig()` in `oauth_service.go` already sets
`EndSessionEndpoint` and `BackchannelLogoutSupported`. However, the endpoint
it advertises (`/oauth/logout`) only accepts POST and does not handle the
RP-initiated GET redirect flow. A dedicated `/oauth/end_session` route is
needed for browser-based RP-initiated logout.

```go
// Add to GetDiscoveryConfig:
EndSessionEndpoint:                base + "/oauth/end_session",
BackchannelLogoutSupported:        true,
FrontchannelLogoutSupported:       false, // enable when implemented
```

> See `oidc-logout-spec-analysis.md` §2 for the request parameter table
> and security considerations.

---

## 3. Back-Channel Logout Implementation

Back-channel logout is the recommended mechanism for production multi-RP
deployments. The OP sends a server-to-server HTTP POST containing a
logout token JWT to each affected RP's registered endpoint.

### 3.1 Logout Token Construction

The logout token is a JWT signed by the OP using the same key used for
ID tokens (RS256 in GGID). It must contain specific claims and must NOT
contain a `nonce` claim.

```go
// GenerateLogoutToken creates an OIDC Back-Channel Logout Token JWT.
// Per spec (OIDC Back-Channel Logout 1.0, §2.4), the token must:
//   - Be signed by the OP
//   - Contain: iss, aud, iat, jti, events
//   - Contain at least one of: sub, sid
//   - NOT contain: nonce
func (s *OAuthService) GenerateLogoutToken(clientID, subject, sessionID string) (string, error) {
	now := time.Now()
	jti, err := crypto.GenerateRandomToken(16)
	if err != nil {
		return "", fmt.Errorf("generate jti: %w", err)
	}

	claims := jwt.MapClaims{
		"iss": s.issuer,
		"aud": clientID,
		"iat": now.Unix(),
		"exp": now.Add(2 * time.Minute).Unix(), // short-lived
		"jti": jti,
		"events": map[string]any{
			"http://schemas.openid.net/event/backchannel-logout": {},
		},
	}

	// At least one of sub or sid must be present.
	if subject != "" {
		claims["sub"] = subject
	}
	if sessionID != "" {
		claims["sid"] = sessionID
	}

	// Sign with the OP's RSA private key.
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyProvider.KeyID()

	return token.SignedString(s.keyProvider.PrivateKey())
}
```

### 3.2 Delivery with Retry

Each RP's `backchannel_logout_uri` receives an HTTP POST with the
`logout_token` parameter. The OP must handle transient failures (5xx,
timeouts) with exponential backoff. NATS JetStream provides durable
retry semantics — if the OP restarts, unacknowledged events are
re-delivered.

```go
// BackChannelNotifier delivers logout tokens to registered RP endpoints.
type BackChannelNotifier struct {
	httpClient *http.Client
	maxRetries int
}

// SendLogoutToken POSTs a logout_token to an RP's back-channel endpoint.
// Returns nil on 200, error on non-200 or network failure.
func (n *BackChannelNotifier) SendLogoutToken(ctx context.Context, endpoint, logoutToken string) error {
	form := url.Values{}
	form.Set("logout_token", logoutToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send logout token: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		return nil // RP acknowledges session destroyed

	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		// 4xx: token invalid or permanent error — do not retry.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("rp rejected logout token (%d): %s", resp.StatusCode, body)

	default:
		// 5xx: transient failure — caller should retry.
		return fmt.Errorf("rp returned %d (retryable)", resp.StatusCode)
	}
}

// DeliverWithBackoff sends a logout token with exponential backoff retry.
// Used by the NATS consumer for durable delivery.
func (n *BackChannelNotifier) DeliverWithBackoff(ctx context.Context, endpoint, logoutToken string) error {
	backoff := time.Second
	for attempt := 1; attempt <= n.maxRetries; attempt++ {
		err := n.SendLogoutToken(ctx, endpoint, logoutToken)
		if err == nil {
			return nil
		}
		if !isRetryable(err) {
			return err // permanent failure, stop
		}

		// Wait with exponential backoff + jitter.
		jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff + jitter):
		}
		backoff *= 2 // 1s, 2s, 4s, 8s, ...
	}
	return fmt.Errorf("max retries (%d) exceeded", n.maxRetries)
}

func isRetryable(err error) bool {
	return !strings.Contains(err.Error(), "rejected") &&
		!strings.Contains(err.Error(), "permanent")
}
```

### 3.3 NATS Consumer for Durable Delivery

```go
// StartLogoutConsumer subscribes to NATS and delivers logout tokens to RPs.
// Runs in a goroutine; survives OP restarts via JetStream durable consumers.
func (p *LogoutPublisher) StartLogoutConsumer(ctx context.Context) error {
	sub, err := p.js.PullSubscribe(
		"GGID.logout.requested",
		"logout-delivery",
		nats.Durable("logout-delivery"),
		nats.MaxDeliver(10),
	)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			msgs, err := sub.Fetch(10, nats.MaxWait(5*time.Second))
			if err != nil {
				continue
			}

			for _, msg := range msgs {
				var evt LogoutEvent
				if err := json.Unmarshal(msg.Data, &evt); err != nil {
					_ = msg.Nak() // malformed — don't retry
					continue
				}

				// Deliver to each RP.
				allOK := true
				for _, clientID := range evt.ClientIDs {
					client, err := p.clientRepo.GetByClientID(clientID)
					if err != nil || client.BackchannelLogoutURI == "" {
						continue // skip RPs without back-channel support
					}

					token, err := p.oauthSvc.GenerateLogoutToken(clientID, evt.Subject, evt.SessionID)
					if err != nil {
						allOK = false
						continue
					}

					if err := p.notifier.DeliverWithBackoff(ctx, client.BackchannelLogoutURI, token); err != nil {
						allOK = false
						p.log.Warn("back-channel logout failed",
							"client_id", clientID,
							"error", err)
					}
				}

				if allOK {
					_ = msg.Ack()
				} else {
					_ = msg.Nak() // re-deliver later
				}
			}
		}
	}()

	return nil
}
```

> See `oidc-logout-spec-analysis.md` §3 for the logout token claim
> requirements, response codes, and sequence diagram.

---

## 4. Front-Channel Logout: Iframe Timing Issues

Front-channel logout relies on the browser loading hidden `<iframe>`
elements for each RP. Each iframe loads the RP's `frontchannel_logout_uri`,
which clears the RP's session cookie via JavaScript. This approach is
fragile for several reasons.

### 4.1 Why Front-Channel Is Fragile

| Problem | Cause | Impact |
|---------|-------|--------|
| **Iframe blocked by CSP** | RP sets `X-Frame-Options: DENY` or `Content-Security-Policy: frame-ancestors` | RP session not destroyed |
| **SameSite cookies** | RP session cookie is `SameSite=Lax` or `Strict`; iframe is a cross-origin request in a third-party context | Cookie not sent; RP cannot identify the session to destroy |
| **Third-party cookie blocking** | ITP (Safari), Chrome's Privacy Sandbox, Firefox ETP block third-party cookies | RP cannot read/write cookies in iframe context |
| **Iframe throttling** | Mobile browsers throttle background iframes | Logout JavaScript may not execute before timeout |
| **Network failures** | RP endpoint slow or unreachable | Iframe times out; RP session survives silently |
| **Race conditions** | Multiple iframes loading concurrently | Order of cookie deletion is non-deterministic; cannot detect partial failures |
| **User navigation** | User clicks a link before all iframes finish | Some RP sessions may survive |

### 4.2 The Core Problem: No Acknowledgment

The OP cannot determine whether an iframe successfully loaded. The
`window.postMessage()` API allows RPs to signal completion, but:

1. It requires JavaScript on both sides (OP page + RP iframe).
2. If the iframe is blocked entirely (CSP), no message is ever sent.
3. There is no retry mechanism — a missed iframe is permanently missed.

This is fundamentally why **back-channel logout is preferred**: the OP
gets an HTTP 200 acknowledgment from each RP, can retry on failure, and
has durable delivery via message queues.

### 4.3 Go: Front-Channel Logout Page (When Back-Channel Unavailable)

Despite its fragility, front-channel logout may be needed for RPs that
cannot expose a server-to-server endpoint (e.g., behind NAT/firewall).

```go
// FrontChannelLogoutPage renders an HTML page with iframes for each RP
// that has a frontchannel_logout_uri but no backchannel_logout_uri.
func (h *Handler) FrontChannelLogoutPage(w http.ResponseWriter, r *http.Request) {
	subject := r.FormValue("sub") // from internal redirect after session destroy
	postLogoutURI := r.FormValue("post_logout_redirect_uri")

	// Get all RPs that need front-channel notification.
	rps := h.getFrontChannelRPs(subject)
	if len(rps) == 0 {
		// No front-channel RPs — redirect immediately.
		if postLogoutURI != "" {
			http.Redirect(w, r, postLogoutURI, http.StatusFound)
		}
		return
	}

	// Build the HTML with iframes.
	var iframes strings.Builder
	for _, rp := range rps {
		// Append the session ID so the RP can identify the session.
		iframeURL := rp.FrontchannelLogoutURI + "?iss=" + url.QueryEscape(h.issuer) +
			"&sid=" + url.QueryEscape(subject)
		iframes.WriteString(fmt.Sprintf(
			`<iframe src="%s" style="display:none" sandbox="allow-scripts"></iframe>`+"\n",
			html.EscapeString(iframeURL),
		))
	}

	// The page waits for all iframes, then redirects.
	// Use postMessage for acknowledgment from each RP iframe.
	page := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Logging out...</title></head>
<body>
<h3>Signing you out of all applications...</h3>
%s
<script>
(function() {
  var pending = %d;
  var redirectUri = "%s";
  var timeout = setTimeout(function() {
    // Force redirect after 5 seconds regardless of iframe status.
    window.location.href = redirectUri;
  }, 5000);

  window.addEventListener("message", function(e) {
    // Each RP iframe sends: { type: "frontchannel_logout_complete" }
    if (e.data && e.data.type === "frontchannel_logout_complete") {
      pending--;
      if (pending <= 0) {
        clearTimeout(timeout);
        window.location.href = redirectUri;
      }
    }
  });

  // If no RPs acknowledge, the timeout handles redirect.
})();
</script>
</body>
</html>`, iframes.String(), len(rps), html.EscapeString(postLogoutURI))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(page))
}
```

### 4.4 RP-Side Front-Channel Handler

The RP's `frontchannel_logout_uri` page runs in the iframe and must:

1. Clear the RP's session cookie.
2. Signal completion via `window.parent.postMessage()`.

```html
<!-- RP: /logout/frontchannel -->
<!DOCTYPE html>
<html>
<script>
  // Clear the session cookie.
  document.cookie = "rp_session=; Path=/; Max-Age=0; SameSite=Lax";

  // Signal the OP that logout is complete.
  window.parent.postMessage({
    type: "frontchannel_logout_complete"
  }, "*");
</script>
</html>
```

> See `oidc-logout-spec-analysis.md` §4 for the front-channel flow
> diagram and security considerations.

---

## 5. Logout Token JWT Verification

RPs that receive back-channel logout tokens must verify them rigorously.
A forged or replayed logout token can be used to force legitimate users
out of their sessions (a denial-of-service vector).

### 5.1 Verification Checklist

| Step | Check | Failure Action |
|------|-------|----------------|
| 1 | Verify JWT signature using OP's JWKS | Reject (400) |
| 2 | `iss` matches expected OP issuer URL | Reject (400) |
| 3 | `aud` matches the RP's `client_id` | Reject (400) |
| 4 | `events` contains `http://schemas.openid.net/event/backchannel-logout` | Reject (400) |
| 5 | `nonce` claim is **absent** | Reject (400) |
| 6 | `iat` is within acceptable time window (not too old) | Reject (400) |
| 7 | `jti` has not been seen before (replay prevention) | Reject (400) |
| 8 | At least one of `sub` or `sid` is present | Reject (400) |
| 9 | Destroy the local session for the identified `sub`/`sid` | Return 200 |

### 5.2 Go: Logout Token Verification on the RP Side

```go
// VerifyLogoutToken verifies an OIDC Back-Channel Logout Token JWT.
// Implements the full verification checklist per OIDC Back-Channel Logout 1.0 §2.4.
func VerifyLogoutToken(
	tokenStr string,
	expectedIssuer string,
	expectedClientID string,
	jwksURL string,
	jtiCache *redis.Client,
) (string, error) {

	// Step 1: Verify signature using the OP's JWKS.
	// We fetch the JWKS and use it to validate the token.
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Ensure RS256 algorithm.
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid header")
		}

		// Fetch the public key from the OP's JWKS endpoint.
		key, err := fetchKeyFromJWKS(jwksURL, kid)
		if err != nil {
			return nil, fmt.Errorf("fetch jwks key: %w", err)
		}
		return key, nil
	})
	if err != nil {
		return "", fmt.Errorf("signature verification failed: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid claims")
	}

	// Step 2: Verify iss.
	iss, _ := claims["iss"].(string)
	if iss != expectedIssuer {
		return "", fmt.Errorf("iss mismatch: expected %s, got %s", expectedIssuer, iss)
	}

	// Step 3: Verify aud.
	aud, _ := claims["aud"].(string)
	if aud != expectedClientID {
		return "", fmt.Errorf("aud mismatch: expected %s, got %s", expectedClientID, aud)
	}

	// Step 4: Verify events claim.
	events, ok := claims["events"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("missing events claim")
	}
	if _, ok := events["http://schemas.openid.net/event/backchannel-logout"]; !ok {
		return "", fmt.Errorf("events does not contain backchannel-logout")
	}

	// Step 5: nonce must be absent.
	if _, ok := claims["nonce"]; ok {
		return "", fmt.Errorf("logout token must not contain nonce")
	}

	// Step 6: Verify iat is not too old (e.g., within last 10 minutes).
	iat, ok := claims["iat"].(float64)
	if !ok {
		return "", fmt.Errorf("missing or invalid iat")
	}
	if time.Now().Unix()-int64(iat) > 600 {
		return "", fmt.Errorf("token too old (iat expired)")
	}

	// Step 7: Check jti for replay.
	jti, _ := claims["jti"].(string)
	if jti == "" {
		return "", fmt.Errorf("missing jti claim")
	}
	ctx := context.Background()
	jtiKey := fmt.Sprintf("oidc:logout_jti:%s", jti)
	set, err := jtiCache.SetNX(ctx, jtiKey, 1, 24*time.Hour).Result()
	if err != nil || !set {
		return "", fmt.Errorf("replay detected: jti already seen")
	}

	// Step 8: Must have sub or sid.
	sub, hasSub := claims["sub"].(string)
	sid, hasSid := claims["sid"].(string)
	if !hasSub && !hasSid {
		return "", fmt.Errorf("must contain sub or sid")
	}

	// Return the subject (or session ID) for session destruction.
	if sub != "" {
		return sub, nil
	}
	return sid, nil
}
```

### 5.3 JTI Replay Cache Considerations

The JTI cache must be:

- **Distributed** (Redis, not in-memory) — if the OP restarts between
  the original token and a replay attempt, the in-memory cache would be
  empty.
- **TTL-bounded** — the cache entry should live at least as long as the
  longest token validity (2-24 hours is typical). Using 24h TTL is a safe
  default.
- **Atomic** — `SetNX` (SET if Not eXists) ensures atomic check-and-set
  even under concurrent token delivery.

### 5.4 GGID's Current State

GGID's `ParseBackchannelLogoutToken()` (oauth_service.go:1326) uses
`jwt.ParseUnverified()` — it does NOT verify the JWT signature. The JTI
replay check uses an in-memory `sync.Map` (`backchannelLogoutList`)
which is lost on process restart. Both are critical security gaps.

---

## 6. GGID Logout Gap Analysis

### 6.1 OAuth Service (`services/oauth/`)

**What exists:**

| Component | Location | Status |
|-----------|----------|--------|
| `/oauth/logout` (POST) | `server.go:425` | Accepts logout_token, parses claims, revokes token. Does NOT handle GET (browser redirect). |
| `/api/v1/oauth/backchannel-logout` (POST) | `server.go:496` | Accepts logout_token, calls `ParseBackchannelLogoutToken`. |
| `/oauth/revoke` (POST) | `server.go:470` | RFC 7009 token revocation. Working. |
| `RPInitiatedLogout()` | `logout.go:31` | Service-level method exists but NOT wired to any HTTP route. |
| `ParseBackchannelLogoutToken()` | `oauth_service.go:1326` | Parses claims but uses `ParseUnverified` — no signature check. |
| `BackchannelLogout()` | `oauth_service.go:1314` | Only marks `sync.Map` entry. Does NOT POST to RPs. |
| `GetDiscoveryConfig()` | `oauth_service.go:357` | Advertises `end_session_endpoint` and `backchannel_logout_supported`. |

**What's missing:**

| Gap | Impact |
|-----|--------|
| No `/oauth/end_session` GET handler for browser redirect | Users cannot trigger RP-initiated logout via browser redirect |
| `ParseBackchannelLogoutToken` uses `ParseUnverified` | Security vulnerability — forged logout tokens accepted |
| `BackchannelLogout` does not POST to RP endpoints | No actual RP notification; sessions at RPs survive |
| No `post_logout_redirect_uris` field on `OAuthClient` | Cannot validate post-logout redirect targets |
| No `backchannel_logout_uri` field on `OAuthClient` | Cannot determine where to send logout tokens |
| No `frontchannel_logout_uri` field on `OAuthClient` | Cannot support front-channel logout |
| No `sid` (session ID) claim in ID tokens | Cannot track which RP sessions are active per user |
| No session-to-RP mapping | Back-channel delivery has no targets |
| `backchannelLogoutList` and `revokedTokens` are `sync.Map` | Lost on restart; replay protection and revocation disappear |

### 6.2 Auth Service (`services/auth/`)

**What exists:**

| Component | Location | Status |
|-----------|----------|--------|
| `Session` domain model | `domain/session.go` | Has `ID`, `UserID`, `TokenHash`, `RevokedAt`, `ExpiresAt`. No `sid` or `client_id` tracking. |
| `SessionService.Revoke()` | `session_service.go:79` | Revokes a single session by ID. |
| `SessionService.RevokeAllForUser()` | `session_service.go:84` | Revokes all sessions for a user. |
| `AuthService.LogoutAll()` | `logout_all.go:12` | Revokes sessions + refresh tokens for a user. No OIDC logout. |
| `AuthService.ForceLogout()` | `session_management.go:148` | Revokes all sessions + tokens. Admin operation. |
| `EnforceSessionLimit()` | `session_management.go:34` | Caps concurrent sessions per user. |

**What's missing:**

| Gap | Impact |
|-----|--------|
| No `sid` generation or tracking in sessions | Cannot correlate OIDC sessions with auth sessions |
| No back-channel logout token generation | OP cannot produce logout tokens for RP notification |
| No NATS integration for logout events | Logout propagation requires manual polling or is missing |
| No Redis-backed session store | Sessions stored only in PostgreSQL; no fast session lookup for logout |

### 6.3 Gateway (`services/gateway/`)

**What exists:**

| Component | Location | Status |
|-----------|----------|--------|
| `/oauth/` route proxy | `config.go:55` | Proxies all `/oauth/` requests to `http://localhost:9005`. |
| Session middleware | `middleware/session.go` | Redis-backed session validation. Excludes `/oauth/` from auth checks. |

**What's missing:**

| Gap | Impact |
|-----|--------|
| No logout-specific routing | The gateway does not distinguish logout from other OAuth requests |
| No cookie clearing on logout | Gateway does not clear `ggid_session` cookie on logout |
| No CORS configuration for post-logout redirect | Cross-origin post-logout redirects may be blocked |

### 6.4 Summary Matrix

```
                         RP-Initiated    Back-Channel    Front-Channel    Token Revocation
                         ────────────    ────────────    ─────────────    ────────────────
Discovery advertises     PARTIAL (1)     YES             NO               YES
Endpoint exists          PARTIAL (2)     YES             NO               YES
Token generation         N/A             NO (3)          N/A              N/A
Token delivery           N/A             NO (4)          NO               N/A
Token verification       N/A             NO (5)          N/A              N/A
Session destroy          YES             YES             N/A              YES
RP notification          NO              NO              NO               N/A
Persistent state         NO (6)          NO (6)          N/A              PARTIAL (7)

(1) end_session_endpoint set in discovery, but endpoint only accepts POST
(2) RPInitiatedLogout() method exists but no HTTP route
(3) No GenerateLogoutToken() in production code
(4) BackchannelLogout() only marks sync.Map; no HTTP POST
(5) ParseBackchannelLogoutToken uses ParseUnverified
(6) sync.Map lost on restart
(7) Token revocation works but revoked list is in-memory
```

---

## 7. Implementation Roadmap

### P0: RP-Initiated Logout (3-5 days)

**Goal:** Users can log out via browser redirect to the OP.

1. Add `post_logout_redirect_uris []string` to `OAuthClient` model + DB
   migration.
2. Add `GET /oauth/end_session` HTTP handler in `server.go` (wire the
   existing `RPInitiatedLogout()` method or write a dedicated handler).
3. Update `GetDiscoveryConfig()` to point `end_session_endpoint` to
   `/oauth/end_session` (currently points to `/oauth/logout` which only
   accepts POST).
4. Clear `ggid_session` cookie on logout.
5. Add `state` parameter passthrough to the redirect.

**Effort:** 3-5 days (model change + handler + migration + tests)

### P0: Persistent Revocation Store (2-3 days)

**Goal:** Token revocation survives OP restarts.

1. Replace `revokedTokens` sync.Map with Redis SET (`ggid:revoked:{jti}`
   with TTL = token expiry).
2. Replace `backchannelLogoutList` sync.Map with Redis SET
   (`ggid:backchannel_logout:{sub}` with 24h TTL).
3. Update `IsTokenRevoked()` and replay check to query Redis.

**Effort:** 2-3 days (Redis integration + migration of checks + tests)

### P1: Back-Channel Logout Delivery (5-8 days)

**Goal:** OP notifies RPs when sessions are destroyed.

1. Add `backchannel_logout_uri string` to `OAuthClient` model + migration.
2. Implement `GenerateLogoutToken()` — sign JWT with RS256, include
   `iss`, `aud`, `iat`, `jti`, `events`, `sub`/`sid`.
3. Add `sid` claim to ID tokens during token issuance.
4. Implement session-to-RP mapping: track which `client_id`s have active
   sessions per user (Redis SET).
5. Implement `BackChannelNotifier` with HTTP POST + exponential backoff.
6. Integrate with NATS JetStream for durable delivery (publish on logout,
   consume + deliver to RPs).
7. Fix `ParseBackchannelLogoutToken()` to verify JWT signature using JWKS.

**Effort:** 5-8 days (most complex phase — model changes + token
generation + delivery infrastructure + signature fix)

### P2: Front-Channel Logout Fallback (2-3 days)

**Goal:** Support RPs that cannot expose back-channel endpoints.

1. Add `frontchannel_logout_uri string` to `OAuthClient` model + migration.
2. Implement iframe-rendering page (see §4.3).
3. Add timeout-based redirect (5 seconds, then redirect regardless).
4. Add `frontchannel_logout_supported: true` to discovery config.

**Effort:** 2-3 days (straightforward HTML rendering + model change)

### P3: JTI Replay Cache Hardening (1-2 days)

**Goal:** Replay prevention survives restarts and works across instances.

1. Replace in-memory JTI tracking with Redis `SetNX` (see §5.2).
2. Set 24h TTL on JTI entries.
3. Add monitoring/metrics for replay detection events.

**Effort:** 1-2 days (Redis integration + tests)

### Roadmap Summary

| Priority | Feature | Effort | Dependencies |
|----------|---------|--------|-------------|
| P0 | RP-Initiated Logout | 3-5 days | Client model change |
| P0 | Persistent Revocation Store | 2-3 days | Redis |
| P1 | Back-Channel Logout Delivery | 5-8 days | Client model, NATS, sid tracking |
| P2 | Front-Channel Logout Fallback | 2-3 days | Client model change |
| P3 | JTI Replay Cache | 1-2 days | Redis |

**Total estimated effort:** 13-21 days for full OIDC logout compliance.

The critical path is P0 (RP-Initiated) + P1 (Back-Channel). Together they
deliver a production-grade logout system. P2 and P3 are hardening and
fallback measures.

---

## References

- [OIDC RP-Initiated Logout 1.0](https://openid.net/specs/openid-connect-rpinitiated-1_0.html)
- [OIDC Back-Channel Logout 1.0](https://openid.net/specs/openid-connect-backchannel-1_0.html)
- [OIDC Front-Channel Logout 1.0](https://openid.net/specs/openid-connect-frontchannel-1_0.html)
- [RFC 7009: OAuth 2.0 Token Revocation](https://datatracker.ietf.org/doc/html/rfc7009)
- [RFC 8417: Security Event Token (SET)](https://datatracker.ietf.org/doc/html/rfc8417)
- See `oidc-logout-spec-analysis.md` for spec-level analysis
