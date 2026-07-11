# Credential Rotation Automation for IAM Systems

> **Scope:** Rotation strategies for SAML certificates, OAuth client secrets,
> database passwords, LDAP bind credentials, PII encryption keys, password
> pepper, and a unified rotation scheduler.
>
> **Out of scope:** JWT signing key rotation — see
> [`key-rotation-iam.md`](./key-rotation-iam.md) (1373 lines) and
> [`jwks-key-rotation.md`](./jwks-key-rotation.md) (438 lines) for JWT
> signing key lifecycle, HSM, HKDF, JWKS caching, dual-key strategy, and
> `kid` propagation.

---

## Table of Contents

1. [Credential Inventory for IAM](#1-credential-inventory-for-iam)
2. [SAML Certificate Rotation](#2-saml-certificate-rotation)
3. [OAuth Client Secret Rotation](#3-oauth-client-secret-rotation)
4. [Database Password Rotation](#4-database-password-rotation)
5. [LDAP Bind Credential Rotation](#5-ldap-bind-credential-rotation)
6. [PII Encryption Key Rotation](#6-pii-encryption-key-rotation)
7. [Password Pepper Rotation](#7-password-pepper-rotation)
8. [Unified Rotation Scheduler](#8-unified-rotation-scheduler)
9. [GGID Credential Rotation Audit](#9-ggid-credential-rotation-audit)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. Credential Inventory for IAM

An IAM system manages a broader credential surface than typical web applications.
Each credential type has different rotation constraints, blast radius, and
automation maturity.

### Full Credential Taxonomy

| # | Credential | Where Stored | Rotation Frequency | Blast Radius | GGID Today |
|---|-----------|-------------|-------------------|-------------|------------|
| 1 | JWT signing keys (RSA) | `configs/rsa_private.pem` | 90 days | Auth outage | Covered in separate docs |
| 2 | SAML IdP/SP signing certs | Metadata XML / filesystem | 1-2 years | SSO breakage | `pkg/saml` — single cert, no rotation |
| 3 | OAuth client secrets | `oauth_clients.client_secret_hash` | 90 days | Per-client outage | `RotateClientSecret` exists (hard cutover) |
| 4 | Database passwords | Env vars / Vault | 30-90 days | Full service outage | Hardcoded `ggid:ggid` in dev compose |
| 5 | LDAP bind credentials | Env vars (`LDAP_BIND_PASSWORD`) | 90 days | LDAP auth outage | Hardcoded `admin123` in dev compose |
| 6 | Redis passwords | Env vars / config | 90 days | Session/token outage | No password in dev compose |
| 7 | NATS credentials | Connection URL | 90 days | Audit pipeline loss | No auth in dev compose |
| 8 | API keys (access keys) | DB / Redis | 60 days | Per-key | Access key page in console |
| 9 | PII encryption keys (DEK) | Env var / KMS | 90 days | Silent data corruption | `crypto.AESEncrypt` — single key, no versioning |
| 10 | Password pepper | Env var | 12 months | Requires re-hash of all passwords | `crypto.SetPepper` — single global, no rotation |

### Classification by Blast Radius

**Critical (system-wide outage):**
- Database passwords — all services down
- Redis password — auth service cannot issue/validate tokens
- Password pepper — all logins fail if changed without migration

**High (feature outage):**
- LDAP bind password — LDAP users cannot authenticate
- NATS credentials — audit pipeline stops
- SAML signing cert — SSO federation breaks for external IdPs

**Medium (per-client/per-user):**
- OAuth client secret — only the affected client breaks
- API keys — only the affected integration breaks

**Low (data-at-rest, delayed impact):**
- PII encryption key — old data still decryptable with old key, new data uses new key

### Classification by Rotation Difficulty

| Difficulty | Credential | Why |
|-----------|-----------|-----|
| Easy | API keys, NATS creds | Stateless, dual-valid period trivial |
| Medium | OAuth client secrets, DB passwords, LDAP | Need dual-valid window + coordinated cutover |
| Hard | SAML certs | External coordination with IdPs |
| Very Hard | PII DEK | Requires background re-encryption of all data |
| Very Hard | Password pepper | Requires every user to log in for re-hash |

---

## 2. SAML Certificate Rotation

### The Problem

SAML relies on X.509 certificates for signing assertions and AuthnRequests.
Unlike JWT keys (which are fetched dynamically via JWKS), SAML metadata is often
cached by IdPs for hours or days. Rotating a SAML certificate without a
dual-cert overlap window causes federation breakage.

### Dual-Cert Overlap Strategy

```
Phase 1 (Publish):    Metadata publishes BOTH old + new cert
                       IdP fetches new metadata during cache refresh window
Phase 2 (Accept):     SP signs with NEW cert; accepts assertions from BOTH
Phase 3 (Revoke):     After confidence interval, old cert removed from metadata
```

**Timing rules:**
- Phase 1 duration: >= metadata cache TTL (typically 24-48 hours)
- Phase 2 duration: >= 1 week for external IdPs to update
- Phase 3: validate no requests signed with old cert for 7+ days before removal

### Coordinating with External IdPs

For enterprise IdPs (Okta, Azure AD, OneLogin):
1. Export updated SP metadata with dual certs
2. Notify IdP administrators with a migration window
3. Confirm IdP has ingested new metadata before Phase 3
4. Maintain a rollback path: re-add old cert to metadata if IdP lags

### Go Code: SAML Cert Rotation Manager

```go
package saml

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"
	"time"
)

// CertRotationPhase represents the current state of a cert rotation.
type CertRotationPhase int

const (
	PhaseStable   CertRotationPhase = iota // single cert, no rotation in progress
	PhasePublish                            // dual cert published in metadata
	PhaseAccept                             // signing with new cert
	PhaseComplete                           // old cert revoked
)

// CertRotationManager manages SAML signing certificate rotation with
// a dual-cert overlap window to prevent federation breakage.
type CertRotationManager struct {
	mu sync.RWMutex

	// activeCert is used for signing AuthnRequests.
	activeCert *x509.Certificate
	activeKey  interface{} // *rsa.PrivateKey or *ecdsa.PrivateKey

	// previousCert is accepted for verification during overlap.
	// nil when no rotation is in progress.
	previousCert *x509.Certificate

	phase       CertRotationPhase
	publishTime time.Time // when Phase 1 started
}

// NewCertRotationManager initializes with a single cert.
func NewCertRotationManager(cert *x509.Certificate, key interface{}) *CertRotationManager {
	return &CertRotationManager{
		activeCert: cert,
		activeKey:  key,
		phase:      PhaseStable,
	}
}

// BeginRotation starts the dual-cert overlap process.
// The new cert is published alongside the old one in metadata.
func (m *CertRotationManager) BeginRotation(newCert *x509.Certificate, newKey interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.phase != PhaseStable {
		return fmt.Errorf("rotation already in progress (phase=%d)", m.phase)
	}

	m.previousCert = m.activeCert
	m.activeCert = newCert
	m.activeKey = newKey
	m.phase = PhasePublish
	m.publishTime = time.Now()
	return nil
}

// AdvanceToAccept switches signing to the new cert while still
// accepting the old cert for verification.
// Call after metadata cache TTL has elapsed (typically 24-48h).
func (m *CertRotationManager) AdvanceToAccept() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase == PhasePublish {
		m.phase = PhaseAccept
	}
}

// CompleteRotation removes the old cert from metadata.
// Call only after confirming no requests use the old cert for 7+ days.
func (m *CertRotationManager) CompleteRotation() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.previousCert = nil
	m.phase = PhaseComplete
	// After a brief settling period, reset to stable.
	go func() {
		time.Sleep(24 * time.Hour)
		m.mu.Lock()
		m.phase = PhaseStable
		m.mu.Unlock()
	}()
}

// GetSigningCert returns the cert used for signing.
func (m *CertRotationManager) GetSigningCert() *x509.Certificate {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeCert
}

// IsCertAccepted returns true if the given cert is valid for verification.
// During overlap, both old and new certs are accepted.
func (m *CertRotationManager) IsCertAccepted(cert *x509.Certificate) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.activeCert.Equal(cert) {
		return true
	}
	if m.previousCert != nil && m.previousCert.Equal(cert) {
		return true
	}
	return false
}

// MetadataCerts returns all certs that should appear in SP metadata.
// During PhasePublish, returns both old and new. Otherwise just active.
func (m *CertRotationManager) MetadataCerts() []*x509.Certificate {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.phase == PhasePublish && m.previousCert != nil {
		return []*x509.Certificate{m.activeCert, m.previousCert}
	}
	return []*x509.Certificate{m.activeCert}
}

// Phase returns the current rotation phase.
func (m *CertRotationManager) Phase() CertRotationPhase {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.phase
}

// ParsePEMCert parses a PEM-encoded certificate.
func ParsePEMCert(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}
```

---

## 3. OAuth Client Secret Rotation

### The Problem

OAuth client secrets grant access to token endpoints. If compromised, an
attacker can mint tokens for the affected client. Secrets should rotate
periodically (every 90 days) and immediately upon suspected compromise.

### Dual-Secret Period

GGID currently has a `RotateClientSecret` method that does a **hard cutover**:
the old secret is immediately invalidated. This is fine for emergency rotation
but causes downtime for client applications that haven't updated their config.

A **dual-secret** approach allows both old and new secrets to be valid during
a migration window:

```
Day 0:  Generate new secret, store BOTH hashes
Day 0-7: Client developers update their config at their convenience
Day 7:  Revoke old secret
```

### Go Code: Dual-Secret Rotation

```go
package oauth

import (
	"context"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
)

// ClientSecretEntry tracks a secret hash and its lifecycle.
type ClientSecretEntry struct {
	Hash      string    // Argon2id hash
	CreatedAt time.Time
	ExpiresAt time.Time // when this secret becomes invalid
	IsActive  bool
}

// DualSecretClient extends OAuthClient with a list of valid secrets.
type DualSecretClient struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	ClientID     string
	Secrets      []ClientSecretEntry // multiple valid during overlap
	IsConfidential bool
}

// RotateClientSecretWithOverlap generates a new secret while keeping
// the old one valid for the specified overlap duration.
// Returns the new plaintext secret (shown once).
func RotateClientSecretWithOverlap(
	ctx context.Context,
	repo ClientRepo,
	tenantID uuid.UUID,
	clientID string,
	overlap time.Duration,
) (string, error) {
	client, err := repo.GetClientByID(ctx, tenantID, clientID)
	if err != nil {
		return "", err
	}

	// Generate new secret.
	newSecret, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return "", err
	}
	newHash, err := crypto.HashPassword(newSecret)
	if err != nil {
		return "", err
	}

	// Mark existing active secrets as expiring.
	now := time.Now()
	for i := range client.Secrets {
		if client.Secrets[i].IsActive {
			client.Secrets[i].ExpiresAt = now.Add(overlap)
		}
	}

	// Add new secret as the active one.
	client.Secrets = append(client.Secrets, ClientSecretEntry{
		Hash:      newHash,
		CreatedAt: now,
		IsActive:  true,
	})

	if err := repo.UpdateClient(ctx, tenantID, clientID, client); err != nil {
		return "", err
	}

	return newSecret, nil
}

// VerifyClientSecretMulti checks the plaintext against all non-expired
// secret hashes. Returns true if any match.
func VerifyClientSecretMulti(plaintext string, client *DualSecretClient) bool {
	now := time.Now()
	for _, entry := range client.Secrets {
		if !entry.IsActive && now.After(entry.ExpiresAt) {
			continue // expired, skip
		}
		ok, _ := crypto.VerifyPassword(plaintext, entry.Hash)
		if ok {
			return true
		}
	}
	return false
}

// PurgeExpiredSecrets removes expired secrets from the client.
// Run this as a daily background job.
func PurgeExpiredSecrets(
	ctx context.Context,
	repo ClientRepo,
	tenantID uuid.UUID,
	clientID string,
) (int, error) {
	client, err := repo.GetClientByID(ctx, tenantID, clientID)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	var kept []ClientSecretEntry
	purged := 0
	for _, entry := range client.Secrets {
		if !entry.IsActive && now.After(entry.ExpiresAt) {
			purged++
			continue
		}
		kept = append(kept, entry)
	}
	client.Secrets = kept

	if purged > 0 {
		if err := repo.UpdateClient(ctx, tenantID, clientID, client); err != nil {
			return 0, err
		}
	}
	return purged, nil
}

// ClientRepo abstracts the client persistence layer.
type ClientRepo interface {
	GetClientByID(ctx context.Context, tenantID uuid.UUID, clientID string) (*DualSecretClient, error)
	UpdateClient(ctx context.Context, tenantID uuid.UUID, clientID string, client *DualSecretClient) error
}
```

### Notification Strategy

When rotating secrets, notify client developers:
1. Send email/webhook with the rotation window deadline
2. Expose a `/oauth/clients/{id}/secret-status` endpoint showing expiry
3. Log a warning when a request authenticates with an expiring secret
4. After expiry, reject the old secret with a clear error message

### GGID Current State

GGID's `services/oauth/internal/service/oauth_service.go` already implements
`RotateClientSecret` (line 874) with hard cutover. It verifies the old secret,
generates a new one, and replaces the hash immediately. The gap: no overlap
period, no dual-secret support, no automated scheduling.

---

## 4. Database Password Rotation

### The Problem

Database passwords are long-lived static credentials embedded in connection
strings. If the database is compromised, the attacker retains access until the
password changes. Compliance frameworks (SOC 2, PCI DSS) require periodic DB
credential rotation (typically every 90 days).

### Zero-Downtime Rotation Procedure

```
Step 1: CREATE new DB role with new password
Step 2: GRANT same privileges to new role
Step 3: Update app config with new connection string
Step 4: Trigger graceful reload (SIGUSR2 or health-check drain)
Step 5: Verify new connections succeed
Step 6: DROP old role (after grace period)
```

The key insight: never change the password on the existing role. Create a new
role and cutover, so the old role remains as a rollback path.

### Go Code: DB Password Rotation

```go
package dbrotate

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// RotationConfig controls database credential rotation.
type RotationConfig struct {
	AdminDSN       string        // superuser DSN for CREATE/DROP ROLE
	AppDBName      string        // e.g. "ggid"
	RolePrefix     string        // e.g. "ggid_app" → roles: ggid_app_a, ggid_app_b
	PasswordLength int           // generated password length in bytes
	GracePeriod    time.Duration // old role kept before DROP
}

// RotateResult captures the outcome of a rotation.
type RotateResult struct {
	NewRole     string
	NewDSN      string
	OldRole     string
	GraceEnds   time.Time
}

// Rotate creates a new DB role with a fresh password and returns
// the new connection string. The old role is kept for GracePeriod
// as a rollback path.
func Rotate(ctx context.Context, cfg RotationConfig) (*RotateResult, error) {
	adminDB, err := sql.Open("pgx", cfg.AdminDSN)
	if err != nil {
		return nil, fmt.Errorf("connect admin DSN: %w", err)
	}
	defer adminDB.Close()

	// Generate new role name and password.
	suffix := randomSuffix(6)
	newRole := fmt.Sprintf("%s_%s", cfg.RolePrefix, suffix)
	password := generatePassword(cfg.PasswordLength)

	// Step 1: Create new role with password.
	_, err = adminDB.ExecContext(ctx,
		fmt.Sprintf(`CREATE ROLE %s WITH LOGIN PASSWORD '%s'`, newRole, password))
	if err != nil {
		return nil, fmt.Errorf("create role %s: %w", newRole, err)
	}

	// Step 2: Grant same privileges as a template role.
	// Assumes a base role "ggid_app_base" has all needed grants.
	_, err = adminDB.ExecContext(ctx,
		fmt.Sprintf(`GRANT ggid_app_base TO %s`, newRole))
	if err != nil {
		return nil, fmt.Errorf("grant to %s: %w", newRole, err)
	}

	// Step 3: Build new DSN.
	newDSN := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		newRole, password, "localhost:5432", cfg.AppDBName)

	// Step 4: Verify new credentials work.
	testDB, err := sql.Open("pgx", newDSN)
	if err != nil {
		return nil, fmt.Errorf("verify new DSN: %w", err)
	}
	if err := testDB.PingContext(ctx); err != nil {
		testDB.Close()
		return nil, fmt.Errorf("ping with new role: %w", err)
	}
	testDB.Close()

	return &RotateResult{
		NewRole:   newRole,
		NewDSN:    newDSN,
		GraceEnds: time.Now().Add(cfg.GracePeriod),
	}, nil
}

// CleanupOldRole drops a role after the grace period.
func CleanupOldRole(ctx context.Context, adminDSN, roleName string) error {
	adminDB, err := sql.Open("pgx", adminDSN)
	if err != nil {
		return err
	}
	defer adminDB.Close()

	// Force-disconnect sessions using the old role.
	_, err = adminDB.ExecContext(ctx,
		fmt.Sprintf(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE usename = '%s'`, roleName))
	if err != nil {
		return fmt.Errorf("terminate sessions for %s: %w", roleName, err)
	}

	_, err = adminDB.ExecContext(ctx,
		fmt.Sprintf(`DROP ROLE IF EXISTS %s`, roleName))
	return err
}

func generatePassword(byteLen int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, byteLen)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func randomSuffix(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[rand.Intn(len(hex))]
	}
	return string(b)
}
```

### Vault Dynamic DB Credentials

HashiCorp Vault's database secrets engine generates short-lived (e.g., 1-hour)
DB credentials on demand. Each service instance requests credentials at startup
and periodically (before TTL expiry). This eliminates static DB passwords
entirely:

```
App startup → Vault /database/creds/ggid-app → returns (username, password)
TTL=1h      → App requests renewal at 45min mark
App shutdown → Vault revokes credentials
```

**Trade-off:** adds Vault as a critical dependency. If Vault is down, services
cannot get DB credentials. Mitigate with Vault HA + credential caching.

---

## 5. LDAP Bind Credential Rotation

### The Problem

GGID's auth service connects to LDAP using a bind DN and password loaded from
environment variables (`LDAP_BIND_DN`, `LDAP_BIND_PASSWORD`). If these are
compromised, an attacker can enumerate users, read attributes, and potentially
modify directory entries (depending on ACLs).

### Rotation Challenges

1. **Shared infrastructure:** LDAP is often managed by a separate directory
   services team. Rotation requires cross-team coordination.
2. **Hard restart:** GGID reads LDAP credentials at startup from env vars.
   Changing the password requires a service restart.
3. **No dual-password:** LDAP bind accounts typically have one password.
   Unlike OAuth, there's no dual-secret window.

### Recommended Procedure

```
1. Coordinate with directory services team — schedule maintenance window
2. Directory team sets NEW password on bind account (old still works briefly)
3. Update GGID deployment env vars with new password
4. Rolling restart auth service pods
5. Verify LDAP connectivity via health check
6. Directory team revokes old password
```

**Alternative:** Use a service account per GGID instance. Each instance has its
own bind DN + password. Rotation affects only one instance.

### Go Code: LDAP Credential Rotation with Pre-Test

```go
package ldapproto

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// LDAPCredConfig holds connection parameters for bind credential rotation.
type LDAPCredConfig struct {
	ServerURL    string // e.g. "ldap://ldap.corp.local:389"
	BindDN       string // e.g. "cn=ggid-svc,ou=service-accounts,dc=corp,dc=local"
	BindPassword string
	StartTLS     bool
}

// RotationManager handles LDAP bind credential rotation.
type RotationManager struct {
	current  LDAPCredConfig
	previous LDAPCredConfig // for rollback
}

// TestBind attempts to connect and bind with the given credentials.
// Returns nil if bind succeeds. This should be called BEFORE updating
// the service configuration.
func TestBind(ctx context.Context, cfg LDAPCredConfig, timeout time.Duration) error {
	opts := []ldap.DialOpt{
		ldap.DialWithTimeout(timeout),
	}

	conn, err := ldap.DialURL(cfg.ServerURL, opts...)
	if err != nil {
		return fmt.Errorf("dial %s: %w", cfg.ServerURL, err)
	}
	defer conn.Close()

	if cfg.StartTLS {
		if err := conn.StartTLS(&tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}); err != nil {
			return fmt.Errorf("start TLS: %w", err)
		}
	}

	// Attempt simple bind with the new credentials.
	if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return fmt.Errorf("bind as %s: %w", cfg.BindDN, err)
	}

	// Verify we can search (confirms read permissions).
	searchReq := ldap.NewSearchRequest(
		"dc=corp,dc=local",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 0, false,
		"(objectClass=*)", []string{"dn"}, nil,
	)
	if _, err := conn.Search(searchReq); err != nil {
		return fmt.Errorf("search test: %w", err)
	}

	return nil
}

// Rotate performs the full rotation:
// 1. Test new credentials against LDAP
// 2. If successful, swap current → previous, new → current
// 3. Return new config for deployment update
func (m *RotationManager) Rotate(ctx context.Context, newCreds LDAPCredConfig) error {
	// Pre-test: verify new credentials work before committing.
	if err := TestBind(ctx, newCreds, 10*time.Second); err != nil {
		return fmt.Errorf("pre-test failed, aborting rotation: %w", err)
	}

	// Swap.
	m.previous = m.current
	m.current = newCreds
	return nil
}

// Rollback reverts to the previous credentials if the new ones fail in
// production.
func (m *RotationManager) Rollback() LDAPCredConfig {
	m.current = m.previous
	return m.current
}

// Current returns the active LDAP credentials.
func (m *RotationManager) Current() LDAPCredConfig {
	return m.current
}
```

### GGID Current State

GGID loads LDAP credentials from `os.Getenv("LDAP_BIND_PASSWORD")` at startup
in `services/auth/cmd/main.go` (line 79). There is no runtime rotation, no
dual-password support, and no pre-test validation. A restart is required for
any password change.

---

## 6. PII Encryption Key Rotation

### The Problem

GGID uses AES-256-GCM (`pkg/crypto.AESEncrypt`) to encrypt sensitive data at
rest. The encryption key is derived via SHA-256 from an arbitrary-length input.
Currently there is no key versioning: all encrypted data uses the same key.

If the encryption key is compromised, all encrypted PII is exposed. Rotating
the key requires **re-encrypting all existing data**.

### Key Versioning Strategy

Every encrypted field stores a key version alongside the ciphertext:

```
[v1][nonce][ciphertext]
```

- `v1`: 1-byte key version identifier
- `nonce`: GCM nonce (12 bytes)
- `ciphertext`: AES-GCM encrypted data

When decrypting, the version determines which key to use. When rotating,
re-encrypt old-version data with the new key.

### Go Code: Versioned DEK Rotation

```go
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
)

// KeyVersion is a 1-byte identifier for the encryption key version.
type KeyVersion uint8

// KeyEntry holds a single DEK and its version.
type KeyEntry struct {
	Version KeyVersion
	Key     []byte // 32 bytes for AES-256
}

// KeyRing manages multiple key versions for rotation.
type KeyRing struct {
	mu          sync.RWMutex
	keys        map[KeyVersion][]byte
	activeVer   KeyVersion
}

// NewKeyRing creates a key ring with the initial key.
func NewKeyRing(initialKey []byte) *KeyRing {
	kr := &KeyRing{
		keys: make(map[KeyVersion][]byte),
	}
	kr.keys[1] = initialKey
	kr.activeVer = 1
	return kr
}

// AddVersion adds a new key version and makes it active for new encrypts.
func (kr *KeyRing) AddVersion(ver KeyVersion, key []byte) {
	kr.mu.Lock()
	defer kr.mu.Unlock()
	kr.keys[ver] = key
	kr.activeVer = ver
}

// GetKey returns the key for a specific version (for decryption).
func (kr *KeyRing) GetKey(ver KeyVersion) ([]byte, error) {
	kr.mu.RLock()
	defer kr.mu.RUnlock()
	key, ok := kr.keys[ver]
	if !ok {
		return nil, fmt.Errorf("unknown key version %d", ver)
	}
	return key, nil
}

// ActiveVersion returns the current encryption key version.
func (kr *KeyRing) ActiveVersion() KeyVersion {
	kr.mu.RLock()
	defer kr.mu.RUnlock()
	return kr.activeVer
}

// Encrypt encrypts plaintext using the active key version.
// Format: [1-byte version][12-byte nonce][ciphertext]
func (kr *KeyRing) Encrypt(plaintext []byte) ([]byte, error) {
	kr.mu.RLock()
	ver := kr.activeVer
	key := kr.keys[ver]
	kr.mu.RUnlock()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Prepend version byte.
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	result := make([]byte, 1+len(ciphertext))
	result[0] = byte(ver)
	copy(result[1:], ciphertext)
	return result, nil
}

// Decrypt decrypts data encrypted with any known key version.
func (kr *KeyRing) Decrypt(data []byte) ([]byte, error) {
	if len(data) < 1 {
		return nil, errors.New("ciphertext too short")
	}
	ver := KeyVersion(data[0])

	key, err := kr.GetKey(ver)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	ciphertext := data[1:]
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short after version byte")
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

// NeedsReEncryption returns true if data was encrypted with a
// non-active key version.
func (kr *KeyRing) NeedsReEncryption(data []byte) bool {
	if len(data) < 1 {
		return false
	}
	return KeyVersion(data[0]) != kr.ActiveVersion()
}

// ReEncrypt decrypts with the old key version and re-encrypts
// with the active key version.
func (kr *KeyRing) ReEncrypt(data []byte) ([]byte, error) {
	plaintext, err := kr.Decrypt(data)
	if err != nil {
		return nil, fmt.Errorf("decrypt for re-encryption: %w", err)
	}
	return kr.Encrypt(plaintext)
}
```

### Background Re-Encryption Job

```go
// ReEncryptJob scans a table for rows encrypted with old key versions
// and re-encrypts them with the active version. Runs in batches to
// avoid locking the table.
type ReEncryptJob struct {
	KeyRing   *KeyRing
	BatchSize int
}

// Run iterates over encrypted columns and re-encrypts stale rows.
// Example for a "user_pii" table with an "ssn_encrypted" column:
//
//	query := `SELECT id, ssn_encrypted FROM user_pii
//	          WHERE LEFT(ssn_encrypted, 1) != $1
//	          LIMIT $2`
//	job.Run(ctx, query, "user_pii", "id", "ssn_encrypted")
func (j *ReEncryptJob) Run(ctx context.Context, db Queryer, table, idCol, encCol string) (int, error) {
	activeByte := make([]byte, 1)
	activeByte[0] = byte(j.KeyRing.ActiveVersion())

	totalReEncrypted := 0
	for {
		query := fmt.Sprintf(
			`SELECT %s, %s FROM %s WHERE %s != $1 LIMIT $2`,
			idCol, encCol, table, encCol)

		rows, err := db.QueryContext(ctx, query, activeByte, j.BatchSize)
		if err != nil {
			return totalReEncrypted, err
		}

		type row struct {
			ID  string
			Enc []byte
		}
		var batch []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.ID, &r.Enc); err != nil {
				rows.Close()
				return totalReEncrypted, err
			}
			batch = append(batch, r)
		}
		rows.Close()

		if len(batch) == 0 {
			break // all rows re-encrypted
		}

		for _, r := range batch {
			reEncrypted, err := j.KeyRing.ReEncrypt(r.Enc)
			if err != nil {
				return totalReEncrypted, fmt.Errorf("re-encrypt %s: %w", r.ID, err)
			}
			updateQuery := fmt.Sprintf(
				`UPDATE %s SET %s = $1 WHERE %s = $2`,
				table, encCol, idCol)
			if _, err := db.ExecContext(ctx, updateQuery, reEncrypted, r.ID); err != nil {
				return totalReEncrypted, err
			}
			totalReEncrypted++
		}
	}
	return totalReEncrypted, nil
}

type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error)
}

