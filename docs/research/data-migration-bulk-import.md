# Data Migration & Bulk User Import: User Lifecycle Migration Toolkit for GGID

> **Focus**: Building a complete user migration toolkit — bulk import, lazy (JIT) migration, password hash compatibility, and session continuity — enabling organizations to migrate users from legacy identity systems (Auth0, Okta, Keycloak, LDAP/AD, custom databases) into GGID with zero downtime.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [The Migration Challenge](#2-the-migration-challenge)
3. [Migration Strategies](#3-migration-strategies)
4. [Password Hash Compatibility](#4-password-hash-compatibility)
5. [Industry Landscape](#5-industry-landscape)
6. [GGID Current State Analysis](#6-ggid-current-state-analysis)
7. [Gap Analysis](#7-gap-analysis)
8. [Proposed Architecture: Migration Toolkit](#8-proposed-architecture-migration-toolkit)
9. [Multi-Hash Password Verifier](#9-multi-hash-password-verifier)
10. [Bulk Import Pipeline](#10-bulk-import-pipeline)
11. [Lazy Migration (JIT Provisioning)](#11-lazy-migration-jit-provisioning)
12. [Database Schema](#12-database-schema)
13. [API Design](#13-api-design)
14. [Console UI Design](#14-console-ui-design)
15. [Performance Considerations](#15-performance-considerations)
16. [Security Considerations](#16-security-considerations)
17. [Competitive Differentiation](#17-competitive-differentiation)
18. [Migration Playbooks](#18-migration-playbooks)
19. [Implementation Backlog](#19-implementation-backlog)

---

## 1. Executive Summary

When an organization adopts GGID, their first task is migrating existing users from their current identity system. This is the **#1 operational blocker** for IAM platform adoption — if migration is painful, organizations delay or avoid switching.

GGID currently supports:
- Argon2id password hashing (`pkg/crypto/crypto.go:68`) with pepper
- SCIM 2.0 Bulk operations (`services/identity/internal/scim/bulk.go:52`)
- Per-tenant IdP configs with auto-provision (`services/identity/internal/idpconfig/idpconfig.go:58`)

But GGID is **missing critical migration capabilities**:
1. No multi-hash password verifier (cannot verify bcrypt/PBKDF2/scrypt hashes during import)
2. No bulk import API (no `/api/v1/users/bulk-import` endpoint)
3. No lazy/JIT migration support (no "login-time migration from legacy DB" flow)
4. No migration job tracking (no async job status, progress, error reporting)
5. No CSV/JSON import wizard in Console UI
6. No password rehashing on login (upgrading legacy hashes to Argon2id transparently)

**Recommendation**: Build a **Migration Toolkit** with three pillars:
1. **Multi-hash password verifier** — support bcrypt, PBKDF2, scrypt, MD5+salt, SHA256+salt during import and lazy migration
2. **Bulk import pipeline** — async job-based API for importing thousands of users from JSON/CSV with progress tracking
3. **Lazy migration engine** — configurable legacy DB connector that migrates users on first login

**Estimated effort**: 4 sprints for MVP (multi-hash + bulk import + lazy migration + Console wizard).

---

## 2. The Migration Challenge

### Why Migration Is Hard

```
┌────────────────────────────────────────────────────────────────────┐
│                    THE MIGRATION PROBLEM                            │
│                                                                    │
│  Legacy System              GGID                                   │
│  ┌──────────────┐           ┌──────────────┐                      │
│  │ 500K users   │           │  0 users     │                      │
│  │ bcrypt hash  │    ───►   │  Argon2id    │                      │
│  │ custom attrs │           │  GGID schema │                      │
│  │ roles/groups │           │  RBAC model  │                      │
│  │ MFA enroll.  │           │  TOTP/WebAuthn│                     │
│  │ OAuth tokens │           │  JWT refresh │                      │
│  └──────────────┘           └──────────────┘                      │
│                                                                    │
│  Challenges:                                                       │
│  1. Password hashes incompatible (bcrypt ≠ Argon2id)              │
│  2. User attribute schema differs                                  │
│  3. Role/permission mapping required                               │
│  4. MFA enrollment data may not transfer                           │
│  5. Active sessions/OAuth tokens must be invalidated              │
│  6. Social login links (Google, GitHub) must be preserved          │
│  7. Cannot afford downtime (100% uptime required)                  │
│  8. Cannot force all users to reset passwords                      │
│  9. Must maintain audit trail across migration                     │
│  10. Rate limits on import APIs constrain throughput               │
└────────────────────────────────────────────────────────────────────┘
```

### Scale Considerations

| Organization Size | User Count | Import Time (Batch) | Key Risk |
|-------------------|-----------|--------------------|---------| 
| Startup | 100-1K | Minutes | Minimal |
| Mid-market | 1K-50K | Hours | Password reset friction |
| Enterprise | 50K-500K | Days | Session disruption, rate limits |
| Large enterprise | 500K-5M | Weeks | Requires phased migration |
| Hyperscale | 5M+ | Months | Custom tooling required |

---

## 3. Migration Strategies

### Strategy 1: Big Bang (Bulk Migration)

All users migrated in a single planned operation.

```
T-0: Export all users from legacy system
T+1h: Transform to GGID format (hash migration, attribute mapping)
T+2h: Bulk import via GGID API
T+4h: Switch DNS / app config to GGID
T+5h: Legacy system → read-only (fallback)
T+7d: Decommission legacy system
```

| Pros | Cons |
|------|------|
| Fastest overall migration | Requires maintenance window |
| Clean cutover | All-or-nothing risk |
| Legacy system decommissioned quickly | Password reset if hashes incompatible |
| Predictable timeline | Large data transfer in one operation |

**Best for**: Small-to-medium user bases (<50K), compatible password hashes, low-traffic periods.

### Strategy 2: Lazy Migration (JIT / Trickle)

Users migrated individually on first login after GGID goes live.

```
T-0: GGID goes live, legacy system stays operational
User logs in to GGID →
  GGID checks: does user exist in GGID? → Yes → authenticate directly
                                → No  → query legacy system
                                     → verify password against legacy hash
                                     → if valid: create user in GGID with GGID hash
                                     → return tokens
                                     → flag for legacy decommission
T+N months: Bulk import remaining inactive users
T+N+1: Disable legacy system
```

| Pros | Cons |
|------|------|
| Zero downtime | Legacy system must stay operational |
| Zero password resets (if hashes compatible) | Migration takes months for inactive users |
| Gradual rollout | Extra latency on first login |
| Can test with real users | Legacy DB connection required |

**Best for**: Large user bases (>50K), incompatible hashes, 24/7 availability requirements.

### Strategy 3: Hybrid (Recommended)

Combines bulk import for active users + lazy migration for inactive users.

```
Phase 1: Pre-migration
  - Export and bulk import users who logged in within last 90 days
  - Use lazy migration for the long tail of inactive users

Phase 2: Transition period (2-4 weeks)
  - Both systems operational
  - New users created only in GGID
  - Lazy migration handles users who didn't appear in export

Phase 3: Cutover
  - Force password reset for any remaining unmigrated users
  - Decommission legacy system
```

| Pros | Cons |
|------|------|
| Balances speed and safety | Requires coordination |
| Most active users migrated upfront | Two systems running temporarily |
| Handles edge cases | More complex than either single strategy |

**Best for**: Enterprise (>50K users), production environments, organizations that want both speed and safety.

---

## 4. Password Hash Compatibility

### The Core Problem

Password hashes are **one-way functions** — you cannot "decrypt" a hash to get the plaintext password. When migrating users, you have three options:

| Option | Description | User Impact |
|--------|-------------|-------------|
| **Import hash as-is** | Store legacy hash, verify with legacy algorithm | Zero impact — password works |
| **Rehash on login** | Import legacy hash, rehash to Argon2id on successful login | Zero impact — transparent upgrade |
| **Force password reset** | Don't import hash, require user to set new password | High friction — users must reset |

### Supported Hash Algorithms (Target)

| Algorithm | Format Example | Used By | Import Support |
|-----------|---------------|---------|----------------|
| **Argon2id** | `argon2id$3$65536$2$salt.hash` | GGID (native), Rust roasts | Native |
| **bcrypt** | `$2a$10$N9qo8uLOickgx2ZMRZoMy...` | Auth0, Node.js, Ruby, Django | **Required** |
| **PBKDF2** | `pbkdf2-sha256:10000:salt:hash` | AWS Cognito, Spring Security | **Required** |
| **scrypt** | `scrypt$N$r$p$salt.hash` | Tarsnap, Litecoin | **Required** |
| **LDAP SSHA** | `{SSHA}base64encodedhash+salt` | OpenLDAP, Active Directory | **Required** |
| **Firebase scrypt** | Custom modified scrypt | Firebase Auth | Optional |
| **Django PBKDF2** | `pbkdf2_sha256$iterations$salt$hash` | Django | Optional |
| **PHP password_hash** | `$2y$10$...` (bcrypt variant) | PHP applications | Via bcrypt |
| **SHA-256 + salt** | `sha256$salt$hash` | Legacy apps | **Required** (migration only) |
| **MD5 + salt** | `md5$salt$hash` | Very old legacy | Optional (insecure) |

### Multi-Hash Verification Flow

When a user logs in and their stored hash is not Argon2id:

```
1. User enters password
2. GGID checks stored hash format:
   - If "argon2id$..." → VerifyPassword(password, hash) [existing path]
   - If "$2a$..." or "$2b$..." → bcrypt.CompareHashAndPassword
   - If "pbkdf2-..." → crypto/pbkdf2 verify
   - If "scrypt$..." → scrypt verify
   - If "{SSHA}..." → LDAP SSHA verify
3. If verification succeeds:
   a. Rehash password with Argon2id: newHash = HashPassword(password)
   b. Update user record: SET password_hash = newHash, hash_algorithm = 'argon2id'
   c. Continue authentication (issue tokens)
4. If verification fails:
   a. Increment failed attempt counter
   b. Return "invalid credentials"
```

This **transparent rehashing** ensures all users are gradually upgraded to Argon2id without any action required on their part.

---

## 5. Industry Landscape

### Auth0 (Okta)

**Bulk Import**: `POST /api/v2/jobs/users-imports` (async, multipart/form-data)
- JSON file format with user objects
- Supports `custom_password_hash` for legacy hashes
- 500KB file size limit per job
- Max 2 concurrent jobs
- Progress tracking via `GET /api/v2/jobs/{id}`

**Lazy Migration**: Custom Database Connections with action scripts
- JavaScript `login(email, password, callback)` script
- Queries legacy DB, verifies password, returns profile
- `Import Users to Auth0` toggle auto-creates user on successful login

**Supported import hashes**: bcrypt, PBKDF2, scrypt, Argon2

### Keycloak

**Bulk Import**: JSON file with user array
- `keycloak-config-cli` or admin API `POST /admin/realms/{realm}/partialImport`
- Supports bcrypt, PBKDF2 password hashes
- No built-in lazy migration

**Lazy Migration**: Custom `User Storage Provider` SPI (Java)
- Implement `UserLookupProvider` + `CredentialInputValidator`
- Federated user queries delegated to legacy DB

### AWS Cognito

**Bulk Import**: CSV file upload via Console or API
- `CreateUserImportJob` API
- CSV with columns matching user pool attributes
- Supports bcrypt and PBKDF2-SHA-256
- Job-based with progress tracking

**Lazy Migration**: Lambda triggers
- `PreSignUp` / `DefineAuthChallenge` triggers
- Custom Lambda queries legacy DB

### LoginRadius

**Bulk Import**: CSV/JSON via Console
- Supports multiple hash formats
- `POST /api/v2/manage/accounts/bulk`
- Lazy migration via custom login handler

### Comparison Matrix

| Feature | Auth0 | Keycloak | AWS Cognito | LoginRadius | **GGID (target)** |
|---------|-------|----------|-------------|-------------|-------------------|
| **Bulk import format** | JSON | JSON | CSV | CSV/JSON | **JSON + CSV** |
| **Async job tracking** | Yes | No | Yes | Yes | **Yes** |
| **Lazy migration** | Custom DB scripts | User Storage SPI | Lambda triggers | Custom handler | **Legacy DB connector** |
| **bcrypt support** | Yes | Yes | Yes | Yes | **Yes** |
| **PBKDF2 support** | Yes | Yes | Yes | Yes | **Yes** |
| **scrypt support** | Yes | No | No | No | **Yes** |
| **LDAP SSHA support** | No | Yes | No | No | **Yes** |
| **Transparent rehash** | No | No | No | No | **Yes** |
| **Console import wizard** | Extension | Admin UI | Console | Console | **Yes** |
| **Open source** | No | Yes | No | No | **Yes (Apache 2.0)** |
| **Concurrent jobs** | 2 | N/A | 1 per pool | Configurable | **Configurable** |
| **Dry-run validation** | No | No | Yes (test) | No | **Yes** |

**Key differentiator**: GGID would be the only IAM with **transparent rehashing** (auto-upgrade legacy hashes to Argon2id on login) and **dry-run validation** for bulk imports.

---

## 6. GGID Current State Analysis

### Existing Infrastructure

| Component | File | Status |
|-----------|------|--------|
| Argon2id hashing | `pkg/crypto/crypto.go:68` | **Implemented** — `HashPassword()` |
| Argon2id verification | `pkg/crypto/crypto.go:90` | **Implemented** — `VerifyPassword()` |
| Pepper support | `pkg/crypto/crypto.go:40` | **Implemented** — `applyPepper()` |
| SCIM 2.0 Bulk | `services/identity/internal/scim/bulk.go:52` | **Implemented** — `HandleBulk()` |
| User creation | `services/identity/internal/service/` | **Implemented** — single user CRUD |
| IdP auto-provision | `services/identity/internal/idpconfig/idpconfig.go:58` | **Implemented** — `Create()` |
| LDAP federation | `services/auth/internal/service/local_provider.go` | **Implemented** — credential check |
| Audit events | `pkg/audit/` | **Implemented** — NATS publisher |

### What GGID Cannot Do Today

| # | Gap | Impact |
|---|-----|--------|
| 1 | No bcrypt/PBKDF2/scrypt verification | Cannot import users from Auth0/Cognito/Django without password reset |
| 2 | No bulk import API | Must create users one-by-one via SCIM or identity API |
| 3 | No migration job tracking | Cannot track import progress, errors, completion |
| 4 | No lazy migration | Cannot migrate users on login from legacy DB |
| 5 | No password rehashing | Legacy hashes remain legacy forever (no upgrade path) |
| 6 | No CSV import | No convenient format for non-technical admins |
| 7 | No Console import wizard | No UI for guided migration |
| 8 | No dry-run mode | Cannot validate import data without committing |
| 9 | No attribute mapping | No way to map legacy fields to GGID schema |
| 10 | No migration audit log | Cannot track what was imported when/by whom |

---

## 7. Gap Analysis

### Real-World Migration Scenarios That Fail

| # | Scenario | Current Behavior | Expected Behavior |
|---|----------|-----------------|-------------------|
| 1 | "Import 50K users from Auth0 with bcrypt hashes" | Fails — `VerifyPassword` only handles Argon2id | Import bcrypt hashes, verify with bcrypt, rehash to Argon2id on login |
| 2 | "Migrate users from LDAP without password reset" | Cannot import SSHA hashes | Import LDAP SSHA hashes, verify on login |
| 3 | "Upload CSV of users exported from Active Directory" | No CSV import endpoint | Console CSV upload with column mapping |
| 4 | "Track migration progress (50K users)" | No job tracking | Async job with progress bar, error list |
| 5 | "Test import without committing" | No dry-run mode | Dry-run returns validation report, no DB writes |
| 6 | "Lazy migrate from legacy MySQL DB" | No legacy connector | Configure DB connection, verify on first login |
| 7 | "Map legacy 'user_type' field to GGID role" | No attribute mapping | Configurable mapping: legacy field → GGID role |
| 8 | "Know which users were imported and when" | No audit trail | Migration log with timestamp, source, result |

---

## 8. Proposed Architecture: Migration Toolkit

### High-Level Architecture

```
                    ┌─────────────────────────────────────────────┐
                    │          Identity Service                    │
                    │                                             │
                    │   ┌─────────────────────────────────────┐   │
                    │   │     Migration Toolkit               │   │
                    │   │                                     │   │
                    │   │  ┌──────────┐  ┌────────────────┐ │   │
                    │   │  │ Bulk     │  │ Lazy Migration │ │   │
                    │   │  │ Import   │  │ Engine         │ │   │
                    │   │  │ Pipeline │  │                │ │   │
                    │   │  └────┬─────┘  └───────┬────────┘ │   │
                    │   │       │                │          │   │
                    │   │  ┌────┴────────────────┴───────┐  │   │
                    │   │  │  Multi-Hash Password        │  │   │
                    │   │  │  Verifier                   │  │   │
                    │   │  │                              │  │   │
                    │   │  │  Argon2id ✓                  │  │   │
                    │   │  │  bcrypt    ✓ (new)           │  │   │
                    │   │  │  PBKDF2    ✓ (new)           │  │   │
                    │   │  │  scrypt    ✓ (new)           │  │   │
                    │   │  │  LDAP SSHA ✓ (new)           │  │   │
                    │   │  │  SHA256+   ✓ (new)           │  │   │
                    │   │  └──────────────────────────────┘  │   │
                    │   │                                     │   │
                    │   │  ┌──────────┐  ┌────────────────┐ │   │
                    │   │  │ Attribute│  │ Migration Job  │ │   │
                    │   │  │ Mapper   │  │ Tracker        │ │   │
                    │   │  └──────────┘  └────────────────┘ │   │
                    │   └─────────────────────────────────────┘   │
                    │                      │                      │
                    │   ┌──────────────────▼──────────────────┐   │
                    │   │   User Repository                   │   │
                    │   │   (with hash_algorithm column)      │   │
                    │   └─────────────────────────────────────┘   │
                    └─────────────────────────────────────────────┘
```

---

## 9. Multi-Hash Password Verifier

### Design

```go
// pkg/crypto/multihash.go

// HashAlgorithm identifies the password hashing algorithm.
type HashAlgorithm string

const (
    HashArgon2id  HashAlgorithm = "argon2id"
    HashBcrypt    HashAlgorithm = "bcrypt"
    HashPBKDF2    HashAlgorithm = "pbkdf2"
    HashScrypt    HashAlgorithm = "scrypt"
    HashLDAPSSHA  HashAlgorithm = "ssha"
    HashSHA256    HashAlgorithm = "sha256"
)

// DetectHashAlgorithm identifies the algorithm from the hash format.
func DetectHashAlgorithm(hash string) HashAlgorithm {
    switch {
    case strings.HasPrefix(hash, "argon2id$"):
        return HashArgon2id
    case strings.HasPrefix(hash, "$2a$"), strings.HasPrefix(hash, "$2b$"), strings.HasPrefix(hash, "$2y$"):
        return HashBcrypt
    case strings.HasPrefix(hash, "pbkdf2"):
        return HashPBKDF2
    case strings.HasPrefix(hash, "scrypt$"):
        return HashScrypt
    case strings.HasPrefix(hash, "{SSHA}"), strings.HasPrefix(hash, "{SHA}"):
        return HashLDAPSSHA
    case strings.HasPrefix(hash, "sha256$"):
        return HashSHA256
    default:
        return HashArgon2id // fallback
    }
}

// VerifyPasswordMulti verifies a password against any supported hash format.
// Returns (valid, algorithm, error).
func VerifyPasswordMulti(password, hash string) (bool, HashAlgorithm, error) {
    algo := DetectHashAlgorithm(hash)
    
    switch algo {
    case HashArgon2id:
        ok, err := VerifyPassword(password, hash) // existing Argon2id path
        return ok, algo, err
        
    case HashBcrypt:
        err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
        return err == nil, algo, nil
        
    case HashPBKDF2:
        ok, err := verifyPBKDF2(password, hash)
        return ok, algo, err
        
    case HashScrypt:
        ok, err := verifyScrypt(password, hash)
        return ok, algo, err
        
    case HashLDAPSSHA:
        ok, err := verifyLDAPSSHA(password, hash)
        return ok, algo, err
        
    case HashSHA256:
        ok, err := verifySHA256Salted(password, hash)
        return ok, algo, err
        
    default:
        return false, algo, fmt.Errorf("unsupported hash format")
    }
}

// ShouldRehash returns true if the hash should be upgraded to Argon2id.
func ShouldRehash(algo HashAlgorithm) bool {
    return algo != HashArgon2id
}

// RehashToArgon2id converts a plaintext password to Argon2id.
// Called after successful legacy hash verification.
func RehashToArgon2id(password string) (string, error) {
    return HashPassword(password) // existing function
}
```

### Integration with Auth Service

In `services/auth/internal/service/local_provider.go`, the login flow is modified:

```go
// Before: only Argon2id
func (p *LocalProvider) Authenticate(ctx context.Context, email, password string) (*User, error) {
    user, err := p.repo.GetByEmail(ctx, email)
    if err != nil { return nil, err }
    
    ok, err := crypto.VerifyPassword(password, user.PasswordHash)
    if !ok || err != nil { return nil, ErrInvalidCredentials }
    
    return user, nil
}

// After: multi-hash with transparent rehashing
func (p *LocalProvider) Authenticate(ctx context.Context, email, password string) (*User, error) {
    user, err := p.repo.GetByEmail(ctx, email)
    if err != nil { return nil, err }
    
    // Multi-hash verification
    ok, algo, err := crypto.VerifyPasswordMulti(password, user.PasswordHash)
    if !ok || err != nil { return nil, ErrInvalidCredentials }
    
    // Transparent rehash to Argon2id if needed
    if crypto.ShouldRehash(algo) {
        newHash, err := crypto.RehashToArgon2id(password)
        if err == nil {
            _ = p.repo.UpdatePasswordHash(ctx, user.ID, newHash, "argon2id")
            // Log: "user %s hash upgraded from %s to argon2id", user.ID, algo
        }
        // Non-fatal: if rehash fails, user still authenticated; will retry next login
    }
    
    return user, nil
}
```

---

## 10. Bulk Import Pipeline

### Architecture

```
    Admin                GGID API              Job Queue           DB
      │                     │                     │                 │
      │ 1. POST /import     │                     │                 │
      │  (JSON/CSV file)    │                     │                 │
      ├────────────────────►│                     │                 │
      │                     │ 2. Create job       │                 │
      │                     ├────────────────────►│                 │
      │   3. 202 Accepted   │                     │                 │
      │   {job_id}          │                     │                 │
      │◄────────────────────┤                     │                 │
      │                     │ 4. Parse + validate │                 │
      │                     │    each user record │                 │
      │                     │ 5. Batch insert     │                 │
      │                     │    (100/batch)      │                 │
      │                     ├─────────────────────────────────────►│
      │                     │                     │ 6. Update       │
      │                     │                     │    progress     │
      │                     │                     │                 │
      │ 7. GET /import/{id} │                     │                 │
      ├────────────────────►│                     │                 │
      │   {status, progress,│                     │                 │
      │    errors}          │                     │                 │
      │◄────────────────────┤                     │                 │
```

### Import File Format

#### JSON Format

```json
[
  {
    "email": "alice@corp.com",
    "email_verified": true,
    "display_name": "Alice Chen",
    "first_name": "Alice",
    "last_name": "Chen",
    "password_hash": "$2a$10$N9qo8uLOickgx2ZMRZoMy...",
    "hash_algorithm": "bcrypt",
    "roles": ["developer"],
    "groups": ["engineering"],
    "department": "Engineering",
    "phone": "+1-555-0100",
    "metadata": {
      "legacy_id": "usr_12345",
      "hire_date": "2023-01-15"
    }
  },
  {
    "email": "bob@corp.com",
    "email_verified": true,
    "display_name": "Bob Smith",
    "password_hash": "argon2id$3$65536$2$salt.hash",
    "hash_algorithm": "argon2id",
    "roles": ["admin"]
  }
]
```

#### CSV Format

```csv
email,email_verified,display_name,first_name,last_name,password_hash,hash_algorithm,roles,department
alice@corp.com,true,Alice Chen,Alice,Chen,$2a$10$N9qo8uLO...,bcrypt,developer,Engineering
bob@corp.com,true,Bob Smith,Bob,Smith,argon2id$3$65536$2$...,argon2id,admin,Operations
```

### Processing Pipeline

```go
// services/identity/internal/service/migration_import.go

// ImportJob represents an async bulk import operation.
type ImportJob struct {
    ID              uuid.UUID
    TenantID        uuid.UUID
    Status          ImportStatus    // pending, validating, importing, completed, failed
    TotalUsers      int
    ProcessedUsers  int
    FailedUsers     int
    Errors          []ImportError
    Source          string          // "json", "csv", "auth0", "keycloak"
    StartedAt       *time.Time
    CompletedAt     *time.Time
    CreatedAt       time.Time
}

type ImportStatus string
const (
    ImportStatusPending     ImportStatus = "pending"
    ImportStatusValidating  ImportStatus = "validating"
    ImportStatusImporting   ImportStatus = "importing"
    ImportStatusCompleted   ImportStatus = "completed"
    ImportStatusFailed      ImportStatus = "failed"
)

type ImportError struct {
    Row     int    // Line number in source file
    Email   string // User email (if available)
    Field   string // Field that caused the error
    Message string // Error description
}

// ProcessImportFile reads, validates, and imports users from a file.
func (s *MigrationService) ProcessImportFile(ctx context.Context, jobID uuid.UUID, data []byte, format string) error {
    // 1. Update job status → validating
    s.updateJobStatus(ctx, jobID, ImportStatusValidating)
    
    // 2. Parse users from file
    users, err := s.parseUsers(data, format)
    if err != nil {
        s.failJob(ctx, jobID, "parse error: "+err.Error())
        return err
    }
    
    s.setJobTotal(ctx, jobID, len(users))
    
    // 3. Validate each user record
    var validUsers []ImportUser
    for i, u := range users {
        if err := s.validateUser(u); err != nil {
            s.addJobError(ctx, jobID, i+1, u.Email, "", err.Error())
            continue
        }
        validUsers = append(validUsers, u)
    }
    
    // 4. Update status → importing
    s.updateJobStatus(ctx, jobID, ImportStatusImporting)
    
    // 5. Batch insert (configurable batch size, default 100)
    batchSize := 100
    for i := 0; i < len(validUsers); i += batchSize {
        end := i + batchSize
        if end > len(validUsers) { end = len(validUsers) }
        
        batch := validUsers[i:end]
        if err := s.batchCreateUsers(ctx, batch); err != nil {
            // Log error, continue with next batch
            s.addJobError(ctx, jobID, i, "", "", err.Error())
        }
        
        s.incrementJobProgress(ctx, jobID, len(batch))
    }
    
    // 6. Mark complete
    s.completeJob(ctx, jobID)
    return nil
}
```

---

## 11. Lazy Migration (JIT Provisioning)

### Architecture

```
    User               GGID Auth Svc        Legacy DB Connector      Legacy DB
      │                     │                      │                    │
      │ 1. POST /login      │                      │                    │
      │  {email, password}  │                      │                    │
      ├────────────────────►│                      │                    │
      │                     │ 2. Check GGID DB     │                    │
      │                     │    user exists?      │                    │
      │                     │    → No              │                    │
      │                     │ 3. Lazy migration    │                    │
      │                     │    enabled?          │                    │
      │                     │    → Yes             │                    │
      │                     ├─────────────────────►│                    │
      │                     │                      │ 4. Query user     │
      │                     │                      ├───────────────────►│
      │                     │                      │ 5. Return user    │
      │                     │                      │    + legacy hash  │
      │                     │◄─────────────────────┤                    │
      │                     │ 6. Verify password   │                    │
      │                     │    against legacy    │                    │
      │                     │    hash (multi-hash) │                    │
      │                     │                      │                    │
      │                     │ 7. If valid:         │                    │
      │                     │    a. HashPassword() │                    │
      │                     │       → Argon2id     │                    │
      │                     │    b. Create user    │                    │
      │                     │       in GGID DB     │                    │
      │                     │    c. Issue tokens   │                    │
      │                     │                      │                    │
      │ 8. 200 + tokens     │                      │                    │
      │◄────────────────────┤                      │                    │
```

### Legacy DB Connector Configuration

```yaml
# Per-tenant lazy migration config
lazy_migration:
  enabled: true
  source:
    type: postgres          # postgres, mysql, ldap, http
    connection: "postgres://legacy:5432/users"
    query: |
      SELECT id, email, password_hash, display_name, department
      FROM users WHERE email = $1
  hash_detection: auto      # auto, bcrypt, pbkdf2, ssha
  attribute_mapping:
    email: email
    display_name: display_name
    department: department
    legacy_id: id
  fallback:
    on_not_found: deny      # deny, allow_as_new, redirect_signup
    on_db_error: deny       # deny, allow_cached
```

---

## 12. Database Schema

```sql
-- Add hash_algorithm column to existing users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS hash_algorithm VARCHAR(32) DEFAULT 'argon2id';
ALTER TABLE users ADD COLUMN IF NOT EXISTS legacy_id VARCHAR(256);
ALTER TABLE users ADD COLUMN IF NOT EXISTS migrated_from VARCHAR(64);  -- 'auth0', 'keycloak', 'ldap'
ALTER TABLE users ADD COLUMN IF NOT EXISTS migrated_at TIMESTAMPTZ;

-- Migration jobs
CREATE TABLE migration_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            VARCHAR(128),               -- "Auth0 migration July 2026"
    source          VARCHAR(64) NOT NULL,        -- 'json', 'csv', 'auth0', 'keycloak', 'ldap', 'lazy'
    status          VARCHAR(32) NOT NULL,        -- 'pending', 'validating', 'importing', 'completed', 'failed'
    total_users     INT DEFAULT 0,
    processed_users INT DEFAULT 0,
    failed_users    INT DEFAULT 0,
    skipped_users   INT DEFAULT 0,
    config_json     JSONB,                       -- Import configuration
    file_name       VARCHAR(256),
    file_size       BIGINT,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NOT NULL                -- Admin user ID
);

-- Migration errors (per-user import failures)
CREATE TABLE migration_errors (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES migration_jobs(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL,
    row_number      INT,                         -- Line in source file
    email           VARCHAR(256),
    error_type      VARCHAR(64),                 -- 'duplicate', 'invalid_email', 'invalid_hash', 'missing_field'
    error_message   TEXT,
    raw_data        JSONB,                       -- The user record that failed
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Lazy migration configuration (per-tenant)
CREATE TABLE lazy_migration_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL UNIQUE,
    enabled         BOOLEAN NOT NULL DEFAULT false,
    source_type     VARCHAR(32) NOT NULL,        -- 'postgres', 'mysql', 'ldap', 'http'
    connection_str  TEXT NOT NULL,               -- Encrypted
    query_template  TEXT,                         -- SQL query template
    hash_detection  VARCHAR(32) DEFAULT 'auto',
    fallback_action VARCHAR(32) DEFAULT 'deny',
    attribute_map   JSONB,                        -- Legacy field → GGID field mapping
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Attribute mapping (reusable across migrations)
CREATE TABLE migration_attribute_maps (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            VARCHAR(128) NOT NULL,
    source_system   VARCHAR(64),                 -- 'auth0', 'keycloak', 'ldap', 'custom'
    mappings        JSONB NOT NULL,               -- { "legacy_field": "ggid_field" }
    role_mappings   JSONB,                        -- { "legacy_role": "ggid_role_key" }
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_migration_jobs_tenant ON migration_jobs (tenant_id, created_at DESC);
CREATE INDEX idx_migration_errors_job ON migration_errors (job_id);
CREATE INDEX idx_migration_errors_tenant ON migration_errors (tenant_id);
CREATE INDEX idx_users_legacy_id ON users (tenant_id, legacy_id) WHERE legacy_id IS NOT NULL;
CREATE INDEX idx_users_migrated_from ON users (migrated_from) WHERE migrated_from IS NOT NULL;
```

---

## 13. API Design

### Bulk Import

```
# Create import job (upload file)
POST /api/v1/identity/migration/import
Content-Type: multipart/form-data

Form fields:
  - file: users.json or users.csv (max 10MB)
  - format: "json" | "csv"
  - source: "auth0" | "keycloak" | "ldap" | "custom"
  - dry_run: true | false
  - upsert: true | false  (update existing users on email match)
  - attribute_map_id: uuid (optional, use saved mapping)

Response: 202 Accepted
{
    "job_id": "uuid",
    "status": "pending",
    "total_users": 0,
    "message": "Import job queued. Poll /migration/import/{job_id} for status."
}

# Check job status
GET /api/v1/identity/migration/import/{job_id}

Response:
{
    "job_id": "uuid",
    "status": "importing",
    "total_users": 50000,
    "processed_users": 35000,
    "failed_users": 12,
    "skipped_users": 3,
    "progress_percent": 70.0,
    "started_at": "2026-07-17T10:00:00Z",
    "estimated_completion": "2026-07-17T10:15:00Z"
}

# Get import errors
GET /api/v1/identity/migration/import/{job_id}/errors?limit=100&offset=0

Response:
{
    "errors": [
        {
            "row": 342,
            "email": "invalid@@email.com",
            "error_type": "invalid_email",
            "error_message": "Email format is invalid",
            "raw_data": { "email": "invalid@@email.com", ... }
        },
        {
            "row": 1250,
            "email": "dup@corp.com",
            "error_type": "duplicate",
            "error_message": "User with this email already exists",
            "raw_data": { ... }
        }
    ],
    "total_errors": 12,
    "has_more": false
}

# Cancel running import job
POST /api/v1/identity/migration/import/{job_id}/cancel

# Download error report (CSV)
GET /api/v1/identity/migration/import/{job_id}/errors/download
```

### Dry-Run Validation

```
# Validate import file without committing
POST /api/v1/identity/migration/import?dry_run=true
(multipart file upload)

Response: 200 OK
{
    "valid": false,
    "total_users": 50000,
    "valid_users": 49985,
    "invalid_users": 15,
    "errors": [
        { "row": 342, "field": "email", "message": "Invalid format" },
        { "row": 1250, "field": "password_hash", "message": "Unrecognized hash format" }
    ],
    "warnings": [
        { "row": 5000, "message": "User has no roles assigned" }
    ],
    "hash_algorithms_detected": {
        "argon2id": 45000,
        "bcrypt": 4800,
        "pbkdf2": 185
    }
}
```

### Lazy Migration Configuration

```
# Configure lazy migration
PUT /api/v1/identity/migration/lazy-config
{
    "enabled": true,
    "source": {
        "type": "postgres",
        "host": "legacy-db.corp.com",
        "port": 5432,
        "database": "users",
        "username": "readonly_user",
        "password_encrypted": "aes-encrypted-blob"
    },
    "query": "SELECT id, email, password_hash, display_name FROM users WHERE email = $1",
    "hash_detection": "auto",
    "fallback": "deny",
    "attribute_mapping": {
        "email": "email",
        "display_name": "display_name",
        "legacy_id": "id"
    }
}

# Test lazy migration connection
POST /api/v1/identity/migration/lazy-config/test
{
    "test_email": "alice@corp.com"
}

Response:
{
    "connected": true,
    "user_found": true,
    "detected_hash_algorithm": "bcrypt",
    "mapped_attributes": {
        "email": "alice@corp.com",
        "display_name": "Alice Chen"
    }
}
```

### Migration Statistics

```
# Get migration overview for tenant
GET /api/v1/identity/migration/stats

Response:
{
    "total_users": 125000,
    "migrated_users": 118500,
    "migration_rate": 94.8,
    "by_source": {
        "bulk_import": 95000,
        "lazy_migration": 23500
    },
    "hash_distribution": {
        "argon2id": 115200,
        "bcrypt": 2800,
        "pbkdf2": 500
    },
    "pending_rehash": 3300,
    "recent_jobs": [
        { "job_id": "...", "status": "completed", "total": 50000, "failed": 12 }
    ]
}
```

---

## 14. Console UI Design

### Migration Dashboard

```
┌──────────────────────────────────────────────────────────────────┐
│  User Migration                                                 │
│                                                                  │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐     │
│  │  Total Users   │  │  Migrated      │  │  Pending       │     │
│  │  125,000       │  │  118,500       │  │  6,500         │     │
│  │                │  │  94.8%         │  │  5.2%          │     │
│  └────────────────┘  └────────────────┘  └────────────────┘     │
│                                                                  │
│  Active Jobs                                                     │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ Auth0 Migration        ████████████████░░░░  70%  35K/50K │  │
│  │ Started: 10:00 AM   Est. completion: 10:15 AM             │  │
│  │ Errors: 12   [View Errors]   [Cancel]                      │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Quick Actions                                                   │
│  + New Import (JSON/CSV)                                        │
│  + Configure Lazy Migration                                     │
│  + View Migration History                                       │
│  + Download Migration Report                                    │
│                                                                  │
│  Hash Distribution                                               │
│  Argon2id  ████████████████████████████░  115,200              │
│  bcrypt    ██░░░░░░░░░░░░░░░░░░░░░░░░░░   2,800               │
│  PBKDF2    █░░░░░░░░░░░░░░░░░░░░░░░░░░░     500               │
│                                                                  │
│  3,300 users pending rehash to Argon2id                         │
└──────────────────────────────────────────────────────────────────┘
```

### Import Wizard

```
Step 1: Choose Source
  ◉ Upload File (JSON/CSV)
  ○ Auth0 Export
  ○ Keycloak Export
  ○ LDAP/AD Export
  ○ Custom Database (Lazy Migration)

Step 2: Upload File
  ┌──────────────────────────────────────────────────┐
  │  [Drop file here or click to browse]              │
  │                                                  │
  │  Supported: .json, .csv (max 10MB)               │
  │  ─────────────────────────────────────────────── │
  │  uploaded: users_export.json (2.3MB, 50,000 users)│
  └──────────────────────────────────────────────────┘

Step 3: Field Mapping
  ┌──────────────────────────────────────────────────┐
  │  Source Field       →  GGID Field                │
  │  email              →  email           ✓         │
  │  name               →  display_name    ✓         │
  │  hashed_password    →  password_hash   ✓         │
  │  user_type          →  role            (map)     │
  │  dept               →  department      ✓         │
  │  + Add custom mapping                              │
  └──────────────────────────────────────────────────┘

Step 4: Role Mapping
  ┌──────────────────────────────────────────────────┐
  │  Source Role        →  GGID Role                 │
  │  "admin"            →  admin            ✓        │
  │  "user"             →  viewer           ✓        │
  │  "superuser"        →  admin            ✓        │
  │  (unmapped)         →  (skip)                    │
  └──────────────────────────────────────────────────┘

Step 5: Validation (Dry-Run)
  ┌──────────────────────────────────────────────────┐
  │  ✓ 49,985 users valid                            │
  │  ✗ 15 users have errors:                         │
  │    Row 342: Invalid email format                 │
  │    Row 1250: Unrecognized hash format            │
  │    Row 3000: Missing required field "email"      │
  │                                                  │
  │  Detected hash algorithms:                       │
  │    bcrypt: 45,000  |  Argon2id: 4,985            │
  │                                                  │
  │  [Download Error Report]                         │
  └──────────────────────────────────────────────────┘

Step 6: Confirm & Import
  ☑ Import 49,985 valid users
  ☐ Skip 15 users with errors
  ☑ Upsert: update existing users on email match
  ☑ Rehash legacy passwords to Argon2id on next login

  [Start Import]
```

---

## 15. Performance Considerations

### Bulk Import Throughput

| Batch Size | Users/Second | 50K Users | Notes |
|-----------|-------------|-----------|-------|
| 10 | ~50 | ~17 min | Safe, low DB pressure |
| 50 | ~200 | ~4 min | Good balance |
| 100 | ~400 | ~2 min | Recommended default |
| 500 | ~1000 | ~50 sec | High DB pressure, may timeout |
| 1000 | ~2000 | ~25 sec | Risk of connection pool exhaustion |

### Optimization Strategies

1. **COPY instead of INSERT**: Use PostgreSQL `COPY` for bulk data loading (10-50x faster than INSERT)

2. **Transaction batching**: Wrap each batch in a transaction to avoid per-row commit overhead

3. **Connection pooling**: Use dedicated connection pool for import jobs (separate from API pool)

4. **Parallel processing**: For very large imports (>100K), split into multiple parallel jobs with configurable concurrency

5. **Memory streaming**: Parse JSON/CSV with streaming parser (not `json.Unmarshal` of entire file) to handle large files

6. **Progress checkpointing**: Persist progress every N rows so interrupted jobs can resume

### Lazy Migration Latency

| Step | Latency | Notes |
|------|---------|-------|
| GGID DB lookup (user exists?) | <2ms | Redis cache hit |
| Legacy DB query | 5-50ms | Network round-trip to legacy DB |
| Multi-hash verify | 1-100ms | bcrypt: ~100ms, PBKDF2: ~10ms |
| Rehash to Argon2id | ~50ms | Argon2id computation |
| Create user in GGID DB | 2-5ms | INSERT |
| **Total first-login overhead** | **60-200ms** | Only on first login; subsequent logins use GGID DB directly |

---

## 16. Security Considerations

### Data Handling

| Risk | Mitigation |
|------|-----------|
| **Password hash exposure during transit** | TLS mandatory; hash at rest encrypted in import file storage (AES-256-GCM) |
| **Import file with plaintext passwords** | Reject — validation checks for plaintext patterns; require hashed format |
| **Legacy DB credentials in config** | Encrypted at rest (`pkg/crypto` AES-256-GCM), never logged, never returned via API |
| **Import job privilege escalation** | Import requires `migration:import` permission; role mapping validated against caller's admin scope |
| **Duplicate user injection** | Upsert mode checks email + tenant_id uniqueness; dry-run catches duplicates |
| **Injection via legacy DB query** | Parameterized queries only; query template validated; no raw string concatenation |

### Audit Trail

Every migration action is logged to the audit service:

```json
{
    "event": "user.migrated",
    "tenant_id": "uuid",
    "actor": "admin@corp.com",
    "target": "alice@corp.com",
    "details": {
        "source": "auth0",
        "method": "bulk_import",
        "job_id": "uuid",
        "hash_algorithm": "bcrypt",
        "roles_assigned": ["developer"]
    },
    "timestamp": "2026-07-17T10:05:23Z"
}
```

---

## 17. Competitive Differentiation

| Feature | GGID (proposed) | Auth0 | Keycloak | AWS Cognito |
|---------|-----------------|-------|----------|-------------|
| **Bulk import (JSON)** | **Yes** | Yes | Yes | CSV only |
| **Bulk import (CSV)** | **Yes** | Extension | No | Yes |
| **Lazy migration** | **Yes** | Yes (JS scripts) | Yes (Java SPI) | Lambda triggers |
| **Multi-hash verify** | **6 algorithms** | 3-4 | 2 | 2 |
| **Transparent rehash** | **Yes** | No | No | No |
| **Dry-run validation** | **Yes** | No | No | No |
| **Console wizard** | **Yes (multi-step)** | Extension | Basic | Basic |
| **Attribute mapping** | **Configurable** | Via script | Via mapper | Via mapping |
| **Role mapping** | **Configurable** | Via script | Via mapper | Via Lambda |
| **Progress tracking** | **Real-time** | Polling | None | Polling |
| **Error reporting** | **Detailed + CSV** | Summary | None | Summary |
| **Open source** | **Yes (Apache 2.0)** | No | Yes | No |

**Key differentiators**:
1. **Transparent rehashing** — no other IAM auto-upgrades legacy hashes to Argon2id
2. **Dry-run validation** — test imports before committing
3. **6 hash algorithms** — widest compatibility (Argon2id, bcrypt, PBKDF2, scrypt, LDAP SSHA, SHA256+salt)
4. **Multi-step Console wizard** — guided migration with field/role mapping
5. **Detailed error reporting** — per-row errors with downloadable CSV

---

## 18. Migration Playbooks

### Playbook 1: Auth0 → GGID

```
1. Export users from Auth0 Management API:
   GET /api/v2/users?include_totals=true (paginate)
   
2. Transform Auth0 format to GGID format:
   - user_id → legacy_id
   - email → email
   - password_hash → password_hash (detect: bcrypt or argon2)
   - app_metadata.roles → roles
   - identities[].provider == 'google' → social_links

3. Dry-run import in GGID Console
4. Fix any validation errors
5. Run bulk import
6. Verify: compare user counts
7. Configure lazy migration as fallback (point to Auth0 custom DB)
8. Switch application config from Auth0 to GGID
9. Monitor for 1 week
10. Decommission Auth0
```

### Playbook 2: Keycloak → GGID

```
1. Export realm JSON from Keycloak admin:
   GET /admin/realms/{realm}/partialExport

2. Transform Keycloak format:
   - username → email (or custom mapping)
   - credentials[].secretData → password_hash (PBKDF2)
   - realmRoles → GGID roles
   - groups → GGID groups

3. Import via GGID Console (JSON format)
4. PBKDF2 hashes auto-detected, transparent rehash on login
```

### Playbook 3: LDAP/AD → GGID

```
1. Configure lazy migration:
   source: ldap
   connection: ldap://dc.corp.com:389
   base_dn: ou=users,dc=corp,dc=com
   filter: (mail={email})

2. Users migrate on next login (SSHA hashes auto-detected)
3. After 30 days, bulk import remaining inactive users
4. Decommission LDAP after all users migrated
```

---

## 19. Implementation Backlog

### P0 — Core Migration Engine (3 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 1 | Multi-hash password verifier | `pkg/crypto/multihash.go` — bcrypt, PBKDF2, scrypt, LDAP SSHA, SHA256 | 4 days |
| 2 | Transparent rehashing | Auto-upgrade to Argon2id on successful legacy verify | 2 days |
| 3 | Auth service integration | Wire multi-hash into `LocalProvider.Authenticate()` | 2 days |
| 4 | Migration job data model | PostgreSQL tables for jobs, errors, lazy configs | 2 days |
| 5 | Bulk import service | Async job-based import with batch processing | 5 days |
| 6 | JSON/CSV parser | Streaming parser for large files | 3 days |
| 7 | Dry-run validation | Validate without committing, return error report | 2 days |
| 8 | Import API endpoints | POST import, GET status, GET errors, POST cancel | 3 days |
| 9 | Unit tests | 90%+ coverage for multi-hash, parser, import service | 4 days |

### P1 — Enhanced Features (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 10 | Lazy migration engine | Legacy DB connector, per-tenant config | 5 days |
| 11 | Attribute mapping engine | Configurable field + role mapping | 3 days |
| 12 | PostgreSQL COPY optimization | Use COPY for 10-50x faster bulk inserts | 2 days |
| 13 | Progress checkpointing | Resume interrupted jobs | 2 days |
| 14 | Migration statistics API | Aggregated stats endpoint | 2 days |
| 15 | Error report download (CSV) | Downloadable error file | 1 day |
| 16 | Integration tests | End-to-end import + lazy migration tests | 3 days |

### P2 — Console UI (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 17 | Migration dashboard | Stats cards, active jobs, hash distribution chart | 3 days |
| 18 | Import wizard (JSON/CSV) | Multi-step: upload → map fields → map roles → validate → import | 5 days |
| 19 | Job progress bar | Real-time progress with WebSocket or polling | 2 days |
| 20 | Error viewer | Sortable/filterable error table with download | 2 days |
| 21 | Lazy migration config UI | Form for legacy DB connection + attribute mapping | 3 days |
| 22 | Migration history | List of all migration jobs with status and details | 2 days |

### P3 — Advanced Features (Future)

| # | Task | Description |
|---|------|-------------|
| 23 | Source-specific importers | Pre-built importers for Auth0/Keycloak/Cognito export formats |
| 24 | Parallel import workers | Multi-worker parallel processing for >1M users |
| 25 | Social login migration | Preserve Google/GitHub/Microsoft identity links |
| 26 | MFA enrollment migration | Transfer TOTP secrets and WebAuthn credentials |
| 27 | Rollback engine | Undo a migration job (delete imported users) |
| 28 | Migration simulation | Simulate entire migration with synthetic data for load testing |
| 29 | Federation coexistence | Run GGID and legacy IdP in parallel with session bridging |

---

## References

- [Auth0: How to Migrate Users](https://auth0.com/blog/how-to-migrate-users-to-auth0-a-technical-guide/) — Bulk vs lazy migration strategies
- [Auth0: Bulk User Imports](https://auth0.com/docs/manage-users/user-migration/bulk-user-imports) — Management API import endpoint
- [SuperTokens: Lazy Migration](https://supertokens.com/blog/migrating-users-without-downtime-in-your-service) — Zero-downtime migration patterns
- [Keycloak: Partial Import](https://www.keycloak.org/docs/latest/server_admin/#_export_import) — Realm JSON import
- [AWS Cognito: User Import](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pools-using-import-tool.html) — CSV import job
- [RFC 7644 Section 3.7: SCIM Bulk](https://datatracker.ietf.org/doc/html/rfc7644#section-3.7) — SCIM bulk operations
- [OWASP: Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) — Hash algorithm recommendations
- [Argon2 RFC 9106](https://www.rfc-editor.org/rfc/rfc9106) — Argon2 specification
- [bcrypt Specification](https://datatracker.ietf.org/doc/html/draft-moyer-bcrypt-spec-00) — bcrypt password hashing
- [PBKDF2: RFC 2898](https://www.rfc-editor.org/rfc/rfc2898) — PBKDF2 key derivation
- [WeTransfer Migration Case Study](https://medium.com/@estebanpintos/migrating-80-million-users-to-auth0) — 80M user migration experience
