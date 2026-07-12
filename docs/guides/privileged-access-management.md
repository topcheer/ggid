# Privileged Access Management

This guide covers PAM overview, privileged account discovery, JIT elevation, break-glass, session recording, command logging, time-boxed access, approval workflow, credential vaulting, and GGID's PAM implementation.

## PAM Overview

### What is PAM?

Privileged Access Management (PAM) secures and monitors access to critical systems by privileged users (administrators, DevOps, security teams). PAM reduces the risk of privileged credential theft, misuse, and insider threats.

### Key Principles

| Principle | Description |
|---|---|
| Least privilege | Grant minimum access needed |
| Just-in-time (JIT) | Access granted only when needed, for limited time |
| Session isolation | Privileged sessions are isolated and monitored |
| Credential vaulting | Passwords stored securely, not known to users |
| Full audit | Every privileged action is logged |

### Privileged Account Types

| Type | Description | Risk |
|---|---|---|
| Platform admin | Full system access | Critical |
| Security admin | Security config, audit access | Critical |
| Database admin | DB access, schema changes | High |
| Network admin | Network config, firewall rules | High |
| DevOps admin | CI/CD, deployment access | High |
| Application admin | App config, user management | Medium |
| Service accounts | Non-human automated accounts | Medium-High |

## Privileged Account Discovery

### Discovery Process

```go
func DiscoverPrivilegedAccounts(tenantID string) []PrivilegedAccount {
    var accounts []PrivilegedAccount

    // 1. Scan role assignments for privileged roles
    for _, role := range getPrivilegedRoles(tenantID) {
        for _, user := range role.Members {
            accounts = append(accounts, PrivilegedAccount{
                UserID:    user.ID,
                UserName:  user.Name,
                RoleID:    role.ID,
                RoleName:  role.Name,
                Source:    "role_assignment",
                RiskLevel: role.RiskLevel,
            })
        }
    }

    // 2. Scan for direct permission grants
    for _, perm := range getDirectGrants(tenantID) {
        if isPrivilegedPermission(perm) {
            accounts = append(accounts, PrivilegedAccount{
                UserID:     perm.UserID,
                Permission: perm.Name,
                Source:     "direct_grant",
                RiskLevel:  perm.RiskLevel,
            })
        }
    }

    // 3. Scan service accounts
    for _, svc := range getServiceAccounts(tenantID) {
        accounts = append(accounts, PrivilegedAccount{
            AccountID: svc.ID,
            Type:      "service",
            Source:    "service_account",
            RiskLevel: "high",
        })
    }

    // 4. Detect shadow admin (users with effective admin via inheritance)
    for _, user := range getAllUsers(tenantID) {
        effectivePerms := computeEffectivePermissions(user)
        if hasAdminPermissions(effectivePerms) && !user.HasAdminRole {
            accounts = append(accounts, PrivilegedAccount{
                UserID:     user.ID,
                Source:     "shadow_admin",
                RiskLevel:  "critical",
                Note:       "Effective admin via permission inheritance",
            })
        }
    }

    return accounts
}
```

## JIT Elevation

### Just-in-Time Access

Instead of standing privileged access, users request temporary elevation:

```
1. User needs admin access for a task
2. User submits elevation request (what, why, how long)
3. Approver reviews request
4. If approved: temporary role granted (e.g., 1 hour)
5. User performs task with elevated access
6. Access automatically expires
7. Session recorded for audit
```

### Request

```bash
POST /api/v1/pam/elevate
Authorization: Bearer <user_token>

{
  "target_role": "database-admin",
  "duration": "1h",
  "reason": "Production database schema migration",
  "ticket_ref": "JIRA-12345",
  "approver": "security-team"
}
```

### Approval Flow