type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
}

type Result interface{}
```

---

## 7. Password Pepper Rotation

### The Problem

GGID uses an optional HMAC-SHA256 pepper (`pkg/crypto.SetPepper`) applied
before Argon2id hashing. The pepper is a server-side secret that protects
against rainbow table attacks if the database is compromised without the
application server.

Rotating the pepper is uniquely difficult: every password hash in the database
was computed with the old pepper. Changing the pepper invalidates all hashes.
There is no way to re-hash passwords without knowing the plaintext — which is
exactly what we don't have.

### Gradual Rotation Strategy

The solution is **lazy re-hashing on login**:

```
1. Store BOTH old and new pepper values
2. On login:
   a. Try verifying with NEW pepper → if success, done
   b. If fail, try OLD pepper → if success, re-hash with NEW pepper, update DB
   c. If both fail, password incorrect
3. After all users have logged in at least once (track via a flag), remove old pepper
4. For users who never log in, force password reset
```

This is transparent to users — they don't notice the rotation. The cost is
a brief period of dual-verifications on every login until all hashes are migrated.

### Go Code: Pepper Rotation

```go
package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"sync"
)

// PepperRotator manages dual-pepper rotation for password hashing.
type PepperRotator struct {
	mu           sync.RWMutex
	newPepper    []byte // active pepper for new hashes
	oldPepper    []byte // previous pepper for legacy verification
	oldPepperSet bool
}

