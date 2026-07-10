# Audit Log Tampering Detection for IAM Systems

> **Focus**: Implementation-grade tamper-evidence techniques for audit logs — hash
> chains, Merkle commitments, append-only storage, RFC 3161 timestamping, external
> commitment, and real-time detection — with concrete Go code and an analysis of
> GGID's current audit infrastructure.
>
> **Companion doc**: `docs/research/audit-log-compliance.md` covers which events
> frameworks require and retention tiers. This document goes deep into *how* to
> make logs cryptographically tamper-evident and detectable at the storage layer.

---

## 1. Threat Model for Audit Tampering

### 1.1 Who tampers with audit logs

| Threat Actor | Access Level | Motivation |
|---|---|---|
| **Malicious insider (DBA/admin)** | Direct DB access | Conceal unauthorized access, data exfiltration, or privilege abuse performed during work hours |
| **Compromised admin account** | Application + DB admin | Attacker pivots from initial foothold to erase traces of lateral movement and privilege escalation |
| **Compromised application** | App-level DB credentials | Attacker who achieves RCE or SQL injection in a service wants to delete or alter the forensic trail before the breach is detected |
| **Rogue operator (cloud)** | Cloud console access | Cloud admin detaches EBS volumes or modifies S3 bucket policies to access and alter log objects |
| **Nation-state / APT** | Supply-chain or 0-day access | Long-term persistence — modify logs to maintain cover for months or years |

### 1.2 How tampering is done

**Direct database manipulation** — the most common vector:

```sql
-- Erase all evidence of a specific action
DELETE FROM audit_events WHERE actor_id = 'target-uuid' AND action = 'role.assign';

-- Alter a specific record to change the recorded result
UPDATE audit_events SET result = 'denied' WHERE id = 'target-event-id';

-- Bulk purge to cover an entire session
DELETE FROM audit_events WHERE created_at BETWEEN '2025-01-15 02:00:00' AND '2025-01-15 03:00:00';

-- Drop and recreate a partition to silently lose a month of records
DROP TABLE audit_events_2025_01;
CREATE TABLE audit_events_2025_01 PARTITION OF audit_events
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

**SQL injection** — if a query endpoint is vulnerable, the attacker uses `UPDATE`/`DELETE`
through the injection point rather than direct DB access.

**Storage-layer manipulation** — modifying the PostgreSQL data files on disk,
restoring a snapshot from before the attack window, or replacing WAL segments.

**Transport-layer manipulation** — intercepting NATS messages in transit and dropping
or replacing them before they reach the consumer.

### 1.3 Why tamper detection matters for IAM

IAM audit logs are the **primary forensic evidence** for identity-related incidents.
If they can be silently modified, every downstream control is compromised:

- **Compliance**: SOC 2, HIPAA, PCI DSS all require logs to be tamper-evident or
  tamper-proof. An auditor finding no integrity control is a **control deficiency**.
- **Forensics**: Incident response depends on accurate timelines. Modified logs lead
  investigators to wrong conclusions about attacker dwell time and blast radius.
- **Legal evidence**: In litigation or criminal prosecution, log admissibility
  requires demonstrating a chain of custody and integrity. A modifiable log has
  zero evidentiary value.
- **Trust**: If an attacker can erase their login records, they can perform
  unlimited unauthorized access with plausible deniability. Tamper detection is
  the **last line of defense** — even if the attack succeeds, the evidence survives.

### 1.4 Defense in depth — layered integrity controls

```
┌─────────────────────────────────────────────────────────┐
│ Layer 5: External Commitment (S3 Object Lock / chain)  │  ← strongest
├─────────────────────────────────────────────────────────┤
│ Layer 4: RFC 3161 Trusted Timestamping                 │
├─────────────────────────────────────────────────────────┤
│ Layer 3: Merkle Tree Root Publication                  │
├─────────────────────────────────────────────────────────┤
│ Layer 2: Hash Chain (sequential HMAC)                  │
├─────────────────────────────────────────────────────────┤
│ Layer 1: Append-Only Storage (DB constraints)          │
├─────────────────────────────────────────────────────────┤
│ Layer 0: Network Integrity (TLS, NATS auth)            │  ← weakest
└─────────────────────────────────────────────────────────┘
```

Each layer addresses a different attacker capability. Layer 1 stops an
application-level attacker with INSERT-only credentials. Layer 2+3 stop a DBA
who can UPDATE/DELETE. Layer 5 stops a cloud admin who can modify storage objects.

---

## 2. Hash Chain Implementation

### 2.1 Sequential hash chaining

Each audit event stores a hash that cryptographically depends on the previous
event's hash, forming an unbreakable chain:

```
event_1.hash = SHA-256(genesis_hash || serialize(event_1))
event_2.hash = SHA-256(event_1.hash || serialize(event_2))
event_3.hash = SHA-256(event_2.hash || serialize(event_3))
...
event_n.hash = SHA-256(event_{n-1}.hash || serialize(event_n))
```

If any event is modified, deleted, or inserted out of order, the chain breaks
at that point. The genesis hash is a well-known constant (e.g., all zeros) that
anchors the chain.

**Per-event vs block-based chains:**

| Approach | Chain Granularity | Verification Cost | Insert Latency | Best For |
|---|---|---|---|---|
| Per-event chain | Every event linked to previous | O(N) full scan | High (read prev hash per insert) | High-security, low-throughput |
| Block-based chain | Events grouped into blocks (e.g., 1000 events or 5 min) | O(N/B) blocks | Low (chain at block close) | High-throughput IAM systems |

For GGID's architecture (events arrive via NATS, persisted in batches by the
consumer), **block-based chaining at batch-close time** is the practical choice.
Each consumer batch becomes a block with a single chain hash, avoiding the need
for per-event read-before-write contention.

### 2.2 Genesis block

The genesis block establishes the chain's anchor. It is a special record with a
known, publicly verifiable starting hash:

```go
// GenesisHash is the well-known starting hash for all audit chains.
// Any verification routine must start from this constant.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"
```

For per-tenant chains, the genesis can incorporate the tenant ID to prevent
cross-tenant chain splicing attacks:

```go
genesisHash := sha256.Sum256([]byte("ggid-audit-genesis:" + tenantID.String()))
```

### 2.3 Go implementation — hash-chained event storage

```go
package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// ChainConfig controls hash chain behavior.
type ChainConfig struct {
	// Secret is the HMAC key. If nil, plain SHA-256 is used.
	// With HMAC, an attacker who can compute SHA-256 still can't forge
	// valid chain hashes without the secret.
	Secret []byte
	// PerTenant enables separate chains per tenant_id, preventing
	// cross-tenant chain splicing.
	PerTenant bool
}

