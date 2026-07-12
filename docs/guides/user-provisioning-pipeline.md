# User Provisioning Pipeline

SCIM API → attribute mapping → validation → provisioning queue → target app sync → error retry → audit, with webhook notifications.

## Pipeline Architecture

```
Source (HR/IdP/SCIM)
    │
    ▼
┌──────────────┐
│ SCIM Ingress │  POST /scim/v2/Users
└──────┬───────┘
       ▼
┌──────────────┐
│ Attribute    │  Map source attrs → GGID canonical schema
│ Mapping      │
└──────┬───────┘
       ▼
┌──────────────┐
│ Validation   │  Email format, uniqueness, required fields
└──────┬───────┘
       ▼
┌──────────────┐
│ Provisioning │  Write to PostgreSQL, assign default roles/groups
│ Queue        │  (async, reliable via NATS)
└──────┬───────┘
       ▼
┌──────────────┐
│ Target App   │  Sync to connected apps (via SCIM/webhook)
│ Sync         │
└──────┬───────┘
       ▼
┌──────────────┐
│ Webhook      │  Notify subscribed apps
│ Notification │
└──────┬───────┘
       ▼
┌──────────────┐
│ Audit Log    │  Record full provisioning trail
└──────────────┘
```

## SCIM Ingress

```bash
# HR system pushes user via SCIM
POST /scim/v2/Users
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "userName": "jane@corp.com",
  "active": true,
  "emails": [{"value": "jane@corp.com", "primary": true}],
  "displayName": "Jane Doe",
  "name": {"givenName": "Jane", "familyName": "Doe"},
  "urn:hr:extension:1.0": {
    "department": "Engineering",
    "employeeId": "EMP-1234",
    "manager": "manager@corp.com",
    "startDate": "2025-01-15"
  }
}
```

## Attribute Mapping

```yaml
mappings:
  # SCIM standard → GGID field
  userName: email
  displayName: display_name
  name.givenName: first_name
  name.familyName: last_name
  emails[primary].value: email
  phoneNumbers[type eq "mobile"].value: phone
  
  # HR extension → GGID field
  urn:hr:extension:1.0.department: department
  urn:hr:extension:1.0.employeeId: employee_id
  urn:hr:extension:1.0.manager: manager_email
  
  # Derived attributes
  groups:
    - if: department == "Engineering"
      then: ["engineering", "developers"]
    - if: department == "Sales"
      then: ["sales", "revenue"]
```

### Transformation Rules

```yaml
transformations:
  - field: email
    rules:
      - lowercase          # Normalize
      - strip_whitespace   # Clean
      - validate_format    # RFC 5322
  
  - field: display_name
    rules:
      - trim               # Remove extra spaces
      - max_length: 100    # Truncate
      - required           # Must be present
  
  - field: department
    rules:
      - default: "General" # If missing
      - enum: ["Engineering", "Sales", "Marketing", "General"]
```

## Validation

```go
type ProvisioningValidator struct{}

func (v *ProvisioningValidator) Validate(user *User) error {
    // Required fields
    if user.Email == "" { return ErrEmailRequired }
    if user.DisplayName == "" { return ErrDisplayNameRequired }
    
    // Format validation
    if !isValidEmail(user.Email) { return ErrInvalidEmail }
    
    // Uniqueness
    if existing, _ := store.FindByEmail(user.Email); existing != nil {
        return ErrDuplicateEmail
    }
    
    // Value constraints
    if len(user.DisplayName) > 100 { return ErrDisplayNameTooLong }
    
    // Department enum
    if !validDepartment(user.Department) { return ErrInvalidDepartment }
    
    return nil
}
```

### Validation Results

| Result | Action |
|--------|--------|
| Valid | Proceed to provisioning |
| Warning (non-blocking) | Proceed + log warning |
| Error (blocking) | Reject + return SCIM error |

## Provisioning Queue

```go
// Publish to NATS for async, reliable provisioning
func QueueProvisioning(user *User) error {
    event := ProvisioningEvent{
        UserID:    user.ID,
        Action:    "create",
        User:      user,
        Timestamp: time.Now(),
    }
    
    _, err := js.Publish("provisioning.user.create",
        event.Marshal(),
        nats.MsgId(event.UserID),   // Dedup
        nats.MaxDeliver(5),         // Retry up to 5x
    )
    return err
}
```

### Queue Processing

