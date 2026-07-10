# Token Status List (TSL) — Research

> **Spec:** `draft-ietf-oauth-status-list-21` | **Status:** IETF Draft, Standards Track (2024–2025), near final
> **Authors:** T. Looker (MATTR), P. Bastian (Bundesdruckerei), C. Bormann (SPRIND)

---

## 1. Overview

**Token Status List (TSL)** defines a privacy-preserving, scalable mechanism for
representing revocation/status of tokens secured by JOSE or COSE — JWTs, SD-JWT
VCs, CWTs, and ISO mdocs.

### Problem Solved

Traditional mechanisms leak which credential is being checked:
- **CRL:** request reveals subject identity (certificate serial numbers)
- **OCSP:** request reveals serial number to a third-party responder
- **OCSP Stapling:** better privacy, but freshness is limited

### TSL Approach

The verifier fetches the **entire** status list (a compressed bitstring) and
checks a bit position locally — no phone-home. The issuer cannot tell which
credential is being verified; all credentials in the same list form an anonymity set.

### Key Properties

- **Privacy-first:** no per-credential query, herd anonymity
- **Scalable:** single list serves millions of credentials
- **Offline-capable:** cached list works for TTL window
- **Extensible:** registry for custom status types and future mechanisms

### Relationship to Existing Specs

- **Token Introspection (RFC 7662):** requires contacting issuer per token — TSL
  eliminates this by batching status into one fetchable artifact
- **SD-JWT VC:** references TSL for verifiable credential status
- **ISO mDL:** uses CWT-based TSL encoding in the Mobile Security Object (MSO)

---

## 2. Bitstring Format

A Status List is a compressed byte array where each credential occupies `bits`
bits (1, 2, 4, or 8). Position is defined by the credential's `idx` assigned at issuance.

### Bit Size Values

| `bits` | Statuses | Use Case |
|--------|----------|----------|
| 1 | 2 (valid/revoked) | Most common — simple revocation |
| 2 | 4 (adds suspended) | Suspension support |
| 4 | 16 | Rich status models |
| 8 | 256 | Full application-defined states |

### Encoding Pipeline

```
1. Build byte array: size = num_credentials * bits / 8
2. Set bit values for each credential at its idx (LSB first)
3. Compress with DEFLATE (RFC 1951) + ZLIB (RFC 1950)
4. Base64url-encode compressed bytes → "lst" field
```

### Example: 16 credentials, bits=1

```
Uncompressed: [0xB9, 0xA3]  (2 bytes)

JSON: { "bits": 1, "lst": "eNrbuRgAAhcBXQ" }
```

### Referenced Token with `status` Claim

```json
{
  "iss": "https://example.com/issuer",
  "iat": 1683000000, "exp": 1883000000,
  "sub": "6c5c0a49-b589-431d-bae7-219122a9ec2c",
  "status": {
    "status_list": { "idx": 0, "uri": "https://example.com/statuslists/1" }
  }
}
```

### Status List Token (JWT, `typ: statuslist+jwt`)

```json
{ "alg": "ES256", "kid": "12", "typ": "statuslist+jwt" }
.
{
  "sub": "https://example.com/statuslists/1",
  "iat": 1686920170, "exp": 2291720170, "ttl": 43200,
  "status_list": { "bits": 1, "lst": "eNrbuRgAAhcBXQ" }
}
```

The Status List Token is itself a signed JWT — integrity preserved even when
served by third-party CDNs or transferred offline.

---

## 3. Privacy Properties

### Why It's Privacy-Preserving

Verifiers fetch the entire list, not a single credential's status. This creates
**herd anonymity** — the issuer learns only that *some* credential from the list
is being verified, not *which one*.

| Mechanism | What's Revealed |
|-----------|----------------|
| CRL | Subject identity (via certificate in the list) |
| OCSP | Certificate serial number to responder |
| OCSP Stapling | No direct query, but freshness/privacy depends on presenter |
| **TSL** | **Nothing** — list fetch is anonymous, check is local |

### K-Anonymity Analysis

All credentials sharing the same status list form an **anonymity set**. Privacy
scales with set size: `privacy ∝ log₂(set_size)`.

- 100 credentials → weak (~6.6 bits)
- 10,000 → moderate (~13.3 bits)
- 1,000,000 → excellent (~19.9 bits)

**Recommendations:**
- Minimum 10,000+ credentials per list for meaningful anonymity
- Partition by credential type or issuance batch, not per-user
- Multiple lists are fine if each remains large enough
- Consider time-based partitioning (e.g., monthly) to bound size
- **Timing attacks:** fetch proactively/regularly, not on-demand per verification

