# JWT Signing Key Rotation Lifecycle for IAM Systems

> Research document covering broader key management: lifecycle states, automated
> rollover, multi-tenant keys, HSM integration, compromise response, key hierarchy,
> and JWKS caching strategy for GGID and similar Go-based IAM platforms.
>
> **Companion document:** `jwks-key-rotation.md` covers the dual-key strategy,
> `kid` propagation mechanics, and zero-downtime rotation procedure in detail.
> This document focuses on **lifecycle state machines, multi-tenant isolation,
> HSM-backed signing, key compromise response, and HKDF-based derivation** —
> topics not covered there.

---

## 1. Key Lifecycle States

A signing key is not a static artifact — it progresses through a well-defined
state machine from creation to destruction. Each state restricts what operations
the key may perform.

### State Machine

```
                         ┌──────────────┐
                         │ pre-active   │  Key generated, published to JWKS.
                         │ (published)  │  Not yet trusted for signing.
                         └──────┬───────┘
                                │ activate (scheduled or manual)
                                ▼
                    ┌──────────────────────┐
          ┌────────│       active         │────────┐
          │        │  Signs + verifies    │        │
          │        └──────────────────────┘        │
          │ suspend (compromise suspected)         │ retire (new key activated)
          ▼                                         ▼
  ┌──────────────┐                       ┌──────────────────┐
  │  suspended   │                       │     retired      │
  │  (no sign,   │                       │  (no sign,       │
  │   no verify) │                       │   still verify   │
  └──────┬───────┘                       │  during overlap) │
         │                                 └────────┬─────────┘
         │ reinstate (false alarm)                   │ overlap expires
         │                                           ▼
         │                                 ┌──────────────────┐
         └────────────────────────────────►│    destroyed     │
           destroy (confirmed compromise)   │ (key wiped, kid  │
                                             │  removed from JWKS│
                                             └──────────────────┘
```

### Valid Operations per State

| State | Sign new tokens? | Verify existing tokens? | In JWKS? |
|---|---|---|---|
| `pre-active` | No | Yes (pre-published for cache warm-up) | Yes |
| `active` | **Yes** | Yes | Yes |
| `suspended` | No | No (blocked pending investigation) | No* |
| `retired` | No | **Yes** (overlap window) | Yes |
| `destroyed` | No | No | No |

*Suspended keys are removed from the JWKS so verifiers cannot validate tokens
signed by a potentially compromised key. A separate allow-list controls
suspended-key verification for forensic purposes.

### Minimum Overlap Window

The retired key must remain verifiable until every token signed with it has
expired. The overlap window is:

```
overlap = max(access_token_ttl, refresh_token_ttl, id_token_ttl)
```

For GGID's current configuration:
- Access token TTL: **15 minutes**
- Refresh token TTL: **30 days** (`30 * 24 * time.Hour`)

Therefore the minimum overlap window is **30 days**. Any key retired today must
remain in the JWKS for at least 30 days to avoid breaking active refresh-token
sessions.

> **Critical insight:** Refresh tokens are opaque in GGID (stored as hashes in
> PostgreSQL/Redis), so they are not signed with the RSA key and don't affect
> the overlap window. If GGID migrates to signed refresh tokens (JWT-based),
> the overlap must include the refresh token TTL.

---

## 2. Automated Rollover System

Manual key rotation is error-prone and often skipped until an audit forces it.
An automated system ensures keys rotate on a fixed schedule with zero human
intervention.

### Design

The scheduler runs as a background goroutine within the Auth Service. It uses
a PostgreSQL table to persist key metadata so state survives restarts.

```go
// KeyRecord represents a signing key's lifecycle state in the database.
type KeyRecord struct {
    ID          string    // UUID
    KID         string    // JWKS key ID (fingerprint)
    PublicKey   []byte    // DER-encoded public key
    PrivateKey  []byte    // AES-GCM encrypted private key (envelope encryption)
    State       KeyState  // pre-active, active, retired, destroyed
    CreatedAt   time.Time
    ActivatedAt *time.Time
    RetiredAt   *time.Time
    DestroyAt   time.Time // scheduled destruction time
}

type KeyState string

const (
    KeyStatePreActive KeyState = "pre-active"
    KeyStateActive    KeyState = "active"
    KeyStateRetired   KeyState = "retired"
    KeyStateDestroyed KeyState = "destroyed"
)

// RotationScheduler manages automated key generation and rotation.
type RotationScheduler struct {
	db          *pgxpool.Pool        // DB-backed key store
	signer      *TokenService        // Active signer (hot-reloads on rotation)
	jwksUpdater func(kid string, pub *rsa.PublicKey) // Callback to update JWKS
	interval    time.Duration        // Rotation interval (e.g., 90 days)
	overlap     time.Duration        // Overlap window (e.g., 30 days)
	cacheWarmup time.Duration        // Pre-activation lead time (e.g., 15 min)
}

// Start runs the rotation scheduler in a background goroutine.
func (s *RotationScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.checkAndRotate(ctx); err != nil {
				log.Printf("rotation check failed: %v", err)
			}
		}
	}
}

// checkAndRotate performs a single rotation cycle.
func (s *RotationScheduler) checkAndRotate(ctx context.Context) error {
	active, err := s.getActiveKey(ctx)
	if err != nil {
		return fmt.Errorf("get active key: %w", err)
	}

	// Check if it's time to rotate
	if active == nil || time.Since(active.ActivatedAt.Add(s.interval)) < 0 {
		// Not yet time — also check for pending pre-active promotion
		return s.promotePendingKeys(ctx)
	}

	// Step 1: Generate new key
	newKey, err := s.generateKey(ctx)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	// Step 2: Publish to JWKS (pre-active state)
	if err := s.publishToJWKS(newKey); err != nil {
		return fmt.Errorf("publish to JWKS: %w", err)
	}

	// Step 3: Wait for cache warmup, then flip active
	time.Sleep(s.cacheWarmup)

	if err := s.activateKey(ctx, newKey.ID, active.ID); err != nil {
		return fmt.Errorf("activate key: %w", err)
	}

	// Step 4: Schedule old key retirement + destruction
	if err := s.scheduleRetirement(ctx, active.ID); err != nil {
		return fmt.Errorf("schedule retirement: %w", err)
	}

	log.Printf("key rotation complete: old_kid=%s → new_kid=%s",
		active.KID, newKey.KID)
	return nil
}

// generateKey creates a new RSA key pair and persists it in pre-active state.
func (s *RotationScheduler) generateKey(ctx context.Context) (*KeyRecord, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	kid := computeKID(&privKey.PublicKey)

	// Encrypt private key with envelope key (master key from KMS/env)
	privDER := x509.MarshalPKCS1PrivateKey(privKey)
	encryptedPriv, err := crypto.AESEncrypt(privDER, s.envelopeKey())
	if err != nil {
		return nil, fmt.Errorf("encrypt private key: %w", err)
	}

	pubDER, _ := x509.MarshalPKIXPublicKey(&privKey.PublicKey)

	record := &KeyRecord{
		ID:         uuid.New().String(),
		KID:        kid,
		PublicKey:  pubDER,
		PrivateKey: encryptedPriv,
		State:      KeyStatePreActive,
		CreatedAt:  time.Now(),
		DestroyAt:  time.Now().Add(s.interval + s.overlap),
	}

	// Persist to DB
	_, err = s.db.Exec(ctx,
		`INSERT INTO signing_keys (id, kid, public_key, private_key, state, created_at, destroy_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		record.ID, record.KID, record.PublicKey, record.PrivateKey,
		record.State, record.CreatedAt, record.DestroyAt)
	if err != nil {
		return nil, fmt.Errorf("persist key: %w", err)
	}

	return record, nil
}

