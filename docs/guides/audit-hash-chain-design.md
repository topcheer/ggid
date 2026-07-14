# Audit Hash Chain Design

This guide covers the block structure, SHA-256 chaining, tamper detection, Merkle tree range proofs, re-anchoring, storage optimization, and verification API for GGID's audit hash chain.

## Overview

The audit hash chain provides cryptographic proof that audit events have not been tampered with. Each event is linked to the previous one via a SHA-256 hash, creating a tamper-evident chain. Any modification to a past event breaks the chain and is immediately detectable.

## Block Structure

### Event Block

Each audit event is wrapped in a block with chaining metadata:

```go
type AuditBlock struct {
    // Event data
    EventID     string          `json:"event_id"`
    EventType   string          `json:"event_type"`
    UserID      string          `json:"user_id"`
    TenantID    string          `json:"tenant_id"`
    Timestamp   time.Time       `json:"timestamp"`
    Payload     json.RawMessage `json:"payload"`

    // Chain metadata
    Sequence    int64           `json:"sequence"`     // Monotonic sequence number
    PrevHash    string          `json:"prev_hash"`    // Hash of previous block
    BlockHash   string          `json:"block_hash"`   // Hash of this block
    BlockTime   time.Time       `json:"block_time"`   // When block was created
}
```

### Block Hash Calculation

```go
func computeBlockHash(block *AuditBlock) string {
    // Hash the event data + chain metadata (excluding BlockHash itself)
    data := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s",
        block.EventID,
        block.EventType,
        block.UserID,
        block.TenantID,
        block.Timestamp.UnixNano(),
        block.Sequence,
        block.PrevHash,
    )
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}
```

## SHA-256 Chaining

### Chain Construction

```
Block 0 (Genesis)
  PrevHash: "0000000000000000000000000000000000000000000000000000000000000000"
  BlockHash: H(Event0 | Seq0 | PrevHash0)

Block 1
  PrevHash: BlockHash of Block 0
  BlockHash: H(Event1 | Seq1 | PrevHash1)

Block 2
  PrevHash: BlockHash of Block 1
  BlockHash: H(Event2 | Seq2 | PrevHash2)

... and so on
```

### Implementation

```go
type HashChain struct {
    mu       sync.Mutex
    lastHash string
    sequence int64
}

func NewHashChain() *HashChain {
    return &HashChain{
        lastHash: strings.Repeat("0", 64),  // Genesis hash
        sequence: 0,
    }
}

func (c *HashChain) AddBlock(event *AuditEvent) (*AuditBlock, error) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.sequence++
    block := &AuditBlock{
        EventID:   event.ID,
        EventType: event.Type,
        UserID:    event.UserID,
        TenantID:  event.TenantID,
        Timestamp: event.Timestamp,
        Payload:   event.Payload,
        Sequence:  c.sequence,
        PrevHash:  c.lastHash,
        BlockTime: time.Now(),
    }

    block.BlockHash = computeBlockHash(block)
    c.lastHash = block.BlockHash

    return block, nil
}
```

## Tamper Detection

### Verification Process

```go
func VerifyChain(blocks []*AuditBlock) (bool, int, error) {
    for i, block := range blocks {
        // Recompute hash
        expectedHash := computeBlockHash(block)
        if expectedHash != block.BlockHash {
            return false, i, fmt.Errorf("block %d hash mismatch: expected %s, got %s",
                i, expectedHash, block.BlockHash)
        }

        // Check chain linkage
        if i > 0 {
            if block.PrevHash != blocks[i-1].BlockHash {
                return false, i, fmt.Errorf("block %d prev_hash mismatch: chain broken",
                    i)
            }
        }

        // Check sequence
        if block.Sequence != int64(i+1) {
            return false, i, fmt.Errorf("block %d sequence mismatch: expected %d, got %d",
                i, i+1, block.Sequence)
        }
    }
    return true, -1, nil
}
```

### Tamper Scenario

