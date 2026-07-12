# MFA Fatigue Defense

This guide covers MFA fatigue attack patterns, detection signals, defense methods, and GGID's implementation.

## Attack Pattern

### What is MFA Fatigue?

MFA fatigue (also called "push bombing" or "MFA bombing") is an attack where the attacker repeatedly sends MFA push notifications to a victim's device, hoping the user will eventually approve one out of frustration, annoyance, or confusion.

### Attack Flow

```
1. Attacker obtains user's password (phishing, breach, etc.)
2. Attacker attempts login → triggers MFA push notification
3. User receives push notification on phone
4. Attacker repeatedly triggers logins → floods user with push notifications
5. User, overwhelmed by notifications, approves one to stop the flood
6. Attacker gains access
```

### Real-World Incidents

| Incident | Year | Target | Method |
|---|---|---|---|
| Uber breach | 2022 | Uber engineer | Repeated push approvals until approved |
| Cisco breach | 2022 | Cisco VPN | MFA fatigue + voice phishing |
| Reddit breach | 2023 | Reddit employee | Phishing + MFA fatigue |

## Detection Signals

### Velocity Signals

| Signal | Threshold | Confidence |
|---|---|---|
| Push notifications per user | >3 in 10 minutes | High |
| Push notifications per user | >10 in 1 hour | Very High |
| Push notifications per IP | >5 in 10 minutes | High |
| Unique target users per IP | >3 in 1 hour | Medium |

### Pattern Signals

| Signal | Description | Detection |
|---|---|---|
| Rapid successive pushes | <2 seconds between pushes | Bot behavior |
| Same source IP | All pushes from same IP | Single attacker |
| Multiple source IPs | Pushes from many IPs | Distributed attack |
| Off-hours pushes | Notifications at 3 AM | Suspicious timing |
| Non-standard geo | Pushes from unusual location | Account compromise |

### User Behavior Signals

| Signal | Description |
|---|---|
| Previously denied pushes | User has been rejecting |
| No recent login activity | No successful logins recently |
| Password just used | Password used shortly before pushes |
| New device fingerprint | Login from unseen device |

## Defense Methods

### 1. Number Matching (Primary Defense)

Instead of a simple approve/deny, the user must match a number displayed on their screen with the number on their phone:

```
Login screen: "Enter the number shown on your device"
Phone: "491527 — Is this the number on your login screen?"
User must: Select the correct number from options
```

```go
type NumberChallenge struct {
    ChallengeID   string
    CorrectNumber int  // 2-digit number shown on login screen
    Options       []int  // Options shown on phone
    ExpiresAt     time.Time
}

func GenerateNumberChallenge() *NumberChallenge {
    correct := randInt(10, 99)
    options := generateUniqueOptions(correct, 3)
    return &NumberChallenge{
        ChallengeID:   uuid.New().String(),
        CorrectNumber: correct,
        Options:       options,
        ExpiresAt:     time.Now().Add(60 * time.Second),
    }
}

func (c *NumberChallenge) Verify(selected int) bool {
    return selected == c.CorrectNumber && time.Now().Before(c.ExpiresAt)
}
```

**Why it works**: The attacker can't see the number on the user's phone, so they can't tell the user which number to select. A fatigued user tapping randomly is unlikely to select the correct number (1 in 3 chance per attempt).

### 2. Conditional Access

Only trigger MFA push in specific contexts:

```yaml
mfa:
  conditional:
    skip_push_when:
      - managed_device: true  # Skip for managed devices
      - corporate_network: true  # Skip on corporate network
      - recent_mfa: 1h  # Skip if MFA verified recently
    require_push_when:
      - new_device: true
      - new_ip: true
      - off_hours: true
```

### 3. Geofencing

Only allow MFA approval from expected geographic locations:

```go
func isGeoAllowed(userIP string, userPatterns *UserPatterns) bool {
    currentCountry := geoLocate(userIP).Country
    if contains(userPatterns.Countries, currentCountry) {
        return true
    }
    // Allow if within expected region
    for _, expectedCountry := range userPatterns.Countries {
        if isSameRegion(currentCountry, expectedCountry) {
            return true
        }
    }
    return false
}
```

### 4. Rate Limiting Push Notifications

```yaml
mfa:
  push_rate_limit:
    per_user:
      max_per_10min: 3
      max_per_hour: 10
      max_per_day: 20
    per_ip:
      max_per_10min: 5
      max_per_hour: 15
    on_exceed:
      action: "block_push"  # Stop sending notifications
      duration: 30m  # Block pushes for 30 minutes
      notify_admin: true
      notify_user: true  # Email user about blocked pushes
```

### 5. Adaptive Challenge Selection

When push fatigue is detected, switch to a different MFA method:

```go
func selectMFAChallenge(riskContext *RiskContext, pushHistory *PushHistory) string {
    // If push fatigue detected, don't use push
    if pushHistory.RecentDenials > 2 || pushHistory.PushCount10Min > 3 {
        // Switch to TOTP or WebAuthn
        if riskContext.HasTOTP {
            return "totp"
        }
        if riskContext.HasWebAuthn {
            return "webauthn"
        }
    }
    return "push"  // Default
}
```

### 6. Notification Content Security

Include context in push notifications to help users identify legitimate vs fraudulent requests:

```
"Approve login to GGID from:
  - App: Company Portal
  - Location: San Francisco, CA
  - IP: 192.168.1.50
  - Time: 2:30 PM PST
  - Device: Chrome on macOS

If you didn't request this, tap Deny."
```

### 7. Deny-First for Suspicious Patterns

After detecting push fatigue patterns, automatically deny subsequent pushes:

```go
func shouldAutoDeny(userID string, pushHistory *PushHistory) bool {
    // Auto-deny if too many pushes in short time
    if pushHistory.PushCount10Min >= 5 {
        log.Security("auto-denying push due to fatigue pattern", 
            "user_id", userID,
            "push_count_10min", pushHistory.PushCount10Min)
        return true
    }
    
    // Auto-deny if user has denied 3+ in a row
    if pushHistory.ConsecutiveDenials >= 3 {
        return true
    }
    
    return false
}
```

## GGID Defense Implementation

### Multi-Layer Defense

```
Login Request → Rate Limit Check → Risk Evaluation
             → Push Fatigue Check → Challenge Selection
             → Number Matching → Allow/Deny
```

### Configuration

```yaml
mfa:
  fatigue_defense:
    enabled: true
    number_matching:
      enabled: true
      number_length: 2  # 2-digit number
      timeout: 60s
      max_attempts: 3
    rate_limit:
      per_user_10min: 3
      per_user_hour: 10
      per_user_day: 20
      block_duration: 30m
    adaptive:
      switch_on_fatigue: true
      fallback_methods: ["totp", "webauthn"]
    notification:
      include_context: true  # Location, IP, app, time
      warn_on_denial: true   # "If you didn't request this, contact IT"
    auto_deny:
      enabled: true
      threshold_10min: 5
      threshold_consecutive_denials: 3
    notify:
      admin_on_fatigue: true
      user_on_block: true  # Email: "Your MFA pushes were blocked"
```

### Middleware

```go
func MFAPushMiddleware(config *FatigueConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID := getUserID(r)
            
            // Check rate limit
            if isRateLimited(userID, config.RateLimit) {
                notifyAdmin(userID, "push_rate_limited")
                notifyUser(userID, "push_blocked_rate_limit")
                writeError(w, 429, "mfa_push_rate_limited")
                return
            }
            
            // Check for auto-deny
            if shouldAutoDeny(userID, getPushHistory(userID)) {
                writeError(w, 403, "mfa_auto_denied")
                return
            }
            
            // Select challenge method
            method := selectMFAChallenge(getRiskContext(r), getPushHistory(userID))
            
            if method == "push" {
                // Generate number matching challenge
                challenge := GenerateNumberChallenge()
                storeChallenge(challenge)
                sendPushNotification(userID, challenge, getContextInfo(r))
            } else {
                // Switch to alternative method
                writeError(w, 403, "mfa_method_changed", map[string]interface{}{
                    "alternative_method": method,
                    "reason": "push_fatigue_detected",
                })
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

## Microsoft Case Study

### Microsoft's Approach

Microsoft implemented several defenses against MFA fatigue in Azure AD (now Entra ID):

1. **Number matching** (2022): Users must match a number shown on screen with their phone
2. **Additional context** (2022): Push notifications show app name, location, and IP
3. **Phishing-resistant MFA** (2023): Promote FIDO2/WebAuthn as primary MFA
4. **Conditional access**: Skip MFA for trusted locations/devices

### Results

- MFA fatigue attacks dropped significantly after number matching rollout
- Legitimate user friction increased slightly (~2% more time per login)
- User satisfaction remained high due to clear context in notifications

### Lessons for GGID

1. Number matching is the single most effective defense
2. Context in notifications helps users identify attacks
3. Promoting WebAuthn reduces reliance on push MFA entirely
4. Rate limiting prevents notification flooding

## Best Practices

1. **Enable number matching** — Most effective single defense
2. **Rate limit push notifications** — Prevent flooding
3. **Include context in notifications** — App, location, IP, time
4. **Auto-deny on fatigue patterns** — Stop the flood automatically
5. **Switch methods on detection** — Fall back to TOTP/WebAuthn
6. **Notify users when blocked** — Email about blocked push attempts
7. **Notify admins on fatigue** — Security team should investigate
8. **Promote phishing-resistant MFA** — WebAuthn eliminates push fatigue
9. **Educate users** — Tell users to never approve unexpected pushes
10. **Log all push events** — Full audit trail for investigation