// activateKey atomically promotes a pre-active key and retires the old one.
func (s *RotationScheduler) activateKey(ctx context.Context, newID, oldID string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// Promote new key to active
	_, err = tx.Exec(ctx,
		`UPDATE signing_keys SET state = $1, activated_at = $2 WHERE id = $3`,
		KeyStateActive, now, newID)
	if err != nil {
		return fmt.Errorf("activate new key: %w", err)
	}

	// Retire old key
	_, err = tx.Exec(ctx,
		`UPDATE signing_keys SET state = $1, retired_at = $2 WHERE id = $3`,
		KeyStateRetired, now, oldID)
	if err != nil {
		return fmt.Errorf("retire old key: %w", err)
	}

	return tx.Commit(ctx)
}

// promotePendingKeys checks for keys whose destruction time has passed.
func (s *RotationScheduler) promotePendingKeys(ctx context.Context) error {
	// Destroy keys past their destroy_at time
	_, err := s.db.Exec(ctx,
		`UPDATE signing_keys
		 SET state = $1, private_key = NULL
		 WHERE state = $2 AND destroy_at < NOW()`,
		KeyStateDestroyed, KeyStateRetired)
	if err != nil {
		return fmt.Errorf("destroy expired keys: %w", err)
	}

	// Remove destroyed keys from JWKS
	destroyed, err := s.db.Query(ctx,
		`SELECT kid FROM signing_keys WHERE state = $1`, KeyStateDestroyed)
	if err != nil {
		return err
	}
	defer destroyed.Close()

	for destroyed.Next() {
		var kid string
		if err := destroyed.Scan(&kid); err == nil {
			s.jwksUpdater(kid, nil) // nil = remove from JWKS
		}
	}

	return nil
}
```

### Key ID Uniqueness

The `kid` is a fingerprint of the public key (SHA-256, first 8 bytes). Because
RSA key generation uses 2048-bit random primes, the probability of a `kid`
collision across rotations is negligible (~2^-64). However, the scheduler
should verify uniqueness:

```go
func (s *RotationScheduler) ensureUniqueKID(ctx context.Context, kid string) error {
	var count int
	err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM signing_keys WHERE kid = $1`, kid).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("kid collision detected: %s (improbable — regenerate)", kid)
	}
	return nil
}
```

### Rotation Interval Guidelines

| Interval | Use Case | Security Posture |
|---|---|---|
| 90 days | Compliance default (PCI DSS 3.5.2) | Standard |
| 30 days | High-security (SOC 2 Type II) | Elevated |
| 7 days | Regulated finance / healthcare | High |
| 24 hours | Suspected threat actor activity | Maximum |
| On-demand | Key compromise (bypass schedule) | Emergency |

---

## 3. Multi-Tenant Key Management

GGID is a multi-tenant IAM platform. The fundamental question: **should each
tenant have its own signing key, or should all tenants share one?**

### Per-Tenant vs Shared Key Tradeoffs

| Factor | Per-Tenant Keys | Shared Key |
|---|---|---|
| **Blast radius** | One tenant compromise = one key | One compromise = all tenants |
| **JWKS complexity** | N endpoints or parameterized endpoint | Single endpoint |
| **Operational cost** | N keys to rotate, store, audit | One rotation cycle |
| **Token portability** | Tenant-bound (forged token can't cross) | Cross-tenant forgery possible |
| **Memory footprint** | O(N) keys in gateway cache | O(1) |
| **Compliance** | Easier per-tenant audit | Shared audit trail |

### Per-Tenant JWKS Endpoints

```
GET /.well-known/jwks.json                    → shared key (default tenant)
GET /{tenant}/.well-known/jwks.json           → tenant-specific key
GET /.well-known/jwks.json?tenant={tenant_id} → tenant-specific (query param)
```

The gateway resolves the tenant from the `X-Tenant-ID` header or token claims,
then fetches the correct JWKS.

### Tenant-Aware Key Manager

```go
// TenantKeyManager manages per-tenant signing keys.
type TenantKeyManager struct {
	db     *pgxpool.Pool
	cache  map[string]*TenantKeys // tenant_id → keys
	mu     sync.RWMutex
	kepdek []byte // KEK for decrypting tenant private keys
}

// TenantKeys holds the active and retired keys for a single tenant.
type TenantKeys struct {
	Active  *rsa.PrivateKey
	ActiveKID string
	Retired map[string]*rsa.PublicKey // kid → public key (for overlap verification)
}

// GetSigningKey returns the active private key for a tenant.
func (m *TenantKeyManager) GetSigningKey(tenantID string) (*rsa.PrivateKey, string, error) {
	m.mu.RLock()
	keys, ok := m.cache[tenantID]
	m.mu.RUnlock()

	if ok {
		return keys.Active, keys.ActiveKID, nil
	}

	// Cache miss — load from DB
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if keys, ok := m.cache[tenantID]; ok {
		return keys.Active, keys.ActiveKID, nil
	}

	keys, err := m.loadTenantKeys(tenantID)
	if err != nil {
		return nil, "", err
	}

	m.cache[tenantID] = keys
	return keys.Active, keys.ActiveKID, nil
}

// GetVerificationKey returns the public key for a given tenant + kid.
func (m *TenantKeyManager) GetVerificationKey(tenantID, kid string) (*rsa.PublicKey, error) {
	m.mu.RLock()
	keys, ok := m.cache[tenantID]
	m.mu.RUnlock()

	if !ok {
		// Load from DB
		m.mu.Lock()
		keys, _ = m.loadTenantKeys(tenantID)
		m.cache[tenantID] = keys
		m.mu.Unlock()
	}

	// Check active key
	if keys.ActiveKID == kid {
		return &keys.Active.PublicKey, nil
	}

	// Check retired keys (overlap window)
	if pub, ok := keys.Retired[kid]; ok {
		return pub, nil
	}

	return nil, fmt.Errorf("key not found for tenant=%s kid=%s", tenantID, kid)
}

// RotateTenant rotates the signing key for a specific tenant.
func (m *TenantKeyManager) RotateTenant(ctx context.Context, tenantID string) error {
	newPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	newKID := computeKID(&newPriv.PublicKey)

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Retire current active key
	_, err = tx.Exec(ctx,
		`UPDATE tenant_signing_keys SET state = 'retired', retired_at = NOW()
		 WHERE tenant_id = $1 AND state = 'active'`, tenantID)
	if err != nil {
		return err
	}

	// Insert new active key
	encryptedPriv, _ := crypto.AESEncrypt(
		x509.MarshalPKCS1PrivateKey(newPriv), m.kepdek)
	pubDER, _ := x509.MarshalPKIXPublicKey(&newPriv.PublicKey)

	_, err = tx.Exec(ctx,
		`INSERT INTO tenant_signing_keys (id, tenant_id, kid, public_key, private_key, state, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'active', NOW())`,
		uuid.New().String(), tenantID, newKID, pubDER, encryptedPriv)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Invalidate cache for this tenant
	m.mu.Lock()
	delete(m.cache, tenantID)
	m.mu.Unlock()

	return nil
}
```

### Tenant-Specific Rotation Schedules

Enterprise tenants may require more frequent rotation than community tenants.
The rotation scheduler queries a per-tenant interval:

```sql
CREATE TABLE tenant_rotation_policy (
    tenant_id     UUID PRIMARY KEY REFERENCES tenants(id),
    interval_days INT NOT NULL DEFAULT 90,
    overlap_days  INT NOT NULL DEFAULT 30,
    auto_rotate   BOOLEAN NOT NULL DEFAULT true
);
```

---

## 4. HSM-Backed Key Storage

### Why HSM?

A Hardware Security Module (HSM) is a tamper-resistant device that generates
and stores cryptographic keys. The private key **never leaves the HSM** — all
signing operations happen inside the hardware. Benefits:

- **Key extraction impossible:** Private keys cannot be exfiltrated even with
  full server compromise.
- **Tamper resistance:** Physical tampering destroys keys.
- **Certified:** FIPS 140-2 Level 3 / Common Criteria EAL4+.
- **Audit trail:** All key usage is logged inside the HSM.

For IAM systems handling authentication tokens, HSM-backed signing is the gold
standard for production and a hard requirement for FedRAMP/PCI environments.

### PKCS#11 Interface

PKCS#11 (Cryptoki) is the standard API for interacting with HSMs. Most HSMs
(AWS CloudHSM, Thales Luna, YubiHSM, Utimaco) provide a PKCS#11 library.

### SoftHSM for Testing

[SoftHSM2](https://www.opendnssec.org/softhsm/) is a software implementation of
PKCS#11 for development and testing. It stores keys on disk but provides the
same API as a real HSM.

```bash
# Install SoftHSM2
brew install softhsm         # macOS
apt install softhsm2         # Debian/Ubuntu

# Create a token
softhsm2-util --init-token --slot 0 --label "ggid-dev" --pin 1234 --so-pin 5678

# Generate an RSA key inside the token (via pkcs11-tool)
pkcs11-tool --module /usr/lib/softhsm/libsofthsm2.so \
  --login --pin 1234 \
  --keypairgen --key-type rsa:2048 \
  --label "jwt-signing-key" --id 01
```

### Go HSM Signing with PKCS#11

```go
import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"io"
	"github.com/ThalesGroup/softhsm/internal/pkcs11"
)

// HSMSigner implements crypto.Signer using an HSM-backed key.
type HSMSigner struct {
	ctx       *pkcs11.Ctx
	session   pkcs11.SessionHandle
	privKey   pkcs11.ObjectHandle
	pubKey    *rsa.PublicKey // Public key loaded from HSM (extractable)
	keyLabel  string
}

// NewHSMSigner opens a PKCS#11 session and loads the signing key.
func NewHSMSigner(libPath, tokenLabel, pin, keyLabel string) (*HSMSigner, error) {
	ctx := pkcs11.New(libPath)
	if ctx == nil {
		return nil, fmt.Errorf("failed to load PKCS#11 library: %s", libPath)
	}
	if err := ctx.Initialize(); err != nil {
		return nil, fmt.Errorf("PKCS#11 initialize: %w", err)
	}

	// Find the token slot
	slots, err := ctx.GetSlotList(true)
	if err != nil {
		return nil, err
	}

	var slot uint
	for _, s := range slots {
		info, err := ctx.GetTokenInfo(s)
		if err != nil {
			continue
		}
		if info.Label == tokenLabel || strings.TrimSpace(info.Label) == tokenLabel {
			slot = s
			break
		}
	}

	// Open RW session
	session, err := ctx.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		return nil, fmt.Errorf("open session: %w", err)
	}

	if err := ctx.Login(session, pkcs11.CKU_USER, pin); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	// Find the private key object by label
	privTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, keyLabel),
	}
	if err := ctx.FindObjectsInit(session, privTemplate); err != nil {
		return nil, err
	}
	objs, _, err := ctx.FindObjects(session, 1)
	if err != nil {
		return nil, err
	}
	ctx.FindObjectsFinal(session)
	if len(objs) == 0 {
		return nil, fmt.Errorf("private key '%s' not found in HSM", keyLabel)
	}

	// Extract public key for JWKS publication
	pubKey, err := extractRSAPublicKey(ctx, session, keyLabel)
	if err != nil {
		return nil, fmt.Errorf("extract public key: %w", err)
	}

	return &HSMSigner{
		ctx:      ctx,
		session:  session,
		privKey:  objs[0],
		pubKey:   pubKey,
		keyLabel: keyLabel,
	}, nil
}

// Public returns the RSA public key (safe to export from HSM).
func (h *HSMSigner) Public() crypto.PublicKey {
	return h.pubKey
}

// Sign performs the signature inside the HSM — the private key never exits.
func (h *HSMSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	hashed := digest
	if len(digest) != sha256.Size {
		// If caller passes pre-hashed data of wrong size, hash it
		h := sha256.Sum256(digest)
		hashed = h[:]
	}

	// PKCS#11 CKM_RSA_PKCS mechanism wraps the DigestInfo + raw RSA signing
	mechanism := []*pkcs11.Mechanism{
		pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS, nil),
	}

	// Prepare the DigestInfo prefix for SHA-256 (PKCS#1 v1.5)
	digestInfo := append(sha256DigestInfoPrefix, hashed...)

	if err := h.ctx.SignInit(h.session, mechanism, h.privKey); err != nil {
		return nil, fmt.Errorf("HSM SignInit: %w", err)
	}

	sig, err := h.ctx.Sign(h.session, digestInfo)
	if err != nil {
		return nil, fmt.Errorf("HSM Sign: %w", err)
	}

	return sig, nil
}

// SHA-256 DigestInfo prefix per RFC 3447
var sha256DigestInfoPrefix = []byte{
	0x30, 0x31, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86,
	0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01, 0x05,
	0x00, 0x04, 0x20,
}

// HSMKeyProvider wraps an HSMSigner to satisfy GGID's KeyProvider interface.
type HSMKeyProvider struct {
	signer *HSMSigner
	kid    string
}

func (p *HSMKeyProvider) PrivateKey() *rsa.PrivateKey {
	// Cannot extract — return nil, callers must use Sign()
	return nil
}

func (p *HSMKeyProvider) PublicKey() *rsa.PublicKey {
	return p.signer.pubKey
}

func (p *HSMKeyProvider) KeyID() string {
	return p.kid
}

// SignJWT signs a JWT using the HSM-backed signer.
func (p *HSMKeyProvider) SignJWT(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = p.kid
	// jwt-go calls signer.Sign() internally — routes to HSM
	return token.SignedString(p.signer)
}
```

### Performance Considerations

| Operation | Software (Go RSA) | HSM (CloudHSM) | HSM (YubiHSM USB) |
|---|---|---|---|
| Sign (2048-bit) | ~0.3 ms | 2-5 ms (network) | 0.5-1 ms (local) |
| Batch sign (1000) | ~300 ms | 2-5 seconds | 500ms-1s |
| Throughput | ~3000/s | ~200-500/s | ~1000-2000/s |

**Mitigation strategies for HSM latency:**
1. **Session pooling:** Open multiple PKCS#11 sessions for parallel signing.
2. **Batch signing:** Sign multiple tokens in a single HSM session.
3. **Hybrid approach:** Use software signing for high-volume access tokens,
   HSM for long-lived refresh tokens or OIDC ID tokens.
4. **Local HSM:** YubiHSM 2 (~$650) provides local HSM performance without
   network latency.

---

## 5. Key Compromise Response

When a signing key is suspected compromised, the response must be immediate,
coordinated, and leave no gap where forged tokens are accepted.

### Detection Signals

| Signal | Source | Severity |
|---|---|---|
| Private key file found in public repo / paste site | Secret scanning (GitGuardian, TruffleHog) | Critical |
| Valid JWT with unknown `kid` | Gateway anomaly detection | High |
| Token usage from impossible travel (geo + timing) | Audit log analysis | High |
| Private key used outside expected service | HSM audit log | Critical |
| Internal threat intel report | Threat feed | Variable |
| Unusually high token issuance rate | Metrics anomaly | Medium |

### Emergency Rotation Procedure

```
T+0:00  Detection confirmed
  │
  ├──► 1. Immediately set compromised key to `suspended` state
  │      (Remove from JWKS — all verification of that kid fails)
  │
  ├──► 2. Generate new key pair → activate immediately
  │      (No pre-activation / cache warmup — accept brief verification gap)
  │
  ├──► 3. Force JWKS cache flush across all gateway instances
  │      (Redis pub/sub: jwks:invalidate → all gateways refresh)
  │
  ├──► 4. Revoke all active sessions for affected tenants
  │      (Session table bulk-revoke; refresh tokens invalidated)
  │
  ├──► 5. Force token re-issuance
  │      (All clients get 401 → must re-authenticate)
  │
  └──► 6. Audit log: key_compromised { kid, detection_source, timestamp, actor }
```

### Emergency Rotation Code

```go
// EmergencyRotator handles key compromise response.
type EmergencyRotator struct {
	keyMgr     *TenantKeyManager
	jwksCache  *JWKSCacheManager
	gatewayPub *redis.Client  // Redis pub/sub for cache invalidation
	sessionDB  *pgxpool.Pool  // For bulk session revocation
}

// EmergencyRotate performs immediate key rotation on compromise.
// All tokens signed by the compromised key are instantly invalid.
func (e *EmergencyRotator) EmergencyRotate(
	ctx context.Context,
	tenantID string,
	compromisedKID string,
	reason string,
) (*RotationResult, error) {
	start := time.Now()
	result := &RotationResult{}

	// Step 1: Generate new key immediately
	newKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate emergency key: %w", err)
	}
	newKID := computeKID(&newKey.PublicKey)
	result.NewKID = newKID

	// Step 2: Atomic DB transaction — suspend old, activate new
	tx, err := e.sessionDB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Suspend compromised key (not retired — no overlap, no verification)
	_, err = tx.Exec(ctx,
		`UPDATE tenant_signing_keys
		 SET state = 'suspended', retired_at = NOW()
		 WHERE tenant_id = $1 AND kid = $2`,
		tenantID, compromisedKID)
	if err != nil {
		return nil, fmt.Errorf("suspend compromised key: %w", err)
	}

	// Activate new key
	encryptedPriv, _ := crypto.AESEncrypt(
		x509.MarshalPKCS1PrivateKey(newKey), e.keyMgr.kepdek)
	pubDER, _ := x509.MarshalPKIXPublicKey(&newKey.PublicKey)

	_, err = tx.Exec(ctx,
		`INSERT INTO tenant_signing_keys
		 (id, tenant_id, kid, public_key, private_key, state, created_at, activated_at)
		 VALUES ($1, $2, $3, $4, $5, 'active', NOW(), NOW())`,
		uuid.New().String(), tenantID, newKID, pubDER, encryptedPriv)
	if err != nil {
		return nil, fmt.Errorf("insert emergency key: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Step 3: Flush JWKS cache across all gateway instances
	if err := e.flushGatewayCache(ctx, tenantID); err != nil {
		log.Printf("WARNING: gateway cache flush failed: %v", err)
	}

	// Step 4: Revoke all active sessions for this tenant
	revoked, err := e.revokeAllSessions(ctx, tenantID)
	if err != nil {
		log.Printf("WARNING: session revocation failed: %v", err)
	}
	result.SessionsRevoked = revoked

	// Step 5: Audit log
	result.Duration = time.Since(start)
	log.Printf("EMERGENCY ROTATION: tenant=%s old_kid=%s new_kid=%s sessions_revoked=%d reason=%s duration=%s",
		tenantID, compromisedKID, newKID, revoked, reason, result.Duration)

	return result, nil
}

// flushGatewayCache publishes a cache invalidation signal to all gateways.
func (e *EmergencyRotator) flushGatewayCache(ctx context.Context, tenantID string) error {
	msg, _ := json.Marshal(map[string]string{
		"action":    "invalidate",
		"tenant_id": tenantID,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})
	return e.gatewayPub.Publish(ctx, "jwks:invalidate", msg).Err()
}

// revokeAllSessions bulk-revokes all sessions for a tenant.
func (e *EmergencyRotator) revokeAllSessions(ctx context.Context, tenantID string) (int64, error) {
	tag, err := e.sessionDB.Exec(ctx,
		`UPDATE sessions SET revoked = true, revoked_at = NOW()
		 WHERE tenant_id = $1 AND revoked = false`,
		tenantID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

type RotationResult struct {
	NewKID          string
	SessionsRevoked int64
	Duration        time.Duration
}
```

### Token Invalidation Cascade

```
DB Transaction (atomic)
    │
    ├── old key → suspended (removed from JWKS)
    ├── new key → active (added to JWKS)
    │
    ▼
Redis Pub/Sub: jwks:invalidate
    │
    ├── Gateway Instance 1 → flush JWKS cache → fetch new JWKS
    ├── Gateway Instance 2 → flush JWKS cache → fetch new JWKS
    └── Gateway Instance N → flush JWKS cache → fetch new JWKS
    │
    ▼
Sessions table: bulk revoke
    │
    ├── Redis: delete session:{id} keys
    └── PostgreSQL: SET revoked = true
    │
    ▼
Result: All old tokens → 401 (kid not in JWKS)
        All sessions → must re-authenticate
        New tokens → signed with new key → verified by new JWKS
```

---

## 6. Key Derivation for Tenant-Specific Signing

### Concept

Instead of generating and storing a unique RSA key pair per tenant, derive a
deterministic signing key from a **master key** using HKDF (HMAC-based Key
Derivation Function, RFC 5869). The tenant's key is derived on-demand from
`HKDF(master_key, tenant_id)` and never persisted.

**Important caveat:** HKDF produces symmetric keys. For asymmetric signing,
we derive a deterministic seed and use it to generate an RSA or Ed25519 key.
The seed is reproducible, so the same key is always derived for the same tenant.

### Benefits

- **Zero per-tenant storage:** No database table needed for tenant keys.
- **Instant provisioning:** New tenant gets a key immediately — no keygen.
- **Instant rotation:** Rotating the master key re-derives all tenant keys.
- **Simplified HSM usage:** Only the master key resides in HSM.

### Risks

- **Master key compromise = total compromise:** All tenant keys exposed.
  Mitigation: HSM-back the master key, envelope-encrypt the derived seeds.
- **No key revocation per tenant:** Cannot revoke one tenant's key without
  rotating the master (which rotates all tenants).
- **Deterministic = non-forward-secret:** The same input always produces the
  same key. If master key leaks, all historical keys are compromised.

### HKDF-Based Tenant Key Derivation

```go
import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"golang.org/x/crypto/hkdf"
)

// DerivedKeyManager generates per-tenant signing keys from a master seed.
// Uses HKDF to derive a 32-byte Ed25519 seed per tenant.
type DerivedKeyManager struct {
	masterKey []byte // 32 bytes, stored in HSM or KMS
	cache     map[string]ed25519.PrivateKey
	mu        sync.RWMutex
}

// NewDerivedKeyManager creates a manager with the given master key.
func NewDerivedKeyManager(masterKey []byte) *DerivedKeyManager {
	return &DerivedKeyManager{
		masterKey: masterKey,
		cache:     make(map[string]ed25519.PrivateKey),
	}
}

// deriveTenantSeed uses HKDF to derive a 32-byte Ed25519 seed for a tenant.
// The seed is deterministic: same master + tenant_id always produces the same seed.
func (m *DerivedKeyManager) deriveTenantSeed(tenantID string) ([]byte, error) {
	// HKDF-SHA256: extract-then-expand
	// salt = empty (master key already has high entropy)
	// info = "ggid:tenant-signing:" + tenantID
	info := append([]byte("ggid:tenant-signing:"), []byte(tenantID)...)

	reader := hkdf.New(sha256.New, m.masterKey, nil, info)
	seed := make([]byte, ed25519.SeedSize) // 32 bytes
	if _, err := io.ReadFull(reader, seed); err != nil {
		return nil, fmt.Errorf("HKDF derive: %w", err)
	}
	return seed, nil
}

// GetSigningKey returns the Ed25519 private key for a tenant.
func (m *DerivedKeyManager) GetSigningKey(tenantID string) (ed25519.PrivateKey, string, error) {
	// Check cache
	m.mu.RLock()
	if key, ok := m.cache[tenantID]; ok {
		kid := computeEd25519KID(key.Public())
		m.mu.RUnlock()
		return key, kid, nil
	}
	m.mu.RUnlock()

	// Derive seed
	seed, err := m.deriveTenantSeed(tenantID)
	if err != nil {
		return nil, "", err
	}

	// Ed25519 private key from seed (deterministic)
	privKey := ed25519.NewKeyFromSeed(seed)

	m.mu.Lock()
	m.cache[tenantID] = privKey
	m.mu.Unlock()

	kid := computeEd25519KID(privKey.Public())
	return privKey, kid, nil
}

// RotateMasterKey rotates the master key. All tenant keys are re-derived.
// Old tokens become invalid immediately (no overlap window by default).
func (m *DerivedKeyManager) RotateMasterKey(newMasterKey []byte) {
	m.mu.Lock()
	m.masterKey = newMasterKey
	m.cache = make(map[string]ed25519.PrivateKey) // Clear cache
	m.mu.Unlock()
}

func computeEd25519KID(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return hex.EncodeToString(h[:8])
}
```

### Key Hierarchy Architecture

```
                    ┌─────────────────────────┐
                    │   Root Master Key       │  ← HSM / KMS (never exported)
                    │   (rotated annually)    │
                    └───────────┬─────────────┘
                                │ HKDF
                    ┌───────────┴─────────────┐
                    │                         │
            ┌───────▼────────┐       ┌────────▼────────┐
            │ Tenant A Key   │       │ Tenant B Key    │  ← Derived on-demand
            │ (Ed25519)      │  ...  │ (Ed25519)       │     Never persisted
            └────────────────┘       └─────────────────┘
```

---

## 7. kid Header Propagation and JWKS Caching

### How kid Enables Key Selection

The JWT header contains a `kid` (Key ID) field that tells the verifier which
key was used to sign the token:

```json
{
  "alg": "RS256",
  "kid": "a1b2c3d4e5f6a7b8",
  "typ": "JWT"
}
```

The verifier flow:
1. Parse JWT header, extract `kid`.
2. Look up `kid` in the JWKS cache.
3. If found, verify signature with that public key.
4. If not found, either refresh JWKS or reject.

### JWKS Caching Strategy

| Strategy | Description | Use Case |
|---|---|---|
| **TTL cache** | Cache keys for N minutes, refresh on expiry | Standard |
| **Cache-miss refresh** | On kid not found, force JWKS fetch | Rotation safety net |
| **Pre-emptive refresh** | Refresh before TTL expires | Zero-downtime |
| **Pub/sub invalidation** | Redis pub/sub triggers immediate flush | Emergency rotation |

### Cache Poisoning Prevention

JWKS endpoints are unauthenticated (public keys). An attacker who can MITM the
JWKS fetch can inject their own public key, then forge tokens. Mitigations:

1. **HTTPS only:** JWKS endpoint must use TLS. Gateway rejects HTTP JWKS URLs.
2. **Response signing:** Optionally sign the JWKS response with a separate key.
3. **Key pinning:** Pin expected key fingerprints; reject unknown kids during
   stable periods.
4. **Strict kid validation:** Reject `kid` values that don't match expected
   format (hex, fixed length).
5. **Rate limiting:** Limit JWKS refresh frequency to prevent DoS.

### Cached JWKS Provider with Fallback

GGID's current `JWKSClient.GetKey()` returns an error immediately on cache miss.
This causes verification failures during rotation. A fallback-aware version:

```go
// CachedJWKSProvider wraps JWKSClient with cache-miss fallback.
type CachedJWKSProvider struct {
	jwksURL    string
	keys       map[string]*rsa.PublicKey
	mu         sync.RWMutex
	httpClient *http.Client
	cacheTTL   time.Duration
	lastFetch  time.Time
	refreshing chan struct{} // Dedup concurrent refreshes
}

// NewCachedJWKSProvider creates a provider with the given cache TTL.
func NewCachedJWKSProvider(jwksURL string, cacheTTL time.Duration) *CachedJWKSProvider {
	return &CachedJWKSProvider{
		jwksURL:    jwksURL,
		keys:       make(map[string]*rsa.PublicKey),
		httpClient: &http.Client{Timeout: 5 * time.Second},
		cacheTTL:   cacheTTL,
	}
}

// GetKey returns the RSA public key for the given kid.
// On cache miss, forces an immediate JWKS refresh before rejecting.
func (p *CachedJWKSProvider) GetKey(kid string) (*rsa.PublicKey, error) {
	// Fast path: check cache
	p.mu.RLock()
	if key, ok := p.keys[kid]; ok {
		p.mu.RUnlock()
		return key, nil
	}
	p.mu.RUnlock()

	// Cache miss — force refresh (single-flight to dedup concurrent calls)
	if err := p.forceRefresh(); err != nil {
		return nil, fmt.Errorf("JWKS refresh on cache miss (kid=%s): %w", kid, err)
	}

	// Retry lookup after refresh
	p.mu.RLock()
	defer p.mu.RUnlock()
	if key, ok := p.keys[kid]; ok {
		return key, nil
	}

	return nil, fmt.Errorf("key not found after refresh: kid=%s", kid)
}

// forceRefresh fetches the JWKS and updates the cache. Uses a channel to
// deduplicate concurrent refresh requests.
func (p *CachedJWKSProvider) forceRefresh() error {
	p.mu.Lock()
	// If another goroutine is already refreshing, wait for it
	if p.refreshing != nil {
		ch := p.refreshing
		p.mu.Unlock()
		<-ch // Wait for the in-flight refresh
		return nil
	}
	p.refreshing = make(chan struct{})
	p.mu.Unlock()

	// Perform the fetch
	err := p.doFetch()

	// Signal completion
	p.mu.Lock()
	close(p.refreshing)
	p.refreshing = nil
	p.lastFetch = time.Now()
	p.mu.Unlock()

	return err
}

// doFetch performs the actual HTTP fetch and cache update.
func (p *CachedJWKSProvider) doFetch() error {
	resp, err := p.httpClient.Get(p.jwksURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var jwks struct {
		Keys []struct {
			KTY string `json:"kty"`
			KID string `json:"kid"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		return err
	}

	newKeys := make(map[string]*rsa.PublicKey)
	for _, k := range jwks.Keys {
		if k.KTY != "RSA" || k.Use != "sig" {
			continue
		}
		pub, err := jwkToRSAPublicKey(k.N, k.E)
		if err != nil {
			continue
		}
		newKeys[k.KID] = pub
	}

	p.mu.Lock()
	if len(newKeys) > 0 {
		p.keys = newKeys
	}
	p.mu.Unlock()

	return nil
}
```

### Recommended Cache Configuration

| Parameter | Recommended Value | Rationale |
|---|---|---|
| Cache TTL | 5-15 minutes | Balance freshness vs fetch load |
| Cache-miss refresh | Enabled | Prevents failures during rotation |
| Fetch timeout | 5 seconds | Fast enough for request-path |
| Background refresh | Every cacheTTL | Keep cache warm |
| Pub/sub invalidation | Enabled | For emergency rotation |

---

## 8. GGID Key Rotation Gap Analysis

### Current State

**Auth Service (`services/auth/internal/service/token_service.go`):**
- Loads a **single RSA key** from disk at startup (`loadOrCreatePrivateKey`).
- Key ID = SHA-256 fingerprint of public key (8 bytes, hex-encoded).
- Generates 2048-bit RSA on first run, persists to `configs/rsa_private.pem`.
- No rotation, no state machine, no DB-backed key metadata.
- Private key stored as **plaintext PEM file** on disk (mode 0600).

**OAuth Service (`services/oauth/internal/server/server.go`):**
- `keyProvider` loads the same RSA key pair from disk.
- Serves a JWKS endpoint with a single key.
- No multi-key support — JWKS always serves exactly one key.
- No rotation trigger, no overlap window management.

**Gateway (`services/gateway/internal/middleware/middleware.go`):**
- `JWKSClient` caches public keys in `map[string]*rsa.PublicKey`.
- Background refresh every 15 minutes via `StartRefresh`.
- `GetKey(kid)` returns error immediately on cache miss — **no fallback refresh**.
- No pub/sub cache invalidation.
- Static key fallback to PEM file if JWKS URL fails.
- No multi-tenant key awareness — all tokens verified against one key set.

**pkg/crypto:**
- Provides Argon2id, AES-256-GCM, random token generation.
- No RSA key generation utilities (done inline in token_service.go).
- No HKDF implementation.
- No envelope encryption helper (raw AES used directly).

### What Exists vs What's Missing

| Capability | Status | Location |
|---|---|---|
| RSA key generation | Exists (inline) | `token_service.go:loadOrCreatePrivateKey` |
| kid fingerprint computation | Exists (duplicated) | `token_service.go`, `middleware.go` |
| JWKS endpoint serving | Exists (single key) | `middleware.go:JWKSHandler` |
| JWKS client with TTL refresh | Exists (15 min) | `middleware.go:JWKSClient` |
| **Automated rotation scheduler** | **Missing** | — |
| **DB-backed key state machine** | **Missing** | — |
| **Multi-key JWKS (active + retired)** | **Missing** | JWKS serves 1 key only |
| **Cache-miss JWKS fallback** | **Missing** | GetKey returns error |
| **Pub/sub cache invalidation** | **Missing** | — |
| **Multi-tenant keys** | **Missing** | Single key for all tenants |
| **HSM / PKCS#11 integration** | **Missing** | Keys on disk (PEM) |
| **Key compromise response** | **Missing** | No emergency rotation API |
| **Envelope encryption for private keys** | **Missing** | Plaintext PEM on disk |
| **HKDF key derivation** | **Missing** | — |
| **Audit logging for key events** | **Missing** | — |
| **Key overlap window management** | **Missing** | — |

### Critical Security Concern

GGID stores private signing keys as **plaintext PEM files on disk** with only
filesystem permissions (0600) for protection. Any process running as the same
user, or anyone with container root access, can extract the private key. For
production IAM systems, this is insufficient — envelope encryption or HSM
storage is required.

---

## 9. Gap Analysis & Recommendations

### Priority Action Items

| # | Action | Effort | Priority | Impact |
|---|---|---|---|---|
| 1 | **Implement cache-miss JWKS fallback in gateway** | 2 hours | P0 | Eliminates verification failures during rotation |
| 2 | **Add DB-backed key state machine** | 3 days | P1 | Foundation for all rotation features |
| 3 | **Add automated rotation scheduler** | 2 days | P1 | Compliance (PCI DSS, SOC 2), eliminates manual rotation risk |
| 4 | **Envelope-encrypt private keys at rest** | 1 day | P1 | Mitigates disk-based key theft |
| 5 | **Add PKCS#11 / HSM provider interface** | 3 days | P2 | FIPS compliance, enterprise requirement |

### Detailed Recommendations

**1. Cache-Miss JWKS Fallback (P0, ~2 hours)**

The simplest, highest-impact improvement. Modify `JWKSClient.GetKey()` to
trigger a synchronous JWKS refresh on cache miss, then retry. This prevents
the 15-minute gap where tokens with a new `kid` fail after rotation. The
`CachedJWKSProvider` code in Section 7 provides the implementation.

**2. DB-Backed Key State Machine (P1, ~3 days)**

Create a `signing_keys` table to track key lifecycle states. The Auth Service
and OAuth Service read the active key from the DB instead of a static file.
This enables hot-reloading keys without restart and provides an audit trail
of all key transitions. Schema:

```sql
CREATE TABLE signing_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kid          TEXT NOT NULL UNIQUE,
    tenant_id    UUID,  -- NULL = global key
    public_key   BYTEA NOT NULL,
    private_key  BYTEA NOT NULL,  -- AES-GCM encrypted
    state        TEXT NOT NULL DEFAULT 'pre-active',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    retired_at   TIMESTAMPTZ,
    destroy_at   TIMESTAMPTZ
);
```

**3. Automated Rotation Scheduler (P1, ~2 days)**

Build on the state machine from item 2. The scheduler (Section 2) runs as a
background goroutine in the Auth Service, checks hourly, and performs rotation
when the active key exceeds the configured interval. Include Redis pub/sub
notification to flush gateway caches immediately after rotation. Default
interval: 90 days; overlap: 30 days.

**4. Envelope Encryption (P1, ~1 day)**

Wrap private keys with AES-256-GCM before storing to disk or DB. The envelope
key (KEK) comes from an environment variable, KMS, or HSM. This ensures that
even if the disk or database is compromised, keys remain encrypted. Modify
`writePrivateKey` and `loadOrCreatePrivateKey` to encrypt/decrypt via the
existing `crypto.AESEncrypt` / `crypto.AESDecrypt` functions.

**5. HSM Provider Interface (P2, ~3 days)**

Define a `KeyProvider` interface (already partially exists in OAuth Service's
`domain.KeyProvider`). Implement three providers:
- `FileKeyProvider` (current behavior, for dev/testing)
- `HSMKeyProvider` (PKCS#11, for production)
- `DerivedKeyProvider` (HKDF, for multi-tenant)

Configure via environment variable `KEY_PROVIDER=file|hsm|derived`. Default
to `file` for backward compatibility. HSM requires `PKCS11_LIB_PATH`,
`PKCS11_TOKEN_LABEL`, `PKCS11_PIN`.

### Roadmap Summary

```
Phase 1 (1 week):   Cache-miss fallback + envelope encryption
                     → Eliminates rotation failures, protects keys at rest

Phase 2 (2 weeks):  DB-backed key state machine + automated scheduler
                     → Compliance-ready rotation with audit trail

Phase 3 (1 week):   HSM provider + multi-tenant key derivation
                     → Enterprise-grade key security
```

---

## References

- RFC 7517: JSON Web Key (JWK)
- RFC 7519: JSON Web Token (JWT)
- RFC 5869: HMAC-based Extract-and-Expand Key Derivation Function (HKDF)
- PKCS#11 v3.0: Cryptographic Token Interface Standard
- NIST SP 800-57: Recommendation for Key Management
- PCI DSS 3.5.2: Key rotation requirements
- FIPS 140-2: Security Requirements for Cryptographic Modules

---

*Research document for GGID — the Go-based multi-tenant IAM platform.
See companion document `jwks-key-rotation.md` for dual-key strategy and
zero-downtime rotation procedure details.*