// NewPepperRotator creates a rotator with a single pepper.
func NewPepperRotator(pepper string) *PepperRotator {
	pr := &PepperRotator{}
	if pepper != "" {
		pr.newPepper = []byte(pepper)
	}
	return pr
}

// BeginRotation sets a new active pepper and moves the current one
// to oldPepper for legacy verification.
func (pr *PepperRotator) BeginRotation(newPepper string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.oldPepper = pr.newPepper
	pr.oldPepperSet = len(pr.oldPepper) > 0
	pr.newPepper = []byte(newPepper)
}

// CompleteRotation removes the old pepper. Call only after confirming
// all users have been re-hashed.
func (pr *PepperRotator) CompleteRotation() {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.oldPepper = nil
	pr.oldPepperSet = false
}

// ApplyForNewHash applies the NEW pepper for new password hashes.
func (pr *PepperRotator) ApplyForNewHash(password string) []byte {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return applyPepperKey(password, pr.newPepper)
}

// VerifyWithRotation tries the new pepper first, then the old one.
// Returns (matched, usedOldPepper). If usedOldPepper is true, the
// caller should re-hash the password with the new pepper.
func (pr *PepperRotator) VerifyWithRotation(
	password, encoded string,
	verifyFn func(pepperedPw []byte, encoded string) (bool, error),
) (matched bool, usedOldPepper bool, err error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	// Try new pepper first.
	newPeppered := applyPepperKey(password, pr.newPepper)
	ok, err := verifyFn(newPeppered, encoded)
	if err != nil {
		return false, false, err
	}
	if ok {
		return true, false, nil
	}

	// Try old pepper.
	if pr.oldPepperSet {
		oldPeppered := applyPepperKey(password, pr.oldPepper)
		ok, err := verifyFn(oldPeppered, encoded)
		if err != nil {
			return false, false, err
		}
		if ok {
			return true, true, nil
		}
	}

	return false, false, nil
}