```go
func (s *PAMService) RequestElevation(req *ElevationRequest) (*ElevationRequest, error) {
    // Validate request
    if req.Duration > s.config.MaxElevationDuration {
        return nil, ErrDurationTooLong
    }
    if !s.hasValidTicket(req.TicketRef) {
        return nil, ErrMissingTicketRef
    }

    // Create request
    request := &ElevationRequest{
        ID:        uuid.New().String(),
        UserID:    req.UserID,
        RoleID:    req.TargetRole,
        Duration:  req.Duration,
        Reason:    req.Reason,
        TicketRef: req.TicketRef,
        Status:    "pending_approval",
        CreatedAt: time.Now(),
    }

    // Route to approver
    approver := s.determineApprover(req)
    s.notifyApprover(approver, request)

    return request, nil
}

func (s *PAMService) ApproveElevation(requestID, approverID string) error {
    req := s.getRequest(requestID)
    req.Status = "approved"
    req.ApprovedBy = approverID
    req.ApprovedAt = time.Now()
    req.ExpiresAt = time.Now().Add(req.Duration)

    // Grant temporary role
    s.grantTemporaryRole(req.UserID, req.RoleID, req.ExpiresAt)

    // Start session recording
    s.startSessionRecording(req)

    // Audit
    audit.Log("pam_elevation_approved", req.UserID, req.RoleID, approverID)

    return nil
}
```

### Auto-Expiration

```go
func (s *PAMService) CleanupExpiredElevations() {
    expired := s.getExpiredElevations()
    for _, req := range expired {
        s.revokeTemporaryRole(req.UserID, req.RoleID)
        s.stopSessionRecording(req)
        audit.Log("pam_elevation_expired", req.UserID, req.RoleID)
    }
}
```

## Break-Glass Access

### What is Break-Glass?

Emergency access for critical situations when normal approval workflow is too slow:

| Scenario | Example |
|---|---|
| Production outage | System down, need immediate admin access |
| Security incident | Active attack, need to lock down systems |
| After-hours emergency | No approver available, critical fix needed |

### Break-Glass Process

```
1. User uses break-glass credential (from vault)
2. User authenticates with break-glass account
3. Full access granted immediately
4. Session recorded with enhanced logging
5. Security team notified in real-time
6. Post-incident review mandatory within 24h
7. Break-glass credential rotated after use
```

### Configuration

```yaml
pam:
  break_glass:
    enabled: true
    accounts:
      - name: "emergency-admin"
        credential: "vault://pam/break-glass/admin"
        max_sessions: 1
        require_mfa: true
        notify:
          - "security-team@example.com"
          - "on-call@example.com"
    post_use:
      mandatory_review: 24h
      credential_rotation: true
      audit_escalation: true
```

## Session Recording

### What to Record

| Data | Description | Storage |
|---|---|---|
| Commands | Every command executed | Audit log (immutable) |
| Keystrokes | Optional keystroke logging | Encrypted blob storage |
| Screen | Optional screen recording | Video storage (30-day retention) |
| I/O | Input/output of privileged commands | Audit log |
| Network | Connections made during session | Network log |

### Implementation

```go
type SessionRecorder struct {
    SessionID  string
    UserID     string
    RoleID     string
    StartTime  time.Time
    Commands   []CommandLog
    Events     []SessionEvent
}

type CommandLog struct {
    Timestamp time.Time
    Command   string
    Args      []string
    Output    string  // Truncated if too long
    ExitCode  int
}

func (r *SessionRecorder) LogCommand(cmd string, args []string, output string, exitCode int) {
    r.Commands = append(r.Commands, CommandLog{
        Timestamp: time.Now(),
        Command:   cmd,
        Args:      args,
        Output:    truncate(output, 10000),  // Limit output size
        ExitCode:  exitCode,
    })
    audit.Log("pam_command", r.UserID, r.RoleID, cmd, exitCode)
}
```

## Command Logging

### Command Filtering

```yaml
pam:
  command_logging:
    log_all: true
    dangerous_commands:
      alert:
        - "rm -rf"
        - "dd if="
        - "mkfs"
        - "shutdown"
        - "reboot"
        - "iptables -F"
      block:
        - "rm -rf /"
        - "chmod 777"
    sensitive_patterns:
      - pattern: "password="
        action: "redact"
      - pattern: "token="
        action: "redact"
      - pattern: "BEGIN.*PRIVATE KEY"
        action: "redact"
```

### Real-Time Alerting

