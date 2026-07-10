# Password Cracking Defense for IAM Systems

> Focused research on **cracking resistance**: parameter tuning, pepper strategy,
> k-anonymity HIBP internals, entropy estimation, and migration architecture.
> For NIST SP 800-63B policy compliance, algorithm comparisons, and basic HIBP
> usage, see `docs/research/password-policy-best-practices.md`.

---

## 1. Password Cracking Threat Model

### 1.1 Offline Attacks (Post-Breach)

The most dangerous scenario: an attacker obtains the password database (hashes +
salts) through SQL injection, backup leak, insider threat, or supply-chain
compromise. At this point, rate limiting and lockout policies are irrelevant —
the attacker can test guesses at full hardware speed.

**Attack pipeline:**

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ DB Leak      │────►│ Hash extract │────►| Offline     │────► Plaintext
│ (SQLi, etc.) │     │ (salt+hash)  │     │ cracking    │     passwords
└──────────────┘     └──────────────┘     └──────────────┘
```

The attacker first runs **dictionary attacks** (top 10 million breached
passwords), then **rule-based attacks** (Hashcat rules: capitalize, append
digits, l33t substitution), and finally **mask attacks** for remaining hashes.

### 1.2 GPU Hashrates (Single RTX 4090)

| Algorithm | Parameters | Hashrate (H/s) | Time for 10M guesses |
|-----------|-----------|----------------|----------------------|
| **MD5** | — | ~164,000,000,000 | 0.06 seconds |
| **SHA-256** | — | ~22,000,000,000 | 0.45 seconds |
| **SHA-512** | — | ~5,400,000,000 | 1.9 seconds |
| **bcrypt** | cost=10 | ~17,000 | ~6.8 days |
| **bcrypt** | cost=12 | ~4,300 | ~27 days |
| **bcrypt** | cost=14 | ~1,070 | ~108 days |
| **bcrypt** | cost=16 | ~267 | ~434 days |
| **scrypt** | N=2^17, r=8, p=1 | ~1,900 | ~61 days |
| **PBKDF2** | 600K iters, SHA-256 | ~5,500 | ~21 days |
| **Argon2id** | m=64MB, t=3, p=2 | ~120 | ~966 days |

> **Key insight:** Argon2id at 64 MB is ~142x slower than bcrypt cost=10, and
> ~45Mx slower than SHA-256. The memory hardness is the critical factor — GPUs
> have limited high-bandwidth memory (24 GB on RTX 4090), so running 375+
> parallel instances at 64 MB each is physically impossible.

### 1.3 Time-to-Crack Reality

For a database of 1 million users with common passwords:

| Scenario | Hash | Single GPU | 8-GPU rig | Botnet (1000 GPUs) |
|----------|------|-----------|-----------|-------------------|
| Top 10K dict | bcrypt-12 | 2.5 hours | 19 min | 9 seconds |
| Top 10K dict | Argon2id 64MB | 23 hours | 2.9 hours | 84 seconds |
| Top 10M dict+rules | bcrypt-12 | 1,090 days | 136 days | 1.1 days |
| Top 10M dict+rules | Argon2id 64MB | 38,000 days | 4,750 days | 38 days |

Argon2id at 64 MB extends offline cracking from days to **decades**, even with
significant compute.

### 1.4 Online Attacks

| Attack Type | Description | Mitigation |
|-------------|-------------|------------|
| **Credential stuffing** | Use breached username/password pairs from other sites | Breach password checking, MFA, anomaly detection |
| **Password spraying** | Try one common password across many accounts | Account lockout, rate limiting, timing-constant responses |
| **Username enumeration** | Different error messages for valid vs. invalid users | Uniform error responses, timing-constant delays |

**Why rate limiting alone is insufficient:**
- Distributed attacks (botnets, residential proxies) bypass per-IP limits.
- Credential stuffing uses *correct* passwords from other breaches — no
  brute-force needed.
- An attacker with the database hash table is completely outside the online
  rate limiter's scope.
- Time-delayed lockouts can cause **denial-of-service**: an attacker locks
  legitimate users out by spraying their accounts.

Rate limiting is necessary but not sufficient. The real defense against offline
cracking is **memory-hard hashing + pepper + breach prevention**.

---

## 2. bcrypt Parameter Tuning

### 2.1 Cost Factor Selection

bcrypt's only tunable parameter is the **cost** (work factor): each increment
doubles the computation time.

| Cost | Hash time (server) | RTX 4090 hashrate | Time for 1M guesses | UX impact |
|------|-------------------|-------------------|---------------------|-----------|
| 10 | ~60 ms | ~17,000 H/s | ~59 sec | Negligible |
| 12 | ~240 ms | ~4,300 H/s | ~3.9 min | Slight login delay |
| 14 | ~960 ms | ~1,070 H/s | ~15.6 min | Noticeable |
| 16 | ~3.8 s | ~267 H/s | ~62 min | Unacceptable for login |

### 2.2 Choosing the Right Cost

- **Minimum acceptable today: cost 12.** OWASP recommends this as the floor.
- **Target for new deployments: cost 14.** ~1 second per hash is acceptable
  for registration and login (users tolerate sub-second waits).
- **Cost should increase over time** to track Moore's law. bcrypt's design
  allows this: the cost is embedded in the hash string, so old hashes verify
  at their original cost while new hashes use the current cost.

### 2.3 Go Implementation with Benchmark

```go
package main