// HasOldPepper returns true if legacy verification is still active.
func (pr *PepperRotator) HasOldPepper() bool {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.oldPepperSet
}

func applyPepperKey(password string, pepper []byte) []byte {
	pw := []byte(password)
	if len(pepper) > 0 {
		mac := hmac.New(sha256.New, pepper)
		mac.Write(pw)
		return mac.Sum(nil)
	}
	return pw
}
```

### Usage in Login Flow

```go
// Example: integrating PepperRotator into the auth service login handler.
func (s *AuthService) Login(ctx context.Context, username, password string) (*TokenPair, error) {
	cred, err := s.credRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// Verify with pepper rotation support.
	matched, usedOldPepper, err := s.pepperRotator.VerifyWithRotation(
		password, cred.PasswordHash,
		func(pepperedPw []byte, encoded string) (bool, error) {
			// Call the existing argon2id verification with the peppered input.
			return verifyArgon2id(pepperedPw, encoded)
		},
	)
	if err != nil || !matched {
		return nil, ErrInvalidCredentials
	}

	// If verified with old pepper, re-hash with new pepper.
	if usedOldPepper {
		newHash, err := hashWithPepper(password, s.pepperRotator)
		if err != nil {
			// Log but don't fail the login.
			s.logger.Warn("failed to re-hash password during pepper rotation",
				"username", username, "error", err)
		} else {
			_ = s.credRepo.UpdatePasswordHash(ctx, cred.UserID, newHash)
		}
	}

	return s.issueTokens(ctx, cred.UserID)
}
```

### Tracking Migration Progress

Add a `pepper_version` column to the `credentials` table:

```sql
ALTER TABLE credentials ADD COLUMN pepper_version SMALLINT DEFAULT 1;
```

After re-hashing on login, set `pepper_version = 2`. When
`SELECT COUNT(*) FROM credentials WHERE pepper_version < 2` reaches zero,
the old pepper can be safely removed.

### GGID Current State

GGID's `crypto.SetPepper` (line 37 in `pkg/crypto/crypto.go`) sets a single
global pepper variable. There is no rotation support, no dual-pepper, and no
versioning. The `docs/tech-debt.md` file notes "No password pepper" as a
known gap — the infrastructure exists but is not wired into the auth service
main.go.

---

## 8. Unified Rotation Scheduler

### The Problem

Credential rotation is a multi-disciplinary task spanning database, LDAP,
SAML, OAuth, and encryption. Without a central scheduler, rotations are ad-hoc,
forgotten, or discovered during audits when credentials are years old.

### Design

A unified rotation scheduler tracks every credential, its rotation interval,
last rotation timestamp, and next-due date. It generates alerts when rotations
are overdue and can trigger automated rotations where possible.

### Go Code: Unified Rotation Scheduler

```go
package rotation

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// CredentialType identifies the kind of credential.
type CredentialType string

