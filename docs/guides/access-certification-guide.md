# Access Certification Guide

This guide covers certification campaigns, reviewer assignment, evidence collection, decision workflow, exception handling, expired access detection, recertification frequency, and GGID's implementation.

## Overview

Access certification (also called access review or recertification) is the process of periodically reviewing and validating user access rights. It ensures that users only retain access that is appropriate for their current role and responsibilities.

## Certification Campaigns

### What is a Campaign?

A campaign is a scheduled review of access rights across a specific scope:

| Campaign Type | Scope | Frequency |
|---|---|---|
| User access review | All users' permissions | Quarterly |
| Privileged access review | Admin/elevated access | Monthly |
| Application access review | App-specific permissions | Semi-annually |
| Contractor access review | Non-employee access | Monthly |
| New hire access review | Users onboarded in last 90 days | Monthly |
| Role certification | Role definitions and memberships | Annually |

### Campaign Structure

```yaml
campaign:
  id: "camp-2026-q3"
  name: "Q3 2026 Access Review"
  type: "user_access"
  frequency: "quarterly"
  start_date: "2026-07-01"
  end_date: "2026-07-31"
  scope:
    tenants: ["tenant-uuid-1"]
    roles: ["admin", "security-admin", "user-admin"]
    exclude_roles: ["viewer"]  # Low-risk roles exempt
  notification:
    email_reviewers: true
    reminder_interval: 7d
    escalate_after: 14d
```

## Reviewer Assignment

### Assignment Models

| Model | Description | Use Case |
|---|---|---|
| Manager review | User's direct manager reviews | Standard employee access |
| Role owner review | Owner of the role reviews members | Role-based access |
| App owner review | Application owner reviews users | Application access |
| Self-review | User reviews own access (with approval) | Low-risk access |
| Peer review | Peer/colleague reviews | Manager access |
| Security team | Security team reviews | Privileged access |

### Assignment Logic

```go
func assignReviewers(campaign *Campaign) []ReviewItem {
    var items []ReviewItem

    for _, user := range campaign.Scope.GetUsers() {
        for _, role := range user.Roles {
            if campaign.Scope.ShouldReview(role) {
                reviewer := determineReviewer(user, role, campaign)
                items = append(items, ReviewItem{
                    UserID:     user.ID,
                    UserName:   user.Name,
                    RoleID:     role.ID,
                    RoleName:   role.Name,
                    ReviewerID: reviewer.ID,
                    AssignedAt: time.Now(),
                    Deadline:   campaign.EndDate,
                })
            }
        }
    }
    return items
}

func determineReviewer(user *User, role *Role, campaign *Campaign) *User {
    switch campaign.ReviewerModel {
    case "manager":
        return user.Manager
    case "role_owner":
        return role.Owner
    case "app_owner":
        return getAppOwner(role.AppID)
    case "security_team":
        return getSecurityAdmin(user.TenantID)
    default:
        return user.Manager  // Default to manager
    }
}
```

## Evidence Collection

### What Evidence to Collect

For each review item, collect context to help the reviewer make an informed decision:

| Evidence | Source | Purpose |
|---|---|---|
| Access history | Audit logs | When/how access was granted |
| Last used date | Usage logs | When access was last exercised |
| Role changes | HR system | Recent role/department changes |
| Policy violations | Security logs | Any policy violations |
| Peer comparison | Org data | What similar roles have |
| Risk score | Risk engine | Current risk assessment |

### Evidence Display

```json
{
  "review_item": {
    "user": "jane.doe@example.com",
    "role": "user-admin",
    "granted_date": "2025-01-15",
    "granted_by": "admin@example.com",
    "granted_reason": "New role assignment",
    "last_used": "2026-06-28",
    "usage_count_30d": 42,
    "last_login": "2026-06-30",
    "department": "Engineering",
    "manager": "john.smith@example.com",
    "risk_score": 0.3,
    "policy_violations": 0,
    "similar_roles": ["user-admin (5 others in Engineering)"]
  }
}
```