import (
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHashWithCost hashes a password at the given cost factor.
func BcryptHashWithCost(password string, cost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(hash), nil
}

// BcryptVerify checks a password against a stored bcrypt hash.
func BcryptVerify(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// BcryptNeedsUpgrade returns true if the hash cost is below the target.
// Used for transparent migration to higher cost factors.
func BcryptNeedsUpgrade(hash string, targetCost int) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return true // malformed hash — force re-hash
	}
	return cost < targetCost
}

// BenchmarkBcryptCost measures hashing time at each cost level.
func BenchmarkBcryptCost() {
	password := "CorrectHorseBatteryStaple!"
	for _, cost := range []int{10, 11, 12, 13, 14, 15, 16} {
		start := time.Now()
		_, err := BcryptHashWithCost(password, cost)
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("cost=%d: error: %v\n", cost, err)
			continue
		}
		fmt.Printf("cost=%-2d  hash_time=%v  log2(H/s)≈%.1f\n",
			cost, elapsed, -1*(elapsed.Seconds())+30)
	}
}
```

### 2.4 bcrypt Limitations

- **72-byte password truncation:** bcrypt silently ignores bytes beyond 72.
  Pre-hash with SHA-256 or use HMAC-SHA256 if supporting long passphrases.
- **Not memory-hard:** GPU and ASIC attacks are cost-effective at scale.
  A single ASIC cluster can hit 100K+ H/s on bcrypt-12.
- **No parallelism within a single hash:** Each GPU core processes one hash
  independently, but a data center can parallelize across millions of hashes.

---

## 3. Argon2id Parameter Tuning

### 3.1 Parameter Overview

| Parameter | Symbol | Description | OWASP Recommended | GGID Current |
|-----------|--------|-------------|-------------------|--------------|
| Memory | m (KiB) | Memory per hash computation | 19,456 (19 MB) | 65,536 (64 MB) |
| Iterations | t | Number of passes over memory | 2 | 3 |
| Parallelism | p | Threads (lanes) | 1 | 2 |
| Key length | | Output hash length (bytes) | 32 | 32 |
| Salt length | | Random salt (bytes) | 16 | 16 |

### 3.2 Why Argon2id Over bcrypt

| Property | bcrypt | Argon2id |
|----------|--------|----------|
| Memory-hard | No | Yes (configurable) |
| GPU-resistant | Partial (CPU-bound) | Strong (memory bandwidth bottleneck) |
| ASIC-resistant | No | Yes (large SRAM needed) |
| Side-channel resistant | Yes (data-independent) | Hybrid (id mode) |
| Password length limit | 72 bytes | Unlimited |
| PHC winner | N/A | Yes (2015) |

**Argon2id** (hybrid mode) uses data-independent memory access for the first
half (resistant to timing side-channels) and data-dependent for the second half
(maximizes cracking cost). This is the mode recommended by RFC 9106.

### 3.3 Tuning Methodology

The goal is to maximize cracking cost while keeping server-side hash time under
~500ms. The tradeoff space:

```
  High memory (m) ──► GPU bottleneck (limited VRAM)
  High iterations (t) ──► CPU bottleneck (time scales linearly)
  High parallelism (p) ──► More lanes but needs matching hardware threads
```

**Recommended tuning procedure:**

1. Start with OWASP defaults: m=19456, t=2, p=1
2. Increase memory until hash time approaches 250-500ms
3. If memory hits system limits, increase t instead
4. Keep p=1 for single-tenant services; p=2-4 for high-throughput servers
5. Benchmark on your actual production hardware

### 3.4 Go Implementation

```go
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/argon2"
)

// Argon2idParams holds tunable Argon2id parameters.
type Argon2idParams struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	KeyLength   uint32
	SaltLength  uint32
}

// DefaultArgon2idParams returns OWASP-recommended parameters.
func DefaultArgon2idParams() Argon2idParams {
	return Argon2idParams{
		Memory:      19456, // 19 MB — OWASP minimum
		Iterations:  2,
		Parallelism: 1,
		KeyLength:   32,
		SaltLength:  16,
	}
}