// ChainVerifier computes and verifies audit hash chains.
type ChainVerifier struct {
	cfg ChainConfig
}

func NewChainVerifier(cfg ChainConfig) *ChainVerifier {
	return &ChainVerifier{cfg: cfg}
}

// ComputeHash returns the chain hash for an event given the previous hash.
func (v *ChainVerifier) ComputeHash(eventJSON []byte, prevHash string) string {
	if len(v.cfg.Secret) > 0 {
		mac := hmac.New(sha256.New, v.cfg.Secret)
		mac.Write([]byte(prevHash))
		mac.Write(eventJSON)
		return hex.EncodeToString(mac.Sum(nil))
	}
	h := sha256.New()
	h.Write([]byte(prevHash))
	h.Write(eventJSON)
	return hex.EncodeToString(h.Sum(nil))
}

// ChainedEvent pairs an audit event with its computed chain hash.
type ChainedEvent struct {
	Event    []byte // serialized JSON of the audit event
	Hash     string // SHA-256(prevHash || Event)
	EventID  uuid.UUID
}

// SealBatch takes a batch of serialized events and the previous chain hash,
// then computes chain hashes for all events in sequence.
// Returns the sealed events and the final hash for the batch.
func (v *ChainVerifier) SealBatch(events [][]byte, prevHash string) ([]ChainedEvent, string) {
	result := make([]ChainedEvent, len(events))
	current := prevHash
	for i, evt := range events {
		h := v.ComputeHash(evt, current)
		result[i] = ChainedEvent{Event: evt, Hash: h}
		current = h
	}
	return result, current
}

// VerifyBatch walks the chain from a starting hash and returns the index
// of the first mismatch (or -1 if all hashes verify).
func (v *ChainVerifier) VerifyBatch(events []ChainedEvent, startHash string) int {
	current := startHash
	for i, ce := range events {
		expected := v.ComputeHash(ce.Event, current)
		if len(v.cfg.Secret) > 0 {
			if !hmac.Equal([]byte(expected), []byte(ce.Hash)) {
				return i // mismatch at position i
			}
		} else {
			if expected != ce.Hash {
				return i
			}
		}
		current = ce.Hash
	}
	return -1 // all verified
}
```

### 2.4 Full chain verification algorithm

```go
// VerifyFullChain performs a complete verification of a tenant's audit chain
// by reading all events in created_at order and checking every link.
func (v *ChainVerifier) VerifyFullChain(
	ctx context.Context,
	repo ChainReader,
	tenantID uuid.UUID,
) (*ChainVerificationReport, error) {
	report := &ChainVerificationReport{
		TenantID:    tenantID,
		VerifiedAt:  time.Now().UTC(),
		TotalEvents: 0,
		BrokenAt:    -1,
	}

	const batchSize = 10000
	offset := 0
	genesis := GenesisHash
	if v.cfg.PerTenant {
		g := sha256.Sum256([]byte("ggid-audit-genesis:" + tenantID.String()))
		genesis = hex.EncodeToString(g[:])
	}

	prevHash := genesis

	for {
		events, err := repo.ReadChainBatch(ctx, tenantID, offset, batchSize)
		if err != nil {
			return nil, fmt.Errorf("read chain batch at offset %d: %w", offset, err)
		}
		if len(events) == 0 {
			break
		}

		breakIndex := v.VerifyBatch(events, prevHash)
		report.TotalEvents += len(events)

		if breakIndex >= 0 {
			report.BrokenAt = offset + breakIndex
			report.BrokenEventID = events[breakIndex].EventID
			report.BrokenExpectedHash = v.ComputeHash(events[breakIndex].Event, prevHash)
			report.BrokenActualHash = events[breakIndex].Hash
			return report, nil
		}

		prevHash = events[len(events)-1].Hash
		offset += len(events)
	}

	report.Valid = true
	return report, nil
}

// ChainReader is the interface a repository must implement for verification.
type ChainReader interface {
	ReadChainBatch(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]ChainedEvent, error)
}

// ChainVerificationReport summarizes a full chain verification pass.
type ChainVerificationReport struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	VerifiedAt         time.Time `json:"verified_at"`
	TotalEvents        int       `json:"total_events"`
	Valid              bool      `json:"valid"`
	BrokenAt           int       `json:"broken_at,omitempty"`
	BrokenEventID      uuid.UUID `json:"broken_event_id,omitempty"`
	BrokenExpectedHash string    `json:"broken_expected_hash,omitempty"`
	BrokenActualHash   string    `json:"broken_actual_hash,omitempty"`
}
```

### 2.5 Trade-offs

- **Single point of failure**: If one event is lost (consumer crash, message
  drop), the chain breaks. Mitigation: NATS JetStream durable consumers with
  at-least-once delivery + idempotent inserts (check event ID before insert).
- **O(N) verification cost**: Full chain scan over millions of events is slow.
  Mitigation: periodic batch verification in a background worker + Merkle trees
  (Section 3) for O(log N) spot checks.
- **HMAC key rotation**: If the secret rotates, the chain must be re-anchored.
  Use a versioned key scheme: chain hash includes a key ID prefix.

---

## 3. Merkle Tree Commitment

### 3.1 Why Merkle trees over linear chains

A linear hash chain requires O(N) verification for the full chain. A Merkle tree
allows **O(log N) inclusion proofs** — you can cryptographically prove that a
single event is part of a batch by providing only log2(N) sibling hashes, without
revealing other events. This is the approach used by Certificate Transparency
logs (RFC 6962).

| Property | Linear Hash Chain | Merkle Tree |
|---|---|---|
| Full verification cost | O(N) | O(N) (must read all leaves) |
| Single-event inclusion proof | O(N) (entire chain) | O(log N) |
| Tamper detection (root mismatch) | O(1) compare | O(1) compare |
| Append cost | O(1) (just prev hash) | O(log N) (rebuild affected path) |
| Batch efficiency | Low (per-event) | High (batch close = one root) |

For IAM audit logs, the best architecture is **both**: a per-tenant hash chain
for real-time tamper detection on insert, plus periodic Merkle root publication
for external verification and inclusion proofs.

### 3.2 Building a Merkle tree from audit events

```go
package audit