```go
sub, _ := js.PullSubscribe("provisioning.user.>", "PROVISIONING_WORKER")

for {
    msgs, _ := sub.Fetch(50, nats.MaxWait(5*time.Second))
    for _, msg := range msgs {
        event := parseProvisioningEvent(msg)
        
        switch event.Action {
        case "create":
            provisionUser(event)
        case "update":
            updateUser(event)
        case "deactivate":
            deactivateUser(event)
        }
        
        msg.Ack()
    }
}

func provisionUser(event ProvisioningEvent) {
    // 1. Create user in PostgreSQL
    user, err := store.Create(event.User)
    if err != nil { log.Error(err); return }
    
    // 2. Assign default roles
    roleSvc.AssignDefaultRoles(user.ID)
    
    // 3. Add to default groups
    groupSvc.AddToDefaultGroups(user.ID)
    
    // 4. Send welcome email
    emailSvc.SendWelcome(user)
    
    // 5. Audit
    audit.Log("user.provisioned", user)
    
    // 6. Trigger target app sync
    queueTargetSync(user)
}
```

## Target App Sync

```go
func syncToTargetApps(user *User) {
    for _, app := range getSubscribedApps(user.TenantID) {
        go func(app App) {
            payload := mapUserToAppSchema(user, app.Mapping)
            
            resp, err := app.SCIMClient.CreateUser(payload)
            if err != nil {
                retrySync(app, user, err)
                return
            }
            
            audit.Log("user.synced_to_app", map[string]interface{}{
                "user_id": user.ID,
                "app_id":  app.ID,
                "result":  "success",
            })
        }(app)
    }
}
```

### Sync Status Tracking

```sql
CREATE TABLE provisioning_sync_status (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    app_id UUID NOT NULL,
    status TEXT NOT NULL,  -- pending, synced, failed
    last_attempt TIMESTAMPTZ,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    UNIQUE(user_id, app_id)
);
```

## Error Retry

```go
func retrySync(app App, user *User, err error) {
    status := getSyncStatus(user.ID, app.ID)
    status.RetryCount++
    
    delay := backoff(status.RetryCount)  // 30s, 2m, 10m, 1h, 6h, 24h
    
    if status.RetryCount >= 8 {
        status.Status = "failed"
        alert.Send("provisioning_sync_failed", app, user, err)
        return
    }
    
    time.AfterFunc(delay, func() {
        syncToTargetApps(user)
    })
}
```

## Webhook Notifications

```bash
# Apps subscribe to provisioning events
POST /api/v1/webhooks/endpoints
{
  "url": "https://app.example.com/provisioning",
  "events": [
    "user.provisioned",
    "user.updated",
    "user.deactivated",
    "user.deprovisioned"
  ]
}
```

### Webhook Payload

```json
{
  "event": "user.provisioned",
  "user_id": "uuid",
  "email": "jane@corp.com",
  "display_name": "Jane Doe",
  "department": "Engineering",
  "groups": ["engineering", "developers"],
  "timestamp": "2025-01-15T10:00:00Z"
}
```

## Audit Trail

Every provisioning step is logged:

```json
[
  {"event": "provisioning.ingested", "source": "scim", "timestamp": "..."},
  {"event": "provisioning.mapped", "mapped_fields": 8, "timestamp": "..."},
  {"event": "provisioning.validated", "result": "valid", "timestamp": "..."},
  {"event": "provisioning.queued", "queue_id": "...", "timestamp": "..."},
  {"event": "provisioning.created", "user_id": "...", "timestamp": "..."},
  {"event": "provisioning.roles_assigned", "roles": ["user", "developer"], "timestamp": "..."},
  {"event": "provisioning.synced", "apps": ["slack", "jira"], "timestamp": "..."},
  {"event": "provisioning.notified", "webhooks_sent": 3, "timestamp": "..."}
]
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Provisioning latency (ingest→sync) | <30s | >2min → backlog |
| Sync failure rate | <1% | >5% → target app down |
| Queue depth | <100 | >500 → scale workers |
| Validation failures | <5% | >10% → source data quality |

## See Also

- [SCIM 2.0 Implementation](scim-2-0-implementation.md)
- [Identity Lifecycle Automation](identity-lifecycle-automation.md)
- [Webhook Delivery Guarantees](webhook-delivery-guarantees.md)
- [Access Request Lifecycle](access-request-lifecycle.md)
