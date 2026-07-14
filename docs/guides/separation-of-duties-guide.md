# Separation of Duties Guide

This guide covers SoD principles, conflict role pairs, rule engine design, conflict detection, auto-remediation, compliance requirements, and GGID's SoD implementation.

## SoD Principle

Separation of Duties ensures no single person can complete a critical task alone. This prevents fraud, errors, and insider threats by requiring multiple people for sensitive operations.

## Conflict Role Pairs

| Pair | Conflict | Risk |
|---|---|---|
| Requester vs Approver | Same person requests and approves access | Unauthorized access |
| Developer vs Deployer | Same person writes and deploys code | Malicious code in prod |
| Admin vs Auditor | Same person administers and audits | Audit tampering |
| Creator vs Reviewer | Same person creates and reviews policy | Weak controls |
| User Creator vs User Deleter | Same person creates and deletes users | Account manipulation |
| Key Generator vs Key User | Same person generates and uses keys | Key compromise |
| Config Editor vs Config Approver | Same person edits and approves config | Misconfiguration |

## SoD Rule Engine

### Rule Definition

```yaml
sod:
  rules:
    - name: "requester_not_approver"
      conflict: ["access_requester", "access_approver"]
      action: "prevent"
      message: "Cannot request and approve own access"
    - name: "dev_not_deployer"
      conflict: ["developer", "deployer"]
      action: "warn"
      message: "Developer should not deploy to production"
    - name: "admin_not_auditor"
      conflict: ["platform-admin", "audit-admin"]
      action: "prevent"
      message: "Admin cannot have audit administration role"
    - name: "key_custodian_not_user"
      conflict: ["key-admin", "service-admin"]
      action: "prevent"
      message: "Key custodian cannot use the keys they manage"
```

### Conflict Detection Algorithm

```go
func detectConflicts(userID string, newRole string) []SoDConflict {
    userRoles := getUserRoles(userID)
    var conflicts []SoDConflict

    for _, rule := range sodRules {
        if contains(rule.Conflict, newRole) {
            // Check if user already has the conflicting role
            for _, existingRole := range userRoles {
                if contains(rule.Conflict, existingRole) && existingRole != newRole {
                    conflicts = append(conflicts, SoDConflict{
                        Rule:        rule.Name,
                        Role1:       existingRole,
                        Role2:       newRole,
                        Action:      rule.Action,
                        Message:     rule.Message,
                    })
                }
            }
        }
    }
    return conflicts
}
```

### Enforcement

```go
func assignRole(userID, roleID string) error {
    conflicts := detectConflicts(userID, roleID)

    for _, c := range conflicts {
        if c.Action == "prevent" {
            return fmt.Errorf("SoD violation: %s", c.Message)
        }
        if c.Action == "warn" {
            audit.Log("sod_warning", userID, c.Message)
            notifyAdmin("SoD warning: " + c.Message)
        }
    }

    return doAssignRole(userID, roleID)
}
```

## Auto-Remediation

| Conflict | Remediation |
|---|---|
| Requester = Approver | Auto-reassign approval to manager |
| Dev = Deployer | Require secondary approval for deploy |
| Admin = Auditor | Remove auditor role, notify CISO |
| Key custodian = User | Remove key access, require separate custodian |

```go
func autoRemediate(conflict SoDConflict, userID string) {
    switch conflict.Rule {
    case "requester_not_approver":
        reassignApproval(userID, getManager(userID))
    case "admin_not_auditor":
        removeRole(userID, conflict.Role2)
        notifyCISO("SoD auto-remediation: removed " + conflict.Role2)
    }
    audit.Log("sod_remediation", userID, conflict.Rule)
}
```

## Compliance Requirements

| Framework | SoD Requirement |
|---|---|
| SOX | Financial system access controls require SoD |
| SOC 2 | CC6.3 requires segregation of duties |
| ISO 27001 | A.9.4.4 requires separation of duties |
| PCI-DSS | 7.2 requires least privilege + SoD |
| NIS2 | Article 20 requires access control with SoD |

## GGID SoD Implementation

```yaml
sod:
  enabled: true
  enforcement: "prevent"  # or "warn"
  rules:
    - name: "requester_not_approver"
      conflict: ["access_requester", "access_approver"]
      action: "prevent"
    - name: "dev_not_deployer"
      conflict: ["developer", "deployer"]
      action: "warn"
    - name: "admin_not_auditor"
      conflict: ["platform-admin", "audit-admin"]
      action: "prevent"
  auto_remediation: true
  audit: true
  periodic_review: "quarterly"
```

## Best Practices

1. **Prevent, don't just warn** — Critical conflicts should block assignment
2. **Review periodically** — Quarterly SoD audit for accumulated conflicts
3. **Auto-remediate where possible** — Remove conflicting roles automatically
4. **Notify on violation** — Alert security team on any SoD breach
5. **Document exceptions** — Temporary SoD exceptions need approval + expiry
6. **Map to compliance** — Link SoD rules to framework requirements
7. **Test rules regularly** — Verify detection works for all conflict pairs
8. **Include in access certification** — SoD check during recertification
9. **Support temporary overrides** — Emergency access with post-review
10. **Track SoD metrics** — Number of violations, remediations, exceptions
