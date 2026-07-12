# Session Binding Design

This guide covers session-to-IP binding, device fingerprint binding, geo-binding, concurrent session policy, session fixation prevention, and GGID's session binding implementation.

## Overview

Session binding ties an authenticated session to contextual properties (IP, device, location) to detect and prevent session hijacking. When a bound property changes mid-session, GGID can require re-authentication or terminate the session.

## Session-to-IP Binding

### How It Works

```
1. User logs in from IP 192.168.1.50
2. Session is bound to IP 192.168.1.50
3. Every subsequent request checks: current IP == bound IP?
4. If IP changes → require step-up or terminate session
```

### Configuration

```yaml
session:
  ip_binding:
    enabled: true
    strict: false  # strict = terminate on change, non-strict = challenge
    allow_xff: true  # Trust X-Forwarded-For behind known proxy
    trusted_proxies: ["10.0.0.0/8"]
```

### Implementation

```go
type Session struct {
    ID          string
    UserID      string
    TenantID    string
    BoundIP     string
    CreatedAt   time.Time
    LastSeen    time.Time
    StepUpAt    time.Time
    IPChangeAt  time.Time
}

func CheckIPBinding(session *Session, currentIP string, config IPBindingConfig) Action {
    if !config.Enabled {
        return ActionAllow
    }

    if session.BoundIP == currentIP {
        return ActionAllow
    }

    // IP changed
    if config.Strict {
        return ActionTerminate
    }

    // Non-strict: require step-up
    if time.Since(session.StepUpAt) < time.Hour {
        // Recently stepped up, allow IP update
        session.BoundIP = currentIP
        session.IPChangeAt = time.Now()
        return ActionAllow
    }

    return ActionStepUp
}
```

### Tradeoffs