import (
	"crypto/sha256"
)

// MerkleTree builds a binary hash tree from leaf data.
type MerkleTree struct {
	// Levels[0] = leaves, Levels[d] = root
	// Each level has half the nodes of the previous level.
	Levels [][][32]byte
}

// NewMerkleTree constructs a Merkle tree from leaf hashes.
// If the number of leaves is odd, the last leaf is duplicated.
func NewMerkleTree(leaves [][]byte) *MerkleTree {
	if len(leaves) == 0 {
		return &MerkleTree{}
	}

	// Level 0: hash each leaf
	level := make([][32]byte, len(leaves))
	for i, leaf := range leaves {
		level[i] = sha256.Sum256(leaf)
	}

	tree := &MerkleTree{Levels: [][][32]byte{level}}

	for len(level) > 1 {
		next := make([][32]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			var left, right [32]byte
			left = level[i]
			if i+1 < len(level) {
				right = level[i+1]
			} else {
				right = level[i] // duplicate last node for odd count
			}
			combined := append(left[:], right[:]...)
			next = append(next, sha256.Sum256(combined))
		}
		tree.Levels = append(tree.Levels, next)
		level = next
	}

	return tree
}

// Root returns the Merkle root hash (top of the tree).
func (t *MerkleTree) Root() [32]byte {
	if len(t.Levels) == 0 {
		return [32]byte{}
	}
	lastLevel := t.Levels[len(t.Levels)-1]
	return lastLevel[0]
}

// MerkleProof contains the sibling hashes needed to verify leaf inclusion.
type MerkleProof struct {
	LeafIndex int          `json:"leaf_index"`
	LeafHash  [32]byte     `json:"leaf_hash"`
	Siblings  []ProofNode  `json:"siblings"`
}

// ProofNode is a sibling hash and its direction (left or right).
type ProofNode struct {
	Hash      [32]byte `json:"hash"`
	IsRight   bool     `json:"is_right"` // true = sibling is on the right
}

// GenerateProof produces a Merkle inclusion proof for the leaf at the given index.
func (t *MerkleTree) GenerateProof(leafIndex int) MerkleProof {
	proof := MerkleProof{
		LeafIndex: leafIndex,
		LeafHash:  t.Levels[0][leafIndex],
		Siblings:  []ProofNode{},
	}

	idx := leafIndex
	for level := 0; level < len(t.Levels)-1; level++ {
		var siblingIdx int
		var isRight bool
		if idx%2 == 0 {
			siblingIdx = idx + 1
			isRight = true
		} else {
			siblingIdx = idx - 1
			isRight = false
		}
		// Handle odd node count: sibling is self (duplicate)
		if siblingIdx >= len(t.Levels[level]) {
			siblingIdx = idx
		}
		proof.Siblings = append(proof.Siblings, ProofNode{
			Hash:    t.Levels[level][siblingIdx],
			IsRight: isRight,
		})
		idx /= 2
	}

	return proof
}

// VerifyProof checks that a Merkle proof is valid for a given root.
func VerifyProof(proof MerkleProof, root [32]byte) bool {
	current := proof.LeafHash
	for _, sibling := range proof.Siblings {
		var combined []byte
		if sibling.IsRight {
			combined = append(current[:], sibling.Hash[:]...)
		} else {
			combined = append(sibling.Hash[:], current[:]...)
		}
		current = sha256.Sum256(combined)
	}
	return current == root
}
```

### 3.3 Root publication and rotation frequency

The Merkle root is published to an **external trust anchor** that the attacker
cannot modify. Options:

1. **S3 Object Lock (compliance mode)** — write the root to a WORM object. Once
   written, not even the root account can delete or overwrite it during the
   retention period.
2. **Blockchain / transparency log** — submit the root hash as an OP_RETURN
   transaction (Bitcoin) or to a transparency log service. Immutable by design.
3. **Separate hardened DB** — a minimal append-only database on a different
   host, with no DELETE/UPDATE grants, that only accepts root hash inserts.
4. **Notarization service** — commercial services likeagenzia entrate or
   Universally verifiable timestamping APIs.

**Rotation cadence** determines the detection window:

| Cadence | Detection Window | Overhead | Use Case |
|---|---|---|---|
| Per-event | Immediate (next insert) | High — publish per event | Ultra-high-security |
| Per-batch (1000 events) | <1 min at 1k EPS | Low | High-throughput |
| Per-minute | 60 seconds | Very low | General purpose |
| Per-hour | 60 minutes | Negligible | Compliance minimum |

For GGID, a **per-batch Merkle root published every consumer batch** (or every 5
minutes, whichever comes first) provides sub-minute tamper detection with
minimal overhead.

### 3.4 Inclusion proof workflow

```
1. Auditor requests proof for event E (by ID)
2. Service finds E's leaf index in the relevant batch tree
3. Service generates MerkleProof (log2(N) sibling hashes)
4. Service returns: { leaf_hash, siblings[], published_root, publication_receipt }
5. Auditor independently verifies: VerifyProof(proof, published_root) == true
6. Auditor checks publication_receipt against external trust anchor
```

This lets an external auditor verify a single event without downloading the
entire audit log.

---

## 4. Append-Only Storage

### 4.1 PostgreSQL append-only enforcement

The application-level defenses (hash chain, Merkle tree) are meaningless if an
attacker can `UPDATE` or `DELETE` rows directly in the database. Append-only
storage enforces immutability at the DB layer.

**Step 1: Revoke UPDATE/DELETE from the application role:**

```sql
-- Create a role that can only INSERT and SELECT
CREATE ROLE audit_writer;

-- Grant only INSERT and SELECT on the audit table
GRANT INSERT, SELECT ON audit_events TO audit_writer;