const (
	CredTypeDatabase     CredentialType = "database_password"
	CredTypeLDAPBind     CredentialType = "ldap_bind_password"
	CredTypeSAML         CredentialType = "saml_signing_cert"
	CredTypeOAuthSecret  CredentialType = "oauth_client_secret"
	CredTypePIIKey       CredentialType = "pii_encryption_key"
	CredTypePepper       CredentialType = "password_pepper"
	CredTypeAPIKey       CredentialType = "api_key"
	CredTypeRedis        CredentialType = "redis_password"
)

// CredentialRecord tracks a single credential's rotation lifecycle.
type CredentialRecord struct {
	ID            string         // unique identifier (e.g., "auth-db", "ldap-bind")
	Name          string         // human-readable name
	Type          CredentialType
	RotationInterval time.Duration // e.g., 90 days
	LastRotated   time.Time
	NextRotation  time.Time       // LastRotated + RotationInterval
	AutoRotate    bool            // can this be rotated automatically?
	Owner         string          // team or person responsible
	Metadata      map[string]string // type-specific info (e.g., client_id, db_name)
}

// Rotator is the interface for automated credential rotation.
type Rotator interface {
	Rotate(ctx context.Context, record *CredentialRecord) error
}

// Scheduler manages all credential rotation tracking.
type Scheduler struct {
	mu        sync.RWMutex
	records   map[string]*CredentialRecord
	rotators  map[CredentialType]Rotator
	alertCh   chan Alert
}

