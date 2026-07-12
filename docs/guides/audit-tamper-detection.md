# Audit Tamper Detection

Hash chain construction, verification API, forensics workflow, tamper evidence types, insertion/reorder detection, and recovery procedures.

## Overview

GGID's audit log uses a cryptographic hash chain to detect tampering. Each event is linked to the previous, creating an immutable chain. Any modification, insertion, or deletion breaks the chain and is immediately detectable.

## Hash Chain Construction

```
Event₁ → hash₁ = SHA256(event₁_data || genesis_hash)
Event₂ → hash₂ = SHA256(event₂_data || hash₁)
Event₃ → hash₃ = SHA256(event₃_data || hash₂)
...
Eventₙ → hashₙ = SHA256(eventₙ_data || hashₙ₋₁)
```

### Implementation

```go
type ChainEntry struct {
    Sequence   int64  // Monotonic sequence number
    EventID    string // UUID of the audit event
    EventHash  string // SHA256 of event data + prev hash
    PrevHash   string // Previous entry's EventHash
    Timestamp  time.Time
}

func computeHash(event AuditEvent, prevHash string) string {
    payload := fmt.Sprintf("%s|%s|%s|%s|%s",
        event.EventID,
        event.Timestamp.Format(time.RFC3339Nano),
        event.Action,
        event.Actor.UserID,
        prevHash,
    )
    h := sha256.Sum256([]byte(payload))
    return hex.EncodeToString(h[:])
}
```

### Genesis Block

```sql
INSERT INTO audit_chain (sequence, event_id, event_hash, prev_hash, timestamp)
VALUES (0, 'genesis', '00000000000000000000000000000000', '', '2025-01-01T00:00:00Z');
```

## Verification API

### Full Chain Verification

```bash
GET /api/v1/audit/verify-chain
# → {
#   "status": "valid",
#   "entries_verified": 1452390,
#   "broken_links": 0,
#   "verification_time_ms": 3400,
#   "last_verified_sequence": 1452390
# }
```

### Range Verification

```bash
GET /api/v1/audit/verify-chain?from=2025-01-01&to=2025-01-31
# → {"status": "valid", "entries_verified": 45000, "broken_links": 0}
```

### Verification Logic

```go
func VerifyChain(from, to int64) (ChainVerificationResult, error) {
    entries := store.GetChainRange(from, to)
    brokenLinks := []BrokenLink{}
    
    expectedPrev := getHashAt(from - 1)
    
    for _, entry := range entries {
        // Recompute hash
        event := store.GetEvent(entry.EventID)
        recomputed := computeHash(event, entry.PrevHash)
        
        // Check 1: Hash matches
        if recomputed != entry.EventHash {
            brokenLinks = append(brokenLinks, BrokenLink{
                Sequence: entry.Sequence,
                Type: "hash_mismatch",
            })
        }
        
        // Check 2: Chain continuity
        if entry.PrevHash != expectedPrev {
            brokenLinks = append(brokenLinks, BrokenLink{
                Sequence: entry.Sequence,
                Type: "chain_break",
            })
        }
        
        expectedPrev = entry.EventHash
    }
    
    return ChainVerificationResult{
        Status:         len(brokenLinks) == 0 ? "valid" : "broken",
        BrokenLinks:    brokenLinks,
    }, nil
}
```

## Tamper Evidence Types

### Type 1: Event Modification

```
Original: Event₁ data = "user.login jane@corp.com"
Tampered: Event₁ data = "user.login admin@corp.com"

Detection: Hash₁ no longer matches recomputed hash
Evidence: Stored hash ≠ recomputed hash
```

### Type 2: Event Deletion

```
Original: Event₁ → Event₂ → Event₃
Tampered: Event₁ → Event₃ (Event₂ deleted)

Detection: Event₃.PrevHash references Event₂, but Event₂ missing
Evidence: Chain break at Event₃
```

### Type 3: Event Insertion

```
Original: Event₁ → Event₂ → Event₃
Tampered: Event₁ → Event_FAKE → Event₂ → Event₃

Detection: Event_FAKE.PrevHash ≠ Event₁.EventHash
         OR Event₂.PrevHash ≠ Event_FAKE.EventHash (if attacker recomputed)
         OR sequence numbers are non-contiguous
Evidence: Chain break or sequence gap
```

### Type 4: Event Reorder

```
Original: Event₁ → Event₂ → Event₃
Tampered: Event₂ → Event₁ → Event₃

Detection: Hash chain depends on order — any reorder breaks hashes
Evidence: All hashes from reorder point become invalid
```

## Forensics Workflow

### Step 1: Detect

```bash
# Automated daily verification
cron: "0 3 * * *"
GET /api/v1/audit/verify-chain
# → If broken_links > 0 → security alert
```

### Step 2: Isolate

```bash
# Freeze audit database (read-only mode)
ALTER DATABASE ggid SET default_transaction_read_only = on;

# Snapshot for forensic analysis
pg_dump ggid > /forensics/audit_snapshot_$(date +%Y%m%d).dump
```

### Step 3: Analyze

```bash
# Get details of broken links
GET /api/v1/audit/verify-chain?detail=true
# → {
#   "broken_links": [
#     {
#       "sequence": 1452380,
#       "type": "hash_mismatch",
#       "stored_hash": "abc123...",
#       "recomputed_hash": "def456...",
#       "event_id": "evt-xyz",
#       "event_timestamp": "2025-01-15T10:30:00Z"
#     }
#   ]
# }
```

### Step 4: Determine Scope

```sql
-- Find all events around the break point
SELECT * FROM audit_events 
WHERE created_at BETWEEN '2025-01-15T10:29:00' AND '2025-01-15T10:31:00'
ORDER BY created_at;

-- Compare with backup to find what changed
SELECT * FROM audit_events_backup 
WHERE event_id = 'evt-xyz';
```

### Step 5: Report

```json
{
  "incident": "audit_tamper_detected",
  "severity": "critical",
  "break_point": "sequence 1452380",
  "tamper_type": "hash_mismatch",
  "affected_events": 1,
  "evidence": "Stored hash differs from recomputed hash",
  "timestamp": "2025-01-15T10:30:00Z",
  "action": "SIEM alerted, database frozen, forensic snapshot taken"
}
```

## Recovery Procedures

### If Tampering Confirmed (Malicious)

1. **Do NOT repair the chain** — the break IS the evidence
2. Export forensic snapshot
3. Notify compliance/legal team (may need to disclose breach)
4. Restore from last known-good backup
5. Investigate how tampering occurred (DB access audit)
6. Harden access controls

### If Chain Corrupted (Non-Malicious)

```bash
# Repair chain from verified backup
POST /api/v1/audit/chain/rebuild
{
  "from_backup": "2025-01-14-nightly",
  "replay_events_since": "2025-01-14T03:00:00Z"
}
# → Reconstructs chain from backup + replays uncorrupted events
```

## Performance

| Operation | Time | Optimization |
|-----------|------|-------------|
| Append (per event) | <0.1ms | Hash computation only |
| Full verify (1M events) | ~3s | Parallel batch verification |
| Range verify (1 day) | <100ms | Indexed by sequence |

## Monitoring

| Metric | Alert |
|--------|-------|
| Chain broken | CRITICAL → immediate security response |
| Verification latency | >5s for full scan → add indexes |
| Chain gap (missing sequence) | Any → data loss |
| Hash mismatch | Any → tampering detected |

## See Also

- [Audit Log Architecture](audit-log-architecture.md)
- [Audit Query API](audit-query-api.md)
- [SIEM Integration](siem-integration.md)
- [Compliance Framework Mapping](compliance-framework-mapping.md)