-- Explicitly do NOT grant UPDATE or DELETE
-- The application connects as audit_writer, so it physically cannot
-- modify or delete audit records even if compromised.
```

**Step 2: INSERT-only trigger as defense-in-depth:**

```sql
-- Even if a DBA grants UPDATE/DELETE, this trigger blocks it.
CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_events is append-only: % operation not permitted', TG_OP;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER no_update BEFORE UPDATE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

CREATE TRIGGER no_delete BEFORE DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

-- Allow TRUNCATE to also be blocked (TRUNCATE bypasses row-level triggers)
-- This requires an event trigger:
CREATE OR REPLACE FUNCTION prevent_truncate()
RETURNS event_trigger AS $$
BEGIN
    RAISE EXCEPTION 'TRUNCATE not permitted on audit tables';
END;
$$ LANGUAGE plpgsql;

CREATE EVENT TRIGGER no_truncate ON ddl_command_end
    WHEN TAG IN ('TRUNCATE TABLE')
    EXECUTE FUNCTION prevent_truncate();
```

### 4.2 Go code for append-only enforcement via migration

```go
package audit

import (
	"context"
	"fmt"
)

// AppendOnlyMigrations contains SQL to enforce append-only behavior.
var AppendOnlyMigrations = []string{
	// 1. Create app role with INSERT + SELECT only
	`DO $$
	BEGIN
	    CREATE ROLE audit_writer LOGIN PASSWORD :'app_password';
	    GRANT INSERT, SELECT ON audit_events TO audit_writer;
	    GRANT USAGE ON SEQUENCE audit_events_id_seq TO audit_writer;
	EXCEPTION WHEN duplicate_object THEN NULL;
	END $$;`,

	// 2. Prevent UPDATE trigger
	`CREATE OR REPLACE FUNCTION prevent_audit_update()
	RETURNS TRIGGER AS $$
	BEGIN
	    RAISE EXCEPTION 'audit_events is append-only: UPDATE denied';
	END;
	$$ LANGUAGE plpgsql;

	CREATE TRIGGER trg_no_audit_update
	    BEFORE UPDATE ON audit_events
	    FOR EACH ROW EXECUTE FUNCTION prevent_audit_update();`,

	// 3. Prevent DELETE trigger
	`CREATE OR REPLACE FUNCTION prevent_audit_delete()
	RETURNS TRIGGER AS $$
	BEGIN
	    RAISE EXCEPTION 'audit_events is append-only: DELETE denied';
	END;
	$$ LANGUAGE plpgsql;

	CREATE TRIGGER trg_no_audit_delete
	    BEFORE DELETE ON audit_events
	    FOR EACH ROW EXECUTE FUNCTION prevent_audit_delete();`,
}

// ApplyAppendOnly enforces append-only constraints on the audit table.
func ApplyAppendOnly(ctx context.Context, db DBExecer) error {
	for i, migration := range AppendOnlyMigrations {
		if _, err := db.Exec(ctx, migration); err != nil {
			return fmt.Errorf("append-only migration %d failed: %w", i, err)
		}
	}
	return nil
}

type DBExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (CommandTag, error)
}

type CommandTag interface {
	RowsAffected() int64
}
```

### 4.3 Partition-based time-windowed immutability

A powerful pattern: **freeze old partitions** by revoking all access except SELECT:

```sql
-- After a partition's month ends, convert it to read-only
ALTER TABLE audit_events_2025_01 NO INHERIT audit_events;

-- Detach and move to a read-only tablespace or schema
ALTER TABLE audit_events_2025_01 SET TABLESPACE audit_readonly_ts;

-- Revoke everything except SELECT from the frozen partition
REVOKE INSERT ON audit_events_2025_01 FROM audit_writer;
```

This creates a **rolling immutability window**: current month is append-only,
all previous months are frozen read-only. An attacker who compromises the app
credentials today cannot touch last month's logs.

### 4.4 WORM storage patterns

- **S3 Object Lock (Compliance mode)**: Once an object is written, it cannot be
  deleted or overwritten by anyone — including the root account — for the
  specified retention period. Export daily Merkle roots or raw event batches to
  S3 Object Lock as the ultimate external commitment.
- **GCP Bucket Lock**: Similar functionality with retention policies.
- **Azure Immutable Blob Storage**: Time-based retention + legal hold policies.
- **On-prem WORM**: Optical media, tape archives, or hardware WORM appliances
  for air-gapped environments.

---

## 5. RFC 3161 Timestamping

### 5.1 How trusted timestamping works

RFC 3161 defines a protocol for requesting a cryptographically verifiable
timestamp from a trusted Time Stamping Authority (TSA). The timestamp proves
that a specific hash existed at a specific time, signed by the TSA's
certificate.

```
┌──────────┐  1. Hash(event_batch)   ┌─────────┐
│  GGID    │ ──────────────────────▶ │   TSA   │
│ Audit    │                         │ (e.g.,  │
│ Service  │  2. Timestamp Token     │ DigiCert│
│          │ ◀────────────────────── │  SectiGo)│
└──────────┘  (signed by TSA cert)   └─────────┘
     │
     │  3. Store token alongside batch
     ▼
┌──────────┐
│ Audit DB │  hash + TSA token + TSA cert chain
└──────────┘
```

The key insight: the TSA does not see the data, only its hash. Privacy is
preserved. But the timestamp token is cryptographic proof that the hash existed
before a specific time.

### 5.2 Timestamping audit batches

For efficiency, GGID does not timestamp every event individually. Instead:

1. At each Merkle tree rotation (e.g., every 5 minutes), take the **Merkle root**.
2. Submit the root hash to a TSA.
3. Store the timestamp token alongside the Merkle root.
4. This proves that the entire batch of events existed before the timestamp time.

### 5.3 Go implementation — requesting and verifying RFC 3161 timestamps

```go
package audit

import (
	"bytes"
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/digitorus/timestamp"
)

