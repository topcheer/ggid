# Data Residency and Sovereignty for IAM Systems

> Research document for the GGID IAM Suite
> Topic: Data residency, sovereignty, and cross-border compliance for identity infrastructure
> Status: Research / Gap Analysis — January 2025

## Table of Contents

1. [Regulatory Landscape](#1-regulatory-landscape)
2. [IAM Data Classification](#2-iam-data-classification)
3. [Tenant Data Pinning](#3-tenant-data-pinning)
4. [PostgreSQL RLS for Regional Isolation](#4-postgresql-rls-for-regional-isolation)
5. [Cross-Region Replication Considerations](#5-cross-region-replication-considerations)
6. [Key Management Per Region](#6-key-management-per-region)
7. [Data Subject Rights (DSR) Implementation](#7-data-subject-rights-dsr-implementation)
8. [GGID Multi-Region Data Residency Design](#8-ggid-multi-region-data-residency-design)
9. [Gap Analysis and Recommendations](#9-gap-analysis-and-recommendations)

---

## 1. Regulatory Landscape

Data residency laws dictate where specific categories of data may be stored,
processed, and transferred. For IAM systems — which hold the most sensitive
personal data in any organisation (credentials, biometrics, access patterns) —
non-compliance can result in regulatory fines, service shutdowns, and loss of
operating licenses.

### 1.1 GDPR (European Union) — Articles 44-49

The General Data Protection Regulation restricts transfers of personal data
outside the European Economic Area (EEA). Article 44 establishes the principle:
personal data may only leave the EEA if an adequate level of protection is
guaranteed. Articles 45-49 define the legal mechanisms:

| Mechanism | Article | Description |
|-----------|---------|-------------|
| Adequacy Decision | Art. 45 | The EU Commission determines a third country provides "essentially equivalent" protection. Current: UK, Switzerland, Japan, South Korea, New Zealand, Uruguay, Argentina, Israel, and (under the EU-US Data Privacy Framework) the US for certified organisations. |
| Standard Contractual Clauses (SCCs) | Art. 46(2)(c) | Pre-approved contractual templates between data exporter and importer. The 2021 SCCs include a Transfer Impact Assessment (TIA) requirement — organisations must evaluate whether the destination country's surveillance laws (e.g., FISA 702) undermine SCC safeguards. |
| Binding Corporate Rules (BCRs) | Art. 47 | Internal rules adopted by multinational groups for intra-group transfers. Approval from a competent supervisory authority is required. Typically takes 12-24 months to obtain. |
| Derogations | Art. 49 | Exceptions for specific situations: explicit consent, contract necessity, important reasons of public interest. These are narrow and should not be the basis for systematic transfers. |

**Schrems II Impact**: Following the CJEU ruling (Case C-311/18), supplementary
measures (encryption, pseudonymisation, contractual overrides) may be required
even when using SCCs if the destination country has invasive surveillance laws.

### 1.2 Russia — Federal Law 242-FZ

Effective September 1, 2015, the Russian Data Localization Law requires that the
"collection, recording, systematisation, accumulation, storage, clarification,
and extraction" of personal data of Russian citizens must occur in databases
located within the Russian Federation. Key points:

- Applies to any organisation processing Russian citizens' personal data,
  regardless of where the organisation is headquartered.
- Cross-border transfer is permitted only after the data is initially stored in
  Russia, and the data subject must be notified.
- Roskomnadzor (the regulator) maintains a registry of violators and can block
  access to non-compliant services from within Russia.

**IAM Impact**: An IAM system serving Russian users must store their user
records, credentials, and authentication logs on Russian soil. A centralised EU
deployment is insufficient.

### 1.3 China — Personal Information Protection Law (PIPL)

Effective November 1, 2021, PIPL imposes strict cross-border transfer
requirements:

- **Security Assessment**: Cross-border transfer of personal information above
  thresholds set by the Cyberspace Administration of China (CAC) requires a
  government security assessment. Thresholds (updated 2022): 1M+ individuals'
  data, 100K+ individuals' sensitive data, or 10,000+ individuals' data with
  cumulative volume exceeding 1GB.
- **Standard Contract**: For smaller volumes, a CAC-approved standard contract
  with the overseas recipient is required, plus a filing with the local CAC
  office.
- **Certification**: Cross-border data processors may obtain certification from
  CAC-recognised bodies (analogous to BCRs).
- **Critical Information Infrastructure Operators (CIIO)**: Subject to the most
  stringent rules — mandatory in-country storage and government security
  assessment for any cross-border transfer regardless of volume.

**Additional regulation**: The Data Security Law (DSL) classifies data into
"important data" and "core data" tiers, with separate export controls. IAM
systems for government or critical infrastructure may fall under these tiers.

### 1.4 India — Digital Personal Data Protection Act (DPDP) 2023

Effective (in phases) from 2023-2024, India's DPDP Act:

- Requires consent for processing personal data, with specific provisions for
  children's data (under 18) requiring verifiable parental consent.
- The Central Government may restrict transfers to specific countries (negative
  list approach, unlike GDPR's positive adequacy approach). As of early 2025,
  no countries have been blacklisted, but the mechanism exists.
- Significant Data Fiduciaries (SDFs) — determined by volume, sensitivity, and
  risk — face additional obligations: Data Protection Impact Assessments (DPIAs),
  annual audits, and appointment of a Data Protection Officer.

### 1.5 Brazil — Lei Geral de Protecao de Dados (LGPD)

Effective September 2020 (enforcement from August 2021):

- Transfer mechanisms mirror GDPR: adequacy decisions, SCCs, BCRs, specific
  consent, and contract necessity.
- ANPD (the Brazilian DPA) is still developing adequacy determinations, creating
  uncertainty for transfers to many countries.
- Fines can reach 2% of revenue (capped at 50 million BRL per violation).

### 1.6 Summary Table

| Jurisdiction | Primary Law | Localization Required? | Transfer Mechanism | Max Penalty |
|---|---|---|---|---|
| EU/EEA | GDPR | No (if adequate protection) | SCCs, Adequacy, BCRs | EUR 20M or 4% global revenue |
| UK | UK GDPR | No | UK IDTA, UK Addendum to SCCs | GBP 17.5M or 4% global revenue |
| Russia | 242-FZ | **Yes** — initial collection in Russia | Notification after local storage | Service blocking, fines up to 18M RUB |
| China | PIPL + DSL | **Yes** for CIIOs and large volumes | Security assessment, standard contract, certification | Up to 50M RUB or 5% prior-year revenue |
| India | DPDP 2023 | Conditional (government may restrict) | Permitted except to blacklisted countries | Up to 250 crore INR (~USD 30M) |
| Brazil | LGPD | No (if adequate protection) | Adequacy, SCCs, BCRs, consent | 2% Brazilian revenue, 50M BRL per violation |
| Australia | Privacy Act (APPs) | No | APP 8 (reasonable steps) | AUD 50M (serious/repeated) |
| Canada | PIPEDA | No (adequacy with EU) | Accountability-based transfer | CAD 100K per violation |
| Singapore | PDPA | No | Transfer Limitation Obligation | SGD 1M or 10% annual turnover |

---

## 2. IAM Data Classification

Not all IAM data carries the same regulatory weight. Effective data residency
design requires classifying data elements by sensitivity tier and mapping them to
applicable regulations.

### 2.1 Data Categories

**Category A: Direct Identifiers (PII)**
- Full name, display name
- Email address
- Phone number
- Physical address
- Government ID numbers (SSN, national insurance)

These are subject to virtually all data protection laws. Under GDPR they are
"personal data"; under PIPL they are "personal information"; under LGPD they
are "personal data." Transfer restrictions apply in all jurisdictions listed
above.

**Category B: Authentication Secrets**
- Password hashes (Argon2id)
- Biometric templates (WebAuthn credential public keys, Face ID vectors)
- TOTP secrets
- OAuth refresh tokens
- SAML assertions and artefacts

Biometric data is classified as "special category" under GDPR Art. 9 and
"sensitive personal information" under PIPL. Processing requires explicit
consent and carries heightened security obligations. Password hashes are personal
data (because they can be cracked to reveal identity) but not special category.

**Category C: Behavioural and Metadata**
- IP addresses (classified as personal data by CJEU in Breyer v. Germany)
- User agent strings (can fingerprint devices)
- Geolocation data (GPS coordinates, GeoIP resolution)
- Login timestamps and patterns
- Device fingerprints (FIDO2 AAGUID, browser fingerprints)

Under GDPR Recital 30, online identifiers including IP addresses, cookie IDs,
and RFID tags are personal data. Under PIPL, device information and location
data are explicitly listed as personal information.

**Category D: Session and Access Tokens**
- JWT access tokens
- Session cookies
- Device codes (RFC 8628)
- Push notification tokens

These may contain embedded PII (user IDs, email claims) or be linkable to
identity through the issuing system. Short-lived tokens reduce residency risk
but do not eliminate it.

**Category E: Audit and Security Logs**
- Authentication event logs (success/failure, timestamps, IP, user agent)
- Authorisation decision logs (policy evaluations, RBAC role assignments)
- Security event logs (anomalous login detection, rate-limit triggers)

Audit logs are paradoxical: regulators require them for compliance (SOC 2,
ISO 27001, GDPR Art. 30-32), but the logs themselves contain regulated personal
data. Retention requirements (often 1-7 years) conflict with erasure rights.

### 2.2 Sensitivity Tier Matrix

| Tier | Data Elements | GDPR | PIPL | 242-FZ | LGPD |
|------|--------------|------|------|--------|------|
| T1 — Critical | Biometric templates, password hashes | Special category (Art. 9) | Sensitive PI | Localise | Sensitive |
| T2 — High | PII (name, email, phone), audit logs with IP | Personal data | Personal information | Localise | Personal data |
| T3 — Medium | Session tokens, refresh tokens | Personal data (if linkable) | Personal information | Localise* | Personal data |
| T4 — Low | Pseudonymised IDs, hashed device IDs | Pseudonymous data | Personal information** | Depends | Pseudonymous |
| T5 — Public | JWKS public keys, OIDC discovery docs | Not personal data | Not personal data | N/A | N/A |

\* Russia's 242-FZ applies to "personal data" broadly defined.
\** China's PIPL treats pseudonymised data as personal information that can be
transferred more freely (PIPL Art. 4), but the practical threshold is ambiguous.

---

## 3. Tenant Data Pinning

### 3.1 The Per-Tenant Region Problem

In a multi-tenant IAM system, different tenants operate under different legal
obligations. A German enterprise tenant requires GDPR-compliant EU storage. A
tenant serving Chinese end-users requires PIPL-compliant in-country storage. A
single-region deployment cannot serve all tenants legally.

The solution is **tenant data pinning**: each tenant is assigned a "home region"
at provisioning time, and all data for that tenant is routed to and stored in
that region exclusively.

### 3.2 Tenant Metadata Extension

GGID's current tenant context (`pkg/tenant/tenant.go`) carries `TenantID`,
`IsolationLevel`, `SchemaName`, and `Settings`. For region-aware routing, the
tenant context must be extended:

```go
package tenant

// Region represents a geographic deployment region with residency guarantees.
type Region string

const (
	RegionEUWest   Region = "eu-west-1" // Frankfurt — GDPR
	RegionEUNorth  Region = "eu-north-1" // Stockholm — GDPR
	RegionUSEast   Region = "us-east-1"  // Virginia — DPF certified
	RegionCNShanghai Region = "cn-shanghai-1" // PIPL
	RegionRUMoscow Region = "ru-moscow-1" // 242-FZ
	RegionINMumbai Region = "in-mumbai-1" // DPDP
	RegionBRSaoPaulo Region = "br-saopaulo-1" // LGPD
)

// Context carries tenant-specific information including data residency region.
type Context struct {
	TenantID       uuid.UUID
	IsolationLevel IsolationLevel
	SchemaName     string
	Settings       map[string]any

	// Data residency fields
	HomeRegion   Region   // primary storage region — data must not leave
	ReplicaRegions []Region // optional: read replicas in compliance-compatible regions
	DataClassification DataClassification // tenant-wide classification level
}

// DataClassification defines the sensitivity tier for a tenant.
type DataClassification int

const (
	ClassificationStandard  DataClassification = iota // T3-T4: pseudonymised, tokens
	ClassificationSensitive                            // T2: PII, audit logs
	ClassificationHighlySensitive                      // T1: biometrics, credentials
)

// CanCrossBorder returns false if tenant data must not leave the home region.
func (c *Context) CanCrossBorder() bool {
	if c.DataClassification >= ClassificationSensitive {
		return false
	}
	if c.HomeRegion == RegionCNShanghai || c.HomeRegion == RegionRUMoscow {
		return false // PIPL and 242-FZ prohibit cross-border by default
	}
	return true
}
```

### 3.3 Region-Aware Request Routing

The gateway must resolve the tenant's home region before forwarding requests.
If the request arrives at a region different from the tenant's home region, the
gateway either proxies to the correct regional deployment or returns an HTTP 451
(Unavailable For Legal Reasons) with a redirect:

```go
package router

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/ggid/ggid/pkg/tenant"
)

// RegionResolver maps a tenant ID to its home region and the upstream URL
// for that region's service deployment.
type RegionResolver interface {
	Resolve(tenantID string) (tenant.Region, string, error) // region, upstreamURL
}

// RegionAwareRouter inspects tenant context and routes to the correct region.
type RegionAwareRouter struct {
	localRegion  tenant.Region
	resolver     RegionResolver
	proxies      map[tenant.Region]*httputil.ReverseProxy
}

func NewRegionAwareRouter(localRegion tenant.Region, resolver RegionResolver) *RegionAwareRouter {
	return &RegionAwareRouter{
		localRegion: localRegion,
		resolver:    resolver,
		proxies:     make(map[tenant.Region]*httputil.ReverseProxy),
	}
}

func (r *RegionAwareRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Extract tenant from JWT or X-Tenant-ID header
	tenantID := req.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		http.Error(w, "missing tenant identifier", http.StatusBadRequest)
		return
	}

	homeRegion, upstreamURL, err := r.resolver.Resolve(tenantID)
	if err != nil {
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}

	// If request arrived at the correct region, no routing needed.
	// In production, the local service handles it directly.
	if homeRegion == r.localRegion {
		// Fall through to local handler
		return
	}

	// If tenant data cannot cross borders, reject the request at the edge
	// rather than proxying. Return a redirect to the correct regional endpoint.
	proxy := r.getOrCreateProxy(homeRegion, upstreamURL)
	if proxy == nil {
		// Return 451 with a Location header pointing to the correct region's endpoint
		w.Header().Set("Location", fmt.Sprintf("https://%s.%s.ggid.io", homeRegion, req.URL.Path))
		w.Header().Set("X-Data-Residency-Region", string(homeRegion))
		w.WriteHeader(http.StatusUnavailableForLegalReasons)
		fmt.Fprintf(w, `{"error":"data_residency_redirect","region":"%s"}`, homeRegion)
		return
	}

	// Proxy to the correct regional deployment
	proxy.ServeHTTP(w, req)
}

func (r *RegionAwareRouter) getOrCreateProxy(region tenant.Region, upstream string) *httputil.ReverseProxy {
	if p, ok := r.proxies[region]; ok {
		return p
	}
	target, err := url.Parse(upstream)
	if err != nil {
		return nil
	}
	p := httputil.NewSingleHostReverseProxy(target)
	r.proxies[region] = p
	return p
}
```

### 3.4 Tenant Region Lookup Service

A global control plane stores tenant-to-region mappings. This metadata is
low-sensitivity (tenant IDs and region codes, no PII) and can be replicated
globally without residency concerns:

```go
// TenantRegionRecord is the global lookup entry for tenant region routing.
// This is the ONLY tenant data that can be safely replicated across all regions.
type TenantRegionRecord struct {
	TenantID    uuid.UUID      `json:"tenant_id"`
	HomeRegion  tenant.Region  `json:"home_region"`
	CreatedAt   time.Time      `json:"created_at"`
	// No PII — just routing metadata
}
```

---

## 4. PostgreSQL RLS for Regional Isolation

### 4.1 Beyond Tenant Isolation: Regional Partitioning

GGID currently uses PostgreSQL Row-Level Security for tenant isolation. The
identity service sets `app.tenant_id` via `SET LOCAL` and the RLS policy filters
rows automatically (see `services/identity/internal/repository/pg_repo.go`,
function `setTenantRLS`). This same mechanism can enforce regional isolation.

The approach combines **declarative partitioning** (for physical separation) with
**RLS policies** (for logical enforcement):

### 4.2 Declarative Partitioning by Region

```sql
-- Regional partitioning for the users table.
-- Each partition lives on a tablespace backed by storage in the correct region.
-- In practice, each regional deployment runs its own PostgreSQL cluster,
-- so partitioning is a logical enforcement layer within a region.

CREATE TABLE users (
    id              UUID DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL,
    region          VARCHAR(20) NOT NULL DEFAULT current_setting('app.region', true),
    username        VARCHAR(64) NOT NULL,
    email           VARCHAR(255) NOT NULL,
    -- ... other columns
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, region)
) PARTITION BY LIST (region);

CREATE TABLE users_eu_west PARTITION OF users
    FOR VALUES IN ('eu-west-1') TABLESPACE ts_eu_west;

CREATE TABLE users_us_east PARTITION OF users
    FOR VALUES IN ('us-east-1') TABLESPACE ts_us_east;

CREATE TABLE users_cn_shanghai PARTITION OF users
    FOR VALUES IN ('cn-shanghai-1') TABLESPACE ts_cn_shanghai;
```

### 4.3 Regional RLS Policies

```sql
-- Enable RLS and add a regional isolation policy.
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

-- Policy: a session can only see rows matching both tenant AND region.
-- The app.region setting is set at connection pool initialisation,
-- making it impossible for application-level bugs to read cross-region data.
CREATE POLICY tenant_region_isolation ON users
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
        AND region = current_setting('app.region', true)
    );

-- Enforce that all INSERTs set the correct region.
-- This prevents data from being written to the wrong partition.
CREATE OR REPLACE FUNCTION enforce_region()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.region IS NULL OR NEW.region = '' THEN
        NEW.region := current_setting('app.region', true);
    ELSIF NEW.region != current_setting('app.region', true) THEN
        RAISE EXCEPTION 'cross-region insert blocked: session region=%, row region=%',
            current_setting('app.region', true), NEW.region
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_enforce_region
    BEFORE INSERT OR UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION enforce_region();
```

### 4.4 Go: Setting Regional Context

```go
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// setRegionalContext sets both tenant and region session variables.
// This must be called at the start of every transaction.
func setRegionalContext(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, region string) error {
	// Set tenant for RLS (existing pattern in GGID)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID.String())); err != nil {
		return fmt.Errorf("set tenant RLS: %w", err)
	}
	// Set region for regional RLS (new)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.region = '%s'", region)); err != nil {
		return fmt.Errorf("set region: %w", err)
	}
	return nil
}

// AfterAcquire is called when a connection is acquired from the pool.
// It sets the region once per connection so individual transactions
// don't need to set it repeatedly (though SET LOCAL is per-transaction safe).
func AfterAcquire(region string) func(ctx context.Context, conn *pgx.Conn) error {
	return func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, fmt.Sprintf("SET app.region = '%s'", region))
		return err
	}
}
```

The `SET LOCAL` approach (per-transaction) is safer than pool-level `SET` because
it ensures every transaction explicitly asserts its region. A misconfigured
connection pool that shares sessions across regional contexts would be caught by
the RLS policy returning zero rows, rather than silently leaking data.

---

## 5. Cross-Region Replication Considerations

### 5.1 Data That Must NOT Cross Borders

The following data categories are subject to strict residency requirements and
must never be replicated or accessible across regions:

| Data Type | Regulation | Rationale |
|-----------|-----------|-----------|
| Biometric templates (WebAuthn keys) | GDPR Art. 9, PIPL | Special category; unique identifiers |
| Raw audit logs with IP addresses | GDPR, 242-FZ, PIPL | Contains behavioural PII; Russian law requires local storage |
| Password hashes | GDPR, PIPL | Linkable to identity; cracking risk |
| User PII (name, email, phone) | All regulations | Core personal data |
| OAuth tokens and refresh tokens | GDPR, PIPL | Session hijacking vector; linkable to identity |

### 5.2 Data That CAN Cross Borders

Some IAM data is low-sensitivity or non-personal and can be distributed globally
for performance and availability:

| Data Type | Crosses Borders? | Rationale |
|-----------|-----------------|-----------|
| JWKS public keys | Yes | Public keys, no PII |
| OIDC discovery metadata | Yes | Configuration, no PII |
| Tenant region mapping | Yes | Just tenant ID + region code |
| Token revocation lists (CRLs) | Yes | Token hashes only, no PII |
| Rate-limit counters | Yes | Aggregated, pseudonymous |
| Public SAML SP metadata | Yes | Configuration, no PII |

### 5.3 Read Replicas vs Compliance

Standard cloud architecture uses cross-region read replicas for disaster
recovery and low-latency global reads. For IAM systems subject to data residency
laws, this approach is dangerous:

```
                    ┌─────────────────────────────────────────┐
                    │          GLOBAL CONTROL PLANE            │
                    │  (tenant→region mapping, JWKS, CRLs)     │
                    │  Replicated to ALL regions               │
                    └───────────┬──────────┬──────────────────┘
                                │          │
                ┌───────────────┘          └───────────────┐
                ▼                                          ▼
    ┌───────────────────────┐                 ┌───────────────────────┐
    │   EU-WEST (Frankfurt)  │                 │  CN-SHANGHAI (China)   │
    │   GDPR Compliance       │                 │  PIPL Compliance       │
    │                         │                 │                         │
    │   • PG Primary (EU)     │   NO CROSS-     │   • PG Primary (CN)     │
    │   • Redis (EU)          │   REGION REPL   │   • Redis (CN)          │
    │   • NATS (EU)           │   FOR PII       │   • NATS (CN)           │
    │   • KMS Keys (EU)       │                 │   • KMS Keys (CN)       │
    │   • Audit Logs (EU)     │                 │   • Audit Logs (CN)     │
    └───────────────────────┘                 └───────────────────────┘
                │                                          │
                ▼                                          ▼
    ┌───────────────────────┐                 ┌───────────────────────┐
    │   US-EAST (Virginia)   │                 │   RU-MOSCOW (Russia)   │
    │   DPF Certified         │                 │   242-FZ Compliance    │
    │                         │                 │                         │
    │   • PG Primary (US)     │                 │   • PG Primary (RU)     │
    │   • Redis (US)          │                 │   • Redis (RU)          │
    │   • NATS (US)           │                 │   • NATS (RU)           │
    │   • KMS Keys (US)       │                 │   • KMS Keys (RU)       │
    └───────────────────────┘                 └───────────────────────┘
```

**Key architectural principle**: Each region is a **silo** with its own
PostgreSQL, Redis, NATS, and KMS. The only cross-region communication is via the
control plane (non-PII metadata) and edge proxy redirects (for mis-routed
requests).

### 5.4 Audit Log Replication Strategy

GGID's audit service currently uses monthly time-based partitioning
(`services/audit/migrations/000002_create_partitions.up.sql`). For multi-region,
audit logs must additionally be region-pinned:

```sql
-- Multi-dimensional partitioning: by region AND by month
CREATE TABLE audit_events (
    id           UUID DEFAULT uuid_generate_v4(),
    tenant_id    UUID NOT NULL,
    region       VARCHAR(20) NOT NULL,
    event_time   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- ... other columns
    PRIMARY KEY (id, region, event_time)
) PARTITION BY LIST (region);

-- Sub-partition each region by month
CREATE TABLE audit_events_eu PARTITION OF audit_events
    FOR VALUES IN ('eu-west-1') PARTITION BY RANGE (event_time);

CREATE TABLE audit_events_eu_2025_01 PARTITION OF audit_events_eu
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

---

## 6. Key Management Per Region

### 6.1 Why Regional Keys

Encryption keys are the last line of defence for data residency. Even if
encrypted data is accidentally replicated across borders, if the decryption key
never leaves the origin region, the data remains cryptographically inaccessible
abroad. This principle — **key-data co-location** — is recognised by:

- GDPR (encryption as a technical measure, Art. 32)
- PIPL (de-identification requirements)
- PCI DSS (key management requirements, Req. 3)
- FIPS 140-2/3 (cryptographic module boundaries)

### 6.2 Envelope Encryption with Regional Master Keys

```go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
)

// RegionalKMS manages per-region master keys and performs envelope encryption.
// The master key NEVER leaves the region — only data encryption keys (DEKs)
// are generated, used, and stored encrypted (wrapped) alongside ciphertext.
type RegionalKMS struct {
	region     string
	masterKey  []byte // only exists in-memory within this region
	wrappedDEKs map[string][]byte // cache: keyID → wrapped DEK
	mu         sync.RWMutex
}

func NewRegionalKMS(region string, masterKey []byte) *RegionalKMS {
	return &RegionalKMS{
		region:      region,
		masterKey:   masterKey,
		wrappedDEKs: make(map[string][]byte),
	}
}

// EncryptedPayload is the on-wire/at-rest format for region-locked encrypted data.
type EncryptedPayload struct {
	Region     string `json:"region"`       // origin region — for routing decisions
	WrappedDEK string `json:"wrapped_dek"`  // DEK encrypted with regional master key
	Ciphertext string `json:"ciphertext"`   // data encrypted with DEK
	KeyID      string `json:"key_id"`       // DEK identifier
}

// Encrypt performs envelope encryption: generates a DEK, encrypts data with it,
// then wraps the DEK with the regional master key.
func (k *RegionalKMS) Encrypt(plaintext []byte) (*EncryptedPayload, error) {
	// 1. Generate a fresh DEK
	dek := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("generate DEK: %w", err)
	}

	// 2. Encrypt the data with the DEK
	ciphertext, err := aesGCMEncrypt(plaintext, dek)
	if err != nil {
		return nil, fmt.Errorf("encrypt data: %w", err)
	}

	// 3. Wrap the DEK with the regional master key
	wrappedDEK, err := aesGCMEncrypt(dek, k.masterKey)
	if err != nil {
		return nil, fmt.Errorf("wrap DEK: %w", err)
	}

	keyID := generateKeyID()

	return &EncryptedPayload{
		Region:     k.region,
		WrappedDEK: base64.StdEncoding.EncodeToString(wrappedDEK),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		KeyID:      keyID,
	}, nil
}

// Decrypt reverses envelope encryption. If the master key is unavailable
// (e.g., data was replicated to a different region without the key),
// decryption fails — this is the intended data residency enforcement.
func (k *RegionalKMS) Decrypt(payload *EncryptedPayload) ([]byte, error) {
	// Enforce that data from a different region cannot be decrypted
	if payload.Region != k.region {
		return nil, fmt.Errorf(
			"cross-region decryption blocked: data region=%s, this region=%s",
			payload.Region, k.region,
		)
	}

	wrappedDEK, err := base64.StdEncoding.DecodeString(payload.WrappedDEK)
	if err != nil {
		return nil, fmt.Errorf("decode wrapped DEK: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	// Unwrap the DEK with the regional master key
	dek, err := aesGCMDecrypt(wrappedDEK, k.masterKey)
	if err != nil {
		return nil, fmt.Errorf("unwrap DEK: %w", err)
	}

	// Decrypt the data with the DEK
	plaintext, err := aesGCMDecrypt(ciphertext, dek)
	if err != nil {
		return nil, fmt.Errorf("decrypt data: %w", err)
	}

	return plaintext, nil
}

// aesGCMEncrypt and aesGCMDecrypt mirror GGID's existing pkg/crypto functions.
func aesGCMEncrypt(plaintext, key []byte) ([]byte, error) {
	h := sha256.Sum256(key)
	block, err := aes.NewCipher(h[:])
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

func aesGCMDecrypt(ciphertext, key []byte) ([]byte, error) {
	h := sha256.Sum256(key)
	block, err := aes.NewCipher(h[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

func generateKeyID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
```

### 6.3 Key Rotation

Regional master keys should be rotated annually or after any security incident.
Rotation requires re-wrapping all DEKs with the new master key — a background
job that iterates through encrypted records. The `KeyID` field in
`EncryptedPayload` allows gradual migration: old payloads with old KeyIDs can be
read until re-encrypted.

---

## 7. Data Subject Rights (DSR) Implementation

### 7.1 The Multi-Region DSR Challenge

GDPR grants data subjects eight rights (access, rectification, erasure,
restriction, portability, objection, automated decision-making, and
notification of breach). PIPL grants similar rights. Fulfilling these in a
multi-region IAM system requires identifying ALL data for a subject across ALL
services and regions, then executing the right in each location.

### 7.2 DSR Fulfilment Architecture

```go
package dsr

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DSRRequest represents a data subject rights request.
type DSRRequest struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	SubjectUserID uuid.UUID // the user exercising their rights
	Type      DSRType
	Regions   []string // regions where this request must be executed
	CreatedAt time.Time
	Deadline  time.Time // legal deadline (GDPR: 1 month)
}

type DSRType string

const (
	DSRAccess     DSRType = "access"      // Art. 15: data export
	DSRErasure    DSRType = "erasure"     // Art. 17: right to be forgotten
	DSRRectify    DSRType = "rectification" // Art. 16
	DSRPortability DSRType = "portability" // Art. 20: machine-readable export
)

// DSRCoordinator orchestrates DSR fulfilment across all services and regions.
type DSRCoordinator struct {
	regions  []string
	handlers map[string][]DSRHandler // region → list of service handlers
}

// DSRHandler is implemented by each service that stores user data.
type DSRHandler interface {
	ServiceName() string
	ExportUser(ctx context.Context, tenantID, userID uuid.UUID) (json.RawMessage, error)
	DeleteUser(ctx context.Context, tenantID, userID uuid.UUID) error
}

// FulfilAccess executes a data export across all regions and services.
// Returns a machine-readable JSON document (GDPR Art. 20 portability format).
func (c *DSRCoordinator) FulfilAccess(ctx context.Context, req *DSRRequest) (*DSRResult, error) {
	result := &DSRResult{
		RequestID: req.ID,
		ExportedAt: time.Now().UTC(),
		Data:      make(map[string]map[string]json.RawMessage), // region → service → data
	}

	for _, region := range req.Regions {
		result.Data[region] = make(map[string]json.RawMessage)
		for _, handler := range c.handlers[region] {
			data, err := handler.ExportUser(ctx, req.TenantID, req.SubjectUserID)
			if err != nil {
				return nil, fmt.Errorf(
					"DSR access %s/%s: %w", region, handler.ServiceName(), err,
				)
			}
			result.Data[region][handler.ServiceName()] = data
		}
	}

	return result, nil
}

// FulfilErasure executes right-to-be-forgotten across all regions and services.
// Order matters: delete derived data first (audit logs, sessions), then core
// records (users), to prevent orphaned references.
func (c *DSRCoordinator) FulfilErasure(ctx context.Context, req *DSRRequest) error {
	// Deletion order: sessions → audit (anonymise) → tokens → user records
	deletionOrder := []string{"auth", "audit", "oauth", "identity", "org"}

	for _, region := range req.Regions {
		handlers := c.sortedHandlers(c.handlers[region], deletionOrder)
		for _, handler := range handlers {
			if err := handler.DeleteUser(ctx, req.TenantID, req.SubjectUserID); err != nil {
				return fmt.Errorf(
					"DSR erasure %s/%s: %w", region, handler.ServiceName(), err,
				)
			}
		}
	}
	return nil
}

type DSRResult struct {
	RequestID  uuid.UUID                            `json:"request_id"`
	ExportedAt time.Time                            `json:"exported_at"`
	Data       map[string]map[string]json.RawMessage `json:"data"` // region → service → data
}
```

### 7.3 Audit Log Anonymisation vs Deletion

GDPR Art. 17 (erasure) conflicts with legal retention obligations for audit
logs (SOC 2, SOX, tax regulations may require 1-7 year retention). The accepted
compromise is **anonymisation**: replace identifiable fields with pseudonymous
or redacted values while retaining the audit trail structure.

```go
// AnonymiseAuditEvent replaces PII in an audit event while preserving
// the event structure for compliance retention.
func AnonymiseAuditEvent(e *AuditEvent) *AuditEvent {
	return &AuditEvent{
		ID:           e.ID,
		TenantID:     e.TenantID,
		ActorType:    e.ActorType,
		ActorID:      hashForRetention(e.ActorID), // one-way hash
		ActorName:    "[REDACTED]",
		Action:       e.Action,
		ResourceType: e.ResourceType,
		ResourceID:   hashForRetention(e.ResourceID),
		IPAddress:    "0.0.0.0",  // nullify
		UserAgent:    "[REDACTED]",
		Metadata:     redactMetadata(e.Metadata),
		CreatedAt:    e.CreatedAt,
	}
}

// hashForRetention produces a deterministic one-way hash so that
// related events can still be correlated for forensic analysis
// without revealing the original identity.
func hashForRetention(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8]) // truncated for brevity
}
```

### 7.4 Deletion Cascades in GGID

GGID's identity service already has `ON DELETE CASCADE` foreign keys
(`user_emails`, `user_external_identities`, `email_verification_tokens` all
cascade from `users`). A DSR erasure of the `users` row will cascade to these
tables automatically. However, the following services need explicit deletion
logic:

- **Auth**: Credentials, MFA secrets, session records
- **OAuth**: Access tokens, refresh tokens, consent records
- **Audit**: Must be anonymised, not deleted (retention obligation)
- **Policy**: Role assignments (delete), policies themselves (tenant-owned, keep)

---

## 8. GGID Multi-Region Data Residency Design

### 8.1 Current Architecture Assessment

GGID's existing architecture has both strengths and gaps for multi-region data
residency.

**Strengths (what helps):**

1. **Tenant context propagation** (`pkg/tenant/tenant.go`): The `Context` struct
   with `TenantID`, `IsolationLevel`, and `Settings` map provides a natural
   extension point for adding `HomeRegion`. The context is already propagated via
   `context.Context` through all service layers.

2. **RLS-based isolation** (`services/identity/internal/repository/pg_repo.go`):
   The `setTenantRLS` function pattern (`SET LOCAL app.tenant_id`) can be
   trivially extended to also set `app.region`, adding regional RLS policies
   without architectural change.

3. **PII masking** (`pkg/pii/pii.go`): The `Obfuscate` function already masks
   emails, phones, IPs, UUIDs, SSNs, and credit card numbers in log output.
   This is essential for any cross-region log shipping.

4. **Audit time-based partitioning**: The audit service already uses declarative
   partitioning (`audit_events_2025_01`, etc.). Adding a region dimension to
   the partition key is a natural extension.

5. **AES-256-GCM encryption** (`pkg/crypto/crypto.go`): The existing `AESEncrypt`
   and `AESDecrypt` functions can serve as the DEK encryption layer in an
   envelope encryption scheme.

6. **Cascade deletes**: The identity schema uses `ON DELETE CASCADE` for
   dependent tables, simplifying DSR erasure.

**Gaps (what hinders):**

1. **No region awareness**: The tenant `Context` has no `HomeRegion` field. No
   service knows which region it's running in. No routing layer distinguishes
   regional deployments.

2. **Single-region database**: All services connect to a single PostgreSQL pool
   via `pgxpool.Pool`. There is no mechanism to route queries to different
   regional databases based on tenant.

3. **Gateway assumes single backend**: The gateway's reverse proxy forwards all
   requests to fixed upstream URLs. There is no tenant-to-region resolution or
   cross-region redirect logic.

4. **No regional key management**: The `AESEncrypt` function takes a single key.
   There is no KMS integration, no envelope encryption, no key-per-region
   concept.

5. **Audit logs contain raw PII**: The audit repository stores `ip_address`,
   `user_agent`, and `actor_name` in plaintext. These would violate residency
   requirements if replicated.

6. **No DSR orchestration**: There is no coordinator for data subject rights
   requests. Each service would need to implement export and deletion
   independently.

7. **Redis and NATS are region-blind**: Session storage (Redis) and event
   streaming (NATS) have no regional affinity. Sessions and events could cross
   borders without detection.

8. **No data classification metadata**: There is no field in the schema or
   tenant context indicating the sensitivity tier of stored data, making it
   impossible to make residency decisions at the data level.

### 8.2 Required Changes for Multi-Region Support

**Tier 1: Foundation (enables single-region compliance reporting)**

Add region awareness to the tenant context and all service configurations. This
doesn't enable multi-region deployment yet, but makes the system "region-aware"
for audit and compliance purposes.

```go
// pkg/tenant/tenant.go — add region fields
type Context struct {
	TenantID       uuid.UUID
	IsolationLevel IsolationLevel
	SchemaName     string
	Settings       map[string]any

	HomeRegion       string   // NEW: primary storage region
	DataResidency    string   // NEW: residency policy ("strict", "standard", "none")
}
```

**Tier 2: Data Pinning (enables per-tenant regional isolation)**

- Add `region` column to all tenant-scoped tables
- Extend RLS policies to include region checks
- Add gateway-level tenant region resolution
- Deploy per-region PostgreSQL clusters

**Tier 3: Full Multi-Region (enables global multi-tenant deployment)**

- Global control plane for tenant-to-region mapping
- Per-region KMS integration
- DSR coordinator across regions
- Cross-region redirect protocol at the gateway
- Per-region Redis and NATS deployments

---

## 9. Gap Analysis and Recommendations

### 9.1 Current Gaps

| Gap | Severity | Effort | Impact |
|-----|----------|--------|--------|
| No region field in tenant Context | Critical | Small (2 days) | Blocks all multi-region work |
| No regional RLS policies | Critical | Medium (1 week) | No database-level residency enforcement |
| Gateway lacks region routing | Critical | Medium (1 week) | Cannot serve tenants in different regions |
| No regional KMS / envelope encryption | High | Large (2-3 weeks) | Keys would cross borders |
| No DSR coordinator | High | Medium (1-2 weeks) | Cannot fulfil GDPR/PIPL data subject rights |
| Audit logs store raw PII | High | Small (2-3 days) | PII leaks across regions on replication |
| No tenant data classification metadata | Medium | Small (2-3 days) | Cannot make per-data residency decisions |
| Redis/NATS are region-blind | Medium | Medium (1 week) | Sessions and events can cross borders |

### 9.2 Implementation Roadmap

**Phase 1: Region Awareness (2-3 weeks, 1 developer)**

1. **Extend `pkg/tenant/Context`** with `HomeRegion` and `DataResidency` fields.
   Add `CanCrossBorder()` method. Update all context propagation paths.

2. **Add `region` column** to all tenant-scoped tables (users, audit_events,
   roles, organisations, credentials). Extend RLS policies to check region:
   ```sql
   CREATE POLICY tenant_region_isolation ON users
       FOR ALL USING (
           tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
           AND region = current_setting('app.region', true)
       );
   ```

3. **Add `setRegionalContext`** to all repository layers, extending the existing
   `setTenantRLS` pattern. Set `app.region` at connection pool initialisation.

**Phase 2: Gateway Region Routing (2 weeks, 1 developer)**

4. **Implement `RegionResolver`** in the gateway: a global lookup service
   (backed by a replicated key-value store or a dedicated control-plane database)
   that maps tenant IDs to home regions. Cache aggressively (tenant region
   rarely changes).

5. **Add region-aware routing middleware**: Before forwarding, resolve the
   tenant's home region. If it differs from the local region, return HTTP 451
   with a redirect to the correct regional endpoint. Include
   `X-Data-Residency-Region` header.

**Phase 3: Regional Key Management (3 weeks, 1 developer + 1 security engineer)**

6. **Implement envelope encryption** (`RegionalKMS`) wrapping GGID's existing
   AES-GCM functions. Integrate with cloud KMS (AWS KMS, GCP KMS, Azure Key
   Vault) for master key storage. Master keys are created per-region and never
   exported.

7. **Encrypt sensitive fields at rest**: `password_hash` is already Argon2id
   (irreversible), but biometric templates, TOTP secrets, and OAuth tokens
   should be encrypted with regional envelope encryption. Add a `region` tag to
   encrypted payloads to enforce co-location.

**Phase 4: DSR Orchestration (2 weeks, 1 developer)**

8. **Implement `DSRCoordinator`** that dispatches access and erasure requests to
   all services across all regions. Each service implements `DSRHandler` with
   `ExportUser` and `DeleteUser` methods.

9. **Implement audit log anonymisation**: Replace the raw PII fields in audit
   events with one-way hashes and `[REDACTED]` placeholders. This satisfies
   GDPR Art. 17 (erasure) while maintaining the audit trail for compliance
   retention obligations.

**Phase 5: Cross-Region Communication Hardening (2 weeks, 1 developer)**

10. **Audit NATS and Redis usage**: Ensure no PII-bearing messages are published
    to globally-replicated NATS subjects. Implement region-pinned NATS
    JetStream streams for audit events. Configure Redis with per-region
    namespaces to prevent session data from crossing borders.

### 9.3 Compliance Checklist

- [ ] Each tenant has a documented home region
- [ ] RLS policies enforce region isolation at the database level
- [ ] Encryption keys are region-pinned and never exported
- [ ] Audit logs with PII are region-pinned and retention-managed
- [ ] DSR requests can be fulfilled across all regions within statutory deadlines
- [ ] Gateway redirects mis-routed requests to the correct region
- [ ] Data classification metadata exists for all stored PII
- [ ] No PII-bearing messages cross regional NATS/Redis boundaries
- [ ] Transfer Impact Assessments are documented for each cross-border mechanism
- [ ] Incident response plan covers multi-region data breach scenarios

---

## Conclusion

GGID's architecture is well-positioned for multi-region data residency: the
tenant context propagation, RLS-based isolation, and PII masking provide strong
foundations. The primary work is adding region awareness throughout the stack —
from the tenant context to the database, gateway, and key management layers.

The most critical gap is the absence of any region concept in the current
system. Without it, the system cannot make data residency decisions at any
level. The recommended phased approach adds region awareness first (low effort,
high leverage), then builds gateway routing, key management, and DSR
orchestration on top.

For organisations operating in Russia, China, or other strict-localisation
jurisdictions, Phase 1 and Phase 2 are prerequisites for legal operation. For
GDPR-only deployments, Phases 1-3 provide a defensible compliance posture with
SCC-based transfers for the remaining gaps.
