# Identity Recovery

Account recovery methods, recovery flow security, time-delayed recovery, audit trail, emergency access (break-glass), and disaster recovery for identity infrastructure.

## Recovery Methods

| Method | Security | Speed | Use Case |
|--------|---------|-------|---------|
| Recovery codes | High | Instant | Pre-generated at MFA enrollment |
| Backup email | Medium | Minutes | Secondary email verification |
| Backup phone | Medium | Minutes | SMS OTP to backup number |
| Social recovery | Low | Hours | Trusted contact vouches |
| Admin-assisted | High | Hours | Identity verification + admin approval |
| Break-glass | Max | Minutes | Emergency, dual approval |

## Recovery Flow Security

```
1. User clicks "Forgot password" / "Lost MFA device"
2. GGID verifies identity:
   a. Recovery code? → Enter code
   b. No code? → Email + phone + security questions
   c. Still stuck? → Admin-assisted (identity proofing)
3. Time delay (24h for admin-assisted) to detect hijacking
4. User re-enrolls MFA factors
5. All previous sessions revoked
6. Audit trail records entire recovery
```

### Time-Delayed Recovery

```go
func initiateRecovery(userID string, method string) error {
    if method == "admin_assisted" {
        // 24h delay to allow detection of social engineering
        recovery := &Recovery{
            UserID:    userID,
            Status:    "pending",
            AvailableAt: time.Now().Add(24 * time.Hour),
        }
        store.Create(recovery)

        // Notify user of pending recovery
        notifyUser(userID, "Recovery initiated. Available in 24h.")
        notifyAdmins("Recovery pending for " + userID)
        return nil
    }
    // Self-service methods proceed immediately
    return selfServiceRecovery(userID, method)
}
```

## Break-Glass (Emergency Access)

```bash
POST /api/v1/admin/break-glass
{
  "user_id": "admin-uuid",
  "reason": "Production outage — need admin access immediately",
  "approver_id": "secondary-admin-uuid"
}
# → Requires DUAL approval
# → Time-boxed: 30 minutes max
# → Full audit recording
# → Auto-revoke after expiry
```

| Rule | Value |
|------|-------|
| Max duration | 30 minutes |
| Requires | 2 approvers (requester + approver) |
| Audit | Every action logged in real-time |
| Auto-revoke | After 30 min, no extension |
| Alert | CISO + security team notified |

## Disaster Recovery for Identity Infrastructure

| Component | RTO | RPO | Strategy |
|-----------|-----|-----|---------|
| PostgreSQL | 15 min | 5 min | WAL streaming + PITR |
| Redis | 5 min | 0 | Replica failover |
| NATS JetStream | 5 min | 0 | Stream replication |
| JWT signing keys | Immediate | 0 | Pre-shared to all regions |
| JWKS endpoint | Immediate | 0 | CDN cached |

### RTO/RPO Definitions

| Term | Meaning |
|------|---------|
| RTO | Recovery Time Objective — max downtime |
| RPO | Recovery Point Objective — max data loss |

## Monitoring

| Metric | Alert |
|--------|-------|
| Recovery attempts | Spike → possible social engineering |
| Break-glass usage | Any → security review |
| DR drill overdue | >6 months → schedule |
| Recovery completion time | Track for SLA |

## See Also

- [Identity Recovery Playbook](identity-recovery-playbook.md)
- [Passkey Recovery Strategy](passkey-recovery-strategy.md)
- [MFA Architecture](mfa-architecture.md)
- [Backup and Restore](backup-and-restore.md)
- [Disaster Recovery Testing](disaster-recovery-testing.md)