// TSAClient requests RFC 3161 timestamp tokens from a TSA.
type TSAClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewTSAClient creates a client for the given TSA endpoint.
// Common public TSAs:
//   - http://timestamp.digicert.com
//   - http://timestamp.sectigo.com
//   - http://tsa.belgium.be/connect
func NewTSAClient(endpoint string) *TSAClient {
	return &TSAClient{
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// TimestampHash requests a timestamp token for the given hash.
// Returns the DER-encoded timestamp token.
func (c *TSAClient) TimestampHash(hash []byte) ([]byte, error) {
	// Build RFC 3161 TimeStampReq
	req := timestamp.Request{
		HashAlgorithm: crypto.SHA256,
		HashedMessage: hash[:],
	}

	reqDER, err := req.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal timestamp request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(reqDER))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/timestamp-query")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("timestamp request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TSA returned %d: %s", resp.StatusCode, body)
	}

	tokenDER, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read TSA response: %w", err)
	}

	return tokenDER, nil
}

// TimestampToken holds a verified RFC 3161 timestamp.
type TimestampToken struct {
	RawDER      []byte    `json:"-"`          // raw DER-encoded token
	Hash        []byte    `json:"hash"`       // hash that was timestamped
	Timestamp   time.Time `json:"timestamp"`  // TSA-verified time
	Serial      []byte    `json:"serial"`     // TSA serial number
	Policy      string    `json:"policy"`     // TSA policy OID
	CertChain   [][]byte  `json:"cert_chain"` // TSA certificate chain
}

// VerifyTimestampToken parses and verifies a timestamp token.
// It checks that the token covers the expected hash and that the
// TSA signature is valid against a trusted certificate pool.
func VerifyTimestampToken(tokenDER []byte, expectedHash []byte, trustedRoots *x509.CertPool) (*TimestampToken, error) {
	ts, err := timestamp.ParseResponse(tokenDER)
	if err != nil {
		return nil, fmt.Errorf("parse timestamp token: %w", err)
	}

	// Verify the token hash matches what we expect
	if !bytes.Equal(ts.HashedMessage, expectedHash) {
		return nil, fmt.Errorf("timestamp token hash mismatch: token does not cover expected data")
	}

	// Build certificate chain from the token
	if len(ts.Certificates) == 0 {
		return nil, fmt.Errorf("timestamp token has no embedded certificates")
	}

	intermediates := x509.NewCertPool()
	leaf := ts.Certificates[0]
	for _, cert := range ts.Certificates[1:] {
		intermediates.AddCert(cert)
	}

	// Verify the TSA certificate chain
	_, err = leaf.Verify(x509.VerifyOptions{
		Roots:         trustedRoots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
	})
	if err != nil {
		return nil, fmt.Errorf("TSA certificate verification failed: %w", err)
	}

	// Verify the signature on the timestamp token
	if err := ts.Verify(leaf.PublicKey); err != nil {
		return nil, fmt.Errorf("timestamp signature verification failed: %w", err)
	}

	token := &TimestampToken{
		RawDER:    tokenDER,
		Hash:      ts.HashedMessage,
		Timestamp: ts.Time,
		Serial:    ts.SerialNumber,
		CertChain: make([][]byte, len(ts.Certificates)),
	}
	for i, cert := range ts.Certificates {
		token.CertChain[i] = cert.Raw
	}

	return token, nil
}

// BatchTimestampRequest summarizes what gets stored per Merkle rotation.
type BatchTimestampRequest struct {
	BatchID    uuid.UUID `json:"batch_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	MerkleRoot []byte    `json:"merkle_root"`
	EventCount int       `json:"event_count"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
}

// TimestampBatch ties a Merkle root to a TSA-verified timestamp and persists both.
func (c *TSAClient) TimestampBatch(req BatchTimestampRequest) (*TimestampToken, error) {
	tokenDER, err := c.TimestampHash(req.MerkleRoot)
	if err != nil {
		return nil, fmt.Errorf("timestamp batch %s: %w", req.BatchID, err)
	}

	// Note: in production, trustedRoots would be loaded from a config or
	// the system trust store, pre-configured with TSA root CAs.
	token, err := VerifyTimestampToken(tokenDER, req.MerkleRoot, x509.NewCertPool())
	if err != nil {
		return nil, fmt.Errorf("verify returned timestamp: %w", err)
	}

	return token, nil
}
```

### 5.4 Timestamp verification in forensics

During an incident investigation, an auditor can:

1. Extract the Merkle root and TSA token for a batch.
2. Independently verify the TSA signature against public TSA root certificates.
3. Confirm the batch existed **before** the timestamp time.
4. This proves the audit log was not retroactively fabricated or modified.

This is critical for legal evidence: a self-signed timestamp from the same system
under attack has no value, but a TSA-verified timestamp from an independent
authority is extremely difficult to dispute.

---

## 6. External Commitment Service

### 6.1 Commitment cadence and detection window

External commitment is the practice of periodically publishing a cryptographic
summary of audit state to a system the attacker cannot reach. The detection
window is the maximum time between tampering and detection.

| Strategy | Publication Point | Detection Window | Bandwidth | Trust Model |
|---|---|---|---|---|
| Per-event | Each event hash → external | 0 (immediate) | Very high | Requires external sync per event |
| Per-batch | Batch Merkle root → external | < 1 min | Low | Strong — batch root covers all events |
| Per-hour | Hourly root → external | 60 min | Negligible | Adequate for compliance |
| Per-day | Daily root → blockchain | 24h | Negligible | Immutable public record |

For GGID, a **per-batch commitment to S3 Object Lock** is recommended:
- S3 Object Lock compliance mode prevents modification even by root account.
- Merkle root is 32 bytes — trivial to publish.
- Detection window is the consumer batch interval (configurable, default 5 min).

### 6.2 Go code — external commitment publisher

```go
package audit

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CommitmentRecord is what gets published externally.
type CommitmentRecord struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	BatchID     uuid.UUID `json:"batch_id"`
	MerkleRoot  string    `json:"merkle_root"`
	EventCount  int       `json:"event_count"`
	ChainHash   string    `json:"chain_hash"` // last hash in the hash chain
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	CommittedAt time.Time `json:"committed_at"`
}

// CommitmentSink is the interface for external commitment targets.
type CommitmentSink interface {
	Commit(ctx context.Context, record CommitmentRecord) error
}

// S3ObjectLockSink publishes commitments to S3 with Object Lock.
type S3ObjectLockSink struct {
	bucket     string
	retention  time.Duration
	s3Client   S3Putter
}