```
Original chain:
  Block1 → Block2 → Block3 → Block4

Attacker modifies Block2's payload:
  Block1 → Block2' → Block3 → Block4

Verification:
  Block1: hash OK, prev OK
  Block2: hash MISMATCH (payload changed, hash no longer matches)
  → Tamper detected at Block2
```

## Merkle Tree Range Proofs

### Why Merkle Trees?

For large audit logs, verifying the entire chain is expensive. Merkle trees allow efficient range proofs — verifying a subset of events without processing the entire chain.

### Tree Construction

```
              Root Hash
             /         \
        H(AB)          H(CD)
        /   \           /  \
    H(A)   H(B)    H(C)   H(D)
     |       |       |       |
   Block1  Block2  Block3  Block4
```

### Range Proof

To prove blocks 2-3 are untampered:

```
Proof: [H(A), H(D), Root Hash]
Verification: H(H(A) + H(H(B) + H(C))) + H(D) = Root Hash
```

### Implementation

```go
type MerkleTree struct {
    leaves  [][]byte
    layers  [][][]byte
    root    []byte
}

func buildMerkleTree(blocks []*AuditBlock) *MerkleTree {
    // Build leaf layer
    leaves := make([][]byte, len(blocks))
    for i, block := range blocks {
        hash, _ := hex.DecodeString(block.BlockHash)
        leaves[i] = hash
    }

    tree := &MerkleTree{leaves: leaves}

    // Build up to root
    current := leaves
    for len(current) > 1 {
        next := make([][]byte, 0)
        for i := 0; i < len(current); i += 2 {
            if i+1 < len(current) {
                combined := append(current[i], current[i+1]...)
                hash := sha256.Sum256(combined)
                next = append(next, hash[:])
            } else {
                next = append(next, current[i])
            }
        }
        tree.layers = append(tree.layers, current)
        current = next
    }
    tree.root = current[0]
    return tree
}
```

## Re-anchoring

### Why Re-Anchor?

Over time, the hash chain grows. Re-anchoring periodically snapshots the current chain state to external storage (e.g., a blockchain, notarization service), providing an independent verification point.

### Re-Anchor Process

```
1. Every 24h: compute current chain head hash
2. Store head hash + sequence number in external system
3. Record anchor reference in audit log
4. On verification: compare external anchor with chain state
```

### Implementation

```go
func (c *HashChain) ReAnchor() error {
    c.mu.Lock()
    defer c.mu.Unlock()

    anchor := &Anchor{
        HeadHash:   c.lastHash,
        Sequence:   c.sequence,
        Timestamp:  time.Now(),
        ExternalRef: "",  // Filled by external system
    }

    // Submit to external notarization
    ref, err := notarizeSubmit(c.lastHash, c.sequence)
    if err != nil {
        return fmt.Errorf("notarization failed: %w", err)
    }
    anchor.ExternalRef = ref

    // Record anchor in audit log
    return storeAnchor(anchor)
}
```

### Verification with Anchors

```go
func VerifyWithAnchors(blocks []*AuditBlock, anchors []*Anchor) error {
    for _, anchor := range anchors {
        // Find block at anchor sequence
        if int(anchor.Sequence) > len(blocks) {
            continue
        }
        block := blocks[anchor.Sequence-1]

        // Verify chain hash matches anchor
        if block.BlockHash != anchor.HeadHash {
            return fmt.Errorf("anchor mismatch at sequence %d", anchor.Sequence)
        }

        // Verify external reference
        if err := notarizeVerify(anchor.ExternalRef, anchor.HeadHash); err != nil {
            return fmt.Errorf("external verification failed: %w", err)
        }
    }
    return nil
}
```

## Batch vs Per-Event

### Per-Event Chaining

Each event is immediately added to the chain:

| Pros | Cons |
|---|---|
| Immediate tamper evidence | High overhead per event |
| No delay in verification | Lock contention under high load |
| Simple implementation | Storage overhead for chain metadata |

### Batch Chaining

Events are collected and chained in batches (e.g., every 100 events or every 1 minute):