---

## 4. JWT vs CWT Status List

### JWT Status List

- Referenced token carries `status.status_list` claim with `idx` + `uri`
- Verifier fetches Status List JWT (`typ: statuslist+jwt`) from `uri`
- Decompresses bitstring, checks bit at `idx`
- JSON-based, human-readable, widely supported

### CWT Status List (CBOR Web Token)

- Same conceptual model, CBOR-encoded for constrained devices
- Status reference in CWT claims (COSE-secured, claim key 65535)
- Status List CWT type: `application/statuslist+cwt`
- More compact binary representation (~30-40% smaller)
- Relevant for ISO mDL and IoT credentials

### Comparison

| Aspect | JWT Status List | CWT Status List |
|--------|----------------|-----------------|
| Encoding | JSON (text) | CBOR (binary) |
| Token format | JOSE (JWS/JWE) | COSE |
| `lst` field | base64url string | raw byte string |
| Status ref claim | `status.status_list` (JSON) | `status_list` (CBOR map, key 65535) |
| Content-Type | `application/statuslist+jwt` | `application/statuslist+cwt` |
| Size | Larger (base64 overhead) | ~30-40% smaller |
| Use case | Web, enterprise, OAuth | IoT, mDL, constrained devices |
| Signed by | JWS (e.g., ES256) | COSE_Sign1 |

---

## 5. Comparison with CRL/OCSP

| Mechanism | Privacy | Latency | Bandwidth | Offline | Complexity |
|-----------|---------|---------|-----------|---------|------------|
| **CRL** | Low (subject visible) | Medium (periodic) | Large | Yes | Low |
| **OCSP** | Low (serial visible) | Low (per-request) | Small | No | Medium |
| **OCSP Stapling** | Better | Low (pre-attached) | Medium | Limited | Medium |
| **Token Status List** | **High** (anonymous bulk) | **Low** (cached, ~16KB) | **Small** (compressed) | **Yes** (TTL window) | Medium |

**When to use each:**
- CRL: legacy PKI, TLS certificates
- OCSP: real-time revocation where privacy is secondary
- TSL: verifiable credentials, OAuth tokens, SD-JWT VCs, mDL — where privacy and scalability matter

---

## 6. GGID Integration Points

### GGID as Issuer (Status List Publisher)

**New endpoint:**
```
GET /oauth/statuslist/{list_id}
Accept: application/statuslist+jwt
→ 200 OK, Content-Type: application/statuslist+jwt
```

**Revocation flow:**
1. Admin/API calls revoke on a credential
2. Service sets the bit at the credential's assigned `idx`
3. Recompresses and re-signs the Status List Token
4. Updated list served on next fetch (CDN-cacheable)

**Storage:**
```sql
CREATE TABLE status_lists (
    id           UUID PRIMARY KEY,
    tenant_id    UUID NOT NULL,
    bits         BYTEA NOT NULL,         -- compressed bitstring
    total_bits   INTEGER NOT NULL,       -- e.g., 131072
    bit_width    SMALLINT DEFAULT 1,     -- 1, 2, 4, or 8
    last_updated TIMESTAMPTZ NOT NULL
);
CREATE TABLE credential_index (
    credential_id UUID PRIMARY KEY,
    list_id       UUID NOT NULL REFERENCES status_lists(id),
    idx           INTEGER NOT NULL,
    assigned_at   TIMESTAMPTZ DEFAULT NOW()
);
```

**Service interface (Go):**
```go
type StatusListService interface {
    AssignIndex(ctx context.Context, credID, listID uuid.UUID) (int, error)
    SetStatus(ctx context.Context, credID uuid.UUID, status uint8) error
    GetStatusListJWT(ctx context.Context, listID uuid.UUID) (string, error)
}
```

### GGID as Verifier

**Gateway middleware** checks token status on high-security operations:
```go
func (m *StatusListMiddleware) Wrap(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractBearerToken(r)
        if ref, ok := token.StatusListClaim(); ok {
            list, err := m.verifier.FetchCached(r.Context(), ref.URI)
            if err == nil && list.IsRevoked(ref.Idx) {
                http.Error(w, "token revoked", 401)
                return
            }
        }
        next.ServeHTTP(w, r)
    })
}
```

**Cache strategy:**
- Redis key: `statuslist:{list_id}` → compressed bytes + JWT
- TTL: honor `ttl` claim (default 5 min for hot paths)
- Lazy refresh on cache miss; background prefetch before TTL expiry

### Integration with Existing Revocation