type S3Putter interface {
	PutObject(ctx context.Context, bucket, key string, body []byte,
		retentionMode string, retentionDays int) error
}

func NewS3ObjectLockSink(bucket string, retention time.Duration, s3 S3Putter) *S3ObjectLockSink {
	return &S3ObjectLockSink{bucket: bucket, retention: retention, s3Client: s3}
}

func (s *S3ObjectLockSink) Commit(ctx context.Context, record CommitmentRecord) error {
	key := fmt.Sprintf("audit-commitments/%s/%s.json",
		record.TenantID, record.BatchID)
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal commitment record: %w", err)
	}

	retentionDays := int(s.retention.Hours() / 24)
	return s.s3Client.PutObject(ctx, s.bucket, key, data, "COMPLIANCE", retentionDays)
}

// CommitmentPublisher periodically commits Merkle roots + chain hashes externally.
type CommitmentPublisher struct {
	tenantID uuid.UUID
	sink     CommitmentSink
	interval time.Duration
}

func NewCommitmentPublisher(tenantID uuid.UUID, sink CommitmentSink, interval time.Duration) *CommitmentPublisher {
	return &CommitmentPublisher{tenantID: tenantID, sink: sink, interval: interval}
}

// CommitBatch commits a single batch's Merkle root and chain hash externally.
func (p *CommitmentPublisher) CommitBatch(
	ctx context.Context,
	batchID uuid.UUID,
	merkleRoot [32]byte,
	chainHash string,
	eventCount int,
	startTime, endTime time.Time,
) error {
	record := CommitmentRecord{
		ID:          uuid.New(),
		TenantID:    p.tenantID,
		BatchID:     batchID,
		MerkleRoot:  hex.EncodeToString(merkleRoot[:]),
		EventCount:  eventCount,
		ChainHash:   chainHash,
		StartTime:   startTime,
		EndTime:     endTime,
		CommittedAt: time.Now().UTC(),
	}

	if err := p.sink.Commit(ctx, record); err != nil {
		return fmt.Errorf("external commitment failed: %w", err)
	}

	return nil
}
```

### 6.3 Verification against external commitment

```go
// VerifyExternalCommitment reads the latest external commitment for a tenant
// and compares it against the current DB state to detect tampering.
func VerifyExternalCommitment(
	ctx context.Context,
	sink CommitmentSink,
	tenantID uuid.UUID,
	localRoot [32]byte,
	localChainHash string,
) error {
	// Fetch the last commitment record from the external system
	// (implementation depends on the sink type)
	records, err := sink.FetchLatest(ctx, tenantID, 1)
	if err != nil {
		return fmt.Errorf("fetch external commitment: %w", err)
	}
	if len(records) == 0 {
		return fmt.Errorf("no external commitment found for tenant %s", tenantID)
	}

	latest := records[0]

	// The local chain hash must match (or extend) the committed chain hash.
	// The local Merkle root for the committed batch must match.
	if latest.ChainHash != localChainHash {
		return fmt.Errorf("CHAIN HASH MISMATCH: external=%s local=%s — TAMPERING DETECTED",
			latest.ChainHash, localChainHash)
	}

	committedRoot := latest.MerkleRoot
	localRootHex := hex.EncodeToString(localRoot[:])
	if committedRoot != localRootHex {
		return fmt.Errorf("MERKLE ROOT MISMATCH: external=%s local=%s — TAMPERING DETECTED",
			committedRoot, localRootHex)
	}

	return nil
}
```

---

## 7. Tamper Detection and Alerting

### 7.1 Real-time detection on INSERT

The most valuable detection is **at write time**: when the consumer persists a
new batch, it computes chain hashes and immediately verifies the link to the
previous batch. If the chain breaks, the event is flagged before it's accepted.

```go
// TamperDetectionMiddleware wraps a repository Insert to verify chain integrity
// in real-time. If the chain is broken, it blocks the insert and raises an alert.
type TamperDetectionMiddleware struct {
	repo      ChainAwareRepo
	verifier  *ChainVerifier
	alerter   TamperAlerter
}

type ChainAwareRepo interface {
	GetLastChainHash(ctx context.Context, tenantID uuid.UUID) (string, error)
	InsertWithHash(ctx context.Context, event []byte, hash string) error
}

type TamperAlerter interface {
	AlertTamperDetected(ctx context.Context, detail TamperAlert) error
}

type TamperAlert struct {
	TenantID    uuid.UUID
	Timestamp   time.Time
	ExpectedHash string
	ActualHash   string
	BatchID      uuid.UUID
	Severity     string // "critical"
}

func (m *TamperDetectionMiddleware) InsertBatch(
	ctx context.Context,
	tenantID uuid.UUID,
	events [][]byte,
) error {
	// Read the last known chain hash for this tenant
	prevHash, err := m.repo.GetLastChainHash(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("read last chain hash: %w", err)
	}
	if prevHash == "" {
		genesis := GenesisHash
		if m.verifier.cfg.PerTenant {
			g := sha256.Sum256([]byte("ggid-audit-genesis:" + tenantID.String()))
			genesis = hex.EncodeToString(g[:])
		}
		prevHash = genesis
	}

	// Seal the batch (compute chain hashes)
	sealed, finalHash := m.verifier.SealBatch(events, prevHash)

	// CRITICAL: verify the first event links to the stored prevHash.
	// This catches tampering that happened between our read and our write.
	firstExpected := m.verifier.ComputeHash(sealed[0].Event, prevHash)
	if firstExpected != sealed[0].Hash {
		alert := TamperAlert{
			TenantID:     tenantID,
			Timestamp:    time.Now().UTC(),
			ExpectedHash: firstExpected,
			ActualHash:   sealed[0].Hash,
			Severity:     "critical",
		}
		_ = m.alerter.AlertTamperDetected(ctx, alert)
		return fmt.Errorf("real-time tamper detection: chain broken at batch start")
	}

	// Persist all events with their chain hashes
	for _, se := range sealed {
		if err := m.repo.InsertWithHash(ctx, se.Event, se.Hash); err != nil {
			return fmt.Errorf("insert chained event: %w", err)
		}
	}

	return nil
}
```

### 7.2 Batch verification — periodic full-chain scan

A background worker runs a full chain verification periodically (e.g., every hour)
to catch tampering that real-time detection might miss (e.g., direct DB UPDATE
that bypasses the application):

```go
// PeriodicChainScanner runs full-chain verification for all tenants on a schedule.
type PeriodicChainScanner struct {
	verifier  *ChainVerifier
	repo      ChainReader
	alerter   TamperAlerter
	tenantSvc TenantLister
}