## Decision Workflow

### Reviewer Actions

| Decision | Action | Result |
|---|---|---|
| Certify | Access is appropriate | Keep access |
| Revoke | Access no longer needed | Remove access |
| Modify | Change access level | Adjust permissions |
| Defer | Need more information | Escalate to security team |

### Workflow

```
1. Campaign starts → reviewers notified
2. Reviewer opens review items
3. Reviewer examines evidence
4. Reviewer makes decision (certify/revoke/modify/defer)
5. If revoke: access removed immediately
6. If modify: role adjusted
7. If defer: escalated to security team
8. Campaign completes → report generated
```

### Implementation

```go
func (s *CertificationService) SubmitDecision(
    reviewerID string,
    itemID string,
    decision Decision,
    comment string,
) error {
    item := s.GetItem(itemID)
    if item.ReviewerID != reviewerID {
        return ErrNotAssignedReviewer
    }

    // Record decision
    item.Decision = decision
    item.Comment = comment
    item.DecidedAt = time.Now()

    // Execute decision
    switch decision {
    case DecisionCertify:
        // No action, access retained
        audit.Log("access_certified", item.UserID, item.RoleID, reviewerID)
        
    case DecisionRevoke:
        // Remove access
        s.revokeAccess(item.UserID, item.RoleID)
        audit.Log("access_revoked", item.UserID, item.RoleID, reviewerID, comment)
        
    case DecisionModify:
        // Adjust permissions (requires additional input)
        return ErrRequiresModificationDetails
        
    case DecisionDefer:
        // Escalate to security team
        s.escalateToSecurity(item)
        audit.Log("access_deferred", item.UserID, item.RoleID, reviewerID, comment)
    }

    // Update campaign progress
    s.updateCampaignProgress(item.CampaignID)
    
    return nil
}
```

## Exception Handling

### Exception Types

| Exception | Description | Handling |
|---|---|---|
| Reviewer unavailable | Manager left/on leave | Reassign to backup/next-level manager |
| User on extended leave | User not active | Auto-revoke or defer until return |
| Conflicting decisions | Multiple reviewers disagree | Escalate to security team |
| Overdue review | Reviewer hasn't responded | Escalate to manager, then security |
| Emergency access needed | User needs immediate access | Temporary grant with post-approval review |

### Escalation

```yaml
certification:
  escalation:
    first_reminder: 7d   # After 7 days, send reminder
    second_reminder: 14d  # After 14 days, escalate to manager's manager
    final_notice: 21d     # After 21 days, escalate to security team
    auto_revoke: 30d      # After 30 days, auto-revoke unreviewed access
    notify_admin: true
```

## Expired Access Detection

### Detection Rules

```yaml
certification:
  expired_detection:
    enabled: true
    rules:
      - name: "unused_90_days"
        condition: "last_used > 90d"
        action: "flag_for_review"
      - name: "orphaned_role"
        condition: "role.members == 0"
        action: "flag_role_for_deletion"
      - name: "stale_account"
        condition: "last_login > 180d"
        action: "disable_account"
      - name: "overprivileged"
        condition: "permissions > role_baseline"
        action: "flag_for_review"
```

### Automated Detection

```go
func (s *CertificationService) DetectExpiredAccess() []ExpiredAccessItem {
    var items []ExpiredAccessItem

    // Find unused permissions
    unused := s.findUnusedPermissions(90 * 24 * time.Hour)  // 90 days
    for _, perm := range unused {
        items = append(items, ExpiredAccessItem{
            UserID:    perm.UserID,
            RoleID:    perm.RoleID,
            Reason:    "unused_90_days",
            LastUsed:  perm.LastUsed,
            Action:    "flag_for_review",
        })
    }

    // Find stale accounts
    stale := s.findStaleAccounts(180 * 24 * time.Hour)  // 180 days
    for _, user := range stale {
        items = append(items, ExpiredAccessItem{
            UserID:    user.ID,
            Reason:    "stale_account",
            LastLogin: user.LastLogin,
            Action:    "disable_account",
        })
    }

    return items
}
```