// HighSecurityArgon2idParams returns high-security parameters (~350ms/hash).
func HighSecurityArgon2idParams() Argon2idParams {
	return Argon2idParams{
		Memory:      65536, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		KeyLength:   32,
		SaltLength:  16,
	}
}

// HashArgon2id produces an encoded hash string with embedded parameters.
// Format: argon2id$t=iter$m=memKB$p=par$salt$hash
func HashArgon2id(password string, params Argon2idParams) (string, error) {
	salt := make([]byte, params.SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password), salt,
		params.Iterations, params.Memory, params.Parallelism,
		params.KeyLength,
	)

	encoded := fmt.Sprintf("argon2id$t=%d$m=%d$p=%d$%s$%s",
		params.Iterations, params.Memory, params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

// BenchmarkArgon2idParams prints hash time for different parameter sets.
func BenchmarkArgon2idParams() {
	testCases := []struct {
		name   string
		params Argon2idParams
	}{
		{"OWASP minimum", {19456, 2, 1, 32, 16}},
		{"OWASP preferred", {65536, 2, 1, 32, 16}},
		{"High security", {65536, 3, 2, 32, 16}},
		{"Extreme", {131072, 4, 2, 32, 16}},
	}

	password := "Tr0ub4dour&3"
	for _, tc := range testCases {
		start := time.Now()
		_, _ = HashArgon2id(password, tc.params)
		elapsed := time.Since(start)
		fmt.Printf("%-20s m=%-7d t=%d p=%d  hash_time=%v\n",
			tc.name, tc.params.Memory, tc.params.Iterations,
			tc.params.Parallelism, elapsed)
	}
}
```

---

## 4. Pepper Strategy

### 4.1 What Is a Pepper?

A **pepper** is a server-side secret key that is combined with the user's
password *before* hashing. Unlike a salt (stored per-user), the pepper is the
same for all users and is **never stored in the database**.

```
Standard:   hash = Argon2id(password, salt)
Peppered:   hash = Argon2id(HMAC-SHA256(pepper, password), salt)
```

### 4.2 Why Pepper Stops Cracking on DB-Only Leaks

If an attacker steals only the database (hashes + salts), they cannot crack
passwords even with infinite compute because the pepper is unknown. Every guess
produces a wrong hash. The pepper must be obtained separately (from the
application config, KMS, or HSM).

**Threat model improvement:**

| Attack | Without pepper | With pepper |
|--------|---------------|-------------|
| DB leak only | Hashes crackable offline | **Uncrackable** |
| DB + app config leak | Hashes crackable offline | Crackable (pepper exposed) |
| DB + HSM compromise | Hashes crackable offline | Crackable |
| Full root compromise | Crackable | Crackable |

The pepper specifically defends against the most common breach scenario:
SQL injection or backup leak that exposes the database but not the application
server's filesystem or KMS.

### 4.3 HMAC-First vs. Concatenation

**HMAC-SHA256 first** (recommended):

```go
hmacInput := hmac.New(sha256.New, pepper)
hmacInput.Write([]byte(password))
pepperedPassword := hex.EncodeToString(hmacInput.Sum(nil))
hash := argon2id.HashArgon2id(pepperedPassword, params)
```

**Why HMAC over concatenation (`pepper + password`):**
- HMAC is a proven PRF — output distribution is uniform regardless of input.
- Concatenation has length-extension vulnerabilities with some hash functions
  (not with Argon2id, but defense-in-depth).
- HMAC output is fixed-length — no ambiguity in where the pepper ends and
  the password begins.
- HMAC handles arbitrary-length peppers and passwords cleanly.

### 4.4 Pepper Storage

| Storage | Security | Complexity | Notes |
|---------|----------|------------|-------|
| **HSM** (Hardware Security Module) | Highest | High | FIPS 140-2 Level 3+. Pepper never leaves hardware. |
| **Cloud KMS** (AWS KMS, GCP KMS, Vault) | High | Medium | Centralized key management, audit logs, IAM policies. |
| **Environment variable / config file** | Medium | Low | Simple but pepper is accessible to anyone with shell access. |
| **Multiple key shards** (Shamir's Secret Sharing) | High | Medium | Requires N of M shards to reconstruct. Split across servers. |

**Recommended:** Cloud KMS for managed deployments, HSM for regulated
environments (PCI DSS, HIPAA). For smaller deployments, environment variable
loaded from a secrets manager (Vault, AWS Secrets Manager) is acceptable.

### 4.5 Pepper Rotation

Pepper rotation re-hashes all user passwords with a new pepper. This is
expensive and disruptive:

```go
// Rotation requires knowing every user's plaintext password — impossible
// without a login event. Strategy:
// 1. Add "pepper_version" column to credentials table
// 2. On next successful login, re-hash with new pepper
// 3. Verify with old pepper until re-hashed
// 4. After migration window (e.g., 90 days), force password reset for
//    accounts still using the old pepper
```

### 4.6 Go Implementation

```go
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// PepperManager holds pepper keys and supports rotation.
type PepperManager struct {
	current  []byte    // active pepper for new hashes
	previous [][]byte  // old peppers still accepted for verification
}

func NewPepperManager(peppers ...[]byte) *PepperManager {
	if len(peppers) == 0 {
		panic("at least one pepper required")
	}
	return &PepperManager{
		current:  peppers[0],
		previous: peppers[1:],
	}
}

// ApplyPepper transforms a password using HMAC-SHA256 with the current pepper.
func (pm *PepperManager) ApplyPepper(password string) string {
	mac := hmac.New(sha256.New, pm.current)
	mac.Write([]byte(password))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyWithAllPeppers tries the current and all previous peppers.
// Returns the peppered password that matched, or empty string if none matched.
// Used during the rotation window.
func (pm *PepperManager) VerifyWithAllPeppers(password string) []string {
	var candidates []string
	candidates = append(candidates, pm.applyPepper(password, pm.current))
	for _, old := range pm.previous {
		candidates = append(candidates, pm.applyPepper(password, old))
	}
	return candidates
}

func (pm *PepperManager) applyPepper(password string, pepper []byte) string {
	mac := hmac.New(sha256.New, pepper)
	mac.Write([]byte(password))
	return hex.EncodeToString(mac.Sum(nil))
}

// Full peppered hashing pipeline:
//   1. HMAC-SHA256(pepper, password) → fixed-length hex string
//   2. Argon2id(peppered, salt, params) → final hash
func PepperedHashPassword(password string, pepper []byte, params Argon2idParams) (string, error) {
	mac := hmac.New(sha256.New, pepper)
	mac.Write([]byte(password))
	peppered := hex.EncodeToString(mac.Sum(nil))
	return HashArgon2id(peppered, params)
}

// PepperedVerifyPassword checks a password against a peppered hash.
func PepperedVerifyPassword(password, encoded string, pepper []byte) (bool, error) {
	mac := hmac.New(sha256.New, pepper)
	mac.Write([]byte(password))
	peppered := hex.EncodeToString(mac.Sum(nil))
	// Then verify the Argon2id hash of the peppered password
	// (reuse the standard VerifyPassword from pkg/crypto)
	return VerifyArgon2id(peppered, encoded)
}
```

---

## 5. k-Anonymity HIBP API in Depth

### 5.1 The Privacy Problem

Checking if a password is breached requires comparing the user's password
against a database of known-breached passwords. Sending the full password (or
full hash) to a third-party API creates a privacy risk — the API operator could
log it.

### 5.2 k-Anonymity Protocol

The k-anonymity model, designed by Junade Ali for HIBP, ensures the API never
sees enough information to identify the user's password:

```
Step 1: Client computes SHA-1(plaintext_password)
        e.g., "password" → 5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8

Step 2: Client sends ONLY first 5 hex characters to API
        GET https://api.pwnedpasswords.com/range/5BAA6

Step 3: Server returns ALL suffixes in that prefix range (~500-800 entries)
        Response body:
        1E4C9B93F3F0682250B6CF8331B7EE68FD8:3303003
        23D8B3C5C7A5E1B4F9D2A6E8C0B3D4F5A6B7C8D9:42
        ... (hundreds more)

Step 4: Client checks locally: does my full suffix appear in the response?
        1E4C9B93F3F0682250B6CF8331B7EE68FD8 → YES, count=3303003
        → Password is breached!
```

**Privacy guarantee:** The API receives only 5 hex chars = 16^5 = 1,048,576
possible prefixes. Each prefix maps to ~500+ real passwords, so the server
cannot distinguish the client's password from any other in the range. The
k-anonymity value k >= ~500.

### 5.3 Go Implementation with Caching

```go
package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// HIBPClient checks passwords against HIBP using k-anonymity.
type HIBPClient struct {
	httpClient *http.Client
	cache      *redis.Client  // nil = no caching
	baseURL    string         // default: https://api.pwnedpasswords.com
}

func NewHIBPClient(cache *redis.Client) *HIBPClient {
	return &HIBPClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cache:      cache,
		baseURL:    "https://api.pwnedpasswords.com",
	}
}

// IsBreached returns true if the password appears in known breaches.
// Uses k-anonymity: only sends first 5 hex chars of SHA-1 hash.
func (c *HIBPClient) IsBreached(ctx context.Context, password string) (bool, int, error) {
	// Step 1: compute SHA-1
	h := sha1.Sum([]byte(password))
	fullHash := strings.ToUpper(hex.EncodeToString(h[:]))

	// Step 2: split into prefix (5 chars) and suffix (35 chars)
	prefix := fullHash[:5]
	suffix := fullHash[5:]

	// Step 3: check Redis cache first
	if c.cache != nil {
		cached, err := c.cache.Get(ctx, "hibp:"+prefix).Result()
		if err == nil {
			return c.parseAndCheck(cached, suffix)
		}
	}

	// Step 4: fetch from HIBP API
	url := fmt.Sprintf("%s/range/%s", c.baseURL, prefix)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, 0, nil // fail-open: don't block on client errors
	}
	req.Header.Set("User-Agent", "GGID-IAM")
	req.Header.Set("Add-Padding", "true") // HIBP padding to prevent size-based analysis

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, 0, nil // fail-open
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, 0, nil // fail-open
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, 0, nil
	}
	bodyStr := string(body)

	// Step 5: cache the response (1 hour TTL — breach data changes slowly)
	if c.cache != nil {
		c.cache.Set(ctx, "hibp:"+prefix, bodyStr, time.Hour)
	}

	return c.parseAndCheck(bodyStr, suffix)
}

