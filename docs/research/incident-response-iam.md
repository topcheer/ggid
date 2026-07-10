# Security Incident Response Playbook for IAM Systems

> **Scope**: GGID Identity and Access Management Suite — 7 microservices (Gateway, Identity, Auth, OAuth, Policy, Org, Audit) with PostgreSQL, Redis, NATS JetStream, and OpenLDAP.

> **Audience**: SREs, security engineers, on-call responders, platform architects.

---

## Table of Contents

1. [Incident Detection for IAM](#1-incident-detection-for-iam)
2. [Incident Classification](#2-incident-classification)
3. [Containment Strategies](#3-containment-strategies)
4. [Eradication and Patching](#4-eradication-and-patching)
5. [Recovery and Verification](#5-recovery-and-verification)
6. [Postmortem Process](#6-postmortem-process)
7. [Communication Plan](#7-communication-plan)
8. [Incident Response Automation](#8-incident-response-automation)
9. [Tabletop Exercises](#9-tabletop-exercises)
10. [GGID IR Playbook](#10-ggid-ir-playbook)
11. [Gap Analysis & Recommendations](#11-gap-analysis--recommendations)

---

## 1. Incident Detection for IAM

### 1.1 Detection Sources

IAM incidents can originate from multiple detection vectors. A mature detection program combines automated signals with human reporting:

| Source | Signal | Latency | Example |
|--------|--------|---------|---------|
| **SIEM alerts** | Failed login spike from single IP | Seconds | 500+ 401s in 60s from 203.0.113.42 |
| **Audit anomalies** | Unusual admin actions (role grants, policy changes) | Minutes | Admin account creates 50 new roles at 3 AM |
| **User reports** | "I can't log in" / "My account was modified" | Hours | Helpdesk ticket spike after credential leak |
| **External notifications** | Breach reports, CERT advisories, vendor notices | Days | HIBP notification: tenant domain in credential dump |
| **Infrastructure metrics** | CPU spike, DB connection exhaustion, NATS lag | Seconds | Auth service CPU 95% → DoS or brute force |
| **Honeypot / deception** | Decoy credentials used in login attempt | Minutes | Canary admin account triggers authentication |

### 1.2 Detection Rules

Six core detection rules should be implemented for IAM-specific threats:

**Rule 1 — Brute Force Detection**: N failed login attempts from a single IP within a time window. GGID's existing `RateLimiter` (login: 5/min) partially addresses this but lacks cross-window correlation.

**Rule 2 — Credential Stuffing**: High volume of login attempts with *different* usernames from distributed IPs, with a mix of successes and failures. Distinct from brute force (one account) — stuffing targets many accounts with leaked credentials.

**Rule 3 — Impossible Travel**: Successful authentication from geographically distant IPs within a timeframe inconsistent with travel. Requires GeoIP data (GGID has `geoip.go` middleware).

**Rule 4 — Mass Token Issuance**: Abnormal volume of JWT access tokens issued in a short window. Could indicate automated session creation or token harvesting.

**Rule 5 — Unusual API Access Patterns**: API calls from a new User-Agent, new ASN, or atypical hour patterns for a user.

**Rule 6 — Privilege Escalation**: Role or policy modifications outside change-management windows, or admin self-elevation.

### 1.3 Detection Rule Engine (Go)

```go
package detection

import (
	"context"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// DetectionRule evaluates audit events and fires alerts when conditions are met.
type DetectionRule interface {
	Name() string
	Severity() string
	Evaluate(ctx context.Context, events []*domain.AuditEvent) []*Alert
}

// Alert represents a triggered detection rule.
type Alert struct {
	RuleName     string    `json:"rule_name"`
	Severity     string    `json:"severity"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Description  string    `json:"description"`
	IPAddresses  []string  `json:"ip_addresses"`
	AffectedUsers []string `json:"affected_users"`
	Evidence     []string  `json:"evidence"` // event IDs
	Timestamp    time.Time `json:"timestamp"`
}

// BruteForceRule detects repeated failed logins from a single source IP.
type BruteForceRule struct {
	FailedThreshold int           // e.g. 20 failures
	Window          time.Duration // e.g. 5 minutes
}

func (r *BruteForceRule) Name() string     { return "brute_force_detection" }
func (r *BruteForceRule) Severity() string { return "SEV-2" }

func (r *BruteForceRule) Evaluate(_ context.Context, events []*domain.AuditEvent) []*Alert {
	ipFailures := make(map[string][]*domain.AuditEvent)
	cutoff := time.Now().Add(-r.Window)

	for _, e := range events {
		if e.CreatedAt.Before(cutoff) {
			continue
		}
		if e.Action == "user.login" && e.Result == domain.ResultFailure {
			ipFailures[e.IPAddress] = append(ipFailures[e.IPAddress], e)
		}
	}

	var alerts []*Alert
	for ip, fails := range ipFailures {
		if len(fails) >= r.FailedThreshold {
			affected := make(map[string]bool)
			var evidence []string
			for _, f := range fails {
				if f.ActorName != "" {
					affected[f.ActorName] = true
				}
				evidence = append(evidence, f.ID.String())
			}
			users := make([]string, 0, len(affected))
			for u := range affected {
				users = append(users, u)
			}
			alerts = append(alerts, &Alert{
				RuleName:      r.Name(),
				Severity:      r.Severity(),
				TenantID:      fails[0].TenantID,
				Description:   "Brute force login detected",
				IPAddresses:   []string{ip},
				AffectedUsers: users,
				Evidence:      evidence,
				Timestamp:     time.Now(),
			})
		}
	}
	return alerts
}

// CredentialStuffingRule detects many usernames from distributed IPs.
type CredentialStuffingRule struct {
	UniqueUserThreshold int           // e.g. 50 distinct usernames
	UniqueIPThreshold   int           // e.g. 20 distinct IPs
	Window              time.Duration
}

func (r *CredentialStuffingRule) Name() string     { return "credential_stuffing_detection" }
func (r *CredentialStuffingRule) Severity() string { return "SEV-2" }

func (r *CredentialStuffingRule) Evaluate(_ context.Context, events []*domain.AuditEvent) []*Alert {
	usernames := make(map[string]bool)
	ips := make(map[string]bool)
	var tenantID uuid.UUID
	cutoff := time.Now().Add(-r.Window)

	for _, e := range events {
		if e.CreatedAt.Before(cutoff) || e.Action != "user.login" {
			continue
		}
		usernames[e.ActorName] = true
		ips[e.IPAddress] = true
		tenantID = e.TenantID
	}

	if len(usernames) >= r.UniqueUserThreshold && len(ips) >= r.UniqueIPThreshold {
		ipList := make([]string, 0, len(ips))
		for ip := range ips {
			ipList = append(ipList, ip)
		}
		return []*Alert{{
			RuleName:    r.Name(),
			Severity:    r.Severity(),
			TenantID:    tenantID,
			Description: "Credential stuffing attack: many usernames from distributed IPs",
			IPAddresses: ipList,
			Timestamp:   time.Now(),
		}}
	}
	return nil
}

// DetectionEngine runs all registered rules against a window of events.
type DetectionEngine struct {
	rules []DetectionRule
}

func NewDetectionEngine(rules ...DetectionRule) *DetectionEngine {
	return &DetectionEngine{rules: rules}
}

func (e *DetectionEngine) Analyze(ctx context.Context, events []*domain.AuditEvent) []*Alert {
	var allAlerts []*Alert
	for _, rule := range e.rules {
		alerts := rule.Evaluate(ctx, events)
		allAlerts = append(allAlerts, alerts...)
	}
	return allAlerts
}
```

---

## 2. Incident Classification

### 2.1 Severity Levels

| Level | Name | Definition | Response Time | Escalation |
|-------|------|------------|---------------|------------|
| **SEV-1** | Critical | Active data breach, auth bypass in production, complete service outage | Immediate (15 min) | CTO, CISO, Legal |
| **SEV-2** | High | Authentication bypass for subset of users, token leak, privilege escalation | 1 hour | Security lead, on-call SRE |
| **SEV-3** | Medium | Rate limiting failure, partial degradation, suspicious activity without confirmed breach | 4 hours | On-call SRE |
| **SEV-4** | Low | Minor config issue, single-user access problem, non-security defect | 1 business day | Engineering team |

### 2.2 Classification Matrix

| Indicator | Affected Scope | Data Exposure | Severity |
|-----------|---------------|---------------|----------|
| JWT signing key leaked | Any | Full auth bypass | **SEV-1** |
| SQL injection in login | Any | PII/credentials | **SEV-1** |
| Auth bypass (any user → admin) | Any | Full system compromise | **SEV-1** |
| Compromised admin account | 1+ tenants | User data, policies | **SEV-1** or SEV-2 |
| Mass credential stuffing | 1 tenant | Potentially limited | **SEV-2** |
| Rate limiter bypass | 1+ endpoints | No direct data exposure | **SEV-3** |
| Tenant isolation failure | 2+ tenants | Cross-tenant data | **SEV-1** |
| Misconfigured CORS | Any | Token theft risk | **SEV-3** |
| Single user locked out | 1 user | None | **SEV-4** |

### 2.3 Incident Severity Classifier (Go)

```go
package classification

import (
	"time"

	"github.com/google/uuid"
)

// Incident represents a classified security incident.
type Incident struct {
	ID            uuid.UUID
	Title         string
	Severity      string // SEV-1 through SEV-4
	Status        string // detected, investigating, contained, resolved
	TenantsAffected int
	UsersAffected  int
	DataExposed    bool
	AuthBypass     bool
	TokenLeak      bool
	ServiceDown    bool
	DetectedAt     time.Time
}

// SeverityClassifier determines incident severity from indicators.
type SeverityClassifier struct{}

func (c *SeverityClassifier) Classify(inc *Incident) string {
	// SEV-1: existential threats
	if inc.AuthBypass || inc.TokenLeak || inc.DataExposed && inc.TenantsAffected > 1 {
		return "SEV-1"
	}

	// SEV-1: complete outage of auth or gateway
	if inc.ServiceDown {
		return "SEV-1"
	}

	// SEV-2: multi-tenant or large-scale impact
	if inc.TenantsAffected > 1 || inc.UsersAffected > 100 {
		return "SEV-2"
	}

	// SEV-2: data exposure in single tenant
	if inc.DataExposed {
		return "SEV-2"
	}

	// SEV-3: limited impact, no confirmed data exposure
	if inc.TenantsAffected == 1 || inc.UsersAffected > 0 {
		return "SEV-3"
	}

	return "SEV-4"
}
```

---

## 3. Containment Strategies

### 3.1 Immediate Containment Actions

When an incident is confirmed, execute containment actions in priority order:

1. **Revoke all tokens** for affected tenant(s) — invalidates every active JWT
2. **Disable compromised accounts** — prevents further authenticated actions
3. **Block suspicious IPs** — adds to gateway IP denylist
4. **Disable vulnerable endpoints** — selectively disable attack-surface endpoints
5. **Rotate signing keys** — new JWT signing key makes all prior tokens invalid

### 3.2 Containment Decision Tree

```
Incident Confirmed
├── Auth bypass / token leak?
│   ├── Rotate JWT signing keys (SEV-1, immediate)
│   ├── Revoke all sessions + refresh tokens
│   └── Force re-authentication for all users
├── Compromised admin account?
│   ├── Disable account
│   ├── Revoke all sessions for that user
│   ├── Audit all actions taken by account (last 7 days)
│   └── Review all role/policy changes made by account
├── Brute force / credential stuffing?
│   ├── Add source IPs to denylist
│   ├── Tighten rate limits (temporarily)
│   ├── Enable CAPTCHA on login
│   └── Force password reset for affected users
├── Tenant isolation breach?
│   ├── Quarantine affected tenants
│   ├── Review RLS policies
│   └── Audit all cross-tenant queries
```

### 3.3 Emergency Token Revocation (Go)

```go
package containment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// EmergencyRevocation provides immediate containment by invalidating tokens
// and sessions for affected scopes.
type EmergencyRevocation struct {
	rdb *redis.Client
}

func NewEmergencyRevocation(rdb *redis.Client) *EmergencyRevocation {
	return &EmergencyRevocation{rdb: rdb}
}

// RevokeTenantTokens marks all JWTs for a tenant as invalid by bumping the
// tenant's token epoch. The gateway checks this epoch on every request and
// rejects tokens issued before the current epoch.
func (er *EmergencyRevocation) RevokeTenantTokens(ctx context.Context, tenantID uuid.UUID) error {
	key := fmt.Sprintf("token_epoch:%s", tenantID)
	newEpoch := time.Now().Unix()
	if err := er.rdb.Set(ctx, key, newEpoch, 0).Err(); err != nil {
		return fmt.Errorf("set token epoch for tenant %s: %w", tenantID, err)
	}
	return nil
}

// BlockIPs adds IP addresses to the gateway denylist in Redis.
// The IP filter middleware reads this set on every request.
func (er *EmergencyRevocation) BlockIPs(ctx context.Context, ips []string, ttl time.Duration) error {
	if len(ips) == 0 {
		return nil
	}
	pipe := er.rdb.Pipeline()
	for _, ip := range ips {
		pipe.SAdd(ctx, "blocked_ips", ip)
	}
	pipe.Expire(ctx, "blocked_ips", ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// DisableAccount marks a user account as disabled and revokes all sessions.
func (er *EmergencyRevocation) DisableAccount(ctx context.Context, tenantID, userID uuid.UUID) error {
	// Mark account disabled
	disabledKey := fmt.Sprintf("account_disabled:%s:%s", tenantID, userID)
	if err := er.rdb.Set(ctx, disabledKey, "1", 0).Err(); err != nil {
		return fmt.Errorf("disable account: %w", err)
	}

	// Revoke all sessions by adding user to revocation set
	revokeKey := fmt.Sprintf("revoke_sessions:%s:%s", tenantID, userID)
	if err := er.rdb.Set(ctx, revokeKey, time.Now().Unix(), 0).Err(); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}

	return nil
}

// RotateSigningKeyFlag signals all auth instances to reload JWT signing keys.
// The auth service watches this key and performs hot reload.
func (er *EmergencyRevocation) RotateSigningKeyFlag(ctx context.Context) error {
	return er.rdb.Set(ctx, "jwt_key_rotation_required", time.Now().Unix(), 0).Err()
}
```

### 3.4 GGID-Specific Containment Levers

GGID already provides several containment mechanisms:

- **`LogoutAll`** (`auth/internal/service/logout_all.go`): revokes all sessions and refresh tokens for a user
- **`IPFilterStore.Set`** (`gateway/internal/middleware/ip_filter.go`): per-tenant IP denylist
- **`RateLimiter`** (`gateway/internal/middleware/ratelimit.go`): tighten per-endpoint limits
- **`JTIReplayTracker`** (`gateway/internal/middleware/jti_replay.go`): anti-replay enforcement

---

## 4. Eradication and Patching

### 4.1 Root Cause Identification

After containment, identify the root cause through:

1. **Code audit**: Review recent commits to affected service. Check `git log --since="7 days ago" -- <service>/`
2. **Log analysis**: Query audit events for the attacker's session timeline. Look for the initial entry point.
3. **Dependency scan**: Check if a known CVE in a dependency was exploited.
4. **Configuration review**: Diff current config against known-good baseline.

### 4.2 Applying the Fix

- Deploy hotfix to affected services via `docker compose up -d --build <service>`
- Rotate all secrets that may have been exposed (DB passwords, JWT keys, API keys)
- Update dependency versions if the root cause was a library vulnerability

### 4.3 Removing Attacker Artifacts

Attackers may leave persistent backdoors:

| Artifact Type | Where to Check | Cleanup Action |
|--------------|----------------|----------------|
| Backdoor admin accounts | `users` table, role assignments | Delete account, revoke all sessions |
| Injected roles/policies | `roles`, `policies` tables | Remove unauthorized entries |
| Rogue OAuth clients | `oauth_clients` table | Revoke client credentials |
| Stolen refresh tokens | Redis `refresh_tokens:*` | Delete token entries |
| Modified audit events | Audit HMAC chain verification | Detect tampered events |

### 4.4 Security Cleanup (Go)

```go
package eradication

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SecurityCleanup removes attacker artifacts from the database.
type SecurityCleanup struct {
	db *pgxpool.Pool
}

func NewSecurityCleanup(db *pgxpool.Pool) *SecurityCleanup {
	return &SecurityCleanup{db: db}
}

// RemoveUnauthorizedAccounts deletes accounts created during the incident window.
func (sc *SecurityCleanup) RemoveUnauthorizedAccounts(ctx context.Context, tenantID uuid.UUID, since time.Time) ([]string, error) {
	rows, err := sc.db.Query(ctx, `
		DELETE FROM users
		WHERE tenant_id = $1 AND created_at >= $2 AND created_by != 'system'
		RETURNING id, email
	`, tenantID, since)
	if err != nil {
		return nil, fmt.Errorf("remove unauthorized accounts: %w", err)
	}
	defer rows.Close()

	var removed []string
	for rows.Next() {
		var id uuid.UUID
		var email string
		if err := rows.Scan(&id, &email); err != nil {
			continue
		}
		removed = append(removed, fmt.Sprintf("%s (%s)", id, email))
	}
	return removed, nil
}

// RemoveInjectedPolicies deletes roles and policies created during the incident.
func (sc *SecurityCleanup) RemoveInjectedPolicies(ctx context.Context, tenantID uuid.UUID, since time.Time) (int, error) {
	// Delete policies created after the incident start
	cmd, err := sc.db.Exec(ctx, `
		DELETE FROM policies
		WHERE tenant_id = $1 AND created_at >= $2
	`, tenantID, since)
	if err != nil {
		return 0, fmt.Errorf("remove injected policies: %w", err)
	}
	return int(cmd.RowsAffected()), nil
}

// RevokeRogueOAuthClients disables OAuth clients created during the incident.
func (sc *SecurityCleanup) RevokeRogueOAuthClients(ctx context.Context, tenantID uuid.UUID, since time.Time) ([]string, error) {
	rows, err := sc.db.Query(ctx, `
		UPDATE oauth_clients
		SET disabled = true
		WHERE tenant_id = $1 AND created_at >= $2
		RETURNING client_id
	`, tenantID, since)
	if err != nil {
		return nil, fmt.Errorf("revoke rogue oauth clients: %w", err)
	}
	defer rows.Close()

	var revoked []string
	for rows.Next() {
		var clientID string
		if err := rows.Scan(&clientID); err != nil {
			continue
		}
		revoked = append(revoked, clientID)
	}
	return revoked, nil
}
```

### 4.5 Fix Verification

After applying a fix, verify the attack vector is closed:

```bash
# Reproduce the attack (in a staging environment)
./scripts/reproduce-attack.sh --vector=auth_bypass --tenant=test-tenant

# Verify rate limits are enforced
curl -X POST https://staging.ggid.local/api/v1/auth/login \
  -d '{"username":"test","password":"wrong"}' \
  --repeat 10 | grep "429"

# Verify token revocation is effective
curl -H "Authorization: Bearer $REVOKED_TOKEN" \
  https://staging.ggid.local/api/v1/users
# Expected: 401 Unauthorized
```

---

## 5. Recovery and Verification

### 5.1 Recovery Sequence

1. Confirm the fix is deployed and verified in staging
2. Deploy to production
3. Re-enable disabled endpoints
4. Gradually lift rate limit restrictions
5. Remove IP blocks (except confirmed malicious IPs)
6. Notify affected users of password reset requirements
7. Begin 72-hour elevated monitoring

### 5.2 Post-Recovery Monitoring Checklist

- [ ] All services passing health checks
- [ ] No new security alerts from detection engine
- [ ] Audit event volume returns to baseline
- [ ] No anomalous admin actions detected
- [ ] Rate limiter blocks at expected levels
- [ ] All critical alerts have owners on-call for next 72h
- [ ] SIEM dashboards show normal login patterns
- [ ] No spike in failed login or token refresh errors

### 5.3 Recovery Verification (Go)

```go
package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// RecoveryVerifier checks that the system has returned to normal operation
// after an incident has been contained and patched.
type RecoveryVerifier struct {
	auditService AuditReader
}

type AuditReader interface {
	ListEvents(ctx context.Context, filter domain.ListFilter, page, pageSize int) ([]*domain.AuditEvent, int, error)
}

func NewRecoveryVerifier(audit AuditReader) *RecoveryVerifier {
	return &RecoveryVerifier{auditService: audit}
}

// VerificationResult holds the outcome of recovery checks.
type VerificationResult struct {
	Healthy          bool
	FailedLoginRate  int   // failures per hour post-recovery
	BaselineRate     int   // expected normal failures per hour
	AnomaliesFound   int
	Details          []string
}

// Verify checks post-incident system health.
func (rv *RecoveryVerifier) Verify(ctx context.Context, tenantID uuid.UUID, incidentResolvedAt time.Time) (*VerificationResult, error) {
	result := &VerificationResult{Healthy: true}

	// Check failed login rate in the last hour
	since := time.Now().Add(-time.Hour)
	filter := domain.ListFilter{
		TenantID:  tenantID,
		Action:    "user.login",
		Result:    domain.ResultFailure,
		StartTime: &since,
	}
	events, total, err := rv.auditService.ListEvents(ctx, filter, 1, 1)
	if err != nil {
		return nil, fmt.Errorf("query failed logins: %w", err)
	}
	_ = events
	result.FailedLoginRate = total

	// Compare against baseline (e.g., <50 failures/hour is normal)
	result.BaselineRate = 50
	if total > result.BaselineRate {
		result.Healthy = false
		result.Details = append(result.Details,
			fmt.Sprintf("Failed login rate %d/hour exceeds baseline %d/hour", total, result.BaselineRate))
	}

	// Check for admin actions post-resolution (should be none outside business hours)
	adminFilter := domain.ListFilter{
		TenantID:  tenantID,
		Action:    "role.assign",
		StartTime: &incidentResolvedAt,
	}
	_, adminActions, err := rv.auditService.ListEvents(ctx, adminFilter, 1, 100)
	if err == nil && adminActions > 0 {
		result.AnomaliesFound += adminActions
		result.Details = append(result.Details,
			fmt.Sprintf("%d admin actions since resolution — verify legitimacy", adminActions))
	}

	if result.AnomaliesFound > 0 || !result.Healthy {
		result.Details = append(result.Details, "Extended monitoring recommended")
	}

	return result, nil
}
```

---

## 6. Postmortem Process

### 6.1 Principles

- **Blameless**: Focus on systems and processes, not individuals
- **Timely**: Draft within 48 hours of resolution, finalize within 1 week
- **Actionable**: Every root cause must produce a concrete action item with an owner
- **Shared**: Published to the entire engineering organization

### 6.2 Root Cause Analysis — Five Whys

Example for a JWT key leak incident:

1. **Why were tokens forgeable?** The JWT signing key was exposed.
2. **Why was the key exposed?** It was committed to a public Git repository.
3. **Why was it in the repo?** The developer hardcoded it in a config file for testing.
4. **Why did the config file get committed?** No `.gitignore` rule for secrets, and no pre-commit hook.
5. **Why no pre-commit hook?** Secret scanning was not part of the CI/CD pipeline setup.

**Root cause**: Missing secret-scanning CI guard. **Action**: Add `gitleaks` pre-commit hook and CI step.

### 6.3 Postmortem Template (Markdown)

```markdown
# Postmortem: [Incident Title]

**Date**: YYYY-MM-DD
**Severity**: SEV-X
**Status**: Resolved
**Authors**: [names]

## Summary

[1-2 paragraph executive summary of what happened, impact, and resolution]

## Timeline (all times UTC)

| Time | Event |
|------|-------|
| 14:03 | SIEM alert: unusual failed login spike from 203.0.113.x |
| 14:08 | On-call SRE acknowledges alert |
| 14:12 | Confirmed credential stuffing attack on tenant acme-corp |
| 14:15 | Containment: IPs blocked, rate limits tightened |
| 14:30 | Root cause identified: leaked credentials from third-party breach |
| 15:00 | Password reset forced for 847 affected users |
| 15:45 | All systems confirmed normal |
| 16:00 | Incident declared resolved |

## Impact

- **Users affected**: 847 accounts
- **Tenants affected**: 1 (acme-corp)
- **Data exposed**: None (attack blocked before successful auth)
- **Downtime**: 0 minutes
- **Duration**: ~2 hours from detection to resolution

## Root Cause

[Detailed root cause analysis with Five Whys]

## Contributing Factors

1. [Factor 1]
2. [Factor 2]
3. [Factor 3]

## What Went Well

- [Positive 1]
- [Positive 2]

## What Went Wrong

- [Negative 1]
- [Negative 2]

## Action Items

| # | Action | Owner | Priority | Due Date | Status |
|---|--------|-------|----------|----------|--------|
| 1 | Add CAPTCHA to login | @frontend | P1 | YYYY-MM-DD | Open |
| 2 | Implement HAVEIBEENPWNED integration | @auth | P2 | YYYY-MM-DD | Open |
| 3 | Add credential stuffing detection rule | @security | P1 | YYYY-MM-DD | Open |

## Lessons Learned

[Key takeaways for the broader team]
```

### 6.4 Postmortem Review Process

1. **Author drafts** within 48 hours using the template above
2. **Peer review** by at least one engineer not involved in the incident
3. **Review meeting** within 1 week — walk through timeline and action items
4. **Publish** to engineering wiki / internal docs
5. **Track action items** in issue tracker with assigned owners and deadlines
6. **Follow-up review** at 30 days to verify action items are complete

---

## 7. Communication Plan

### 7.1 Internal Communication

| Channel | Purpose | Audience |
|---------|---------|----------|
| `#incident-active` (Slack/Teams) | Real-time coordination during incident | On-call, responders, stakeholders |
| `#incident-updates` | Status updates every 30 minutes | Broader engineering team |
| Status page (internal) | Service health visibility | All employees |
| Email distribution list | Post-incident summary | Leadership, all engineering |

### 7.2 External Communication

| Scenario | Channel | Deadline | Template |
|----------|---------|----------|----------|
| Data breach (PII) | Customer email + regulatory | 72 hours (GDPR Art. 33) | Breach notification |
| Service outage | Status page + Twitter | 15 min from detection | Outage notice |
| Security advisory | Blog post + security mailing | 24-48 hours | Advisory |
| Resolved incident | Status page update + email | 1 hour after resolution | Resolution notice |

### 7.3 Customer Notification Template (GDPR Breach)

```
Subject: Security Incident Notification — [Date]

Dear [Customer Name],

We are writing to inform you of a security incident that was detected on
[Date] at [Time UTC]. Based on our investigation, [description of what
happened and what data was affected].

**What happened**: [Brief factual description]

**Data involved**: [Specific categories of personal data]

**What we have done**: [Containment actions taken]

**What you should do**: [Specific recommendations — e.g., reset passwords]

We have notified the relevant data protection authority as required under
Article 33 of the GDPR. We are committed to transparency and will provide
updates as our investigation continues.

For questions, contact: security@[company].com

Sincerely,
[Name], [Title]
```

### 7.4 Escalation Matrix

| Severity | First Responder | Escalate To (30 min) | Executive Notify |
|----------|----------------|---------------------|-----------------|
| SEV-1 | On-call SRE | Security lead, VP Eng | CTO, CISO, CEO |
| SEV-2 | On-call SRE | Security lead | VP Eng |
| SEV-3 | On-call SRE | Engineering manager | — |
| SEV-4 | Assigned engineer | — | — |

---

## 8. Incident Response Automation

### 8.1 Automated Response Pipeline

The goal is to reduce mean time to contain (MTTC) from hours to minutes through automated playbooks:

```
Detection Alert
    │
    ▼
┌─────────────┐
│ Classify     │ ── Severity + scope assessment
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Auto-Contain │ ── Based on severity: block IPs, revoke tokens
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Notify       │ ── Alert on-call, create incident channel
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Escalate     │ ── If not acknowledged in SLA, page next
└─────────────┘
```

### 8.2 NATS Event-Driven Automated Containment

GGID's NATS JetStream infrastructure provides the perfect backbone for event-driven IR automation. The audit service already consumes events from NATS. An IR automation service can subscribe to the same stream and trigger containment:

```go
package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// AutoResponseConfig defines automated containment thresholds.
type AutoResponseConfig struct {
	AutoBlockIPs           bool          // Block IPs with > threshold failures
	BruteForceThreshold    int           // Failures before auto-block
	AutoBlockTTL           time.Duration // How long to keep IP blocked
	AutoRevokeOnSeverity   string        // Auto-revoke tokens at this severity ("SEV-1")
	NotifyChannel          string        // NATS subject for notifications
}

// DefaultAutoResponseConfig returns production-safe defaults.
func DefaultAutoResponseConfig() AutoResponseConfig {
	return AutoResponseConfig{
		AutoBlockIPs:         true,
		BruteForceThreshold:  50,
		AutoBlockTTL:         24 * time.Hour,
		AutoRevokeOnSeverity: "SEV-1",
		NotifyChannel:        "ir.alerts",
	}
}

// AutoResponsePipeline listens for detection alerts on NATS and executes
// automated containment actions.
type AutoResponsePipeline struct {
	js     jetstream.JetStream
	nc     *nats.Conn
	cfg    AutoResponseConfig
	stopCh chan struct{}
}

func NewAutoResponsePipeline(nc *nats.Conn, js jetstream.JetStream, cfg AutoResponseConfig) *AutoResponsePipeline {
	return &AutoResponsePipeline{
		nc:     nc,
		js:     js,
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

// AlertMessage is the NATS message format for IR alerts.
type AlertMessage struct {
	RuleName      string   `json:"rule_name"`
	Severity      string   `json:"severity"`
	TenantID      string   `json:"tenant_id"`
	IPAddresses   []string `json:"ip_addresses"`
	AffectedUsers []string `json:"affected_users"`
	Description   string   `json:"description"`
}

// Start begins consuming IR alerts and executing automated responses.
func (p *AutoResponsePipeline) Start(ctx context.Context) error {
	cons, err := p.js.CreateOrUpdateConsumer(ctx, "AUDIT_EVENTS", jetstream.ConsumerConfig{
		Name:          "ir-automation",
		Durable:       "ir-automation",
		FilterSubject: "ir.alerts",
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("create ir consumer: %w", err)
	}

	go func() {
		for {
			select {
			case <-p.stopCh:
				return
			case <-ctx.Done():
				return
			default:
			}

			batch, err := cons.FetchNoWait(10)
			if err != nil {
				if err == jetstream.ErrNoMessages {
					time.Sleep(time.Second)
					continue
				}
				log.Printf("IR automation: fetch error: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			for msg := range batch.Messages() {
				var alert AlertMessage
				if err := json.Unmarshal(msg.Data(), &alert); err != nil {
					log.Printf("IR automation: decode error: %v", err)
					msg.Ack()
					continue
				}

				if err := p.handleAlert(ctx, &alert); err != nil {
					log.Printf("IR automation: handle error: %v", err)
					msg.Nak()
				} else {
					msg.Ack()
				}
			}
		}
	}()

	return nil
}

func (p *AutoResponsePipeline) handleAlert(ctx context.Context, alert *AlertMessage) error {
	// Automated IP blocking
	if p.cfg.AutoBlockIPs && len(alert.IPAddresses) > 0 {
		blockMsg, _ := json.Marshal(map[string]any{
			"action":    "block_ips",
			"ips":       alert.IPAddresses,
			"ttl":       p.cfg.AutoBlockTTL.String(),
			"reason":    alert.RuleName,
			"timestamp": time.Now().UTC(),
		})
		if err := p.nc.Publish("gateway.cmd.block_ips", blockMsg); err != nil {
			return fmt.Errorf("publish block_ips command: %w", err)
		}
		log.Printf("IR automation: blocked %d IPs for %s", len(alert.IPAddresses), alert.RuleName)
	}

	// Automated token revocation for SEV-1
	if alert.Severity == p.cfg.AutoRevokeOnSeverity && alert.TenantID != "" {
		revokeMsg, _ := json.Marshal(map[string]any{
			"action":    "revoke_tenant_tokens",
			"tenant_id": alert.TenantID,
			"reason":    alert.RuleName,
			"timestamp": time.Now().UTC(),
		})
		if err := p.nc.Publish("auth.cmd.revoke_tokens", revokeMsg); err != nil {
			return fmt.Errorf("publish revoke command: %w", err)
		}
		log.Printf("IR automation: revoked all tokens for tenant %s", alert.TenantID)
	}

	// Publish notification
	notifyMsg, _ := json.Marshal(alert)
	if err := p.nc.Publish(p.cfg.NotifyChannel, notifyMsg); err != nil {
		log.Printf("IR automation: notify error: %v", err)
	}

	return nil
}

func (p *AutoResponsePipeline) Stop() {
	close(p.stopCh)
}
```

### 8.3 SOAR Integration Points

GGID's NATS subjects provide clean integration points for SOAR platforms (Splunk SOAR, Tines, Shuffle):

| NATS Subject | Direction | Purpose |
|-------------|-----------|---------|
| `ir.alerts` | Audit → SOAR | Forward detection alerts |
| `gateway.cmd.block_ips` | SOAR → Gateway | Block IP addresses |
| `auth.cmd.revoke_tokens` | SOAR → Auth | Revoke tokens |
| `auth.cmd.disable_account` | SOAR → Auth | Disable user account |
| `ir.notifications` | SOAR → Slack/Teams | Send alert notifications |

---

## 9. Tabletop Exercises

### 9.1 Why Tabletop Exercises

Tabletop exercises are the most cost-effective way to test incident response readiness. They expose gaps in process, tooling, and communication before a real incident does.

### 9.2 IAM-Specific Scenarios

**Scenario A — Compromised Admin Account**

> At 2:47 AM, the SIEM fires an alert: an admin account created 47 new roles and modified 12 policies in the last hour. The admin's typical login time is 9 AM–5 PM. The login originated from an IP in a country where the admin has never authenticated before.

Discussion questions:
- How do you confirm the account is compromised?
- What containment actions do you take first?
- How do you audit what the attacker changed?
- How do you identify and remove injected policies?
- How do you communicate with the affected tenant?

**Scenario B — JWT Signing Key Leak**

> A developer discovers that the JWT private key was accidentally committed to a public GitHub repository 3 days ago. The key has not been rotated. Anyone with the key can forge valid JWTs for any user in any tenant.

Discussion questions:
- What is the blast radius?
- How quickly can you rotate the signing key?
- What happens to existing valid user sessions during key rotation?
- How do you detect if forged tokens were already used?
- What is the customer communication strategy?

**Scenario C — Multi-Tenant Data Breach**

> A bug report reveals that tenant A's API call returned data belonging to tenant B. The RLS policy on the `users` table was accidentally disabled during a migration.

Discussion questions:
- How do you determine which tenants' data was exposed?
- What is the GDPR notification timeline?
- How do you prevent further cross-tenant access?
- How do you restore RLS policies safely?

**Scenario D — Auth Service DoS**

> The auth service is receiving 10,000 login requests per second. CPU is at 100%, and legitimate users cannot log in. The requests come from 500+ IPs across multiple ASNs.

Discussion questions:
- How do you distinguish DoS from a credential stuffing attack?
- What rate limiting and IP blocking strategies do you deploy?
- Do you take the auth service offline or try to stay up?
- How do you communicate with users who cannot log in?

### 9.3 Exercise Facilitation Guide

1. **Prepare** (1 week before): Define scenario, objectives, and participant roles. Create a shared document for notes.
2. **Brief** (15 min): Explain the scenario, rules of engagement, and expected outcomes.
3. **Execute** (60-90 min): Walk through the scenario timeline. Inject new information at intervals. Participants discuss what they would do.
4. **Debrief** (30 min): What worked? What didn't? What are the gaps?
5. **Document** (within 48h): Write up findings as action items.

### 9.4 Scoring Rubric

| Dimension | Weight | Score (1-5) |
|-----------|--------|-------------|
| Detection speed | 20% | How quickly was the incident detected? |
| Classification accuracy | 15% | Was severity correctly assessed? |
| Containment speed | 25% | How fast was the threat contained? |
| Communication effectiveness | 15% | Were stakeholders notified timely? |
| Process adherence | 15% | Did the team follow the IR playbook? |
| Documentation quality | 10% | Was the incident well documented? |

---

## 10. GGID IR Playbook

### 10.1 Existing Detection Capabilities

Based on source code review of the GGID codebase:

| Capability | Location | Status |
|-----------|----------|--------|
| **Audit event logging** | `services/audit/internal/domain/models.go` | Complete — HMAC hash chain, actor tracking, result tracking |
| **NATS JetStream audit pipeline** | `services/audit/internal/consumer/nats_consumer.go` | Complete — 72h retention, 1GB limit, durable consumer |
| **Failed login tracking** | `services/audit/internal/domain/stats.go` | Partial — `FailedLogins24h` tracked in Stats, but no alerting |
| **Audit query API** | `services/audit/internal/service/audit_service.go` | Complete — List, Filter, GetStats, CleanupOldEvents |
| **Rate limiting** | `services/gateway/internal/middleware/ratelimit.go` | Complete — per-endpoint (login: 5/min, register: 3/min) |
| **IP allow/deny lists** | `services/gateway/internal/middleware/ip_filter.go` | Complete — per-tenant CIDR filtering |
| **JTI anti-replay** | `services/gateway/internal/middleware/jti_replay.go` | Complete — in-memory tracking (needs Redis for HA) |
| **Bot detection** | `services/gateway/internal/middleware/botdetect.go` | Complete — known patterns + behavioral detection |
| **Token revocation** | `services/auth/internal/service/logout_all.go` | Partial — revokes sessions + refresh tokens per user, not per-tenant |
| **Behavioral rate limiting** | `services/gateway/internal/middleware/sliding_ratelimit.go` | Complete — sliding window per-tenant |
| **Tiered rate limiting** | `services/gateway/internal/middleware/tier_ratelimit.go` | Complete — tier-based limits |
| **Circuit breaker** | `services/gateway/internal/middleware/circuitbreaker.go` | Complete — protects downstream services |
| **GeoIP tracking** | `services/gateway/internal/middleware/geoip.go` | Available — supports impossible travel detection |
| **WebSocket audit stream** | `services/audit/internal/server/ws.go` | Complete — real-time audit event streaming |
| **Audit HMAC chain** | `services/audit/` (Hash field in AuditEvent) | Available — tamper detection |

### 10.2 Existing Response Capabilities

| Capability | Implementation | Gaps |
|-----------|---------------|------|
| **Token revocation (per user)** | `LogoutAll` revokes sessions + refresh tokens | No tenant-wide or global revocation |
| **IP blocking** | `IPFilterStore.Set` + per-tenant denylist | Not automated — requires manual config |
| **Rate limit tightening** | Configurable `RateLimitConfig` | No runtime adjustment API |
| **Account disabling** | Auth service user management | Not integrated with automated IR |
| **Key rotation** | `TokenService` loads from file | No hot-reload; requires restart |

### 10.3 Missing Capabilities (Gap Assessment)

1. **No detection rule engine** — GGID collects audit data but has no rules to trigger alerts. The `FailedLogins24h` stat is computed but never compared against thresholds.

2. **No automated containment** — All containment actions (IP blocking, token revocation, account disabling) require manual intervention. There is no NATS-driven automated response pipeline.

3. **No tenant-wide token revocation** — `LogoutAll` revokes per-user but there is no mechanism to revoke all tokens for an entire tenant (needed for SEV-1 containment).

4. **No JWT signing key hot-rotation** — Key rotation requires restarting the auth service. In a SEV-1 incident, this means downtime during containment.

5. **No impossible travel detection** — GeoIP data is collected but never correlated with login events to detect impossible travel.

6. **No SIEM integration** — Audit events are stored in PostgreSQL and streamed via WebSocket, but there is no forwarder to external SIEM (Splunk, Elastic, Datadog).

7. **No incident tracking system** — No internal tool to track incident lifecycle (detected → investigating → contained → resolved → postmortem).

---

## 11. Gap Analysis & Recommendations

### 11.1 Priority Action Items

| # | Action | Effort | Priority | Impact |
|---|--------|--------|----------|--------|
| 1 | **Implement detection rule engine** — Create `pkg/detection` with brute force, credential stuffing, impossible travel, and privilege escalation rules. Subscribe to NATS audit stream. Publish alerts to `ir.alerts`. | Medium (3-5 days) | P1 | Reduces MTTD from hours to minutes |
| 2 | **Add tenant-wide token revocation** — Extend `TokenService` with `RevokeTenantTokens` that bumps a Redis token epoch checked by the gateway on every request. | Small (1-2 days) | P1 | Enables SEV-1 containment in seconds |
| 3 | **Implement JWT key hot-rotation** — Watch a Redis key for rotation signal. On change, reload signing keys from filesystem or secret manager without restart. | Medium (2-3 days) | P1 | Eliminates downtime during key rotation |
| 4 | **Build NATS-driven auto-response pipeline** — Subscribe to `ir.alerts`, auto-block IPs for brute force, auto-revoke for SEV-1, auto-create incident Slack channel. | Medium (3-5 days) | P2 | Reduces MTTC from hours to minutes |
| 5 | **Add SIEM forwarder** — Create `pkg/siem` that forwards audit events via Syslog or HTTP to external SIEM (Splunk HEC, Elastic, Datadog Logs). | Small (2-3 days) | P2 | Enables enterprise security monitoring |
| 6 | **Conduct quarterly tabletop exercise** — Use scenarios from Section 9. Rotate facilitators. Track action items in issue tracker. | Small (recurring) | P3 | Validates IR readiness over time |

### 11.2 Recommended Implementation Sequence

```
Phase 1 (Week 1-2): Detection foundation
├── pkg/detection/rule_engine.go (brute force + credential stuffing)
├── pkg/detection/impossible_travel.go (GeoIP correlation)
└── NATS alert publisher (ir.alerts subject)

Phase 2 (Week 2-3): Containment capability
├── TokenService.RevokeTenantTokens (Redis epoch)
├── Gateway epoch check middleware
└── JWT key hot-rotation (Redis watch + reload)

Phase 3 (Week 3-4): Automation
├── Auto-response pipeline (NATS consumer)
├── Slack/Teams incident channel creation
└── PagerDuty/Opsgenie integration

Phase 4 (Ongoing): Process
├── Tabletop exercise (quarterly)
├── Postmortem template adoption
└── IR playbook documentation maintenance
```

### 11.3 Metrics to Track

| Metric | Target | Current |
|--------|--------|---------|
| Mean Time to Detect (MTTD) | < 5 minutes | Unknown (manual) |
| Mean Time to Contain (MTTC) | < 30 minutes | Unknown (manual) |
| Mean Time to Recover (MTTR) | < 4 hours | Unknown |
| Detection rule coverage | 6 core rules | 0 rules |
| Automated containment actions | 3+ scenarios | 0 scenarios |
| Tabletop exercises per year | 4 (quarterly) | 0 |

---

## Appendix A: Quick Reference — GGID IR Commands

```bash
# View recent failed logins for a tenant
curl -H "X-Tenant-ID: $TENANT_ID" \
  "http://localhost:8072/api/v1/audit/events?action=user.login&result=failure&start_time=$(date -d '1 hour ago' -u +%Y-%m-%dT%H:%M:%SZ)"

# Block an IP (manual, via gateway admin API)
curl -X POST "http://localhost:8080/admin/block-ip" \
  -d '{"ip":"203.0.113.42","ttl":"24h"}'

# Revoke all sessions for a user
curl -X POST "http://localhost:9001/api/v1/auth/logout-all" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"user_id":"<uuid>"}'

# Check NATS audit stream health
nats stream info AUDIT_EVENTS --server nats://localhost:4222

# View real-time audit events via WebSocket
wscat -c "ws://localhost:8072/ws/audit?tenant_id=$TENANT_ID"
```

## Appendix B: References

- NIST SP 800-61 Rev. 2: Computer Security Incident Handling Guide
- GDPR Article 33: Notification of personal data breach to supervisory authority
- ISO/IEC 27035: Information security incident management
- SANS Incident Response Process: Preparation → Identification → Containment → Eradication → Recovery → Lessons Learned