| Aspect | Strict | Non-Strict |
|---|---|---|
| Security | High (immediate termination) | Medium (challenge) |
| UX | Poor (mobile switching WiFi/cellular) | Better (challenge but don't kick out) |
| False positives | High | Low |
| Recommendation | High-security tenants | General purpose |

## Device Fingerprint Binding

### Fingerprint Collection

```javascript
// Client-side fingerprint collection
const fingerprint = {
    userAgent: navigator.userAgent,
    screen: `${screen.width}x${screen.height}x${screen.colorDepth}`,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    language: navigator.language,
    platform: navigator.platform,
    fonts: detectInstalledFonts(),
    canvas: canvasFingerprint(),
    webgl: webglFingerprint(),
};
// Send to server during login
```

### Fingerprint Hash

```go
func computeDeviceFingerprint(fp DeviceFingerprint) string {
    data := fmt.Sprintf("%s|%s|%s|%s|%s",
        fp.UserAgent,
        fp.Screen,
        fp.Timezone,
        fp.Language,
        fp.Platform,
    )
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}
```

### Binding Enforcement

```go
func CheckDeviceBinding(session *Session, currentFP string, config DeviceBindingConfig) Action {
    if !config.Enabled {
        return ActionAllow
    }

    if session.DeviceFingerprint == currentFP {
        return ActionAllow
    }

    // Device fingerprint changed — possible session hijacking
    // Check if it's a minor variation (browser update)
    similarity := fingerprintSimilarity(session.DeviceFingerprint, currentFP)
    if similarity > 0.8 {
        // Minor change, update fingerprint
        session.DeviceFingerprint = currentFP
        return ActionAllow
    }

    // Major change — require step-up
    return ActionStepUp
}
```

### Fingerprint Stability

| Component | Stability | False Positive Risk |
|---|---|---|
| User-Agent | Low (updates frequently) | High |
| Screen resolution | High | Low |
| Timezone | High | Low |
| Canvas fingerprint | Medium (browser updates) | Medium |
| WebGL | Medium (driver updates) | Medium |
| Fonts | High | Low |
| Platform | High | Low |

**Recommendation**: Use only stable components (screen, timezone, platform) for strict binding. Use all components for risk scoring.

## Geo-Binding

### How It Works

```
1. User logs in from New York
2. Session bound to geo: "US-NY"
3. Request from California → same country, allow
4. Request from China → different country, challenge
```

### Configuration

```yaml
session:
  geo_binding:
    enabled: true
    granularity: "country"  # country, state, city
    allow_same_country: true
    challenge_on_country_change: true
    block_high_risk_countries: true
    high_risk_countries: ["XX", "YY"]
```

### Implementation

```go
func CheckGeoBinding(session *Session, currentIP string, config GeoBindingConfig) Action {
    if !config.Enabled {
        return ActionAllow
    }

    currentGeo := geoLocate(currentIP)

    switch config.Granularity {
    case "country":
        if session.GeoCountry == currentGeo.Country {
            return ActionAllow
        }
    case "state":
        if session.GeoState == currentGeo.State {
            return ActionAllow
        }
    case "city":
        if session.GeoCity == currentGeo.City {
            return ActionAllow
        }
    }

    // Geo changed
    if config.BlockHighRiskCountries && isHighRisk(currentGeo.Country) {
        return ActionTerminate
    }

    if config.AllowSameCountry && session.GeoCountry == currentGeo.Country {
        return ActionAllow
    }

    if config.ChallengeOnCountryChange {
        return ActionStepUp
    }

    return ActionAllow
}
```

## Concurrent Session Policy

### Policy Types

| Policy | Description | Use Case |
|---|---|---|
| Single session | Only one active session per user | High-security |
| Limited sessions | Max N concurrent sessions | Enterprise |
| Unlimited | No limit on concurrent sessions | Consumer |

### Configuration

```yaml
session:
  concurrent:
    max_per_user: 5
    max_per_device: 2
    on_exceed: "evict_oldest"  # or "deny_new", "step_up"
    notify_on_exceed: true
```

### Implementation

```go
func EnforceConcurrentSession(userID string, config ConcurrentConfig) error {
    sessions := getActiveSessions(userID)
    if len(sessions) < config.MaxPerUser {
        return nil  // Under limit
    }

    switch config.OnExceed {
    case "evict_oldest":
        // Terminate oldest session
        oldest := sessions[0]
        terminateSession(oldest.ID)
        if config.NotifyOnExceed {
            notifyUser(userID, "Session terminated due to concurrent limit")
        }
        return nil

    case "deny_new":
        return ErrConcurrentLimitExceeded

    case "step_up":
        return ErrStepUpRequired

    default:
        return nil
    }
}
```

### Session Types

```yaml
session:
  types:
    web:
      max_per_user: 3
      lifetime: 8h
    mobile:
      max_per_user: 2
      lifetime: 30d
    api:
      max_per_user: 10
      lifetime: 1h
    admin:
      max_per_user: 1
      lifetime: 1h
      on_exceed: "deny_new"
```

## Session Fixation Prevention

### Attack Pattern

```
1. Attacker obtains a valid session ID (e.g., via XSS, URL parameter)
2. Attacker tricks user into using that session ID
3. User logs in, session ID stays the same
4. Attacker uses the same session ID to access the account
```

### Prevention

**Always rotate session ID after login:**

```go
func LoginHandler(w http.ResponseWriter, r *http.Request) {
    // ... validate credentials ...

    // CRITICAL: Rotate session ID after login
    oldSessionID := getSessionID(r)
    if oldSessionID != "" {
        // Destroy old session
        destroySession(oldSessionID)
    }

    // Create new session with new ID
    newSession := createSession(user.ID, clientIP(r))
    setSessionCookie(w, newSession.ID, secureAttrs)

    // ... issue tokens ...
}
```

### Additional Protections

| Protection | Implementation |
|---|---|
| Session ID rotation | New ID after every login |
| Secure cookie flags | HttpOnly, Secure, SameSite=Strict |
| No session ID in URL | Only use cookies, never query params |
| Session ID length | 32+ bytes from crypto/rand |
| Session timeout | Idle timeout + absolute timeout |

```go
func generateSessionID() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}
```

## Binding Strength vs UX

| Binding | Strength | UX Impact | False Positive Rate |
|---|---|---|---|
| IP strict | Very strong | High (mobile users) | High |
| IP non-strict | Strong | Medium | Medium |
| Device fingerprint | Strong | Low | Low |
| Geo (country) | Medium | Low | Low |
| Geo (city) | Medium | Medium | Medium |
| No binding | None | None | N/A |

### Recommended Combinations

| Tenant Type | IP | Device | Geo | Concurrent |
|---|---|---|---|---|
| High-security (finance) | Strict | Enabled | Country | Single |
| Enterprise | Non-strict | Enabled | Country | Limited (3) |
| Consumer | Disabled | Enabled | Disabled | Unlimited |
| Admin accounts | Strict | Enabled | City | Single |

## Step-Up for Binding Change

When a binding property changes, GGID can require step-up authentication:

```go
func CheckBindings(session *Session, r *http.Request, config *BindingConfig) BindingResult {
    result := BindingResult{Allow: true}

    // IP check
    ipAction := CheckIPBinding(session, clientIP(r), config.IP)
    if ipAction == ActionTerminate {
        return BindingResult{Allow: false, Terminate: true}
    }
    if ipAction == ActionStepUp {
        result.RequireStepUp = true
        result.Reason = "ip_change"
    }

    // Device check
    devAction := CheckDeviceBinding(session, deviceFP(r), config.Device)
    if devAction == ActionStepUp {
        result.RequireStepUp = true
        result.Reason = "device_change"
    }

    // Geo check
    geoAction := CheckGeoBinding(session, clientIP(r), config.Geo)
    if geoAction == ActionStepUp {
        result.RequireStepUp = true
        result.Reason = "geo_change"
    }
    if geoAction == ActionTerminate {
        return BindingResult{Allow: false, Terminate: true}
    }

    return result
}
```

## GGID Session Binding

### Configuration

```yaml
session:
  binding:
    ip:
      enabled: true
      strict: false
    device:
      enabled: true
      stable_components_only: true
    geo:
      enabled: true
      granularity: "country"
      challenge_on_change: true
  concurrent:
    max_per_user: 5
    on_exceed: "evict_oldest"
  fixation:
    rotate_on_login: true
    secure_cookie: true
    same_site: "strict"
  timeout:
    idle: 30m
    absolute: 8h
    sliding: true  # Reset idle timeout on activity
```

### Middleware

```go
func SessionBindingMiddleware(config *BindingConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            session := getSession(r)
            if session == nil {
                next.ServeHTTP(w, r)
                return
            }

            result := CheckBindings(session, r, config)
            if result.Terminate {
                terminateSession(session.ID)
                writeError(w, 401, "session_terminated_binding_violation")
                return
            }
            if result.RequireStepUp {
                writeError(w, 403, "step_up_required_"+result.Reason)
                return
            }

            // Update last seen
            session.LastSeen = time.Now()
            updateSession(session)

            next.ServeHTTP(w, r)
        })
    }
}
```

## Best Practices

1. **Rotate session ID on login** — Always, no exceptions
2. **Use non-strict IP binding** — Strict causes too many false positives
3. **Use stable fingerprint components** — Don't bind to volatile attributes
4. **Geo-binding at country level** — Finer granularity causes false positives
5. **Set concurrent limits** — Prevent session farming attacks
6. **Use secure cookie flags** — HttpOnly, Secure, SameSite=Strict
7. **Step-up on binding change** — Don't terminate, give user a chance to verify
8. **Notify on changes** — Email users when their session context changes
9. **Audit binding events** — Log all binding changes and step-up triggers
10. **Test with real scenarios** — Mobile switching WiFi/cellular is common