// parseAndCheck searches the response body for the given suffix.
func (c *HIBPClient) parseAndCheck(body, suffix string) (bool, int, error) {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[0] == suffix {
			var count int
			fmt.Sscanf(parts[1], "%d", &count)
			return true, count, nil
		}
	}
	return false, 0, nil
}
```

### 5.4 Caching and Rate Limiting

| Concern | Strategy |
|---------|----------|
| **Prefix caching** | Cache each prefix response in Redis with 1h TTL. ~1M possible prefixes, each ~25 KB = ~25 GB worst case. In practice, far fewer unique prefixes per deployment. |
| **Rate limiting** | HIBP API has no published rate limit, but good practice: max 10 requests/second per IP. Use a client-side rate limiter. |
| **Add-Padding header** | HIBP supports `Add-Padding: true` to pad responses to a uniform size, preventing response-size-based analysis. |
| **Fail-open policy** | If the API is unreachable, allow registration (fail-open) rather than blocking all signups during an outage. Log the failure for monitoring. |
| **Offline mode** | Download the full HIBP list (~30 GB compressed) and host locally. Suitable for air-gapped or high-volume environments. |

---

## 6. Password Entropy Estimation

### 6.1 Why Character Classes Fail

Traditional strength meters count character classes (uppercase, lowercase,
digits, symbols). This is misleading:

| Password | Classes | Strength | Reality |
|----------|---------|----------|---------|
| `Password1!` | 4/4 | "Strong" | In every breach dictionary. ~0 bits of effective entropy. |
| `correct-horse-battery-staple` | 2/4 | "Weak" | ~44 bits of entropy. ~109 years to crack with bcrypt-12. |
| `Tr0ub4dour&3` | 4/4 | "Strong" | ~28 bits. In crackstation wordlist with rules. |

### 6.2 zxcvbn Algorithm

Dropbox's zxcvbn is a **context-aware entropy estimator** that decomposes
passwords into recognizable patterns and sums their entropy:

1. **Dictionary matching:** Checks against ranked word lists (common passwords,
   English words, names, surnames). Each match contributes log2(rank) bits.
2. **L33t detection:** Maps `@`→`a`, `3`→`e`, `$`→`s`, etc. The substitution
   itself adds minimal entropy (1-2 bits).
3. **Keyboard pattern detection:** `qwerty`, `asdfgh`, `1qaz2wsx`. Entropy =
   log2(positions^length / repeats).
4. **Sequence detection:** `1234`, `abcd`, `zyxw`. Entropy based on sequence
   length.
5. **Repeat detection:** `aaaa`, `abcabc`. Entropy = log2(repeat_count).
6. **Date matching:** `1990`, `01012020`. Low entropy (~12-16 bits).
7. **Combination:** Dynamic programming finds the minimum-entropy decomposition
   across all matchers.

**Score interpretation:**

| Score | Guesses (log10) | Time to crack (bcrypt-12, single GPU) | Label |
|-------|-----------------|---------------------------------------|-------|
| 0 | < 10^3 | < 1 second | Very weak |
| 1 | 10^3–10^6 | Seconds to minutes | Weak |
| 2 | 10^6–10^8 | Minutes to hours | Fair |
| 3 | 10^8–10^10 | Hours to days | Strong |
| 4 | > 10^10 | Weeks+ | Very strong |

### 6.3 Enforcing Minimum Entropy

Rather than enforcing character classes, reject passwords below an entropy
threshold:

| Threshold | Policy |
|-----------|--------|
| **30 bits** | Minimum acceptable for low-security services |
| **40 bits** | Recommended minimum for IAM systems |
| **50 bits** | High-security (financial, healthcare) |

### 6.4 Go Implementation

```go
package main

