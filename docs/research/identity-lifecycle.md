# Identity Lifecycle Management for IAM Systems

> Research document covering lifecycle state machines, SCIM 2.0 provisioning
> implementation patterns, inbound/outbound federation, dormant account detection,
> event-driven lifecycle architecture, and account reconciliation.
>
> **Related docs:**
> - `identity-lifecycle-automation.md` — JIT provisioning, deprovisioning cascade, role
>   mining, and IGA integration (Joiner-Mover-Leaver automation). This document does not
>   duplicate those topics.
> - `scim-conformance-testing.md` — SCIM 2.0 conformance test suites and remediation
>   roadmap. This document focuses on implementation patterns, not test coverage.

---

## Table of Contents

1. [Identity Lifecycle States](#1-identity-lifecycle-states)
2. [SCIM 2.0 Provisioning Implementation](#2-scim-20-provisioning-implementation)
3. [SCIM 2.0 Bulk Operations](#3-scim-20-bulk-operations)
4. [Inbound Federation Provisioning](#4-inbound-federation-provisioning)
5. [Outbound Provisioning](#5-outbound-provisioning)
6. [Dormant Account Detection](#6-dormant-account-detection)
7. [Lifecycle Event-Driven Architecture](#7-lifecycle-event-driven-architecture)
8. [Account Reconciliation](#8-account-reconciliation)
9. [GGID Lifecycle Gap Analysis](#9-ggid-lifecycle-gap-analysis)
10. [Gap Analysis and Recommendations](#10-gap-analysis-and-recommendations)

---

## 1. Identity Lifecycle States

An IAM system must model the complete lifecycle of an identity from initial creation
through final deletion. Each state constrains what operations are permitted and triggers
side-effects (session revocation, notifications, audit events).

### 1.1 State Machine

```
                         ┌─────────────────┐
                         │ pre-provisioned │  ← Created by SCIM/HR sync, no credential
                         └────────┬────────┘
                                  │ credential set / first login
                                  ▼
                         ┌─────────────────┐
              ┌──────────│     active      │──────────┐
              │          └────────┬────────┘          │
              │                   │                   │
     security policy        admin action         admin action
     (dormant, risk)        (suspension)         (lockout)
              │                   │                   │
              ▼                   ▼                   ▼
    ┌──────────────────┐ ┌───────────────┐  ┌───────────────┐
    │   deprecated     │ │  suspended    │  │    locked     │
    │ (grace period)   │ │ (admin hold)  │  │ (auto-lockout)│
    └────────┬─────────┘ └───────┬───────┘  └───────┬───────┘
             │                   │                   │
             │ reactivate        │ reactivate        │ unlock
             │ or auto-suspend   │                   │
             ▼                   │                   │
    ┌──────────────────┐         │                   │
    │   suspended      │         │                   │
    └────────┬─────────┘         │                   │
             │                   │                   │
             └───────────────────┴───────────────────┘
                                 │
                          admin / policy
                                 │
                                 ▼
                         ┌─────────────────┐
                         │    deleted      │  ← Soft delete, retained for audit
                         │ (retention: Nd) │
                         └─────────────────┘
                                 │
                          retention expired
                                 │
                                 ▼
                         ┌─────────────────┐
                         │  hard deleted   │  ← Irreversible, GDPR right-to-erasure
                         └─────────────────┘
```

### 1.2 State Definitions

| State | Description | Can Authenticate? | Trigger |
|-------|-------------|-------------------|---------|
| `pre-provisioned` | Account created by SCIM/HR sync but no credential set | No | SCIM POST, HR sync |
| `active` | Normal operating state | Yes | Credential set, admin activation |
| `suspended` | Admin-initiated hold (e.g. leave of absence) | No | Admin action |
| `locked` | Security lockout (brute force, MFA failures) | No | Auto policy (max failed attempts) |
| `deprecated` | Grace period before suspension (dormant account) | Yes (with warning) | Dormant threshold reached |
| `deleted` | Soft-deleted, retained for audit/compliance | No | Admin action, HR termination |

### 1.3 GGID Current State Model

GGID's `domain.UserStatus` currently defines four states:

```go
// From services/identity/internal/domain/user.go
const (
    UserStatusActive   UserStatus = "active"
    UserStatusLocked   UserStatus = "locked"
    UserStatusDisabled UserStatus = "disabled"
    UserStatusDeleted  UserStatus = "deleted"
)
```

**Missing states:** `pre-provisioned` (no distinct state for SCIM-created accounts
without credentials) and `suspended` (no admin-hold state separate from disabled).
The `deprecated`/grace-period state for dormant accounts also does not exist.

---

## 2. SCIM 2.0 Provisioning Implementation

SCIM (System for Cross-domain Identity Management) 2.0, defined in RFC 7643/7644, is
the standard protocol for automated user provisioning between identity providers and
service providers.

### 2.1 Endpoint Overview

| Method | Endpoint | Operation |
|--------|----------|-----------|
| POST | `/scim/v2/Users` | Create user |
| GET | `/scim/v2/Users/{id}` | Read user |
| GET | `/scim/v2/Users?filter=...` | Search/list with pagination |
| PUT | `/scim/v2/Users/{id}` | Full replacement |
| PATCH | `/scim/v2/Users/{id}` | Partial update (add/replace/remove) |
| DELETE | `/scim/v2/Users/{id}` | Deprovision |

### 2.2 POST — Create User with Validation

```go
func (h *Handler) createUser(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    bodyBytes, _ := io.ReadAll(r.Body)
    var rawBody map[string]json.RawMessage
    if err := json.Unmarshal(bodyBytes, &rawBody); err != nil {
        writeSCIMErrorWithType(w, 400, ScimTypeInvalidSyntax, "invalid request body")
        return
    }

    var scimUser SCIMUser
    if err := json.Unmarshal(bodyBytes, &scimUser); err != nil {
        writeSCIMErrorWithType(w, 400, ScimTypeInvalidSyntax, "invalid request body")
        return
    }

    if scimUser.UserName == "" {
        writeSCIMErrorWithType(w, 400, ScimTypeInvalidSyntax, "userName is required")
        return
    }

    email := ""
    if len(scimUser.Emails) > 0 {
        email = scimUser.Emails[0].Value
    }

    user, err := h.svc.CreateUser(ctx, &domain.CreateUserInput{
        Username:    scimUser.UserName,
        Email:       email,
        Password:    generateTempPassword(),
        DisplayName: scimUser.DisplayName,
        ExternalID:  scimUser.ExternalID,
    })
    if err != nil {
        writeSCIMErrorWithType(w, 409, ScimTypeUniqueness, "user already exists")
        return
    }

    resp := toSCIMUser(user)
    w.Header().Set("Location", "/scim/v2/Users/"+user.ID.String())
    writeSCIMJSON(w, 201, resp)
}
```

### 2.3 GET — List with Pagination, Filtering, and Sorting

SCIM list responses must follow the `ListResponse` schema with `totalResults`,
`itemsPerPage`, `startIndex`, and `Resources` array.

```go
func (h *Handler) listUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    startIndex, _ := strconv.Atoi(r.URL.Query().Get("startIndex"))
    if startIndex <= 0 {
        startIndex = 1
    }
    count, _ := strconv.Atoi(r.URL.Query().Get("count"))
    if count <= 0 || count > 100 {
        count = 20
    }

    sortBy := mapSCIMSortAttr(r.URL.Query().Get("sortBy"))
    sortDesc := strings.EqualFold(r.URL.Query().Get("sortOrder"), "descending")

    // Parse SCIM filter — GGID supports externalId eq "value"
    filterParam := r.URL.Query().Get("filter")
    externalID := ""
    if filterParam != "" {
        externalID = parseExternalIdFilter(filterParam)
    }

    result, err := h.svc.ListUsers(ctx, &domain.ListUsersFilter{
        PageSize:   count,
        Offset:     startIndex - 1,
        SortBy:     sortBy,
        SortDesc:   sortDesc,
        ExternalID: externalID,
    })
    if err != nil {
        writeSCIMError(w, 500, err.Error())
        return
    }

    resources := make([]SCIMUser, 0, len(result.Users))
    for _, u := range result.Users {
        resources = append(resources, toSCIMUser(u))
    }

    writeSCIMJSON(w, 200, ListResponse{
        Schemas:      []string{scimListSchema},
        TotalResults: result.Total,
        ItemsPerPage: count,
        StartIndex:   startIndex,
        Resources:    resources,
    })
}

func mapSCIMSortAttr(scimAttr string) string {
    switch strings.ToLower(scimAttr) {
    case "username":          return "username"
    case "displayname":       return "display_name"
    case "meta.created":      return "created_at"
    case "meta.lastmodified": return "updated_at"
    case "emails.value":      return "email"
    default:                  return ""
    }
}
```

### 2.4 PATCH — Partial Update with Path Expressions

SCIM PATCH supports three operations: `add`, `replace`, `remove`. Each operation
targets a path that may include value filters on multi-valued attributes:
`emails[type eq "work"].value`.

```go
// PatchOperation represents a single SCIM PATCH operation.
type PatchOperation struct {
    Op    string          `json:"op"`    // "add", "replace", "remove"
    Path  string          `json:"path"`  // "displayName", "emails[type eq \"work\"].value"
    Value json.RawMessage `json:"value"` // value for add/replace
}

// ApplyPatch applies SCIM PATCH operations to an attribute map.
func ApplyPatch(attrs map[string]any, ops []PatchOperation) (map[string]any, error) {
    result := make(map[string]any)
    for k, v := range attrs {
        result[k] = v
    }
    for i, op := range ops {
        switch strings.ToLower(op.Op) {
        case "add":
            if err := applyAdd(result, op.Path, op.Value); err != nil {
                return nil, fmt.Errorf("op[%d]: %w", i, err)
            }
        case "replace":
            if err := applyReplace(result, op.Path, op.Value); err != nil {
                return nil, fmt.Errorf("op[%d]: %w", i, err)
            }
        case "remove":
            if err := applyRemove(result, op.Path); err != nil {
                return nil, fmt.Errorf("op[%d]: %w", i, err)
            }
        default:
            return nil, fmt.Errorf("op[%d]: unsupported %q", i, op.Op)
        }
    }
    return result, nil
}
```

**Path parsing example:**

```
Path: emails[type eq "work"].value
  attrName = "emails"
  filter   = `type eq "work"`
  subPath  = "value"

Path: name.familyName
  attrName = "name"
  subPath  = "familyName"
  filter   = ""
```

### 2.5 SCIM Filter Engine

GGID implements a full SCIM filter parser supporting all RFC 7644 operators (`eq`,
`ne`, `co`, `sw`, `ew`, `pr`, `gt`, `ge`, `lt`, `le`) with logical `and`/`or`/`not`
and parenthetical grouping. The AST evaluates against attribute maps:

```go
type FilterExpr interface {
    Evaluate(attrs map[string]any) bool
    String() string
}

// AttrExpression: userName eq "john"
// AndExpr:        (active eq true) and (emails[type eq "work"])
// NotExpr:        not (userName sw "temp_")
```

**Security note:** The filter engine operates on Go maps after data is retrieved
from the database. For server-side filtering (pushing predicates to SQL), use
parameterized queries — never concatenate filter values into SQL strings.

---

## 3. SCIM 2.0 Bulk Operations

RFC 7644 Section 3.7 defines a bulk endpoint (`POST /scim/v2/Bulk`) that allows
multiple operations in a single HTTP request. This is critical for initial
provisioning when an IdP pushes thousands of users.

### 3.1 Request Format

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
  "failOnErrors": 5,
  "Operations": [
    {
      "method": "POST",
      "path": "/Users",
      "bulkId": "tid-001",
      "data": { "userName": "alice", "emails": [{"value": "alice@example.com"}] }
    },
    {
      "method": "PATCH",
      "path": "/Users/550e8400-e29b-41d4-a716-446655440000",
      "data": { "Operations": [{"op": "replace", "path": "active", "value": false}] }
    }
  ]
}
```

### 3.2 Failure Handling

The `failOnErrors` field controls whether the server stops after N errors:

- `failOnErrors` absent or <= 0: process all operations, return per-op results
- `failOnErrors` = 1: all-or-nothing (first error aborts)
- `failOnErrors` = N: process until N errors accumulate, then stop

### 3.3 Bulk Handler Implementation

```go
func (h *Handler) HandleBulk(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    var req BulkRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeSCIMErrorWithType(w, 400, ScimTypeInvalidSyntax, "invalid bulk body")
        return
    }
    if len(req.Operations) > maxBulkOperations {
        writeSCIMErrorWithType(w, 413, ScimTypeTooMany,
            fmt.Sprintf("exceeded maximum of %d operations", maxBulkOperations))
        return
    }

    failOnErrors := -1
    if req.FailOnErrors != nil {
        failOnErrors = *req.FailOnErrors
    }

    responses := make([]BulkOperationResponse, 0, len(req.Operations))
    errorCount := 0

    for _, op := range req.Operations {
        resp, err := h.executeBulkOp(ctx, op)
        if err != nil {
            errorCount++
            resp.Response, _ = json.Marshal(ErrorResponse{
                Schemas: []string{scimErrSchema},
                Detail:  err.Error(),
                Status:  resp.Status,
            })
        }
        responses = append(responses, resp)

        if failOnErrors > 0 && errorCount >= failOnErrors {
            break // stop processing remaining operations
        }
    }

    writeSCIMJSON(w, 200, BulkResponse{
        Schemas:    []string{bulkResponseSchema},
        Operations: responses,
    })
}

func (h *Handler) executeBulkOp(ctx context.Context, op BulkOperationRequest) (BulkOperationResponse, error) {
    resp := BulkOperationResponse{Method: op.Method, BulkID: op.BulkID}
    switch op.Method {
    case "POST":   return h.bulkCreateUser(ctx, op)
    case "PUT":    return h.bulkReplaceUser(ctx, op)
    case "PATCH":  return h.bulkPatchUser(ctx, op)
    case "DELETE": return h.bulkDeleteUser(ctx, op)
    default:
        resp.Status = "400"
        return resp, fmt.Errorf("unsupported method %q", op.Method)
    }
}
```

### 3.4 BulkID Cross-Referencing

RFC 7644 allows operations within a bulk request to reference results of earlier
operations using `bulkId`. For example, a POST creating a user with `bulkId: "u1"`
can be referenced by a later operation as `path: "/Users/bulkId:u1"`. This requires
the handler to maintain a `bulkId → created resource ID` map during processing.

---

## 4. Inbound Federation Provisioning

Inbound federation provisioning creates local user accounts automatically when a
user authenticates via an external identity provider (SAML, OIDC, LDAP) for the
first time. See `identity-lifecycle-automation.md` Section 2 for the JIT concept
and attribute mapping table — this section focuses on implementation patterns.

### 4.1 JIT from SAML Assertion

When a user completes SAML SSO, the IdP returns an assertion containing attributes.
The service provider extracts these attributes to create or update the local user.

```go
// SAMLAttributeMap maps SAML assertion attribute names to local user fields.
type SAMLAttributeMap struct {
    NameID        string // maps to Username
    Email         string // maps to Email (usually "http://schemas.../emailaddress")
    DisplayName   string // maps to DisplayName (usually "http://schemas.../name")
    GivenName     string
    Surname       string
    Groups        string // maps to role assignments
}

// JITProvisionFromSAML creates or updates a user from SAML assertion attributes.
func (s *IdentityService) JITProvisionFromSAML(
    ctx context.Context,
    tenantID uuid.UUID,
    idpEntityID string,
    nameID string,
    attributes map[string][]string,
) (*domain.User, error) {
    // Check if user already exists by external identity
    existing, _ := s.repo.FindExternalIdentity(ctx, tenantID, "saml:"+idpEntityID, nameID)
    if existing != nil {
        // User exists — update attributes if changed (Mover phase)
        user, err := s.repo.GetUserByID(ctx, tenantID, existing.UserID)
        if err != nil {
            return nil, err
        }
        s.updateUserFromSAMLAttrs(ctx, user, attributes)
        return user, nil
    }

    // First login — create new user (Joiner phase)
    email := getFirstAttr(attributes, "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress")
    displayName := getFirstAttr(attributes, "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name")
    if displayName == "" {
        displayName = nameID
    }

    user := &domain.User{
        ID:            uuid.New(),
        TenantID:      tenantID,
        Username:      nameID,
        Email:         email,
        Status:        domain.UserStatusActive,
        EmailVerified: true, // IdP-asserted emails are considered verified
        DisplayName:   displayName,
        Locale:        "en",
    }

    if err := s.repo.CreateUser(ctx, user); err != nil {
        return nil, fmt.Errorf("create SAML user: %w", err)
    }

    // Link external identity
    ei := &domain.ExternalIdentity{
        ID:         uuid.New(),
        UserID:     user.ID,
        TenantID:   tenantID,
        Provider:   "saml:" + idpEntityID,
        ExternalID: nameID,
        Metadata:   flattenAttrs(attributes),
    }
    s.repo.LinkExternalIdentity(ctx, ei)

    return user, nil
}

func getFirstAttr(attrs map[string][]string, key string) string {
    if vals, ok := attrs[key]; ok && len(vals) > 0 {
        return vals[0]
    }
    return ""
}
```

### 4.2 JIT from OIDC Token

OIDC JIT is simpler — the ID token is a JWT with standard claims:

```go
type OIDCIDToken struct {
    Sub           string `json:"sub"`
    Email         string `json:"email"`
    EmailVerified bool   `json:"email_verified"`
    Name          string `json:"name"`
    GivenName     string `json:"given_name"`
    FamilyName    string `json:"family_name"`
    Picture       string `json:"picture"`
    Locale        string `json:"locale"`
}

func (s *IdentityService) JITProvisionFromOIDC(
    ctx context.Context,
    tenantID uuid.UUID,
    issuer string,
    claims OIDCIDToken,
) (*domain.User, error) {
    existing, _ := s.repo.FindExternalIdentity(ctx, tenantID, "oidc:"+issuer, claims.Sub)
    if existing != nil {
        return s.repo.GetUserByID(ctx, tenantID, existing.UserID)
    }

    user := &domain.User{
        ID:            uuid.New(),
        TenantID:      tenantID,
        Username:      claims.Sub,
        Email:         claims.Email,
        Status:        domain.UserStatusActive,
        EmailVerified: claims.EmailVerified,
        DisplayName:   claims.Name,
        Locale:        claims.Locale,
        AvatarURL:     claims.Picture,
    }
    s.repo.CreateUser(ctx, user)

    s.repo.LinkExternalIdentity(ctx, &domain.ExternalIdentity{
        ID:         uuid.New(),
        UserID:     user.ID,
        TenantID:   tenantID,
        Provider:   "oidc:" + issuer,
        ExternalID: claims.Sub,
    })
    return user, nil
}
```

### 4.3 Security Risks of Auto-Provisioning

| Risk | Mitigation |
|------|------------|
| Account takeover via IdP spoofing | Validate IdP signature, restrict trusted IdPs per tenant |
| Privilege escalation via attribute injection | Never accept role/group claims without verification; use explicit allow-list |
| Account enumeration via JIT | Return same response whether user exists or was just created |
| Orphaned accounts when IdP user deleted | Reconciliation against IdP directory (see Section 8) |
| Email collision with existing local user | Match by external identity first; if email matches, link rather than create |

GGID currently supports LDAP JIT provisioning via `ProvisionFromLDAP` in the identity
service. SAML/OIDC JIT is not yet implemented — see Section 9.

---

## 5. Outbound Provisioning

Outbound provisioning pushes user data from the IAM system to downstream applications
via SCIM client connections. This enables centralized user management — create a user
in GGID, and it automatically appears in Slack, Salesforce, and custom apps.

### 5.1 SCIM Client Architecture

```go
// SCIMClient pushes user updates to a downstream SCIM 2.0 endpoint.
type SCIMClient struct {
    endpoint   string       // https://api.slack.com/scim/v2
    authToken  string       // bearer token
    httpClient *http.Client
    retryPolicy RetryPolicy
}

type RetryPolicy struct {
    MaxRetries  int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
}

// CreateUser pushes a new user to the downstream system.
func (c *SCIMClient) CreateUser(ctx context.Context, user *domain.User) error {
    scimUser := toSCIMUser(user)
    body, _ := json.Marshal(scimUser)

    return c.doWithRetry(ctx, func() (*http.Response, error) {
        req, _ := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/Users", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/scim+json")
        req.Header.Set("Authorization", "Bearer "+c.authToken)
        return c.httpClient.Do(req)
    }, 201) // expect 201 Created
}

// UpdateUser pushes a PUT replacement to the downstream system.
func (c *SCIMClient) UpdateUser(ctx context.Context, user *domain.User) error {
    scimUser := toSCIMUser(user)
    body, _ := json.Marshal(scimUser)
    url := fmt.Sprintf("%s/Users/%s", c.endpoint, user.ID)

    return c.doWithRetry(ctx, func() (*http.Response, error) {
        req, _ := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/scim+json")
        req.Header.Set("Authorization", "Bearer "+c.authToken)
        return c.httpClient.Do(req)
    }, 200)
}

// DeactivateUser disables a user at the downstream system via PATCH.
func (c *SCIMClient) DeactivateUser(ctx context.Context, userID string) error {
    patch := map[string]any{
        "schemas":    []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
        "Operations": []map[string]any{
            {"op": "replace", "path": "active", "value": false},
        },
    }
    body, _ := json.Marshal(patch)
    url := fmt.Sprintf("%s/Users/%s", c.endpoint, userID)

    return c.doWithRetry(ctx, func() (*http.Response, error) {
        req, _ := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/scim+json")
        req.Header.Set("Authorization", "Bearer "+c.authToken)
        return c.httpClient.Do(req)
    }, 200)
}

func (c *SCIMClient) doWithRetry(ctx context.Context, fn func() (*http.Response, error), expectedStatus int) error {
    var lastErr error
    for attempt := 0; attempt <= c.retryPolicy.MaxRetries; attempt++ {
        resp, err := fn()
        if err != nil {
            lastErr = err
        } else if resp.StatusCode == expectedStatus {
            resp.Body.Close()
            return nil
        } else if resp.StatusCode >= 500 {
            lastErr = fmt.Errorf("downstream returned %d", resp.StatusCode)
            resp.Body.Close()
        } else {
            // 4xx — don't retry, it won't help
            body, _ := io.ReadAll(resp.Body)
            resp.Body.Close()
            return fmt.Errorf("downstream SCIM error %d: %s", resp.StatusCode, string(body))
        }

        // Exponential backoff with jitter
        delay := c.retryPolicy.BaseDelay * time.Duration(1<<attempt)
        if delay > c.retryPolicy.MaxDelay {
            delay = c.retryPolicy.MaxDelay
        }
        delay += time.Duration(rand.Intn(int(delay / 2)))

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(delay):
        }
    }
    return fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### 5.2 Provisioning Webhooks

For downstream apps that don't support SCIM, GGID can deliver lifecycle events via
webhooks. The webhook payload follows the CloudEvents format:

```json
{
  "specversion": "1.0",
  "type": "ggid.user.created",
  "source": "/tenants/00000000-0000-0000-0000-000000000001",
  "id": "event-uuid",
  "time": "2025-01-15T10:30:00Z",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "alice",
    "email": "alice@example.com",
    "display_name": "Alice Chen"
  }
}
```

### 5.3 Reconciliation Loop

Outbound provisioning is eventually consistent. A periodic reconciliation job
compares the IAM user list against the downstream system's user list and remediates
drift:

```go
func (m *Reconciler) reconcileDownstream(ctx context.Context, client *SCIMClient, tenantID uuid.UUID) error {
    // Get all active users from IAM
    iamUsers, err := m.identitySvc.ListUsers(ctx, &domain.ListUsersFilter{
        TenantID: tenantID,
        Status:   &domain.UserStatusActive,
        PageSize: 1000,
    })
    if err != nil {
        return err
    }

    // Get all users from downstream SCIM endpoint
    downstreamUsers, err := client.ListAllUsers(ctx)
    if err != nil {
        return err
    }

    // Build lookup maps
    downstreamMap := make(map[string]bool)
    for _, u := range downstreamUsers {
        downstreamMap[u.ID] = true
    }

    iamMap := make(map[string]bool)
    for _, u := range iamUsers.Users {
        iamMap[u.ID.String()] = true
    }

    // Push missing users to downstream
    for _, u := range iamUsers.Users {
        if !downstreamMap[u.ID.String()] {
            if err := client.CreateUser(ctx, u); err != nil {
                log.Printf("reconcile: failed to push user %s: %v", u.ID, err)
            }
        }
    }

    // Deactivate users in downstream that no longer exist in IAM
    for _, du := range downstreamUsers {
        if !iamMap[du.ID] {
            if err := client.DeactivateUser(ctx, du.ID); err != nil {
                log.Printf("reconcile: failed to deactivate %s: %v", du.ID, err)
            }
        }
    }

    return nil
}
```

---

## 6. Dormant Account Detection

Dormant accounts — accounts with no login activity for an extended period — are a
significant security risk. Attackers target dormant accounts because they are less
likely to be monitored, and password resets or MFA device changes may go unnoticed.

### 6.1 Detection Policy

Each tenant should be able to configure dormancy thresholds:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `dormant_threshold_days` | 90 | Days without login before account is flagged dormant |
| `grace_period_days` | 14 | Days in `deprecated` state before auto-suspension |
| `notification_enabled` | true | Send email before/after suspension |
| `exclude_service_accounts` | true | Skip accounts marked as service accounts |
| `exclude_admin_accounts` | false | Skip admin accounts (usually not) |

### 6.2 Dormant Account Scanner

```go
// DormantScanner identifies and processes dormant accounts.
type DormantScanner struct {
    identitySvc  IdentityServiceInterface
    auditPub     *audit.Publisher
    notifier     Notifier
    config       DormantConfig
}

type DormantConfig struct {
    DormantThresholdDays int
    GracePeriodDays      int
    ExcludeServiceAccts  bool
}

type DormantReport struct {
    ScannedAt         time.Time
    TotalScanned      int
    FlaggedDormant    int
    Suspended         int
    Notified          int
    Users             []DormantUser
}

type DormantUser struct {
    UserID      uuid.UUID
    Username    string
    Email       string
    LastLoginAt *time.Time
    DaysInactive int
    Action      string // "flagged", "notified", "suspended"
}

// Scan identifies dormant accounts and applies the configured policy.
func (s *DormantScanner) Scan(ctx context.Context, tenantID uuid.UUID) (*DormantReport, error) {
    threshold := time.Now().AddDate(0, 0, -s.config.DormantThresholdDays)

    // Query users whose last login is before the threshold
    result, err := s.identitySvc.ListUsers(ctx, &domain.ListUsersFilter{
        TenantID:       tenantID,
        Status:         &domain.UserStatusActive,
        LastLoginAfter: &time.Time{}, // nil start — get all
        PageSize:       1000,
    })
    if err != nil {
        return nil, err
    }

    report := &DormantReport{ScannedAt: time.Now()}
    for _, user := range result.Users {
        if s.config.ExcludeServiceAccts && isServiceAccount(user) {
            continue
        }

        // Skip users who never logged in but were created recently (< threshold)
        if user.LastLoginAt == nil && time.Since(user.CreatedAt) < time.Duration(s.config.DormantThresholdDays)*24*time.Hour {
            continue
        }

        var daysInactive int
        if user.LastLoginAt != nil {
            daysInactive = int(time.Since(*user.LastLoginAt).Hours() / 24)
        } else {
            daysInactive = int(time.Since(user.CreatedAt).Hours() / 24)
        }

        if daysInactive < s.config.DormantThresholdDays {
            continue
        }

        report.TotalScanned++
        report.FlaggedDormant++

        // Check if already in grace period
        if isUserInGracePeriod(user) {
            graceElapsed := daysSince(user.GracePeriodStart)
            if graceElapsed >= s.config.GracePeriodDays {
                // Suspend the account
                s.identitySvc.DisableUser(ctx, user.ID)
                report.Suspended++
                report.Users = append(report.Users, DormantUser{
                    UserID: user.ID, Username: user.Username,
                    Email: user.Email, DaysInactive: daysInactive,
                    Action: "suspended",
                })
                s.publishLifecycleEvent(ctx, tenantID, "user.suspended", user, "dormant_policy")
            }
            continue
        }

        // Enter grace period and send notification
        s.identitySvc.StartGracePeriod(ctx, user.ID)
        s.notifier.NotifyDormant(ctx, user, daysInactive, s.config.GracePeriodDays)
        report.Notified++
        report.Users = append(report.Users, DormantUser{
            UserID: user.ID, Username: user.Username,
            Email: user.Email, DaysInactive: daysInactive,
            Action: "notified",
        })
    }

    return report, nil
}
```

### 6.3 Re-activation Flow

When a dormant user attempts to authenticate:

1. Authentication fails with a "dormant account" error
2. User is directed to a re-activation flow (email verification or admin approval)
3. On verification, account status returns to `active`, grace period cleared
4. `LastLoginAt` updated, dormancy counter reset

```go
func (s *IdentityService) ReactivateDormantUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
    user, err := s.repo.GetUserByID(ctx, tc.TenantID, userID)
    if err != nil {
        return nil, err
    }
    if user.Status != domain.UserStatusDisabled {
        return nil, fmt.Errorf("user is not dormant")
    }

    // Reactivate and clear grace period
    user.Status = domain.UserStatusActive
    user.GracePeriodStart = nil
    user.UpdatedAt = time.Now()
    return s.repo.UpdateUserDirect(ctx, user)
}
```

---

## 7. Lifecycle Event-Driven Architecture

Lifecycle state transitions should emit events to a message bus (NATS JetStream in
GGID) so downstream services can react asynchronously. This decouples the identity
service from session management, notification, and audit systems.

### 7.1 Event Catalog

| Event | Trigger | Consumers |
|-------|---------|-----------|
| `user.created` | SCIM POST, registration, JIT | Audit, Notification (welcome email) |
| `user.activated` | Admin activation, unlock | Audit, Notification |
| `user.locked` | Max failed attempts | Audit, Alerting (possible attack) |
| `user.suspended` | Admin action, dormant policy | Session revoker, Subscription canceller |
| `user.deleted` | Admin action, HR termination | Session revoker, Data archiver |
| `user.role_changed` | Role assignment/removal | Audit, Access review |

### 7.2 Lifecycle Event Publisher

```go
// LifecyclePublisher emits user lifecycle events to NATS JetStream.
type LifecyclePublisher struct {
    publisher *audit.Publisher
}

type LifecycleEvent struct {
    EventType   string         `json:"event_type"`
    TenantID    uuid.UUID      `json:"tenant_id"`
    UserID      uuid.UUID      `json:"user_id"`
    Username    string         `json:"username"`
    OldStatus   string         `json:"old_status,omitempty"`
    NewStatus   string         `json:"new_status,omitempty"`
    Reason      string         `json:"reason"` // "admin_action", "dormant_policy", "security_policy"
    ActorID     uuid.UUID      `json:"actor_id"`
    ActorType   string         `json:"actor_type"` // "user", "system", "api_key"
    Timestamp   time.Time      `json:"timestamp"`
    Metadata    map[string]any `json:"metadata,omitempty"`
}

// PublishTransition emits a lifecycle state transition event.
func (p *LifecyclePublisher) PublishTransition(
    ctx context.Context,
    tenantID uuid.UUID,
    user *domain.User,
    oldStatus, newStatus domain.UserStatus,
    reason string,
    actorID uuid.UUID,
) error {
    event := audit.Event{
        ID:           uuid.New(),
        TenantID:     tenantID,
        ActorType:    "system",
        ActorID:      actorID,
        Action:       fmt.Sprintf("user.%s", newStatus),
        ResourceType: "user",
        ResourceID:   user.ID,
        ResourceName: user.Username,
        Result:       "success",
        CreatedAt:    time.Now().UTC(),
        Metadata: map[string]any{
            "old_status": string(oldStatus),
            "new_status": string(newStatus),
            "reason":     reason,
            "email":      user.Email,
        },
    }
    return p.publisher.Publish(ctx, event)
}
```

### 7.3 Downstream Consumer Example

A session revoker subscribes to `user.suspended` and `user.deleted` events:

```go
func (r *SessionRevoker) Subscribe(ctx context.Context, js jetstream.JetStream) error {
    cons, err := js.CreateOrUpdateConsumer(ctx, "AUDIT", jetstream.ConsumerConfig{
        FilterSubject:    "audit.events",
        Durable:          "session-revoker",
        AckPolicy:        jetstream.AckExplicitPolicy,
        MaxDeliveries:    3,
    })
    if err != nil {
        return err
    }

    _, err = js.Consume(ctx, func(msg jetstream.Msg) {
        var event audit.Event
        if err := json.Unmarshal(msg.Data(), &event); err != nil {
            msg.Nak()
            return
        }

        // Revoke sessions on suspend/delete
        if event.Action == "user.suspended" || event.Action == "user.deleted" {
            if err := r.revokeUserSessions(ctx, event.TenantID, event.ResourceID); err != nil {
                log.Printf("session revoke failed for %s: %v", event.ResourceID, err)
                msg.Nak()
                return
            }
        }
        msg.Ack()
    }, cons, nil)
    return err
}
```

---

## 8. Account Reconciliation

Reconciliation is the periodic sync between a source-of-truth system (HR, Active
Directory) and the IAM database. It detects drift — users that exist in one system
but not the other — and applies remediation policies.

### 8.1 Reconciliation Engine

```go
// ReconciliationEngine compares source-of-truth data against IAM state.
type ReconciliationEngine struct {
    identitySvc IdentityServiceInterface
    hrClient    HRSource
    policy      ReconciliationPolicy
}

type ReconciliationPolicy struct {
    OnUserInHRNotInIAM  string // "create", "report", "disable"
    OnUserInIAMNotInHR  string // "disable", "report", "delete"
    OnAttributeMismatch string // "update", "report"
    DryRun              bool
}

type ReconResult struct {
    Created   int
    Updated   int
    Disabled  int
    Deleted   int
    Drift     []DriftItem
}

type DriftItem struct {
    Type       string // "orphan_in_iam", "missing_from_iam", "attribute_mismatch"
    UserID     string
    HRData     map[string]any
    IAMData    map[string]any
    Mismatch   map[string]string // field → "hr_value → iam_value"
}

// Reconcile compares HR source against IAM and applies policy.
func (e *ReconciliationEngine) Reconcile(ctx context.Context, tenantID uuid.UUID) (*ReconResult, error) {
    // Fetch all employees from HR system
    hrEmployees, err := e.hrClient.GetAllEmployees(ctx)
    if err != nil {
        return nil, fmt.Errorf("fetch HR data: %w", err)
    }

    // Fetch all users from IAM
    iamResult, err := e.identitySvc.ListUsers(ctx, &domain.ListUsersFilter{
        TenantID: tenantID,
        PageSize: 10000,
    })
    if err != nil {
        return nil, fmt.Errorf("fetch IAM users: %w", err)
    }

    result := &ReconResult{}

    // Build lookup by employee ID / external ID
    hrByExtID := make(map[string]*HREmployee)
    for _, emp := range hrEmployees {
        hrByExtID[emp.EmployeeID] = emp
    }

    iamByExtID := make(map[string]*domain.User)
    for _, u := range iamResult.Users {
        if u.ExternalID != "" {
            iamByExtID[u.ExternalID] = u
        }
    }

    // Phase 1: Users in HR but not in IAM → create or report
    for extID, emp := range hrByExtID {
        if _, exists := iamByExtID[extID]; !exists {
            if emp.Status == "terminated" {
                continue // skip terminated employees who never had accounts
            }
            if e.policy.DryRun || e.policy.OnUserInHRNotInIAM == "report" {
                result.Drift = append(result.Drift, DriftItem{
                    Type:   "missing_from_iam",
                    UserID: extID,
                    HRData: emp.ToMap(),
                })
            } else if e.policy.OnUserInHRNotInIAM == "create" {
                e.createUserFromHR(ctx, tenantID, emp)
                result.Created++
            }
        }
    }

    // Phase 2: Users in IAM but not in HR → disable or report (orphaned accounts)
    for extID, user := range iamByExtID {
        if _, exists := hrByExtID[extID]; !exists {
            result.Drift = append(result.Drift, DriftItem{
                Type:   "orphan_in_iam",
                UserID: user.ID.String(),
                IAMData: map[string]any{
                    "username":  user.Username,
                    "email":     user.Email,
                    "status":    string(user.Status),
                },
            })

            if e.policy.DryRun || e.policy.OnUserInIAMNotInHR == "report" {
                continue
            }
            if e.policy.OnUserInIAMNotInHR == "disable" && user.Status == domain.UserStatusActive {
                e.identitySvc.DisableUser(ctx, user.ID)
                result.Disabled++
            }
        }
    }

    // Phase 3: Attribute mismatches → update or report
    for extID, emp := range hrByExtID {
        user, exists := iamByExtID[extID]
        if !exists {
            continue
        }
        mismatches := detectMismatches(user, emp)
        if len(mismatches) > 0 {
            result.Drift = append(result.Drift, DriftItem{
                Type:     "attribute_mismatch",
                UserID:   user.ID.String(),
                Mismatch: mismatches,
            })
            if !e.policy.DryRun && e.policy.OnAttributeMismatch == "update" {
                e.applyAttributeUpdates(ctx, user, mismatches)
                result.Updated++
            }
        }
    }

    return result, nil
}

func detectMismatches(user *domain.User, emp *HREmployee) map[string]string {
    m := map[string]string{}
    if user.Email != emp.Email && emp.Email != "" {
        m["email"] = fmt.Sprintf("%s → %s", emp.Email, user.Email)
    }
    if user.DisplayName != emp.DisplayName && emp.DisplayName != "" {
        m["display_name"] = fmt.Sprintf("%s → %s", emp.DisplayName, user.DisplayName)
    }
    if emp.Status == "terminated" && user.Status == domain.UserStatusActive {
        m["status"] = "terminated → active (should be disabled)"
    }
    return m
}
```

### 8.2 Reconciliation Scheduling

Reconciliation should run on a schedule — daily for large organizations, weekly for
smaller ones. Using a cron-based scheduler:

```go
// Run daily at 2 AM
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    for range ticker.C {
        ctx := context.Background()
        result, err := engine.Reconcile(ctx, defaultTenantID)
        if err != nil {
            log.Printf("reconciliation failed: %v", err)
            continue
        }
        log.Printf("reconciliation: +%d ~%d -%d !%d drift items",
            result.Created, result.Updated, result.Disabled, len(result.Drift))
    }
}()
```

---

## 9. GGID Lifecycle Gap Analysis

### 9.1 What Exists

| Feature | Status | Location |
|---------|--------|----------|
| User CRUD (Create/Read/Update/Delete) | Implemented | `services/identity/internal/service/identity_service.go` |
| User states (active, locked, disabled, deleted) | Implemented | `services/identity/internal/domain/user.go` |
| Lock/Unlock/Disable/Activate | Implemented | `services/identity/internal/service/identity_service.go` |
| SCIM /Users CRUD (POST/GET/PUT/PATCH/DELETE) | Implemented | `services/identity/internal/scim/handler.go` |
| SCIM PATCH engine (add/replace/remove) | Implemented | `services/identity/internal/scim/patch.go` |
| SCIM filter parser (eq, co, sw, pr, gt, etc.) | Implemented | `services/identity/internal/scim/filter.go` |
| SCIM bulk operations | Implemented | `services/identity/internal/scim/bulk.go` |
| SCIM pagination + sorting | Implemented | `services/identity/internal/scim/handler.go` |
| SCIM Groups | Implemented | `services/identity/internal/scim/groups.go` |
| SCIM ETag/If-Match/If-None-Match | Implemented | `services/identity/internal/scim/etag.go` |
| External identity linking | Implemented | `services/identity/internal/domain/external_identity.go` |
| LDAP JIT provisioning | Implemented | `services/identity/internal/service/identity_service.go` |
| Audit event publisher (NATS) | Implemented | `pkg/audit/publisher.go` |
| LastLoginAt field | Exists on User model | `services/identity/internal/domain/user.go` |

### 9.2 What's Missing

| Feature | Priority | Effort |
|---------|----------|--------|
| **SAML JIT provisioning** | P0 | 3 days — Add `JITProvisionFromSAML` to identity service, wire from OAuth service SAML handler |
| **OIDC JIT provisioning** | P0 | 2 days — Add `JITProvisionFromOIDC`, wire from OAuth service OIDC callback |
| **`pre-provisioned` state** | P1 | 1 day — Add to `UserStatus`, set on SCIM-created accounts without credentials |
| **`suspended` state** | P1 | 1 day — Distinct from `disabled`; admin-hold semantics with reactivation path |
| **Dormant account detection** | P0 | 3 days — Scanner job, per-tenant config, grace period workflow, notifications |
| **Lifecycle event publishing** | P1 | 2 days — Wire audit.Publisher into LockUser/SuspendUser/DeleteUser to emit lifecycle events |
| **Outbound SCIM client** | P2 | 3 days — Client for pushing users to downstream apps, reconciliation loop |
| **Account reconciliation** | P2 | 3 days — HR sync engine, drift detection, automated remediation |
| **SCIM filter → SQL translation** | P1 | 2 days — Currently only `externalId eq` is pushed to DB; full filters are post-query |
| **Lifecycle state machine validation** | P1 | 1 day — Enforce valid transitions (e.g. cannot go from deleted → active without restore) |

### 9.3 Code-Level Observations

1. **SCIM PATCH handler is limited**: The `patchUser` method in `handler.go` only
   handles `displayName` and `active` paths. Other SCIM attributes (emails,
   phoneNumbers, name) are silently ignored. The `ApplyPatch` engine in `patch.go`
   is more complete but is not fully wired to the handler.

2. **SCIM createUser uses hardcoded temp password**: `Password: "TempPass123!"` —
   should generate a cryptographically random password and force a change on first
   login.

3. **No lifecycle event publishing on state transitions**: `LockUser`, `DisableUser`,
   `DeleteUser` update the database but do not publish events to NATS. Downstream
   services cannot react to suspensions or deletions.

4. **LastLoginAt exists but is not used for dormancy**: The field is on the User
   model but no scanner or policy enforcement exists.

5. **No state transition validation**: `setStatus` allows any transition without
   checking validity. For example, a deleted user can be locked without being
   restored first.

---

## 10. Gap Analysis and Recommendations

### Recommendation 1: Implement SAML/OIDC JIT Provisioning (P0, ~5 days)

**Rationale:** Without JIT provisioning, federated users cannot log in unless an
admin pre-creates their account. This blocks enterprise SSO adoption.

**Action items:**
- Add `JITProvisionFromSAML` and `JITProvisionFromOIDC` to `IdentityService`
- Wire from OAuth service callback handlers
- Add per-tenant config flag: `jit_provisioning_enabled`
- Implement attribute mapping table (per Section 4)

### Recommendation 2: Build Dormant Account Scanner (P0, ~3 days)

**Rationale:** Dormant accounts are a top-5 OWASP identity risk. GGID already has
`LastLoginAt` but doesn't act on it.

**Action items:**
- Implement `DormantScanner` (per Section 6) as a background job
- Add per-tenant config for threshold, grace period, exclusions
- Wire `audit.Publisher` to emit `user.dormant_flagged` events
- Add re-activation flow endpoint

### Recommendation 3: Wire Lifecycle Event Publishing (P1, ~2 days)

**Rationale:** State transitions (`LockUser`, `DisableUser`, `DeleteUser`) currently
update the DB silently. Without events, downstream services (session revocation,
notification) cannot react.

**Action items:**
- Inject `audit.Publisher` into `IdentityService`
- Call `PublishTransition` in `setStatus` for all state changes
- Add a session revoker consumer in the gateway service

### Recommendation 4: Expand SCIM PATCH Handler Coverage (P1, ~2 days)

**Rationale:** Current PATCH only handles `displayName` and `active`. IdPs like
Azure AD send PATCH operations for emails, phone numbers, and name changes that
are silently dropped.

**Action items:**
- Wire the existing `ApplyPatch` engine to handle all SCIM attributes
- Add domain model update for emails, phone numbers, name components
- Add integration tests with real Azure AD PATCH payloads

### Recommendation 5: Add State Transition Validation (P1, ~1 day)

**Rationale:** Without validation, invalid transitions (e.g. `deleted → locked`)
can corrupt account state and confuse downstream consumers.

**Action items:**
- Define allowed transitions in a state machine table
- Add validation in `setStatus` before applying
- Return `gerr.InvalidArgument` for illegal transitions
- Add unit tests for all valid/invalid transition pairs

---

## References

- RFC 7643: SCIM Core Schema — https://datatracker.ietf.org/doc/html/rfc7643
- RFC 7644: SCIM Protocol — https://datatracker.ietf.org/doc/html/rfc7644
- GGID SCIM implementation: `services/identity/internal/scim/`
- GGID identity service: `services/identity/internal/service/identity_service.go`
- GGID audit publisher: `pkg/audit/publisher.go`
- `identity-lifecycle-automation.md` — JML automation, JIT concept, IGA integration
- `scim-conformance-testing.md` — SCIM conformance test suites and roadmap