| Mechanism | Scope | Use Case |
|-----------|-------|----------|
| **Redis revocation set** (current) | Internal tokens/sessions | Immediate revocation of GGID JWTs |
| **Token Status List** | Cross-domain | VCs, tokens shared with external RPs |

Complementary: Redis for internal (instant), TSL for cross-domain (privacy-preserving).
Gateway checks both.

---

## 7. Implementation Design

### Bit Operations

```go
func GetBit(data []byte, idx, bitWidth int) (uint8, error) {
    bitsPerByte := 8 / bitWidth
    byteOffset := idx / bitsPerByte
    if byteOffset >= len(data) {
        return 0, fmt.Errorf("index %d out of bounds", idx)
    }
    mask := uint8((1 << bitWidth) - 1)
    return (data[byteOffset] >> ((idx % bitsPerByte) * bitWidth)) & mask, nil
}

func SetBit(data []byte, idx, bitWidth int, value uint8) error {
    bitsPerByte := 8 / bitWidth
    byteOffset := idx / bitsPerByte
    if byteOffset >= len(data) {
        return fmt.Errorf("index %d out of bounds", idx)
    }
    bitOffset := (idx % bitsPerByte) * bitWidth
    mask := uint8((1 << bitWidth) - 1)
    data[byteOffset] &^= mask << bitOffset       // clear
    data[byteOffset] |= (value & mask) << bitOffset // set
    return nil
}
```

### Compress / Decompress

```go
func Compress(data []byte) ([]byte, error) {
    var buf bytes.Buffer
    w := zlib.NewWriter(&buf)
    w.Write(data); w.Close()
    return buf.Bytes(), nil
}

func Decompress(data []byte) ([]byte, error) {
    r, err := zlib.NewReader(bytes.NewReader(data))
    if err != nil { return nil, err }
    defer r.Close()
    return io.ReadAll(r)
}
```

### IssueStatusListJWT (Issuer)

```go
func (s *Service) IssueStatusListJWT(ctx context.Context, listID uuid.UUID) (string, error) {
    list, err := s.repo.Get(ctx, listID)
    if err != nil { return "", err }
    claims := jwt.MapClaims{
        "sub": fmt.Sprintf("%s/oauth/statuslist/%s", s.baseURL, listID),
        "iat": list.UpdatedAt.Unix(),
        "exp": list.UpdatedAt.Add(12 * time.Hour).Unix(),
        "ttl": 300,
        "status_list": map[string]any{
            "bits": list.BitWidth,
            "lst":  base64.RawURLEncoding.EncodeToString(list.Bits),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
    token.Header["typ"] = "statuslist+jwt"
    return token.SignedString(s.signingKey)
}
```

### FetchAndCache (Verifier)

```go
func (v *Verifier) FetchCached(ctx context.Context, uri string) (*DecodedStatusList, error) {
    cacheKey := "statuslist:" + uri
    if cached, err := v.redis.Get(ctx, cacheKey).Bytes(); err == nil {
        return v.parseToken(cached)
    }
    req, _ := http.NewRequestWithContext(ctx, "GET", uri, nil)
    req.Header.Set("Accept", "application/statuslist+jwt")
    resp, err := v.client.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    raw, _ := io.ReadAll(resp.Body)
    v.redis.Set(ctx, cacheKey, raw, 5*time.Minute)
    return v.parseToken(raw)
}
```

---

## 8. Roadmap

| Phase | Scope | Effort |
|-------|-------|--------|
| **1** | Status List issuer: endpoint + bitstring management + DB schema | 3 days |
| **2** | Verifier: Redis cache + gateway middleware + check logic | 2 days |
| **3** | Multi-tenant partitioning: per-tenant lists, lifecycle mgmt | 2 days |
| **4** | CWT support: CBOR encoding for IoT/mDL | 2 days |

**Phase 1-2: ~1 week** — delivers working issuer + verifier pipeline.

**Dependencies:** Go stdlib (`compress/zlib`, `encoding/base64`, `crypto/ecdsa`),
existing Redis. No new external dependencies.

**Testing:** unit (bit round-trip, compress/decompress, edge cases) → integration
(issue → revoke → verify) → E2E through gateway.

---

*References:* [draft-ietf-oauth-status-list-21](https://datatracker.ietf.org/doc/draft-ietf-oauth-status-list/) · [RFC 7662](https://datatracker.ietf.org/doc/html/rfc7662) · [SD-JWT VC](https://datatracker.ietf.org/doc/draft-ietf-oauth-sd-jwt-vc/) · [RFC 8392](https://datatracker.ietf.org/doc/html/rfc8392)