import (
	"fmt"
	"math"

	"github.com/nbutton23/zxcvbn-go"
)

// EntropyResult holds the result of entropy estimation.
type EntropyResult struct {
	Score        int     // 0-4 (zxcvbn score)
	GuessesLog10 float64 // log10 of estimated guess count
	CrackingTime string  // human-readable time to crack
	Suggestions  []string // improvement suggestions
	EntropyBits  float64 // estimated entropy in bits
}

// EstimateEntropy evaluates password strength using zxcvbn.
// The userInputs parameter provides personal context (email, name, tenant)
// that should reduce the password's effective entropy.
func EstimateEntropy(password string, userInputs []string) EntropyResult {
	result := zxcvbn.PasswordStrength(password, userInputs)

	// guessesLog10 → entropy bits (log2 = log10 * log2(10))
	entropyBits := result.GuessesLog10 * math.Log2(10)

	return EntropyResult{
		Score:        result.Score,
		GuessesLog10: result.GuessesLog10,
		CrackingTime: result.CrackTimeDisplay,
		Suggestions:  result.Feedback.Suggestions,
		EntropyBits:  entropyBits,
	}
}

// MinEntropyValidator rejects passwords below the given entropy threshold.
func MinEntropyValidator(password string, userInputs []string, minBits float64) error {
	result := EstimateEntropy(password, userInputs)
	if result.EntropyBits < minBits {
		return fmt.Errorf(
			"password entropy is %.1f bits (minimum required: %.1f). "+
				"Suggestions: %v",
			result.EntropyBits, minBits, result.Suggestions,
		)
	}
	return nil
}