| Pros | Cons |
|---|---|
| Lower overhead | Delay in tamper detection |
| Better throughput | More complex implementation |
| Reduced lock contention | Batch must be sealed atomically |

### GGID Approach

```yaml
audit:
  hash_chain:
    mode: "batch"  # or "per_event"
    batch_size: 100      # Events per batch
    batch_timeout: 1m    # Max time before batch is sealed
    seal_on_shutdown: true  # Seal current batch on graceful shutdown
```

## Storage Optimization

### Compression

```go
// Store blocks in compressed batches
func storeBatch(batch []*AuditBlock) error {
    data, _ := json.Marshal(batch)
    compressed := gzipCompress(data)
    return db.Store("audit_batch:"+batch[0].Sequence, compressed)
}
```

### Tiered Storage

| Age | Storage | Access |
|---|---|---|
| 0-7 days | Hot (PostgreSQL) | Real-time query |
| 7-90 days | Warm (S3) | On-demand query |
| 90+ days | Cold (Glacier) | Batch retrieval |

### Indexing

```sql
CREATE INDEX idx_audit_tenant_time ON audit_events (tenant_id, timestamp DESC);
CREATE INDEX idx_audit_user_time ON audit_events (user_id, timestamp DESC);
CREATE INDEX idx_audit_type_time ON audit_events (event_type, timestamp DESC);
CREATE INDEX idx_audit_sequence ON audit_blocks (sequence);
```

## Verification API

### Endpoint

```bash
GET /api/v1/audit/verify-chain?start=1&end=1000
Authorization: Bearer <admin_token>

Response:
{
  "verified": true,
  "blocks_checked": 1000,
  "first_block": 1,
  "last_block": 1000,
  "head_hash": "abc123...",
  "anchors_verified": 2,
  "tamper_detected": false
}
```

### Single Block Verification

```bash
GET /api/v1/audit/verify-block/123
Authorization: Bearer <admin_token>

Response:
{
  "block_sequence": 123,
  "block_hash": "abc123...",
  "prev_hash": "def456...",
  "hash_valid": true,
  "chain_intact": true,
  "anchor_verified": true
}
```

## GGID Implementation

### Configuration

```yaml
audit:
  hash_chain:
    enabled: true
    algorithm: "sha256"
    mode: "batch"
    batch_size: 100
    batch_timeout: 1m
    re_anchor_interval: 24h
    anchor_provider: "internal"  # or "blockchain", "notary"
    storage:
      hot_retention: 7d
      warm_retention: 90d
      cold_retention: 7y
    verification:
      auto_verify_interval: 1h
      alert_on_tamper: true
```

### Hash Chain Service

```go
type HashChainService struct {
    chain    *HashChain
    store    AuditStore
    anchors  AnchorStore
    config   HashChainConfig
}

func (s *HashChainService) RecordEvent(event *AuditEvent) error {
    block, err := s.chain.AddBlock(event)
    if err != nil {
        return err
    }
    return s.store.StoreBlock(block)
}

func (s *HashChainService) VerifyRange(start, end int64) (*VerificationResult, error) {
    blocks, err := s.store.GetBlocks(start, end)
    if err != nil {
        return nil, err
    }
    valid, tamperedAt, err := VerifyChain(blocks)
    return &VerificationResult{
        Verified:     valid,
        BlocksChecked: len(blocks),
        TamperAt:     tamperedAt,
        Error:        err,
    }, nil
}
```

## Best Practices

1. **Seal batches atomically** — Don't leave partial batches
2. **Re-anchor periodically** — External verification point prevents chain rewrite
3. **Auto-verify regularly** — Run verification hourly to catch tamper early
4. **Alert on tamper** — Immediate security team notification
5. **Use tiered storage** — Don't keep years of audit data in hot storage
6. **Compress old blocks** — Reduce storage costs
7. **Index for query** — Optimize for audit queries, not just chain verification
8. **Handle genesis carefully** — The genesis block hash must be immutable
9. **Log anchor references** — Keep external verification references accessible
10. **Test tamper detection** — Regularly verify that modifications are caught