// Alert is raised when a credential rotation is overdue.
type Alert struct {
	CredentialID string
	Name         string
	Type         CredentialType
	DaysOverdue  int
	Severity     string // "warning" (7 days), "critical" (30 days)
}

// NewScheduler creates a new rotation scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		records:  make(map[string]*CredentialRecord),
		rotators: make(map[CredentialType]Rotator),
		alertCh:  make(chan Alert, 100),
	}
}

// Register adds a credential to track.
func (s *Scheduler) Register(rec *CredentialRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec.LastRotated.IsZero() {
		rec.LastRotated = time.Now()
	}
	rec.NextRotation = rec.LastRotated.Add(rec.RotationInterval)
	s.records[rec.ID] = rec
}

// RegisterRotator associates a rotator implementation with a credential type.
func (s *Scheduler) RegisterRotator(credType CredentialType, r Rotator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rotators[credType] = r
}

// MarkRotated updates the last-rotated timestamp for a credential.
func (s *Scheduler) MarkRotated(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[id]
	if !ok {
		return
	}
	rec.LastRotated = time.Now()
	rec.NextRotation = rec.LastRotated.Add(rec.RotationInterval)
}

// CheckOverdue scans all credentials and returns overdue ones.
func (s *Scheduler) CheckOverdue() []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	var alerts []Alert
	for _, rec := range s.records {
		if now.Before(rec.NextRotation) {
			continue
		}
		daysOverdue := int(now.Sub(rec.NextRotation).Hours() / 24)
		severity := "warning"
		if daysOverdue >= 30 {
			severity = "critical"
		}
		alerts = append(alerts, Alert{
			CredentialID: rec.ID,
			Name:         rec.Name,
			Type:         rec.Type,
			DaysOverdue:  daysOverdue,
			Severity:     severity,
		})
	}
	return alerts
}