// Usage during registration:
//   err := MinEntropyValidator(password,
//       []string{userEmail, userFirstName, userLastName, tenantName},
//       40.0, // minimum 40 bits of entropy
//   )
//   if err != nil { return err }
```

### 6.5 User Input Context

zxcvbn accepts a `userInputs` array — words that should be treated as zero-cost
matches. Always include:

- User's email address (and common substrings)
- First name, last name
- Tenant/organization name
- Username

This catches passwords like `john@Acme2024!` that pass character-class checks
but are trivially guessable.

---

## 7. Cracking-Resistant Password Storage Architecture

### 7.1 Defense-in-Depth Stack

```
┌─────────────────────────────────────────────────────────────┐
│                  REGISTRATION / PASSWORD CHANGE              │
│                                                              │
│  1. Entropy check (zxcvbn >= 40 bits)                        │
│  2. Breach check (HIBP k-anonymity)                          │
│  3. Apply pepper: HMAC-SHA256(pepper, password)              │
│  4. Hash: Argon2id(peppered_password, salt, m=64MB, t=3)     │
│  5. Store: hash in DB, pepper in KMS                         │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│                        LOGIN VERIFICATION                     │
│                                                              │
│  1. Apply pepper: HMAC-SHA256(pepper, password)              │
│  2. Argon2id verify (constant-time comparison)               │
│  3. If legacy hash: transparent upgrade on success           │
│  4. Rate limit + anomaly detection                           │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│                      HONEYPOT DEFENSE                         │
│                                                              │
│  - Deception hashes: fake credentials that trigger alerts    │
│  - If honeypot hash is attempted: alert security team        │
│  - Attacker likely has the DB dump                           │
└─────────────────────────────────────────────────────────────┘
```

### 7.2 Legacy Hash Migration Handler

When upgrading from bcrypt to Argon2id (or increasing Argon2id parameters),
use transparent re-hashing on login:

```go
package main

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HashMigrator handles transparent password hash upgrades.
type HashMigrator struct {
	targetParams  Argon2idParams
	pepperManager *PepperManager
}

// VerifyAndMaybeUpgrade verifies a password against any hash format.
// If the hash is a legacy format (bcrypt or old Argon2id params),
// it re-hashes with current parameters after successful verification.
func (hm *HashMigrator) VerifyAndMaybeUpgrade(
	password, storedHash string,
) (verified bool, newHash string, err error) {

	// Detect hash format by prefix
	switch {
	case strings.HasPrefix(storedHash, "argon2id"):
		// Current format — verify directly
		peppered := hm.pepperManager.ApplyPepper(password)
		verified, err = VerifyArgon2id(peppered, storedHash)
		if verified {
			// Check if params need upgrade
			if hm.argon2idNeedsUpgrade(storedHash) {
				newHash, err = HashArgon2id(peppered, hm.targetParams)
			}
		}
		return verified, newHash, err

	case strings.HasPrefix(storedHash, "$2a$") || strings.HasPrefix(storedHash, "$2b$"):
		// Legacy bcrypt format
		peppered := hm.pepperManager.ApplyPepper(password)
		bcryptErr := bcrypt.CompareHashAndPassword(
			[]byte(storedHash), []byte(peppered),
		)
		if bcryptErr != nil {
			return false, "", nil
		}
		// Upgrade to Argon2id
		newHash, err = HashArgon2id(peppered, hm.targetParams)
		if err != nil {
			return true, "", fmt.Errorf("hash upgrade failed: %w", err)
		}
		return true, newHash, nil

	default:
		// Unknown format — try plain bcrypt as fallback
		bcryptErr := bcrypt.CompareHashAndPassword(
			[]byte(storedHash), []byte(password),
		)
		if bcryptErr == nil {
			newHash, err = HashArgon2id(
				hm.pepperManager.ApplyPepper(password),
				hm.targetParams,
			)
			return true, newHash, err
		}
		return false, "", nil
	}
}