```go
func (r *SessionRecorder) checkDangerousCommand(cmd string) {
    for _, pattern := range r.config.AlertPatterns {
        if strings.Contains(cmd, pattern) {
            alertSecurity(r.UserID, r.RoleID, cmd, "dangerous_command")
            break
        }
    }
    for _, pattern := range r.config.BlockPatterns {
        if strings.Contains(cmd, pattern) {
            alertSecurity(r.UserID, r.RoleID, cmd, "blocked_command")
            r.terminateSession()
            break
        }
    }
}
```

## Time-Boxed Access

### Access Windows

```yaml
pam:
  time_boxing:
    max_elevation_duration: 4h
    default_duration: 1h
    extensions:
      allowed: true
      max_extension: 2h
      require_reapproval: true
    automatic_expiration: true
    grace_period: 5m  # 5-minute warning before expiration
```

## Approval Workflow

### Approval Models

| Model | Description | Use Case |
|---|---|---|
| Single approver | One person approves | Standard elevation |
| Multi-approver | 2+ approvals required | Critical systems |
| Auto-approve | Based on policy | Low-risk, short duration |
| Break-glass | No approval, post-review | Emergency |

### Configuration

```yaml
pam:
  approval:
    models:
      low_risk:
        duration_max: 30m
        model: "auto_approve"
        conditions:
          - managed_device: true
          - business_hours: true
      medium_risk:
        duration_max: 2h
        model: "single_approver"
        approver: "manager"
      high_risk:
        duration_max: 4h
        model: "multi_approver"
        approvers: ["security_admin", "manager"]
      critical:
        model: "multi_approver"
        approvers: ["security_admin", "cto"]
        require_ticket: true
        require_mfa: true
```

## Credential Vaulting

### How It Works

```
1. Privileged credentials stored in encrypted vault
2. User requests access → credential retrieved from vault
3. Credential injected into session (user never sees it)
4. Session uses credential for authentication
5. After session: credential rotated in vault
6. User never knows the actual password
```

### Vault Integration

```go
func (s *PAMService) GetVaultedCredential(userID, targetSystem string) (string, error) {
    // Check if user has active elevation
    elevation := s.getActiveElevation(userID)
    if elevation == nil {
        return "", ErrNoActiveElevation
    }

    // Retrieve credential from vault
    cred, err := s.vault.Get(fmt.Sprintf("pam/%s/%s", targetSystem, elevation.RoleID))
    if err != nil {
        return "", err
    }

    // Audit credential access
    audit.Log("pam_credential_accessed", userID, targetSystem, elevation.RoleID)

    // Schedule rotation after session
    s.scheduleRotation(targetSystem, elevation.RoleID)

    return cred, nil
}
```

## GGID PAM Implementation

### API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/pam/discover` | GET | Discover privileged accounts |
| `/api/v1/pam/elevate` | POST | Request elevation |
| `/api/v1/pam/elevate/{id}/approve` | POST | Approve elevation |
| `/api/v1/pam/elevate/{id}/deny` | POST | Deny elevation |
| `/api/v1/pam/sessions` | GET | List active sessions |
| `/api/v1/pam/sessions/{id}` | GET | Get session details |
| `/api/v1/pam/sessions/{id}/commands` | GET | Get session commands |
| `/api/v1/pam/break-glass` | POST | Break-glass access |

### Configuration

```yaml
pam:
  enabled: true
  jit:
    max_duration: 4h
    default_duration: 1h
    auto_expire: true
  break_glass:
    enabled: true
    post_review: 24h
  session_recording:
    enabled: true
    log_commands: true
    log_io: true
    screen_recording: false  # Enable for high-security
  command_alerting:
    enabled: true
    block_dangerous: true
  credential_vaulting:
    enabled: true
    rotation_after_use: true
  approval:
    models: ["single", "multi", "auto", "break-glass"]
```

## Best Practices

1. **Use JIT everywhere** — No standing privileged access
2. **Require approval for all elevation** — Except break-glass
3. **Time-box all access** — Maximum 4 hours per elevation
4. **Record everything** — Commands, I/O, optionally screen
5. **Alert on dangerous commands** — Real-time security notification
6. **Rotate credentials after use** — Never reuse vaulted passwords
7. **Conduct post-use reviews** — Break-glass mandatory within 24h
8. **Discover shadow admins** — Find effective privileged access via inheritance
9. **Separate duties** — Approver ≠ requester
10. **Regular access reviews** — Monthly certification of privileged access