type TenantLister interface {
	ListAll(ctx context.Context) ([]uuid.UUID, error)
}

func (s *PeriodicChainScanner) ScanAll(ctx context.Context) {
	tenants, err := s.tenantSvc.ListAll(ctx)
	if err != nil {
		return
	}

	for _, tenantID := range tenants {
		report, err := s.verifier.VerifyFullChain(ctx, s.repo, tenantID)
		if err != nil {
			continue
		}

		if !report.Valid {
			alert := TamperAlert{
				TenantID:     tenantID,
				Timestamp:    report.VerifiedAt,
				ExpectedHash: report.BrokenExpectedHash,
				ActualHash:   report.BrokenActualHash,
				Severity:     "critical",
			}
			_ = s.alerter.AlertTamperDetected(ctx, alert)
		}
	}
}

// Run starts the periodic scanner.
func (s *PeriodicChainScanner) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.ScanAll(ctx)
		}
	}
}
```

### 7.3 Alerting — SIEM integration, admin notification, service halt

When tampering is detected, the response must be immediate and unavoidable:

```go
// MultiChannelAlerter sends tamper alerts through multiple independent channels
// to ensure the alert reaches someone even if one channel is compromised.
type MultiChannelAlerter struct {
	channels []TamperAlerter
}

func (m *MultiChannelAlerter) AlertTamperDetected(ctx context.Context, detail TamperAlert) error {
	var errs []error
	for _, ch := range m.channels {
		if err := ch.AlertTamperDetected(ctx, detail); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == len(m.channels) {
		return fmt.Errorf("all alert channels failed")
	}
	return nil
}

// SIEMAlerter sends a CEF-formatted alert to a SIEM/SOC platform.
type SIEMAlerter struct {
	endpoint string
	client   *http.Client
}

func (s *SIEMAlerter) AlertTamperDetected(ctx context.Context, detail TamperAlert) error {
	cef := fmt.Sprintf(
		"CEF:0|GGID|IAM|1.0|AUDIT-TAMPER|Audit Chain Tamper Detected|10|"+
			"src=audit-service act=audit.tamper_detected "+
			"tenant=%s severity=critical "+
			"expected_hash=%s actual_hash=%s "+
			"rt=%s",
		detail.TenantID, detail.ExpectedHash, detail.ActualHash,
		detail.Timestamp.Format(time.RFC3339),
	)
	// POST to SIEM endpoint (Splunk HEC, Elastic _bulk, etc.)
	return s.postToSIEM(ctx, cef)
}

// HaltOnTamper wraps the application to shut down if tampering is detected.
// This is the nuclear option: rather than risk more tampered data being
// written, the service halts and requires manual investigation.
type HaltOnTamper struct {
	delegate TamperAlerter
	halted   bool
	mu       sync.Mutex
}

func (h *HaltOnTamper) AlertTamperDetected(ctx context.Context, detail TamperAlert) error {
	h.mu.Lock()
	h.halted = true
	h.mu.Unlock()

	// Send the alert through the delegate
	_ = h.delegate.AlertTamperDetected(ctx, detail)

	// Halt the process — a tampered audit log is a critical security incident.
	// The service must stop writing until a human investigates.
	log.Panicf("CRITICAL: Audit tamper detected. Service halting. Tenant=%s Expected=%s Actual=%s",
		detail.TenantID, detail.ExpectedHash, detail.ActualHash)

	return nil
}

// IsHalted returns true if the service should stop accepting new events.
func (h *HaltOnTamper) IsHalted() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.halted
}
```

### 7.4 Alert severity matrix

| Detection Method | Latency | Catches | Response |
|---|---|---|---|
| Real-time chain check on INSERT | < 1 ms | Modified previous event | Block insert, alert, halt |
| Periodic full-chain scan | 1 hour | Direct DB UPDATE/DELETE | Alert all channels, halt |
| External commitment mismatch | 5 min (batch) | DB + storage tampering | Alert, investigate, publish root diff |
| Merkle root external verification | On-demand | Any historical tampering | Forensic audit, legal escalation |

---

## 8. GGID Audit Chain Analysis

### 8.1 Current implementation review

The following analysis is based on direct source code examination of
`services/audit/` and `pkg/audit/`:

**Domain model** (`services/audit/internal/domain/models.go`):
- `AuditEvent` struct includes a `Hash string` field with the comment
  `"HMAC chain hash for tamper detection"`.
- **However, this field is never populated.** It is always an empty string.

**Repository** (`services/audit/internal/repository/audit_repo.go`):
- `Insert()` does an `INSERT INTO audit_events (...)` with **13 columns** — the
  `hash` column is NOT included in the INSERT statement.
- `GetByID()` and `List()` SELECT 15 columns but do NOT select a `hash` column.
- `DeleteOlderThan()` provides a `DELETE FROM audit_events WHERE created_at < $1`
  operation — **this is a built-in tampering vector**. Any code path that calls
  this can erase audit history.

**Consumer** (`services/audit/internal/consumer/nats_consumer.go`):
- `processMessage()` unmarshals the JSON event and calls `repo.Insert()` directly.
- **No hash computation** occurs. No chain verification. No Merkle tree. The
  `Hash` field from the message (if any) is never validated or stored.

**Database schema** (`services/audit/migrations/000001_init_extensions.up.sql`):
- The `audit_events` table has **no `hash` column** at all. The domain model's
  `Hash` field has no corresponding database column.
- No triggers exist to prevent UPDATE or DELETE.
- No role-based restrictions: the application connects with full DML privileges.
- `DELETE FROM audit_events` is fully permitted by the schema.

**Publisher** (`pkg/audit/publisher.go`):
- `Publish()` and `PublishAsync()` send events to NATS.
- **No hash is computed** at publish time. Events are fire-and-forget.
- NATS JetStream is used with `MaxAge: 72h` — a 72-hour buffer, not a retention
  guarantee. Messages older than 72 hours are purged by JetStream regardless of
  whether they were consumed.

**External commitment**: None. No S3, no blockchain, no notarization.

**RFC 3161 timestamping**: None. No TSA integration exists.

**Append-only enforcement**: None. The table allows full CRUD.

### 8.2 Tamper protection summary

| Control | Exists? | Details |
|---|---|---|
| Hash chain field | Partially | `Hash` field in Go struct, no DB column, never populated |
| Hash chain computation | No | Consumer does not compute any hash |
| Hash chain verification | No | No `VerifyChain` API exists |
| Append-only DB constraints | No | No triggers, no REVOKE, no WORM |
| Merkle tree | No | Not implemented anywhere |
| Merkle root publication | No | No external commitment |
| RFC 3161 timestamping | No | No TSA integration |
| External commitment (S3/blockchain) | No | Not implemented |
| Real-time tamper detection | No | No middleware on Insert path |
| Periodic chain scan | No | No background worker |
| Tamper alerting | No | No alert pipeline |
| Delete protection | **Negative** | `DeleteOlderThan` provides easy tampering |
| TLS on NATS | Configurable | NATS supports TLS but not enforced by default |

### 8.3 Attack surface assessment

The current GGID audit system has **zero tamper resistance**. An attacker with
any of the following can erase or modify audit logs undetectably:

1. **Application DB credentials** — can UPDATE/DELETE any audit_events row.
2. **NATS access** — can drop or replace messages in transit.
3. **PostgreSQL superuser** — can DROP TABLE, TRUNCATE, or modify rows directly.
4. **Cloud console access** — can restore DB snapshots from before the attack.

The `DeleteOlderThan` method is particularly dangerous — it provides a
convenient, sanctioned code path for bulk deletion that could be exploited if
the method is ever exposed through an API or scheduled job.

---

## 9. Gap Analysis and Recommendations

### 9.1 Priority action items

| # | Action | Effort | Priority | Impact |
|---|---|---|---|---|
| 1 | **Add `hash` column + populate hash chain at consume time** | 3 days | P0 | Enables all other tamper-detection layers |
| 2 | **Enforce append-only storage (triggers + REVOKE)** | 2 days | P0 | Blocks application-level UPDATE/DELETE |
| 3 | **Remove or gate `DeleteOlderThan`** | 0.5 days | P0 | Eliminates sanctioned tampering vector |
| 4 | **Add periodic chain verification worker** | 3 days | P1 | Detects direct DB tampering within 1 hour |
| 5 | **Publish Merkle roots to S3 Object Lock** | 5 days | P1 | External trust anchor, < 5 min detection window |
| 6 | **Integrate RFC 3161 timestamping** | 3 days | P2 | Legally admissible proof of log existence |
| 7 | **Wire tamper alerts to SIEM + halt-on-detect** | 2 days | P1 | Immediate incident response |

**Total estimated effort**: ~18.5 engineer-days for a complete tamper-evidence
system spanning all five defense layers.

### 9.2 Implementation order

```
Phase 1 (Week 1): Foundation
  ├── Add hash column to audit_events table (migration)
  ├── Compute chain hash in consumer before Insert
  ├── Store hash in DB alongside event
  └── Add VerifyChain gRPC API endpoint