func (hm *HashMigrator) argon2idNeedsUpgrade(hash string) bool {
	// Parse stored memory and iterations, compare to target
	// If memory < target or iterations < target → needs upgrade
	// (implementation depends on hash format)
	return false // simplified
}
```

### 7.3 Honeypot / Deception Hashes

```go
// DeceptionHashChecker monitors login attempts against known-fake accounts.
// If a deception credential is targeted, the database has likely been
// compromised and the attacker is testing extracted hashes.
type DeceptionHashChecker struct {
	canaryHashes map[string]string // identifier → fake hash
	alertSink    func(identifier string)
}

func (d *DeceptionHashChecker) IsCanary(identifier string) bool {
	_, ok := d.canaryHashes[identifier]
	return ok
}

func (d *DeceptionHashChecker) OnCanaryHit(identifier string) {
	if d.alertSink != nil {
		d.alertSink(identifier)
	}
}

// Deployment:
//   1. Create 5-10 fake user accounts with random passwords
//   2. Their hashes are in the DB but no human ever logs in with them
//   3. If any canary receives a login attempt → security incident
//   4. Alert: page the on-call, rotate pepper, force password reset for all users
```

---

## 8. GGID Password Hashing Audit

### 8.1 Current Implementation (`pkg/crypto/crypto.go`)

```go
const (
    argonMemory      = 64 * 1024 // 64 MB
    argonIterations  = 3
    argonParallelism = 2
    argonKeyLength   = 32
    argonSaltLength  = 16
)
```

**Strengths:**

| Aspect | Status | Detail |
|--------|--------|--------|
| Algorithm | Strong | Argon2id — the recommended choice (RFC 9106) |
| Memory (64 MB) | Strong | Exceeds OWASP minimum (19 MB), matches preferred tier |
| Iterations (3) | Strong | Exceeds OWASP minimum (2) |
| Parallelism (2) | Good | Reasonable for server-side |
| Key length (32 bytes) | Strong | 256-bit output |
| Salt (16 bytes, crypto/rand) | Strong | Per-password random salt |
| Constant-time comparison | Implemented | `constantTimeCompare()` in `crypto.go:144` |
| Hash format encodes params | Good | `argon2id$iter$mem$par$salt.hash` |
| Breach check (HIBP) | Exists | `password_breach.go` implements k-anonymity |

**Weaknesses:**

| Aspect | Status | Risk | Detail |
|--------|--------|------|--------|
| **No pepper** | Missing | High | DB-only leak leaves hashes fully crackable |
| **Parameters hardcoded** | Suboptimal | Medium | Cannot tune m/t/p at runtime via config or env vars |
| **Breach check not wired** | Missing | High | `CheckPasswordBreach` exists but `Register()` does not call it |
| **HIBP client: no timeout** | Weak | Medium | Uses `http.DefaultClient` (no timeout) |
| **HIBP client: no caching** | Missing | Medium | Every registration queries the API — DDoS risk and latency |
| **HIBP: no Add-Padding header** | Missing | Low | Response-size analysis possible without padding |
| **No entropy estimation** | Missing | Medium | Relies on composition rules instead of real entropy |
| **No bcrypt migration path** | Missing | Medium | If legacy bcrypt hashes exist, no transparent upgrade |
| **No honeypot detection** | Missing | Low | No canary accounts for breach detection |

### 8.2 Registration Flow Audit (`auth_service.go:176-215`)

```go
func (s *AuthService) Register(...) error {
    // 1. Validate password against policy ← composition rules only
    if err := s.passwordService.Validate(password); err != nil { ... }
    // 2. Check if credential already exists ← OK
    // 3. Hash password and create credential
    hash, err := crypto.HashPassword(password) ← no pepper, no breach check
    ...
}
```

**Missing steps:**
1. `CheckPasswordBreach(ctx, password)` — exists but never called during registration
2. Entropy estimation (zxcvbn) — not implemented
3. Pepper application — not implemented
4. Hash format versioning for migration — no version column

### 8.3 Login Flow Audit (`auth_service.go:83-168`, `local_provider.go:30-74`)

```go
func (p *LocalProvider) Authenticate(ctx, creds) {
    // Find credential → check enabled → check locked
    match, err := crypto.VerifyPassword(creds.Password, cred.Secret)
    // ← no pepper applied, no migration check
    if match {
        cred.ResetFailedAttempts()
        // ← no transparent re-hash if params changed
    }
}
```

**Missing steps:**
1. Apply pepper before verify
2. Check if hash needs parameter upgrade (transparent re-hash)
3. No honeypot/canary detection

### 8.4 Password Breach Service Audit (`password_breach.go`)

The existing HIBP implementation is functional but incomplete:

```go
// Current issues:
// 1. Uses http.DefaultClient → no timeout, leaks goroutines on slow responses
// 2. No Redis caching → every call hits HIBP API
// 3. No Add-Padding header → size-based analysis possible
// 4. Fails open silently (returns nil) → no logging/alerting
// 5. Not called from Register() → dead code
```

### 8.5 Configuration Audit (`conf.go`)

```go
PasswordPolicy{
    MinLength:      12,    // OK (NIST min is 8, stricter is fine)
    RequireUpper:   true,  // NIST says REMOVE — non-compliant
    RequireLower:   true,  // Non-compliant
    RequireDigit:   true,  // Non-compliant
    RequireSpecial: false, // Compliant
    // No pepper field
    // No Argon2id parameter config
    // No entropy threshold
    // No breach check toggle
}
```

---

## 9. Gap Analysis & Recommendations

### Priority Action Items

| # | Action | Effort | Impact | Priority |
|---|--------|--------|--------|----------|
| **1** | **Wire breach check into registration:** Call `CheckPasswordBreach()` from `Register()` and `SetPassword()`. Add HIBP client timeout (10s) + Redis caching. Add `Add-Padding` header. | S (4h) | High — NIST-mandated | P0 |
| **2** | **Implement pepper support:** Add `PEPPER` env var, HMAC-SHA256 before hashing in `HashPassword()`/`VerifyPassword()`. Store pepper in KMS/Vault in production. | M (1d) | Critical — defends against DB-only leak | P0 |
| **3** | **Make Argon2id params configurable:** Move hardcoded constants to `PasswordPolicy` or a dedicated `HashConfig` struct. Add env var overrides (`ARGON_MEMORY`, `ARGON_ITERATIONS`, `ARGON_PARALLELISM`). | S (2h) | Medium — enables runtime tuning | P1 |
| **4** | **Add zxcvbn entropy estimation:** Replace composition rules with minimum 40-bit entropy threshold. Add `github.com/nbutton23/zxcvbn-go` dependency. Use user context (email, name) as additional inputs. | M (1d) | High — real strength vs. fake complexity rules | P1 |
| **5** | **Implement transparent hash migration:** Add `hash_version` column to credentials table. On login, detect old format and re-hash with current params. Add bcrypt→Argon2id migration handler. | M (1-2d) | Medium — future-proofing + legacy support | P2 |

### Estimated Total Effort

- **P0 items (breach check wiring + pepper):** ~1.5 days
- **P1 items (configurable params + entropy):** ~1.5 days
- **P2 items (migration handler):** ~1-2 days
- **Total: ~4-5 developer-days**

### Quick Wins (deploy today)

1. Add `Add-Padding: true` header to `password_breach.go` (1 line)
2. Replace `http.DefaultClient` with a client that has a 10s timeout (3 lines)
3. Call `CheckPasswordBreach(ctx, password)` at the end of `Register()` (2 lines)

---

## References

- OWASP Password Storage Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
- Argon2 RFC 9106: https://datatracker.ietf.org/doc/rfc9106/
- HIBP Pwned Passwords API: https://haveibeenpwned.com/API/v3
- HIBP k-Anonymity Paper (Junade Ali): https://blog.cryptographyengineering.com/2018/02/23/the-k-anonymity-protocol/
- bcrypt Cost Factor Guide: https://www.commonlounge.com/discussion/6d28ddc0b5a74903bb93e83b9dd0633f
- zxcvbn (Dropbox): https://github.com/dropbox/zxcvbn
- zxcvbn-go: https://github.com/nbutton23/zxcvbn-go
- Hashcat Benchmark Data: https://hashcat.net/hashcat/
- RTX 4090 Benchmark Suite: https://gist.github.com/Chick3nman/32e662a5d04b5a98a89ff5fb04c54c5b
- Pepper vs. Salt Discussion: https://security.stackexchange.com/questions/3272/password-hashing-add-salt-pepper-or-is-salt-enough
- NIST SP 800-63B: https://pages.nist.gov/800-63-3/sp800-63b.html