// RunAutoRotate attempts automatic rotation for all credentials
// that are due and have a registered Rotator.
func (s *Scheduler) RunAutoRotate(ctx context.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	for _, rec := range s.records {
		if !rec.AutoRotate || now.Before(rec.NextRotation) {
			continue
		}
		rotator, ok := s.rotators[rec.Type]
		if !ok {
			continue
		}
		if err := rotator.Rotate(ctx, rec); err != nil {
			log.Printf("auto-rotate %s failed: %v", rec.ID, err)
			continue
		}
		s.MarkRotated(rec.ID)
		log.Printf("auto-rotated %s (%s)", rec.ID, rec.Type)
	}
}

// Start launches the scheduler's periodic check loop.
func (s *Scheduler) Start(ctx context.Context, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			alerts := s.CheckOverdue()
			for _, a := range alerts {
				log.Printf("[ROTATION ALERT] %s: %s is %d days overdue (%s)",
					a.Severity, a.Name, a.DaysOverdue, a.Type)
			}
			s.RunAutoRotate(ctx)
		}
	}
}

// Status returns a snapshot of all credential rotation statuses.
func (s *Scheduler) Status() []CredentialRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]CredentialRecord, 0, len(s.records))
	for _, rec := range s.records {
		result = append(result, *rec)
	}
	return result
}

// DaysUntilRotation returns days until the next rotation is due.
func (r *CredentialRecord) DaysUntilRotation() int {
	d := r.NextRotation.Sub(time.Now())
	if d < 0 {
		return 0
	}
	return int(d.Hours() / 24)
}

// IsOverdue returns true if the credential should have been rotated.
func (r *CredentialRecord) IsOverdue() bool {
	return time.Now().After(r.NextRotation)
}

// String returns a human-readable status line.
func (r *CredentialRecord) String() string {
	status := "OK"
	if r.IsOverdue() {
		days := int(time.Since(r.NextRotation).Hours() / 24)
		status = fmt.Sprintf("OVERDUE (%dd)", days)
	} else {
		status = fmt.Sprintf("%dd until due", r.DaysUntilRotation())
	}
	return fmt.Sprintf("%-25s %-20s %s", r.Name, r.Type, status)
}
```

---

## 9. GGID Credential Rotation Audit

### Hardcoded Credentials Found

| Location | Credential | Value | Risk |
|----------|-----------|-------|------|
| `deploy/docker-compose.yaml:8` | PostgreSQL password | `ggid` | Dev only, but pattern copied to prod |
| `deploy/docker-compose.yaml:51,68` | LDAP admin password | `admin123` | Dev only |
| `deploy/docker-compose.yaml:174` | LDAP bind password | `admin123` | Used by auth service in dev |
| `deploy/docker-compose.yaml:235,264,295` | DB password (policy/org/audit) | `ggid` | Dev only |
| `services/auth/internal/conf/conf.go:85` | Default DB URL | `postgres://ggid:ggid@...` | Falls back if env not set |
| `services/identity/cmd/main.go:26` | Default DB URL | `postgres://ggid:ggid@...` | Falls back if env not set |
| `services/oauth/internal/conf/conf.go:48` | Default DB URL | `postgres://ggid:ggid@...` | Falls back if env not set |

### How Secrets Are Loaded Today

**Auth Service** (`services/auth/cmd/main.go`):
- DB: `cfg.Database.URL` from `DATABASE_URL` env var, fallback to hardcoded default
- Redis: `cfg.Redis.Password` from config struct (no env override)
- JWT keys: `JWT_PRIVATE_KEY_PATH` / `JWT_PUBLIC_KEY_PATH` env vars
- LDAP: `LDAP_BIND_PASSWORD` env var (line 79) — raw `os.Getenv`, no secret manager
- Pepper: NOT loaded — `crypto.SetPepper` is never called in main.go

**OAuth Service** (`services/oauth/cmd/main.go`):
- DB: `DATABASE_URL` env var, fallback to hardcoded default
- RSA keys: `OAUTH_PRIVATE_KEY_PATH` / `OAUTH_PUBLIC_KEY_PATH` env vars