## Recertification Frequency

### Risk-Based Frequency

| Access Type | Risk Level | Frequency |
|---|---|---|
| Platform admin | Critical | Monthly |
| Security admin | Critical | Monthly |
| User admin | High | Quarterly |
| App admin | High | Quarterly |
| Developer access | Medium | Semi-annually |
| Standard user | Low | Annually |
| Read-only | Low | Annually |
| Contractor | Variable | Monthly (regardless of role) |

### Configuration

```yaml
certification:
  frequency:
    by_risk_level:
      critical: "monthly"
      high: "quarterly"
      medium: "semiannually"
      low: "annually"
    by_user_type:
      employee: "role_based"
      contractor: "monthly"
      service_account: "quarterly"
    by_access_type:
      privileged: "monthly"
      standard: "quarterly"
      read_only: "annually"
```

## GGID Implementation

### API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/certification/campaigns` | POST | Create campaign |
| `/api/v1/certification/campaigns` | GET | List campaigns |
| `/api/v1/certification/campaigns/{id}` | GET | Get campaign details |
| `/api/v1/certification/campaigns/{id}/items` | GET | Get review items |
| `/api/v1/certification/items/{id}/decision` | POST | Submit decision |
| `/api/v1/certification/expired` | GET | Get expired access items |
| `/api/v1/certification/reports/{id}` | GET | Get campaign report |

### Campaign Creation

```bash
POST /api/v1/certification/campaigns
Authorization: Bearer <admin_token>

{
  "name": "Q3 2026 Access Review",
  "type": "user_access",
  "scope": {
    "roles": ["admin", "user-admin"],
    "exclude_roles": ["viewer"]
  },
  "reviewer_model": "manager",
  "deadline": "2026-07-31",
  "notify_reviewers": true
}
```

### Service Implementation

```go
type CertificationService struct {
    store    CertificationStore
    audit    AuditService
    policy   PolicyService
    config   CertificationConfig
}

func (s *CertificationService) CreateCampaign(req *CreateCampaignRequest) (*Campaign, error) {
    campaign := &Campaign{
        ID:        uuid.New().String(),
        Name:      req.Name,
        Type:      req.Type,
        Status:    "active",
        StartDate: time.Now(),
        EndDate:   req.Deadline,
        Scope:     req.Scope,
    }

    // Assign review items
    items := assignReviewers(campaign)
    campaign.ReviewItems = items

    // Store campaign
    if err := s.store.SaveCampaign(campaign); err != nil {
        return nil, err
    }

    // Notify reviewers
    for _, item := range items {
        s.notifyReviewer(item.ReviewerID, campaign, item)
    }

    // Audit
    audit.Log("campaign_created", campaign.ID, len(items))

    return campaign, nil
}
```

### Configuration

```yaml
certification:
  enabled: true
  default_frequency: "quarterly"
  reviewer_model: "manager"
  escalation:
    first_reminder: 7d
    second_reminder: 14d
    final_notice: 21d
    auto_revoke: 30d
  expired_detection:
    enabled: true
    unused_threshold: 90d
    stale_threshold: 180d
  notify:
    email: true
    slack: true
    admin_on_overdue: true
```

## Best Practices

1. **Automate campaign scheduling** — Don't rely on manual triggers
2. **Use risk-based frequency** — High-risk access reviewed more often
3. **Provide rich evidence** — Help reviewers make informed decisions
4. **Set clear deadlines** — With escalation for overdue reviews
5. **Auto-revoke on no response** — Don't let access persist unreviewed
6. **Track completion rates** — Monitor campaign health
7. **Audit all decisions** — Full trail of who certified/revoked what
8. **Integrate with HR** — Detect role changes that affect access
9. **Generate executive reports** — Show compliance status to leadership
10. **Learn from results** — Use review outcomes to improve access policies