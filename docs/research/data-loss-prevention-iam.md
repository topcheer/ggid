# Data Loss Prevention for IAM Systems

> Research document for the GGID project — a Go-based Identity and Access Management suite.
> Focus: preventing PII exfiltration through API responses, logs, audit trails, and database exports.

---

## Table of Contents

1. [DLP Threat Model for IAM](#1-dlp-threat-model-for-iam)
2. [PII Detection Patterns](#2-pii-detection-patterns)
3. [Field-Level Encryption](#3-field-level-encryption)
4. [Tokenization](#4-tokenization)
5. [Data Masking Policies](#5-data-masking-policies)
6. [Log DLP](#6-log-dlp)
7. [API Response DLP](#7-api-response-dlp)
8. [Database Query DLP](#8-database-query-dlp)
9. [GGID PII Handling Audit](#9-ggid-pii-handling-audit)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. DLP Threat Model for IAM

### Why IAM DLP Is Critical

An Identity and Access Management system is the central repository for **all** user PII within an organisation. Unlike a line-of-business application that stores operational data, the IAM stores the master record for every identity — name, email, phone, employment status, group memberships, authentication credentials, and often demographic data. A single exfiltration event from the IAM database can expose the complete identity profile of every employee, contractor, and customer in the organisation.

This makes the IAM the highest-value target for data theft. Attackers who gain even read access can:

- **Correlate identities across systems** — the IAM's user ID and email link to every downstream application.
- **Harvest credentials** — password hashes, MFA secrets, OAuth tokens, WebAuthn credentials.
- **Map organisational structure** — groups, roles, and hierarchy reveal reporting chains and access scopes.
- **Enable targeted phishing** — verified email addresses plus display names enable spear-phishing.
- **Facilitate account takeover** — knowing which accounts exist and their status narrows brute-force and credential-stuffing attacks.

### Where PII Leaks in IAM

| Vector | Description | Example |
|--------|-------------|---------|
| **API responses** | Endpoints returning full user objects with all fields, including phone, email, and internal IDs. | `GET /api/v1/users` returns `email`, `phone`, `primary_email_id` for every user. |
| **Error messages** | Database or ORM errors that include table names, column names, or query fragments. | `duplicate key value violates unique constraint "users_email_key"` reveals schema. |
| **Audit logs** | Audit events that capture full request bodies or response payloads in metadata. | `user.create` audit event stores the entire request body including plaintext password. |
| **Debug endpoints** | Health, metrics, or debug endpoints that echo configuration or environment variables. | `/debug/pprof` or `/healthz` leaking connection strings. |
| **Log files** | Application logs that include request/response bodies, stack traces, or structured fields with PII. | `log.Printf("login attempt for %s from %s", email, ip)` writes email to stdout. |
| **gRPC metadata** | Inter-service calls that pass user data in metadata fields visible in tracing systems. | Gateway → Identity call includes `X-User-Email` in gRPC metadata, captured by Jaeger. |
| **Database backups** | Unencrypted pg_dump or WAL archives stored in object storage. | Nightly `pg_dump` written to S3 bucket without SSE-KMS. |
| **Memory/core dumps** | Process crashes that write heap dumps containing decrypted PII. | Go panic handler writing goroutine dump with request body on stack. |

### Threat Actor Profiles

| Actor | Motivation | Access Method |
|-------|------------|---------------|
| **Insider (low-privilege)** | Browse colleague data, stalking, curiosity | Legitimate API access with over-broad list permissions |
| **Insider (admin)** | Mass export before leaving company | Database query tools, API scripting, backup access |
| **Compromised account** | Lateral movement, data harvesting | Stolen JWT or session token with valid scopes |
| **External attacker** | Identity theft, credential sale | SQL injection, SSRF, broken access control |
| **Third-party integration** | Unintended data exposure via SCIM/OAuth | Over-scoped API key, misconfigured webhook |

---

## 2. PII Detection Patterns

### Pattern-Based Detection

Pattern-based detection uses regular expressions to identify known PII formats within free-text data. GGID's existing `pkg/pii` package already implements basic regex-based masking for emails, phone numbers, IPs, UUIDs, SSNs, and credit cards.

| PII Type | Pattern | Confidence | False Positive Rate |
|----------|---------|------------|---------------------|
| Email | `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}` | High | Low |
| Phone (US) | `\+?1?[-.]?\(?[0-9]{3}\)?[-.]?[0-9]{3}[-.]?[0-9]{4}` | Medium | Medium (matches other 10-digit numbers) |
| SSN (US) | `\b\d{3}-\d{2}-\d{4}\b` | High | Low (requires dash format) |
| Credit Card | Luhn-validated 13-19 digit sequences | High (with Luhn) | Very Low |
| IP Address | `\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b` | Low | High (matches versions, IPs in logs) |
| IBAN | `[A-Z]{2}\d{2}[A-Z0-9]{1,30}` | High | Low |

### Context-Based Detection

Context-based detection examines field names and surrounding data structure rather than value patterns. This is critical for structured data (JSON, SQL columns) where the field name itself reveals the data type.

```go
// sensitiveFieldNames maps field name patterns to PII categories.
var sensitiveFieldNames = map[string]string{
    "email":         "email",
    "phone":         "phone",
    "mobile":        "phone",
    "ssn":           "ssn",
    "social_security": "ssn",
    "address":       "address",
    "street":        "address",
    "zip":           "postal_code",
    "postal":        "postal_code",
    "dob":           "date_of_birth",
    "birthday":      "date_of_birth",
    "password":      "credential",
    "secret":        "credential",
    "token":         "credential",
    "api_key":       "credential",
    "passport":      "government_id",
    "license":       "government_id",
}

// isSensitiveField checks whether a JSON field name indicates PII.
func isSensitiveField(name string) (string, bool) {
    lower := strings.ToLower(name)
    for pattern, category := range sensitiveFieldNames {
        if strings.Contains(lower, pattern) {
            return category, true
        }
    }
    return "", false
}
```

### Luhn Check for Credit Cards

```go
// luhnValid performs the Luhn algorithm check on a digit string.
func luhnValid(number string) bool {
    // Strip non-digit characters.
    var digits []int
    for _, r := range number {
        if r >= '0' && r <= '9' {
            digits = append(digits, int(r-'0'))
        }
    }
    if len(digits) < 13 || len(digits) > 19 {
        return false
    }

    sum := 0
    alternate := false
    for i := len(digits) - 1; i >= 0; i-- {
        d := digits[i]
        if alternate {
            d *= 2
            if d > 9 {
                d -= 9
            }
        }
        sum += d
        alternate = !alternate
    }
    return sum%10 == 0
}
```

### Go PII Scanner

A comprehensive scanner that combines pattern-based and context-based detection:

```go
package dlp

import (
    "regexp"
    "strings"
)

// Finding represents a detected PII item.
type Finding struct {
    Field     string // JSON field name or "" for free-text
    Category  string // email, phone, ssn, credit_card, ip, credential
    Value     string // the matched value
    Position  int    // byte offset in the input
    Confidence float64
}

var (
    emailPattern     = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
    phonePattern     = regexp.MustCompile(`\+?[0-9]{1,3}?[-.\s]?\(?[0-9]{3}\)?[-.\s]?[0-9]{3,4}[-.\s]?[0-9]{4}`)
    ssnPattern       = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
    cardPattern      = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
    ipPattern        = regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)
    jwtPattern       = regexp.MustCompile(`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`)
    apiKeyPattern    = regexp.MustCompile(`(?i)(api[_-]?key|secret|password|token)\s*[=:]\s*\S+`)
)

// ScanText scans a free-text string for PII patterns.
func ScanText(input string) []Finding {
    var findings []Finding

    scan := func(re *regexp.Regexp, category string, conf float64) {
        for _, m := range re.FindAllStringIndex(input, -1) {
            val := input[m[0]:m[1]]
            if category == "credit_card" && !luhnValid(val) {
                continue // skip non-Luhn-valid numbers
            }
            findings = append(findings, Finding{
                Category:   category,
                Value:      val,
                Position:   m[0],
                Confidence: conf,
            })
        }
    }

    scan(emailPattern, "email", 0.95)
    scan(ssnPattern, "ssn", 0.90)
    scan(phonePattern, "phone", 0.60)
    scan(cardPattern, "credit_card", 0.85)
    scan(ipPattern, "ip", 0.40)
    scan(jwtPattern, "credential", 0.95)
    scan(apiKeyPattern, "credential", 0.70)

    return findings
}

// ScanJSON recursively scans a map[string]any for sensitive fields.
func ScanJSON(data map[string]any, prefix string) []Finding {
    var findings []Finding
    for key, val := range data {
        fullKey := key
        if prefix != "" {
            fullKey = prefix + "." + key
        }
        if category, ok := isSensitiveField(key); ok {
            if s, ok := val.(string); ok && s != "" {
                findings = append(findings, Finding{
                    Field:      fullKey,
                    Category:   category,
                    Value:      s,
                    Confidence: 0.99,
                })
            }
        }
        // Recurse into nested objects.
        if nested, ok := val.(map[string]any); ok {
            findings = append(findings, ScanJSON(nested, fullKey)...)
        }
        // Check string values for embedded PII.
        if s, ok := val.(string); ok {
            textFindings := ScanText(s)
            for i := range textFindings {
                textFindings[i].Field = fullKey
            }
            findings = append(findings, textFindings...)
        }
    }
    return findings
}
```

### ML-Based Detection

For unstructured data (free-text notes, support tickets, chat messages), regex patterns miss context-dependent PII. ML-based detection uses Named Entity Recognition (NER) models to identify:

- Person names (PER)
- Organisations (ORG)
- Locations (GPE)
- Dates that could be dates of birth
- Medical record numbers

In a Go environment, ML-based detection is typically deployed as a sidecar service (Python/gRPC) that the Go application calls:

```go
// NERClient calls an external NER service for unstructured PII detection.
type NERClient struct {
    endpoint string
    client   *http.Client
}

func (c *NERClient) DetectPII(ctx context.Context, text string) ([]Finding, error) {
    req, _ := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/detect",
        strings.NewReader(`{"text":"`+text+`"}`))
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Findings []Finding `json:"findings"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return result.Findings, nil
}
```

---

## 3. Field-Level Encryption

### Concept

Field-level encryption (FLE) encrypts specific PII columns at the application layer before writing to the database. Unlike full-database encryption (TDE), FLE ensures that PII is encrypted even if an attacker gains direct database access (e.g., via SQL injection, backup theft, or a compromised DBA account).

### What to Encrypt

| Field | Encryption | Blind Index | Rationale |
|-------|------------|-------------|-----------|
| email | AES-256-GCM | Yes (HMAC-SHA256) | Needs equality search for login and uniqueness |
| phone | AES-256-GCM | Yes | Needs equality search for lookup |
| date_of_birth | AES-256-GCM | No | Rarely searched by exact value |
| display_name | AES-256-GCM | No | Free-text, not searched |
| address_line1 | AES-256-GCM | No | Free-text |

### Blind Index for Encrypted Search

Encrypted data cannot be searched directly. A **blind index** solves this by storing a deterministic, truncated HMAC alongside the ciphertext. To search for `email = 'alice@example.com'`, compute the blind index of the search term and query `WHERE email_blind_idx = $1`.

```go
package dlp

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/hmac"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "io"
)

// FieldCipher provides AES-256-GCM encryption and blind index generation.
type FieldCipher struct {
    encKey  []byte // 32 bytes for AES-256
    idxKey  []byte // 32 bytes for HMAC blind index
    idxBits int    // truncated HMAC bits for blind index (default 80)
}

// NewFieldCipher creates a FieldCipher from two 32-byte keys.
func NewFieldCipher(encKey, idxKey []byte) (*FieldCipher, error) {
    if len(encKey) != 32 || len(idxKey) != 32 {
        return nil, fmt.Errorf("keys must be 32 bytes")
    }
    return &FieldCipher{encKey: encKey, idxKey: idxKey, idxBits: 80}, nil
}

// Encrypt encrypts plaintext and returns base64-encoded ciphertext.
func (fc *FieldCipher) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(fc.encKey)
    if err != nil {
        return "", err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded ciphertext.
func (fc *FieldCipher) Decrypt(b64 string) (string, error) {
    ciphertext, err := base64.StdEncoding.DecodeString(b64)
    if err != nil {
        return "", err
    }
    block, err := aes.NewCipher(fc.encKey)
    if err != nil {
        return "", err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    ns := gcm.NonceSize()
    if len(ciphertext) < ns {
        return "", fmt.Errorf("ciphertext too short")
    }
    plaintext, err := gcm.Open(nil, ciphertext[:ns], ciphertext[ns:], nil)
    if err != nil {
        return "", err
    }
    return string(plaintext), nil
}

// BlindIndex generates a deterministic, truncated HMAC for equality search.
// The truncation prevents rainbow-table attacks on the index.
func (fc *FieldCipher) BlindIndex(plaintext string) string {
    mac := hmac.New(sha256.New, fc.idxKey)
    mac.Write([]byte(plaintext))
    full := mac.Sum(nil)
    // Truncate to idxBits/8 bytes (default 80 bits = 10 bytes).
    byteLen := fc.idxBits / 8
    return base64.StdEncoding.EncodeToString(full[:byteLen])
}
```

### Database Schema

```sql
ALTER TABLE users
    ADD COLUMN email_encrypted TEXT,
    ADD COLUMN email_blind_idx  TEXT,
    ADD COLUMN phone_encrypted  TEXT,
    ADD COLUMN phone_blind_idx  TEXT;

-- Equality search uses blind index
SELECT id, email_encrypted FROM users WHERE email_blind_idx = $1;
```

---

## 4. Tokenization

### Concept

Tokenization replaces PII with non-sensitive tokens. The real value is stored in a secure vault or derived deterministically. Unlike encryption, tokens cannot be reversed without access to the vault or the derivation key. This is preferred for compliance with PCI-DSS, where the goal is to ensure that a database breach does not expose card data.

### Tokenization Strategies

| Strategy | Reversible | Deterministic | Format-Preserving | Use Case |
|----------|------------|---------------|-------------------|----------|
| Vault tokenization | Yes (via vault) | No | No | PAN storage, SSN |
| FPE (FF1/FF3) | Yes (via key) | Yes | Yes | Credit cards in legacy systems |
| Stateless (HMAC) | No | Yes | No | Analytics, deduplication |

### Vault Tokenization

A token vault is a separate, hardened data store that maps tokens to real values. The application never sees the real value after tokenization; detokenization requires explicit vault access.

```go
package dlp

import (
    "crypto/hmac"
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "sync"
)

// VaultTokenizer stores token→value mappings in memory.
// Production would use a separate database or HSM-backed vault.
type VaultTokenizer struct {
    mu       sync.RWMutex
    store    map[string]string // token → plaintext
    reverse  map[string]string // plaintext → token (for deterministic issuance)
    hmacKey  []byte
}

func NewVaultTokenizer(key []byte) *VaultTokenizer {
    return &VaultTokenizer{
        store:   make(map[string]string),
        reverse: make(map[string]string),
        hmacKey: key,
    }
}

// Tokenize replaces plaintext with a token. If the same plaintext was
// tokenized before, the same token is returned (deterministic within vault).
func (vt *VaultTokenizer) Tokenize(plaintext string) (string, error) {
    vt.mu.Lock()
    defer vt.mu.Unlock()

    // Check if already tokenized.
    if token, ok := vt.reverse[plaintext]; ok {
        return token, nil
    }

    // Generate a random token.
    raw := make([]byte, 16)
    if _, err := rand.Read(raw); err != nil {
        return "", err
    }
    token := "tok_" + hex.EncodeToString(raw)

    vt.store[token] = plaintext
    vt.reverse[plaintext] = token
    return token, nil
}

// Detokenize retrieves the original value. Only accessible to authorised services.
func (vt *VaultTokenizer) Detokenize(token string) (string, bool) {
    vt.mu.RLock()
    defer vt.mu.RUnlock()
    val, ok := vt.store[token]
    return val, ok
}

// Delete removes a token-value pair (for data retention compliance).
func (vt *VaultTokenizer) Delete(token string) {
    vt.mu.Lock()
    defer vt.mu.Unlock()
    if val, ok := vt.store[token]; ok {
        delete(vt.store, token)
        delete(vt.reverse, val)
    }
}
```

### Format-Preserving Tokenization (FPE)

FPE produces tokens that maintain the format of the original value (same length, same character set). This is essential for legacy systems that validate field formats.

```go
// FPETokenizer uses HMAC-based FPE for demonstration.
// Production should use NIST-approved FF1 or FF3 algorithms.
type FPETokenizer struct {
    key []byte
}

// TokenizeCard returns a token that looks like a credit card number.
func (ft *FPETokenizer) TokenizeCard(pan string) string {
    mac := hmac.New(sha256.New, ft.key)
    mac.Write([]byte(pan))
    digest := mac.Sum(nil)

    // Take the first 16 digits from the HMAC output, preserving format.
    var token []byte
    for i := 0; i < 16 && i < len(digest); i++ {
        token = append(token, '0'+digest[i]%10)
    }
    // Preserve last 4 digits for customer recognition.
    suffix := pan[len(pan)-4:]
    copy(token[12:], suffix)

    // Fix the Luhn check digit.
    // (Implementation omitted for brevity.)

    return string(token)
}
```

### Stateless Tokenization

Stateless tokenization uses a deterministic HMAC so no vault is needed. It is irreversible (one-way) and used when only deduplication or equality comparison is needed, never display.

```go
func StatelessToken(plaintext string, key []byte) string {
    mac := hmac.New(sha256.New, key)
    mac.Write([]byte(plaintext))
    return hex.EncodeToString(mac.Sum(nil))[:32]
}
```

---

## 5. Data Masking Policies

### Masking Strategies

| Strategy | Input | Output | Use Case |
|----------|-------|--------|----------|
| Partial mask | `john@example.com` | `j***@example.com` | UI display, logs |
| Full mask | `+1-555-123-4567` | `************` | High-sensitivity fields in logs |
| Redaction | `4532-1234-5678-9012` | `REDACTED` | Any field, conservative default |
| Hash | `john@example.com` | `a1b2c3d4e5...` | Deduplication without exposure |
| Truncation | `John Smith` | `John S.` | Display names in shared contexts |

### Per-Role Masking Policy

Different roles require different levels of PII visibility:

```go
package dlp

import (
    "encoding/json"
    "net/http"
    "strings"
)

// MaskPolicy defines masking rules per role per field.
type MaskPolicy struct {
    // role → field → mask type
    rules map[string]map[string]MaskType
}

type MaskType int

const (
    MaskNone MaskType = iota // no masking (show full value)
    MaskPartial              // partial mask (j***@example.com)
    MaskFull                 // full mask (***)
    MaskRedact               // replace with "REDACTED"
    MaskHide                 // remove field entirely
)

func NewMaskPolicy() *MaskPolicy {
    return &MaskPolicy{
        rules: map[string]map[string]MaskType{
            "admin": {
                "email":         MaskNone,
                "phone":         MaskNone,
                "display_name":  MaskNone,
                "password_hash": MaskHide,
                "ssn":           MaskPartial,
            },
            "manager": {
                "email":         MaskPartial,
                "phone":         MaskFull,
                "display_name":  MaskNone,
                "password_hash": MaskHide,
                "ssn":           MaskRedact,
            },
            "user": {
                "email":         MaskPartial,
                "phone":         MaskFull,
                "display_name":  MaskNone,
                "password_hash": MaskHide,
                "ssn":           MaskHide,
            },
            "anonymous": {
                "email":         MaskHide,
                "phone":         MaskHide,
                "display_name":  MaskPartial,
                "password_hash": MaskHide,
                "ssn":           MaskHide,
            },
        },
    }
}

// ApplyMask masks a value based on role and field name.
func (mp *MaskPolicy) ApplyMask(role, field, value string) any {
    fieldRules, ok := mp.rules[role]
    if !ok {
        return MaskValue(MaskRedact, value) // default: redact for unknown roles
    }
    maskType, ok := fieldRules[field]
    if !ok {
        return value // unknown fields pass through
    }
    return MaskValue(maskType, value)
}

// MaskValue applies a mask type to a value.
func MaskValue(mt MaskType, value string) any {
    switch mt {
    case MaskNone:
        return value
    case MaskPartial:
        return partialMask(value)
    case MaskFull:
        return strings.Repeat("*", len(value))
    case MaskRedact:
        return "REDACTED"
    case MaskHide:
        return nil // field will be omitted in JSON
    default:
        return "REDACTED"
    }
}

// partialMask applies context-aware partial masking.
func partialMask(value string) string {
    if strings.Contains(value, "@") {
        return maskEmail(value)
    }
    if len(value) <= 4 {
        return strings.Repeat("*", len(value))
    }
    return string(value[0]) + "***" + string(value[len(value)-2:])
}

func maskEmail(email string) string {
    parts := strings.SplitN(email, "@", 2)
    if len(parts) != 2 || len(parts[0]) == 0 {
        return "***"
    }
    return string(parts[0][0]) + "***@" + parts[1]
}
```

### Response Masking Middleware

```go
// MaskingMiddleware applies per-role field masking to JSON API responses.
func MaskingMiddleware(policy *MaskPolicy, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        role := r.Header.Get("X-User-Role") // set by gateway after JWT validation
        if role == "" {
            role = "anonymous"
        }

        mw := &maskingWriter{
            ResponseWriter: w,
            role:           role,
            policy:         policy,
        }
        next.ServeHTTP(mw, r)

        // Apply masking to buffered JSON response.
        if mw.buffer.Len() > 0 {
            masked := mw.applyMasking()
            w.Header().Set("Content-Type", "application/json")
            w.Write(masked)
        }
    })
}

type maskingWriter struct {
    http.ResponseWriter
    buffer bytes.Buffer
    role   string
    policy *MaskPolicy
    headerWritten bool
}

func (mw *maskingWriter) Write(b []byte) (int, error) {
    return mw.buffer.Write(b) // buffer instead of writing through
}

func (mw *maskingWriter) applyMasking() []byte {
    var data map[string]any
    if err := json.Unmarshal(mw.buffer.Bytes(), &data); err != nil {
        return mw.buffer.Bytes() // not JSON, pass through
    }
    maskedData := mw.maskMap(data)
    result, _ := json.Marshal(maskedData)
    return result
}

func (mw *maskingWriter) maskMap(data map[string]any) map[string]any {
    result := make(map[string]any)
    for key, val := range data {
        masked := mw.policy.ApplyMask(mw.role, key, fmt.Sprintf("%v", val))
        if masked != nil {
            result[key] = masked
        }
        // Recurse into nested objects.
        if nested, ok := val.(map[string]any); ok {
            result[key] = mw.maskMap(nested)
        }
    }
    return result
}
```

---

## 6. Log DLP

### The Problem

Application logs are a primary exfiltration channel. Developers log debug information, error handlers log stack traces, and middleware logs request/response bodies. Any of these can contain PII that persists in log aggregation systems (ELK, Datadog, Splunk) long after the data was removed from the application database.

### Secret Redaction

The highest-priority items to redact are secrets that enable account takeover:

```go
package dlp

import (
    "regexp"
    "strings"
)

// LogRedactor removes PII and secrets from log messages.
type LogRedactor struct {
    patterns []*redactionPattern
}

type redactionPattern struct {
    re       *regexp.Regexp
    category string
    replace  string
}

func NewLogRedactor() *LogRedactor {
    return &LogRedactor{
        patterns: []*redactionPattern{
            // JWT tokens (three base64 segments separated by dots).
            {re: regexp.MustCompile(`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`),
             category: "jwt", replace: "[REDACTED:JWT]"},
            // Password fields in key=value or JSON contexts.
            {re: regexp.MustCompile(`(?i)("?password"?\s*[:=]\s*"?)([^"\s,}]+)`),
             category: "password", replace: "${1}[REDACTED]"},
            // API keys.
            {re: regexp.MustCompile(`(?i)(api[_-]?key\s*[:=]\s*)(\S+)`),
             category: "api_key", replace: "${1}[REDACTED]"},
            // Bearer tokens.
            {re: regexp.MustCompile(`(?i)(bearer\s+)(\S+)`),
             category: "bearer", replace: "${1}[REDACTED]"},
            // Authorization headers.
            {re: regexp.MustCompile(`(?i)(authorization\s*[:=]\s*)(\S+)`),
             category: "auth_header", replace: "${1}[REDACTED]"},
            // Emails.
            {re: regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
             category: "email", replace: "[REDACTED:EMAIL]"},
            // Phone numbers.
            {re: regexp.MustCompile(`\+?[0-9]{1,3}?[-.\s]?\(?[0-9]{3}\)?[-.\s]?[0-9]{3,4}[-.\s]?[0-9]{4}`),
             category: "phone", replace: "[REDACTED:PHONE]"},
            // SSN.
            {re: regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
             category: "ssn", replace: "[REDACTED:SSN]"},
            // Credit card numbers.
            {re: regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`),
             category: "credit_card", replace: "[REDACTED:CARD]"},
        },
    }
}

// Redact applies all redaction patterns to a log message.
func (lr *LogRedactor) Redact(input string) string {
    for _, p := range lr.patterns {
        if strings.Contains(p.replace, "${1}") {
            // Preserve the prefix, redact only the value.
            input = p.re.ReplaceAllString(input, strings.Replace(p.replace, "${1}", "$1", 1))
        } else {
            input = p.re.ReplaceAllString(input, p.replace)
        }
    }
    return input
}
```

### Structured Log Redaction

For structured logging (zerolog, zap), field-level redaction is more precise:

```go
// RedactFields redacts sensitive keys in a structured log field map.
func RedactFields(fields map[string]any) map[string]any {
    sensitiveKeys := map[string]bool{
        "password": true, "token": true, "jwt": true, "api_key": true,
        "secret": true, "authorization": true, "cookie": true,
        "email": true, "phone": true, "ssn": true,
    }

    result := make(map[string]any, len(fields))
    for key, val := range fields {
        lower := strings.ToLower(key)
        if sensitiveKeys[lower] || strings.Contains(lower, "password") ||
           strings.Contains(lower, "token") || strings.Contains(lower, "secret") {
            result[key] = "[REDACTED]"
        } else {
            result[key] = val
        }
    }
    return result
}
```

### Integration with GGID

GGID's `pkg/pii.Obfuscate()` already provides regex-based masking for email, phone, IP, UUID, SSN, and credit card. It should be wired into:

1. The audit publisher's `Metadata` field before serialisation.
2. All `log.Printf` calls that include user data.
3. The gateway's access log middleware.
4. gRPC interceptors for inter-service calls.

---

## 7. API Response DLP

### Over-Posting (Mass Assignment) Prevention

Mass assignment occurs when an API accepts more fields than intended, allowing clients to set fields they should not control (e.g., setting `is_admin` or `email_verified` via a user-update endpoint).

GGID's identity handler already mitigates this by using separate input structs with only the allowed fields:

```go
// In http.go updateUser — only phone, display_name, locale, timezone are accepted.
var req struct {
    Phone       *string `json:"phone"`
    DisplayName *string `json:"display_name"`
    Locale      *string `json:"locale"`
    Timezone    *string `json:"timezone"`
}
```

### Per-Endpoint Field Whitelists

Each API endpoint should define which fields it is allowed to return. Fields not in the whitelist are stripped before serialisation:

```go
package dlp

import (
    "encoding/json"
    "net/http"
)

// ResponseFilter strips non-whitelisted fields from JSON API responses.
type ResponseFilter struct {
    // endpoint pattern → allowed fields
    whitelists map[string][]string
}

func NewResponseFilter() *ResponseFilter {
    return &ResponseFilter{
        whitelists: map[string][]string{
            "/api/v1/users":    {"id", "username", "display_name", "status", "created_at"},
            "/api/v1/users/me": {"id", "username", "email", "phone", "display_name", "status", "locale", "timezone", "created_at"},
            "/api/v1/users/*":  {"id", "username", "display_name", "status"},
        },
    }
}

// FilterMiddleware applies field whitelists to responses.
func (rf *ResponseFilter) FilterMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Determine the whitelist for this path.
        allowed := rf.getWhitelist(r.URL.Path)
        if allowed == nil {
            next.ServeHTTP(w, r) // no filtering if no whitelist defined
            return
        }

        fw := &filteringWriter{
            ResponseWriter: w,
            allowedFields:  allowed,
        }
        next.ServeHTTP(fw, r)

        // Filter the buffered response.
        if fw.buffer.Len() > 0 {
            filtered := rf.filterResponse(fw.buffer.Bytes(), allowed)
            w.Header().Set("Content-Type", "application/json")
            w.Write(filtered)
        }
    })
}

func (rf *ResponseFilter) getWhitelist(path string) []string {
    // Try exact match first.
    if wl, ok := rf.whitelists[path]; ok {
        return wl
    }
    // Try wildcard patterns.
    for pattern, wl := range rf.whitelists {
        if strings.HasSuffix(pattern, "/*") {
            prefix := strings.TrimSuffix(pattern, "/*")
            if strings.HasPrefix(path, prefix+"/") {
                return wl
            }
        }
    }
    return nil
}

func (rf *ResponseFilter) filterResponse(data []byte, allowed []string) []byte {
    var obj map[string]any
    if err := json.Unmarshal(data, &obj); err != nil {
        return data // not a JSON object, pass through
    }
    // Handle list responses.
    if users, ok := obj["users"].([]any); ok {
        for i, u := range users {
            if m, ok := u.(map[string]any); ok {
                obj["users"].([]any)[i] = filterMap(m, allowed)
            }
        }
    } else {
        obj = filterMap(obj, allowed)
    }
    result, _ := json.Marshal(obj)
    return result
}

func filterMap(data map[string]any, allowed []string) map[string]any {
    allowedSet := make(map[string]bool, len(allowed))
    for _, f := range allowed {
        allowedSet[f] = true
    }
    result := make(map[string]any)
    for key, val := range data {
        if allowedSet[key] {
            result[key] = val
        }
    }
    return result
}
```

### Projection-Based Filtering

A more flexible approach uses query parameters to let clients request specific fields (GraphQL-style projection for REST):

```
GET /api/v1/users/123?fields=id,username,display_name
```

```go
// ApplyProjection filters a response map to only include requested fields.
func ApplyProjection(data map[string]any, fields []string) map[string]any {
    if len(fields) == 0 {
        return data // no projection requested
    }
    result := make(map[string]any, len(fields))
    for _, f := range fields {
        if val, ok := data[f]; ok {
            result[f] = val
        }
    }
    return result
}
```

---

## 8. Database Query DLP

### Bulk Export Prevention

A single compromised API key can exfiltrate an entire user database via repeated `GET /api/v1/users?page_size=1000` calls. Database-level DLP prevents this.

### Query Volume Limiter

```go
package dlp

import (
    "context"
    "fmt"
    "sync"
    "time"
)

// QueryLimiter limits the number of rows a client can retrieve within a time window.
type QueryLimiter struct {
    mu          sync.Mutex
    counters    map[string]*rowCounter // clientID → counter
    maxRows     int                     // max rows per window
    window      time.Duration
}

type rowCounter struct {
    count     int
    resetTime time.Time
}

func NewQueryLimiter(maxRows int, window time.Duration) *QueryLimiter {
    return &QueryLimiter{
        counters: make(map[string]*rowCounter),
        maxRows:  maxRows,
        window:   window,
    }
}

// CheckLimit verifies the client has not exceeded the row quota.
// Returns an error if the limit is exceeded.
func (ql *QueryLimiter) CheckLimit(ctx context.Context, clientID string, requestedRows int) error {
    ql.mu.Lock()
    defer ql.mu.Unlock()

    now := time.Now()
    counter, ok := ql.counters[clientID]
    if !ok || now.After(counter.resetTime) {
        counter = &rowCounter{count: 0, resetTime: now.Add(ql.window)}
        ql.counters[clientID] = counter
    }

    if counter.count+requestedRows > ql.maxRows {
        return fmt.Errorf("DLP: row limit exceeded (%d/%d rows used in current window)",
            counter.count, ql.maxRows)
    }

    counter.count += requestedRows
    return nil
}

// QueryDLP wraps a database list function with volume limiting.
func (ql *QueryLimiter) QueryDLP(
    ctx context.Context,
    clientID string,
    pageSize int,
    queryFn func() (int, error), // returns total rows
) (int, error) {
    if err := ql.CheckLimit(ctx, clientID, pageSize); err != nil {
        return 0, err
    }
    return queryFn()
}
```

### Export Detection Heuristics

Beyond volume limits, pattern-based detection flags suspicious access:

| Pattern | Threshold | Action |
|---------|-----------|--------|
| Sequential pagination through all users | >5 consecutive pages in 60s | Alert + throttle |
| Filter returning >10% of total users | Ratio check | Require admin scope |
| Repeated queries for different user IDs | >50 distinct IDs in 5 min | Alert |
| Access outside business hours | Time-based rule | Log for review |
| Access from new geo-location | IP geo lookup | Step-up auth |

```go
// ExportDetector monitors query patterns for exfiltration behaviour.
type ExportDetector struct {
    mu       sync.Mutex
    sessions map[string]*querySession
    maxPages int
    window   time.Duration
}

type querySession struct {
    pages     int
    distinctUsers map[string]bool
    startedAt time.Time
    lastQuery time.Time
}

// RecordQuery tracks a list-query request for export detection.
func (ed *ExportDetector) RecordQuery(clientID, userID string) (bool, string) {
    ed.mu.Lock()
    defer ed.mu.Unlock()

    now := time.Now()
    session, ok := ed.sessions[clientID]
    if !ok || now.Sub(session.startedAt) > ed.window {
        session = &querySession{
            distinctUsers: make(map[string]bool),
            startedAt:     now,
        }
        ed.sessions[clientID] = session
    }

    session.pages++
    session.lastQuery = now
    if userID != "" {
        session.distinctUsers[userID] = true
    }

    // Alert on excessive pagination.
    if session.pages > ed.maxPages {
        return true, fmt.Sprintf("excessive pagination: %d pages in %v",
            session.pages, now.Sub(session.startedAt))
    }

    // Alert on mass user ID access.
    if len(session.distinctUsers) > 50 {
        return true, fmt.Sprintf("mass user access: %d distinct users in %v",
            len(session.distinctUsers), now.Sub(session.startedAt))
    }

    return false, ""
}
```

### Backup DLP

```go
// EncryptBackup encrypts a database backup using AES-256-GCM before writing to storage.
func EncryptBackup(plaintext []byte, key []byte) ([]byte, error) {
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
    return gcm.Seal(nonce, nonce, plaintext, nil), nil
}
```

---

## 9. GGID PII Handling Audit

### Existing PII Infrastructure (`pkg/pii`)

GGID has a `pkg/pii` package that provides:

- **MaskEmail**: `user@example.com` → `u***@e***.com`
- **MaskPhone**: `+1-234-567-8901` → `************8901`
- **MaskIP**: `192.168.1.100` → `192.168.x.x`
- **MaskUUID**: `550e8400-e29b-...` → `550e8400-****-...`
- **Obfuscate**: applies all masking rules to a free-text string

The package uses regex to find PII patterns and replaces them with masked versions. It has comprehensive test coverage (96.6%).

### Findings

#### 1. `pkg/pii` Is Not Wired Into Any Runtime Code Path

The grep search `pii\.(Obfuscate|MaskEmail|MaskPhone|MaskIP|MaskUUID)` returns **zero hits in Go source code** — only in research docs. The masking functions exist and are tested but are never called at runtime. This means:

- Audit events published via `pkg/audit/publisher.go` serialize `Metadata map[string]any` without any PII redaction.
- The `ActorName` field in audit events can contain user emails or display names in plaintext.
- No log statements use `pii.Obfuscate` before writing to stdout.

**Risk**: HIGH. All PII that flows through audit events and log statements is stored in plaintext.

#### 2. Identity Service API Returns All User Fields

The `userToJSON` function in `services/identity/internal/server/http.go` returns **every** field of the user object:

```go
func userToJSON(u *domain.User) map[string]any {
    m := map[string]any{
        "id":             u.ID.String(),
        "tenant_id":      u.TenantID.String(),
        "username":       u.Username,
        "email":          u.Email,       // PII: plaintext
        "phone":          u.Phone,       // PII: plaintext
        "status":         string(u.Status),
        "email_verified": u.EmailVerified,
        "display_name":   u.DisplayName,
        "locale":         u.Locale,
        "timezone":       u.Timezone,
        "created_at":     u.CreatedAt,
        "updated_at":     u.UpdatedAt,
    }
```

This function is called for:
- `GET /api/v1/users/{id}` — single user
- `GET /api/v1/users` — list all users (paginated, default 50 per page)
- `POST /api/v1/users` — after creation
- `POST /api/v1/users/{id}/lock`, `/unlock`, `/activate`, `/deactivate` — status changes
- `POST /api/v1/users/me` — self-service profile

**Risk**: MEDIUM. Any authenticated user with list permissions sees full email and phone of all users. There is no per-role masking or field filtering.

#### 3. `PasswordHash` Is Not Exposed in API Responses

The `User` domain struct contains `PasswordHash string`, but `userToJSON` does **not** include it. This is correct and safe.

**Status**: OK.

#### 4. `LastLoginIP` Is Not Exposed in API Responses

The `User` struct has `LastLoginIP *netip.Addr`, which is not included in `userToJSON`. However, the audit event's `IPAddress` field **is** populated and is not masked.

**Status**: Partially OK. API does not leak IP, but audit events do.

#### 5. Import CSV Handler Returns Error Messages with PII

The `handleImportCSV` function returns per-line results that include error messages from `h.svc.CreateUser`. If CreateUser fails with "user already exists", the error includes the username/email:

```go
results = append(results, importResult{Line: i + 1, Status: "error", Message: err.Error()})
```

**Risk**: LOW. Only accessible to admins, but error messages should be sanitised.

#### 6. Audit Event Structure Contains Potentially Sensitive Metadata

The `Event.Metadata map[string]any` field in `pkg/audit/publisher.go` is a free-form map. There is no schema enforcement or PII filtering on this field. Services that populate Metadata with request bodies, user objects, or error details will write PII to the audit stream in plaintext.

**Risk**: HIGH. The audit stream is persisted to PostgreSQL and NATS JetStream (72-hour retention).

#### 7. No Field-Level Encryption

User PII (email, phone) is stored in PostgreSQL in plaintext. No application-layer encryption or blind index is applied. If an attacker gains direct database access (SQL injection, backup theft, compromised DBA), all PII is immediately readable.

**Risk**: HIGH for compliance (GDPR, CCPA), MEDIUM for operational security.

#### 8. Log Statements Are Minimal

The identity service has only four `log.Printf` calls (in `server.go` and `cmd/main.go`), all for server lifecycle messages. No request/response bodies are logged. This is good but also means there is no structured logging infrastructure that could be extended with DLP filtering.

**Status**: LOW risk currently, but will become a problem when structured logging is added.

---

## 10. Gap Analysis & Recommendations

### Summary of Gaps

| # | Gap | Severity | Current State |
|---|-----|----------|---------------|
| G1 | `pkg/pii.Obfuscate()` never called at runtime | HIGH | Code exists, zero callers |
| G2 | No field-level encryption for PII columns | HIGH | All PII stored in plaintext |
| G3 | API returns all user fields without masking | MEDIUM | No per-role filtering |
| G4 | Audit Metadata has no PII filtering | HIGH | Free-form map, no schema |
| G5 | No query volume limiter for bulk export | MEDIUM | Unbounded pagination |
| G6 | No tokenization for compliance-regulated data | MEDIUM | Not implemented |

### Action Items

#### Action 1: Wire `pkg/pii.Obfuscate` into audit publisher and log middleware
**Effort**: 2 days

Apply `pii.Obfuscate()` to the `ActorName` field and recursively to `Metadata` values in `audit.Publisher.Publish()` before marshalling. Add a `slog.Handler` wrapper that calls `Obfuscate` on all string attributes. This immediately reduces PII leakage in the two highest-risk vectors (audit trail and log files) with minimal code changes.

#### Action 2: Add per-role response masking middleware to the gateway
**Effort**: 3 days

Implement the `MaskingMiddleware` described in Section 5. Wire it into the gateway's handler chain after JWT verification so that the role is available. Define masking policies for `email`, `phone`, and `display_name` with different rules for `admin`, `manager`, `user`, and `anonymous` roles. The identity service's `userToJSON` function should remain unchanged — masking happens at the gateway level, keeping it centralised.

#### Action 3: Implement field-level encryption for email and phone columns
**Effort**: 5 days

Add the `FieldCipher` from Section 3 to `pkg/crypto`. Modify the identity repository to encrypt email and phone before writing and decrypt after reading. Add `email_blind_idx` and `phone_blind_idx` columns for equality search. Update migrations. This protects PII at rest even if the database is directly compromised. Requires a key management strategy (environment variable for development, KMS/HSM for production).

#### Action 4: Add query volume limiter to identity list endpoint
**Effort**: 2 days

Implement the `QueryLimiter` from Section 8. Track per-client row counts with a sliding window. Set a default limit of 5000 rows per 5-minute window per authenticated principal. Return HTTP 429 with a `Retry-After` header when the limit is exceeded. This prevents silent bulk export via API pagination.

#### Action 5: Add PII schema enforcement to audit event Metadata
**Effort**: 3 days

Define a `MetadataSchema` that whitelists allowed keys per action type (e.g., `user.login` allows `method`, `mfa_used`, `ip_address` but not `email` or `password`). The audit publisher validates Metadata against the schema before publishing. Unknown keys are dropped. This prevents services from accidentally embedding full request bodies or user objects in audit events.

---

## Appendix: Regulatory Context

| Regulation | Requirement | Relevant Section |
|------------|-------------|------------------|
| GDPR Art. 32 | Encryption of personal data at rest | Field-Level Encryption (S3) |
| GDPR Art. 25 | Data protection by design | API Response DLP (S7), Masking (S5) |
| CCPA | Right to deletion, data minimisation | Tokenization (S4), Field Filtering (S7) |
| HIPAA | PHI access logging and minimum necessary | Log DLP (S6), Query DLP (S8) |
| PCI-DSS Req. 3 | Protect stored cardholder data | Tokenization (S4), Field-Level Encryption (S3) |
| SOC 2 CC6.1 | Logical access controls over data | All sections |
| ISO 27001 A.8.2 | Information classification and handling | Masking Policies (S5), API DLP (S7) |

---

*Document version: 1.0 | Last updated: 2025-07-11 | Author: GGID Security Research*