**Gateway** (`services/gateway/cmd/main.go`):
- JWT public key: `JWT_PUBLIC_KEY_PATH` env var
- No database, Redis, or LDAP credentials

**Policy / Org / Audit** (`services/*/cmd/main.go`):
- DB: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_DATABASE` env vars
- NATS: `NATS_URL` env var (no auth credentials)

### Existing Rotation Capability

| Credential | Rotation Capability | Status |
|-----------|-------------------|--------|
| JWT signing keys | Manual file rotation + JWKS refresh | Covered in separate docs |
| OAuth client secrets | `RotateClientSecret()` — hard cutover, no overlap | Partial |
| Refresh tokens | Automatic rotation on use (rotation chain in DB) | Complete |
| SAML certs | None — single static cert in SP config | Missing |
| DB passwords | None — requires restart + manual coordination | Missing |
| LDAP bind | None — requires restart + manual coordination | Missing |
| Redis password | None — requires restart | Missing |
| NATS credentials | None — connection URL hardcoded | Missing |
| PII encryption key | None — `AESEncrypt` uses single key, no versioning | Missing |
| Password pepper | None — `SetPepper` never called in main.go | Missing |
| API keys | Manual creation/deletion via console | Manual |

### Production Compose (docker-compose.prod.yaml)

The prod compose file (`deploy/docker-compose.prod.yaml`) is significantly
better — it uses `${POSTGRES_PASSWORD}`, `${REDIS_PASSWORD}`, and
`${NEXTAUTH_SECRET}` with `:?` required-var checks, forcing operators to set
secrets via `.env` file. However, there is still no Vault integration, no
automatic rotation, and no rotation tracking.

---

## 10. Gap Analysis & Recommendations

### Summary of Gaps

| Gap | Severity | Complexity |
|-----|----------|-----------|
| No SAML cert rotation manager | High | Medium |
| OAuth secret rotation is hard-cutover (no overlap) | Medium | Low |
| No DB password rotation automation | Critical | Medium |
| No LDAP bind rotation (requires restart) | High | Medium |
| PII encryption has no key versioning | Critical | High |
| Password pepper not wired (never called) | High | Low |
| No unified rotation scheduler/tracker | High | Medium |
| No Vault or secret manager integration | Medium | High |

### Action Items

**1. Wire password pepper into auth service main.go**
- **Effort:** 1 hour
- **Impact:** Enables server-side secret for password hashing
- **Change:** Add `crypto.SetPepper(os.Getenv("PASSWORD_PEPPER"))` in
  `services/auth/cmd/main.go` after config load
- **Risk:** Low — existing hashes without pepper still verify (pepper is
  backward-compatible; without pepper, `applyPepper` returns identity)

**2. Add dual-secret overlap to OAuth client rotation**
- **Effort:** 1-2 days
- **Impact:** Zero-downtime secret rotation for client developers
- **Change:** Extend `OAuthClient` domain model with `Secrets []ClientSecretEntry`,
  update `RotateClientSecret` to keep old hash with expiry, add
  `PurgeExpiredSecrets` background job
- **Risk:** Medium — schema migration + dual-hash verification logic

**3. Implement PII encryption key versioning**
- **Effort:** 3-5 days
- **Impact:** Enables encryption key rotation without data loss
- **Change:** Replace `crypto.AESEncrypt`/`AESDecrypt` with versioned
  `KeyRing.Encrypt`/`Decrypt`. Add `key_version` to encrypted columns. Build
  background re-encryption job
- **Risk:** High — requires schema migration, backward compatibility for
  existing un-versioned ciphertext, and careful batch processing

**4. Build unified credential rotation dashboard**
- **Effort:** 2-3 days
- **Impact:** Operational visibility into credential hygiene
- **Change:** Implement `rotation.Scheduler`, register all credential types,
  expose `/api/v1/admin/credentials/rotation-status` endpoint, add console page
  showing next-due dates and overdue alerts
- **Risk:** Low — read-only tracking, no credential exposure in API responses

**5. Integrate with HashiCorp Vault for dynamic secrets**
- **Effort:** 1-2 weeks
- **Impact:** Eliminates static DB/LDAP passwords entirely
- **Change:** Add Vault client to service startup, request DB credentials
  from Vault database secrets engine, implement automatic renewal, add Vault
  HA fallback
- **Risk:** High — adds critical infrastructure dependency, requires Vault
  deployment and operational expertise

### Priority Order

```
Phase 1 (Immediate, <1 day):
  → Wire pepper into auth service
  → Document rotation procedures in runbook

Phase 2 (Short-term, 1-2 weeks):
  → Dual-secret OAuth rotation
  → Credential rotation dashboard/scheduler
  → SAML cert rotation manager

Phase 3 (Medium-term, 1-2 months):
  → PII encryption key versioning + re-encryption job
  → DB password rotation automation
  → LDAP bind rotation with pre-test

Phase 4 (Long-term, 2-3 months):
  → Vault integration for dynamic secrets
  → Full automated rotation pipeline
```

### References

- [NIST SP 800-57 Part 1 Rev 5](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-57pt1r5.pdf) — Key management recommendations
- [OAuth 2.0 Security Best Current Practice](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-security-topics) — Client secret rotation
- [SAML 2.0 Metadata](https://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf) — Certificate management
- [HashiCorp Vault Database Secrets Engine](https://developer.hashicorp.com/vault/docs/secrets/databases) — Dynamic DB credentials
- GGID source: `pkg/crypto/crypto.go` — `SetPepper`, `AESEncrypt`, `HashPassword`
- GGID source: `services/oauth/internal/service/oauth_service.go:874` — `RotateClientSecret`
- GGID source: `pkg/saml/sp.go` — `GenerateSPMetadata`, `ServiceProvider`
- GGID docs: [`key-rotation-iam.md`](./key-rotation-iam.md) — JWT key rotation (not duplicated here)
- GGID docs: [`jwks-key-rotation.md`](./jwks-key-rotation.md) — JWKS dual-key strategy (not duplicated here)