Phase 2 (Week 1-2): Append-Only
  ├── Create INSERT-only DB role for the audit consumer
  ├── Add UPDATE/DELETE prevention triggers
  ├── Gate DeleteOlderThan behind admin-only role + audit logging
  └── Add partition-freezing for completed months

Phase 3 (Week 2): Detection
  ├── Real-time chain check on Insert (reject batch if chain breaks)
  ├── Background chain scanner (hourly full-tenant scan)
  ├── Multi-channel tamper alerting (SIEM CEF + admin email + halt)
  └── VerifyChain gRPC API exposed for external auditors

Phase 4 (Week 3): External Commitment
  ├── Merkle tree construction per batch
  ├── S3 Object Lock commitment publisher (per-batch)
  ├── External commitment verification endpoint
  └── RFC 3161 TSA integration (per Merkle root)

Phase 5 (Week 3): Hardening
  ├── Halt-on-detect for critical chain breaks
  ├── Forensic inclusion proof API (Merkle proof per event ID)
  ├── Integration tests with tamper simulation
  └── Documentation for auditors (how to verify GGID logs independently)
```

### 9.3 Key design decisions for GGID

1. **HMAC secret management**: The chain HMAC key must be stored in a KMS or
   HashiCorp Vault, not in the database. If the DB is compromised, the attacker
   must not be able to forge valid chain hashes.

2. **Per-tenant chains**: Use separate chain anchors per `tenant_id` to prevent
   cross-tenant chain splicing attacks. The genesis hash includes the tenant ID.

3. **Gap tolerance**: The chain verifier must handle legitimate gaps (consumer
   restart, NATS redelivery) without false positives. Use idempotent inserts
   (check event UUID before insert) and gap-tolerant verification that reports
   the first broken link without failing on missing events.

4. **Retention vs immutability tension**: GGID's `DeleteOlderThan` is needed for
   GDPR/retention compliance, but conflicts with immutability. Resolution: old
   partitions should be **archived to WORM storage** (S3 Object Lock) before
   being dropped from the DB. The archive preserves the chain for the full
   legal retention period.

5. **NATS as weak link**: NATS JetStream with `MaxAge: 72h` is only a buffer.
   If the consumer is down for > 72 hours, events are permanently lost — a gap
   in the chain. Consider increasing `MaxAge` or adding a dead-letter store for
   undeliverable audit events.

---

## References

- RFC 3161 — Internet X.509 PKI Time-Stamp Protocol (TSP)
- RFC 6962 — Certificate Transparency (Merkle tree logging)
- NIST SP 800-92 — Guide to Computer Security Log Management
- PCI DSS v4.0 Requirement 10 — Track and Monitor All Access
- AWS S3 Object Lock documentation — WORM storage patterns
- Certificate Transparency log specification — Merkle tree commitment design
