# Multi-Tenant SAML Federation — Design & Implementation

> **Document Type**: Research / Architecture
> **Audience**: GGID platform engineers, security architects, integration developers
> **Status**: Design specification
> **Related Packages**: `pkg/saml/`, `pkg/tenant/`, `services/auth/`

---

## Table of Contents

1. [Overview](#1-overview)
2. [SAML Basics Recap](#2-saml-basics-recap)
3. [Multi-Tenant SAML Architecture](#3-multi-tenant-saml-architecture)
4. [SP-Initiated Flow (Multi-Tenant)](#4-sp-initiated-flow-multi-tenant)
5. [IdP-Initiated Flow (Multi-Tenant)](#5-idp-initiated-flow-multi-tenant)
6. [Attribute Mapping](#6-attribute-mapping)
7. [Just-in-Time (JIT) Provisioning](#7-just-in-time-jit-provisioning)
8. [GGID Implementation Design](#8-ggid-implementation-design)
9. [Security Considerations](#9-security-considerations)
10. [Comparison with Other Implementations](#10-comparison-with-other-multi-tenant-saml-implementations)
11. [GGID Roadmap](#11-ggid-roadmap)
12. [Appendix: SAML XML Examples](#appendix-saml-xml-examples)
13. [References](#references)

---

## 1. Overview

### 1.1 The Enterprise SSO Problem

Enterprise customers adopting a SaaS platform like GGID rarely start with a blank
identity slate. They already operate a corporate Identity Provider (IdP) — Okta,
Microsoft Entra ID (Azure AD), OneLogin, PingFederate, ADFS, Google Workspace, or a
self-hosted Shibboleth instance. Their users exist in that IdP, password policies are
enforced there, MFA is configured there, and onboarding/offboarding workflows are wired
to HR systems through that IdP.

For a SaaS vendor, requiring users to create *separate* credentials for every application
is unacceptable. It creates:

- **Password fatigue** — users juggle dozens of credentials, leading to reuse.
- **Onboarding friction** — manual account creation per application.
- **Offboarding risk** — when an employee leaves, their accounts on individual
  applications remain active if not separately deactivated.
- **Compliance gaps** — centralized audit logs from the IdP don't cover applications
  that bypass it.

SAML 2.0 (Security Assertion Markup Language) solves this by enabling federated
identity. The user authenticates once at their corporate IdP, and that authentication
is trusted by the SaaS application (Service Provider) via a cryptographically signed
XML assertion.

### 1.2 The Multi-Tenant Challenge

In a multi-tenant platform like GGID, the SaaS instance serves many organizations
(tenants) simultaneously. Each tenant may use a **different IdP**, with different
configurations:

| Tenant    | IdP               | Entity ID                                  | Certificate | Attribute Names         |
|-----------|-------------------|--------------------------------------------|-------------|-------------------------|
| Acme Corp | Okta              | `http://www.okta.com/exk1abc`             | Cert A      | `email`, `firstName`    |
| Globex    | Azure AD          | `https://sts.windows.net/uuid/`           | Cert B      | `mail`, `givenname`     |
| Initech   | ADFS              | `https://adfs.initech.com/adfs/services/trust` | Cert C   | `http://.../emailaddress` |
| Umbrella  | OneLogin          | `https://umbrella.onelogin.com/saml/metadata` | Cert D   | `User.email`            |

This means GGID, as the Service Provider (SP), must maintain:

- **Per-tenant SP entity IDs** — each tenant's IdP needs a distinct entity ID to
  route assertions correctly.
- **Per-tenant ACS URLs** — the Assertion Consumer Service endpoint varies per
  tenant so GGID knows which tenant's config to use when an assertion arrives.
- **Per-tenant IdP trust certificates** — GGID must verify each assertion against
  the correct IdP's public certificate.
- **Per-tenant attribute mappings** — because no two IdPs name attributes the same way.
- **Per-tenant signing keys** (optionally) — the AuthnRequest GGID sends can be
  signed with a tenant-specific key pair.

### 1.3 GGID's Role

- **GGID = Service Provider (SP)** — consumes SAML assertions and grants sessions.
- **Tenant's IdP = Identity Provider** — authenticates the user and issues assertions.
- **User = Principal** — the person logging in.

GGID's existing `pkg/saml/` package provides assertion parsing, condition validation,
attribute extraction, and signature element checking. The `pkg/tenant/` package provides
multi-tenant context propagation. This document specifies how to combine them into a
complete multi-tenant SAML federation layer.

### 1.4 Goals

1. Support **unlimited tenants**, each with their own IdP configuration.
2. Support both **SP-initiated** and **IdP-initiated** SSO flows.
3. Support **Just-in-Time (JIT) provisioning** — auto-create users on first SAML login.
4. Support **certificate rotation** with zero downtime.
5. Support **dynamic metadata exchange** — GGID fetches IdP metadata automatically.
6. Maintain **strict tenant isolation** — one tenant cannot access another's SAML config
   or user data.
7. Leverage existing `pkg/saml/` assertion parsing primitives.

---

## 2. SAML Basics Recap

### 2.1 Roles

SAML 2.0 defines three roles in the SSO triangle:

```
                         ┌──────────────┐
                    (1)  │              │  (2)
        User  ──────────►│  IdP         │──────────► Assertion
       (Principal)       │  (Identity   │           (XML, signed)
                         │   Provider)  │
                         └──────┬───────┘
                                │
                                │ (3) Trust Relationship
                                │     (pre-configured:
                                │      entity IDs, certs)
                                ▼
                         ┌──────────────┐
                         │  SP          │
                         │  (Service    │
                         │   Provider)  │
                         │  = GGID      │
                         └──────────────┘
```

- **Identity Provider (IdP)**: The system that authenticates the user and issues
  SAML assertions. Examples: Okta, Azure AD, ADFS, OneLogin, Google Workspace.
  The IdP holds the user's credentials, enforces password/MFA policy, and knows
  the user's attributes (email, name, groups, department).

- **Service Provider (SP)**: The application that consumes assertions and grants
  access. In our context, GGID is always the SP. The SP trusts the IdP because it
  has been pre-configured with the IdP's entity ID, SSO URL, and signing certificate.

- **Principal / User**: The end user who wants to access the SP. The user may
  start at the SP (SP-initiated) or at the IdP (IdP-initiated).

### 2.2 SAML Profiles

A **profile** defines how SAML messages are combined to achieve a specific use case.

#### Web Browser SSO Profile

The most widely deployed profile. Enables single sign-on for a user accessing a web
application through their browser. Two initiation modes:

**SP-Initiated SSO:**
```
User ──► SP (GGID) ──► IdP ──► SP (GGID)
  (1) Access GGID
  (2) GGID sends AuthnRequest to IdP
  (3) User authenticates at IdP
  (4) IdP posts assertion back to GGID's ACS URL
  (5) GGID validates assertion, creates session
```

**IdP-Initiated SSO:**
```
User ──► IdP ──► SP (GGID)
  (1) User authenticates at IdP
  (2) IdP sends unsolicited assertion to GGID's ACS URL
  (3) GGID validates assertion, creates session
```

#### Other Profiles (less common in SaaS)

- **Single Logout (SLO)** — propagates logout across all SPs and IdP.
- **Assertion Query/Request** — SP directly queries IdP for an assertion via
  back-channel SOAP.
- **Artifact Resolution** — exchange a small artifact for the full SAML message
  via a back-channel SOAP call.
- **Name Identifier Management** — update or terminate NameID mappings.

### 2.3 SAML Messages

#### AuthnRequest (SP → IdP)

The SP sends this to request authentication. It is an XML document containing:

```xml
<samlp:AuthnRequest
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    ID="_id-abc123"
    Version="2.0"
    IssueInstant="2024-01-15T10:30:00Z"
    Destination="https://idp.example.com/sso"
    ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
    AssertionConsumerServiceURL="https://ggid.example.com/saml/{tenant_id}/acs">
  <saml:Issuer
      xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">
      https://ggid.example.com/saml/{tenant_id}
  </saml:Issuer>
  <samlp:NameIDPolicy
      AllowCreate="true"
      Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"/>
</samlp:AuthnRequest>
```

Key fields:
- **ID** — unique request identifier (used to match the response via `InResponseTo`).
- **Destination** — the IdP's SSO service URL.
- **AssertionConsumerServiceURL** — where the IdP should post the response back.
- **Issuer** — the SP's entity ID (per-tenant in GGID).
- **NameIDPolicy** — what NameID format the SP expects.

The AuthnRequest **may be signed** by the SP. Some IdPs require signed requests;
others accept unsigned. GGID should support both, defaulting to signed.

#### SAML Response (IdP → SP)

Wraps one or more assertions:

```xml
<samlp:Response
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    ID="_response-xyz789"
    InResponseTo="_id-abc123"
    Version="2.0"
    IssueInstant="2024-01-15T10:30:05Z"
    Destination="https://ggid.example.com/saml/{tenant_id}/acs">
  <saml:Issuer>https://idp.example.com</saml:Issuer>
  <samlp:Status>
    <samlp:StatusCode
        Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </samlp:Status>
  <saml:Assertion>
    <!-- Assertion details in §2.3 below -->
  </saml:Assertion>
</samlp:Response>
```

- **InResponseTo** — must match the AuthnRequest ID (for SP-initiated). Absent for
  IdP-initiated (unsolicited).
- **Status** — `Success` or an error code.

#### SAML Assertion

The core security token:

```xml
<saml:Assertion
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="_assertion-001"
    Version="2.0"
    IssueInstant="2024-01-15T10:30:05Z">
  <saml:Issuer>https://idp.example.com</saml:Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <!-- XML Digital Signature over the assertion -->
  </ds:Signature>
  <saml:Subject>
    <saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">
      alice@acme.com
    </saml:NameID>
    <saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
      <saml:SubjectConfirmationData
          InResponseTo="_id-abc123"
          NotOnOrAfter="2024-01-15T10:35:05Z"
          Recipient="https://ggid.example.com/saml/{tenant_id}/acs"/>
    </saml:SubjectConfirmation>
  </saml:Subject>
  <saml:Conditions
      NotBefore="2024-01-15T10:30:05Z"
      NotOnOrAfter="2024-01-15T10:35:05Z">
    <saml:AudienceRestriction>
      <saml:Audience>https://ggid.example.com/saml/{tenant_id}</saml:Audience>
    </saml:AudienceRestriction>
  </saml:Conditions>
  <saml:AuthnStatement
      AuthnInstant="2024-01-15T10:30:04Z"
      SessionIndex="_session-index-001">
    <saml:AuthnContext>
      <saml:AuthnContextClassRef>
        urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
      </saml:AuthnContextClassRef>
    </saml:AuthnContext>
  </saml:AuthnStatement>
  <saml:AttributeStatement>
    <saml:Attribute Name="email">
      <saml:AttributeValue>alice@acme.com</saml:AttributeValue>
    </saml:Attribute>
    <saml:Attribute Name="groups">
      <saml:AttributeValue>engineers</saml:AttributeValue>
      <saml:AttributeValue>admins</saml:AttributeValue>
    </saml:Attribute>
  </saml:AttributeStatement>
</saml:Assertion>
```

Key security-relevant elements:
- **Signature** — XML Digital Signature. GGID verifies this against the IdP's cert.
- **Subject/NameID** — the authenticated user's identifier.
- **SubjectConfirmation** — proves the assertion was issued for this specific
  request (via `InResponseTo`) and this specific endpoint (via `Recipient`).
- **Conditions** — validity window (`NotBefore`, `NotOnOrAfter`).
- **AudienceRestriction** — the SP entity ID that this assertion is valid for.
- **AttributeStatement** — user attributes for mapping/JIT.

### 2.4 Bindings

Bindings define how SAML XML messages are transported:

| Binding | Mechanism | Used For |
|---------|-----------|----------|
| **HTTP-Redirect** | SAML message is base64-encoded, DEFLATE-compressed, placed in URL query param (`?SAMLRequest=...`) | AuthnRequest SP→IdP (small messages) |
| **HTTP-POST** | SAML message is base64-encoded, placed in HTML form's hidden field, auto-submitted via JavaScript | Response IdP→SP (large messages with signatures) |
| **HTTP-Artifact** | A short reference (artifact) is exchanged via redirect/POST; full message resolved via back-channel SOAP | Large assertions, reduces front-channel payload |
| **SOAP** | SAML message in SOAP body, exchanged server-to-server over TLS | Artifact resolution, attribute queries, SLO |

Typical flow: AuthnRequest via **HTTP-Redirect**, Response via **HTTP-POST**. GGID
should implement HTTP-Redirect and HTTP-POST at minimum.

**Encoding details:**

HTTP-Redirect binding:
```
1. Serialize SAML message to XML
2. DEFLATE (no header, raw)
3. Base64 encode
4. URL-encode
5. Append as query parameter: ?SAMLRequest=<encoded>
   (or ?SAMLResponse=<encoded> for responses)
6. Optionally add: &RelayState=<state>&SigAlg=<algorithm>&Signature=<signature>
```

HTTP-POST binding:
```
1. Serialize SAML message to XML
2. Base64 encode (NOT deflated)
3. Place in hidden form field: <input type="hidden" name="SAMLResponse" value="<encoded>"/>
4. Auto-submit form via JavaScript onload
```

---

## 3. Multi-Tenant SAML Architecture

### 3.1 Per-Tenant SP Configuration

In a single-tenant SAML setup, the SP has one entity ID, one ACS URL, and one
metadata endpoint. In multi-tenant GGID, each tenant needs distinct SP configuration:

```
Tenant: acme-corp (tenant_id: 00000000-0000-0000-0000-000000000001)

  SP Entity ID:  https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001
  ACS URL:       https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001/acs
  Metadata URL:  https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001/metadata
  SLO URL:       https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001/slo
```

**Design decision: tenant_id in URL path vs. subdomain.**

| Approach | Example | Pros | Cons |
|----------|---------|------|------|
| Path-based | `/saml/{tenant_id}/acs` | Simple routing, no DNS needed, works behind LB | Tenant ID exposed in URL |
| Subdomain-based | `{tenant}.ggid.com/saml/acs` | Clean URLs, tenant branding | Requires wildcard DNS cert, tenant resolution from host |

GGID's existing architecture uses `X-Tenant-ID` header for tenant resolution in
API requests. For SAML endpoints (which are browser-redirect targets), the tenant
must be resolvable from the URL itself, since browser redirects don't carry custom
headers. **Path-based tenant identification** (`/saml/{tenant_id}/...`) is the
recommended approach for the SAML endpoints.

However, for **user-facing URLs** (the login page), subdomain-based resolution
(`acme.ggid.com`) provides a better UX. GGID can support both:

```
User visits: acme.ggid.com/login
  → Middleware resolves subdomain → tenant_id
  → Redirect to: /saml/{tenant_id}/login (SP-initiated)
  → IdP posts back to: /saml/{tenant_id}/acs
```

### 3.2 Per-Tenant IdP Configuration

Each tenant configures their IdP connection to GGID. The tenant provides:

```
IdP Configuration:
  idp_entity_id:   http://www.okta.com/exk1abc (unique to this tenant's Okta app)
  idp_sso_url:     https://acme.okta.com/app/ggid/exk1abc/sso/saml
  idp_cert_pem:    -----BEGIN CERTIFICATE-----\nMIID...(IdP's signing cert)
  idp_slo_url:     https://acme.okta.com/app/ggid/exk1abc/slo/saml (optional)
  name_id_format:  urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress
```

**How the tenant provides this configuration:**

1. **Metadata URL** (recommended): The tenant provides the IdP's metadata endpoint
   URL. GGID fetches and parses the XML metadata to extract entity ID, SSO URL,
   and certificate automatically. GGID periodically refreshes (default: every 24h)
   to pick up certificate rotations.

2. **Metadata XML upload**: The tenant downloads the IdP metadata XML file and
   uploads it to GGID's admin console. GGID parses it at upload time.

3. **Manual configuration**: The tenant's IT admin enters the entity ID, SSO URL,
   and certificate PEM directly in a form. This is the most error-prone method.

GGID's existing `IdPConfig` struct (`services/auth/internal/service/idp_federation.go`)
already models this:

```go
type IdPConfig struct {
    ID            string            `json:"id"`
    Name          string            `json:"name"`
    Protocol      string            `json:"protocol"`       // "saml" or "oidc"
    EntityID      string            `json:"entity_id"`      // SAML EntityID
    SSOURL        string            `json:"sso_url"`        // IdP SSO endpoint
    CertPEM       string            `json:"cert_pem"`       // IdP signing certificate
    NameIDFormat  string            `json:"name_id_format"` // NameID format URI
    AttrMap       map[string]string `json:"attr_map"`       // IdP attr → GGID field
    AutoProvision bool              `json:"auto_provision"` // JIT provisioning
    Enabled       bool              `json:"enabled"`
    CreatedAt     time.Time         `json:"created_at"`
}
```

This struct needs to be extended with tenant scoping and multi-key support for
production multi-tenant SAML.

### 3.3 Certificate Management

Certificate management is the most operationally complex aspect of SAML federation.
Each side of the trust relationship has signing certificates, and both rotate.

#### IdP Certificate (per-tenant)

Each tenant's IdP signs assertions with its own private key. GGID must have the
matching public certificate to verify those signatures.

**Challenges:**
- IdPs rotate certificates periodically (Azure AD: ~2 years, Okta: configurable).
- During rotation, both old and new certificates may be valid simultaneously.
- If GGID only has the old cert and the IdP switches to the new one, authentication
  breaks immediately.

**Multi-key support:**
GGID should store **multiple certificates** per tenant IdP and accept assertions
signed by any of them. When verifying, try each certificate in order:

```go
func VerifyAssertionSignature(assertion *SAMLAssertion, certs []*x509.Certificate) error {
    for _, cert := range certs {
        if err := verifyWithCert(assertion, cert); err == nil {
            return nil // verified successfully
        }
    }
    return fmt.Errorf("assertion signature did not match any trusted certificate")
}
```

**Metadata-driven rotation:**
When GGID refreshes IdP metadata, it should:
1. Parse new certificates from the metadata.
2. Compare against stored certificates.
3. If a new certificate appears, add it alongside the existing one (do not remove
   the old one yet).
4. Log a warning that a new cert has been detected.
5. After a grace period (configurable, default 7 days), remove certificates that
   are no longer in the metadata.

#### SP Certificate (GGID's signing cert)

GGID signs AuthnRequests. The question: **one global key pair or per-tenant?**

| Strategy | Pros | Cons | Recommendation |
|----------|------|------|----------------|
| Global SP key | Simple, one rotation, one cert in all metadata | Less isolation | Small deployments |
| Per-tenant SP key | Full isolation, compromise of one key doesn't affect others | Complex rotation, more keys to manage | Enterprise/multi-tenant |

**Recommendation: per-tenant SP key pair** for enterprise deployments. Each tenant's
metadata publishes only that tenant's SP certificate. This provides cryptographic
isolation between tenants.

```
Tenant acme-corp:
  SP private key: /secrets/saml/acme-corp/sp.key
  SP certificate: /secrets/saml/acme-corp/sp.crt

Tenant globex:
  SP private key: /secrets/saml/globex/sp.key
  SP certificate: /secrets/saml/globex/sp.crt
```

For smaller deployments or MVP, a global key pair with per-tenant entity IDs is
acceptable.

#### Key Rotation Process (SP side)

```
Phase 1: Generate new key pair
  → Store new key + cert alongside old ones
  → Do NOT publish yet

Phase 2: Publish new cert in metadata
  → Metadata now contains BOTH old and new cert
  → IdPs that refresh metadata will see both
  → Assertions may be signed with either key (prefer new)

Phase 3: Wait for propagation
  → Typically 1-2 metadata refresh cycles (24-48h)
  → This ensures all IdPs have the new cert

Phase 4: Switch signing
  → Start signing AuthnRequests with the new key
  → Assertions signed with old key still accepted during grace period

Phase 5: Remove old cert
  → Remove old cert from metadata
  → Purge old key from key store
  → Log completion
```

### 3.4 Dynamic Metadata Exchange

SAML metadata is the standard mechanism for exchanging configuration between SP
and IdP. It's an XML document that describes:

- Entity ID
- SSO service URL and bindings
- Signing certificates
- Supported NameID formats
- Artifact resolution service URL
- Single Logout service URL

#### IdP Metadata (GGID fetches from tenant's IdP)

```xml
<EntityDescriptor
    xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="http://www.okta.com/exk1abc">
  <IDPSSODescriptor
      protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>MIID...(base64 DER)...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <SingleSignOnService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://acme.okta.com/app/ggid/exk1abc/sso/saml"/>
    <SingleSignOnService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://acme.okta.com/app/ggid/exk1abc/sso/saml"/>
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
  </IDPSSODescriptor>
</EntityDescriptor>
```

GGID fetches this metadata on a schedule (default every 24h) and parses:
- `entityID` → `idp_entity_id`
- `SingleSignOnService Location` → `idp_sso_url`
- `X509Certificate` → `idp_cert_pem` (appends to certificate list)
- `NameIDFormat` → `name_id_format`

**Metadata validation:**
- Fetch over HTTPS only (never HTTP for production).
- Validate XML signature if metadata is signed.
- Verify the entity ID matches the expected value.
- Reject metadata with certificates expiring within 7 days (warn).

#### SP Metadata (GGID publishes for tenant's IdP admin)

GGID generates SP metadata per tenant. The tenant's IdP admin imports this
metadata to configure their side:

```xml
<EntityDescriptor
    xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="https://ggid.example.com/saml/{tenant_id}">
  <SPSSODescriptor
      protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"
      AuthnRequestsSigned="true"
      WantAssertionsSigned="true">
    <KeyDescriptor use="signing">
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>...(GGID SP cert for this tenant)...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <AssertionConsumerService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://ggid.example.com/saml/{tenant_id}/acs"
        index="0"
        isDefault="true"/>
    <AssertionConsumerService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Artifact"
        Location="https://ggid.example.com/saml/{tenant_id}/acs"
        index="1"/>
    <SingleLogoutService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://ggid.example.com/saml/{tenant_id}/slo"/>
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
  </SPSSODescriptor>
</EntityDescriptor>
```

This metadata is served at `GET /saml/{tenant_id}/metadata` and can be consumed
directly by the IdP's metadata import feature.

**Metadata signing:**
For maximum security, GGID can sign its SP metadata XML. This prevents tampering
if the metadata is transferred via insecure channels. The signature uses the same
SP signing key:

```xml
<ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
  <ds:SignedInfo>
    <ds:CanonicalizationMethod
        Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
    <ds:SignatureMethod
        Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
    <ds:Reference>
      <ds:Transforms>
        <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
        <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
      </ds:Transforms>
      <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
      <ds:DigestValue>...</ds:DigestValue>
    </ds:Reference>
  </ds:SignedInfo>
  <ds:SignatureValue>...</ds:SignatureValue>
</ds:Signature>
```

---

## 4. SP-Initiated Flow (Multi-Tenant)

This is the primary SSO flow. The user starts at GGID, is redirected to their
IdP, authenticates, and returns with a valid assertion.

### Step-by-Step

1. **User accesses GGID** with tenant context. This could be:
   - `https://acme.ggid.com/dashboard` (subdomain → tenant resolution)
   - `https://ggid.example.com/saml/{tenant_id}/login` (explicit path)
   - `https://ggid.example.com/login?tenant=acme` (query param)

2. **GGID resolves tenant_id** from the URL, subdomain, or header. The tenant
   context is loaded using `pkg/tenant.FromContext()`.

3. **GGID looks up SAML config** for the tenant from the database:
   ```go
   config, err := samlService.GetConfig(ctx, tenantID)
   ```

4. **GGID generates an AuthnRequest** signed with the tenant's SP private key.
   The request includes:
   - Unique `ID` (stored in session/cache for `InResponseTo` matching)
   - The tenant's SP entity ID as `Issuer`
   - The tenant's ACS URL as `AssertionConsumerServiceURL`
   - `RelayState` — carries a return URL or tenant context to restore after SSO

5. **User's browser is redirected** to the tenant's IdP SSO URL via HTTP-Redirect
   binding:
   ```
   302 Location: https://acme.okta.com/app/ggid/exk1abc/sso/saml?SAMLRequest=<encoded>&RelayState=<state>&SigAlg=<alg>&Signature=<sig>
   ```

6. **User authenticates at the IdP.** This may involve:
   - Password entry
   - MFA challenge (TOTP, push notification, biometric)
   - Session reuse (if the user already has an IdP session, they may skip
     authentication entirely — true SSO)

7. **IdP posts SAML Response** to GGID's ACS URL via HTTP-POST binding:
   ```html
   <form method="POST" action="https://ggid.example.com/saml/{tenant_id}/acs">
     <input type="hidden" name="SAMLResponse" value="<base64 XML>"/>
     <input type="hidden" name="RelayState" value="<state>"/>
   </form>
   <script>document.forms[0].submit();</script>
   ```

8. **GGID validates the assertion:**
   - **Issuer** matches the tenant's configured IdP entity ID.
   - **Signature** verifies against the tenant's stored IdP certificate(s).
   - **Conditions** (NotBefore, NotOnOrAfter) are within the acceptable window
     (accounting for clock skew).
   - **Audience** matches the tenant's SP entity ID.
   - **InResponseTo** matches the stored AuthnRequest ID (prevents CSRF/replay).
   - **Recipient** matches the ACS URL.
   - **Assertion ID** is not a replay (checked against consumed-assertion cache).

9. **GGID maps IdP attributes** to local user fields using the tenant's
   `attr_map` configuration.

10. **GGID creates or looks up the user:**
    - If JIT is enabled and the user doesn't exist → create account.
    - If the user exists → link/update.
    - Assign tenant-scoped roles/groups.

11. **GGID creates a session** and redirects the user to their original destination
    (from `RelayState`):
    ```
    302 Location: https://acme.ggid.com/dashboard
    Set-Cookie: session=<jwt>; HttpOnly; Secure; SameSite=Lax
    ```

### ASCII Sequence Diagram

```
  User          Browser           GGID (SP)           IdP (Okta)
   │               │                  │                    │
   │  (1) Visit     │                  │                    │
   │─ ─ ─ ─ ─ ─ ─►│                  │                    │
   │               │  GET /dashboard  │                    │
   │               │─ ─ ─ ─ ─ ─ ─ ─ ─►│                    │
   │               │                  │                    │
   │               │                  │ (2) Resolve tenant │
   │               │                  │     Load SAML cfg  │
   │               │                  │                    │
   │               │                  │ (3) Gen AuthnReq   │
   │               │                  │     Sign w/ SP key │
   │               │                  │     Store req ID   │
   │               │                  │                    │
   │               │  302 Redirect    │                    │
   │               │◄─ ─ ─ ─ ─ ─ ─ ─ ─│                    │
   │               │  Location: IdP   │                    │
   │               │  ?SAMLRequest=   │                    │
   │               │                  │                    │
   │               │  (4) Follow redirect                   │
   │               │─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─►│
   │               │                  │                    │
   │               │                  │  (5) Login page    │
   │               │◄─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│
   │               │                  │                    │
   │  (6) Enter    │                  │                    │
   │  credentials  │                  │                    │
   │─ ─ ─ ─ ─ ─ ─►│                  │                    │
   │               │  POST /login     │                    │
   │               │─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─►│
   │               │                  │                    │
   │               │                  │  (7) Authenticate  │
   │               │                  │      Check MFA     │
   │               │                  │      Gen assertion │
   │               │                  │      Sign w/ IdP   │
   │               │                  │      key           │
   │               │                  │                    │
   │               │  (8) POST form   │                    │
   │               │  SAMLResponse=   │                    │
   │               │  <base64 XML>    │                    │
   │               │◄─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│
   │               │                  │                    │
   │               │  (9) POST /acs   │                    │
   │               │  SAMLResponse    │                    │
   │               │─ ─ ─ ─ ─ ─ ─ ─ ─►│                    │
   │               │                  │                    │
   │               │                  │ (10) Validate:     │
   │               │                  │  - Issuer match    │
   │               │                  │  - Signature       │
   │               │                  │  - Conditions      │
   │               │                  │  - Audience        │
   │               │                  │  - InResponseTo    │
   │               │                  │  - Not replayed    │
   │               │                  │                    │
   │               │                  │ (11) Map attrs     │
   │               │                  │      Find/create   │
   │               │                  │      user          │
   │               │                  │      Create session│
   │               │                  │                    │
   │               │  302 Redirect    │                    │
   │               │  Set-Cookie      │                    │
   │               │◄─ ─ ─ ─ ─ ─ ─ ─ ─│                    │
   │               │                  │                    │
   │  (12) Dashboard│                 │                    │
   │◄─ ─ ─ ─ ─ ─ ─│                  │                    │
   │               │                  │                    │
```

---

## 5. IdP-Initiated Flow (Multi-Tenant)

In this flow, the user starts at their IdP (e.g., they click the GGID app icon
in their Okta dashboard). The IdP sends an unsolicited SAML assertion to GGID.

### Step-by-Step

1. **User authenticates at their IdP** (e.g., logs into Okta).

2. **User clicks the GGID app** in the IdP's app launcher. The IdP generates a
   SAML assertion **without** a corresponding `AuthnRequest` (hence "unsolicited").

3. **IdP posts the SAML Response** to GGID's ACS URL. The response has:
   - No `InResponseTo` value (or it's omitted).
   - The `Issuer` field set to the IdP's entity ID.
   - The `Destination` set to the tenant's ACS URL.

4. **GGID determines the tenant** from the URL path:
   ```
   POST /saml/{tenant_id}/acs
   ```
   Alternatively, GGID can determine the tenant from the assertion's `Issuer` field
   by looking up which tenant has that IdP entity ID configured.

5. **GGID validates the assertion:**
   - Same checks as SP-initiated, **except** `InResponseTo` is not checked
     (or must be empty/absent).
   - This is the security-sensitive difference.

6. **GGID creates the session** and redirects.

### Security Implications of IdP-Initiated SSO

IdP-initiated SSO is **inherently riskier** than SP-initiated:

- **No CSRF protection from InResponseTo.** Since there's no prior AuthnRequest,
   there's no `InResponseTo` to match. An attacker could trick a user into
   submitting a pre-captured assertion.
- **Spoofing risk.** If an attacker can obtain a valid assertion (e.g., via their
   own IdP account), they might try to submit it to access a different tenant.
   The `AudienceRestriction` check mitigates this — the assertion's audience
   must match the SP entity ID.

### Mitigations

| Risk | Mitigation |
|------|------------|
| Missing InResponseTo | Require the tenant to explicitly enable IdP-initiated SSO (default: disabled) |
| Cross-tenant spoofing | Validate `Audience` matches the tenant's SP entity ID |
| Replay | Track assertion IDs, reject consumed ones (TTL = NotOnOrAfter) |
| Unintended login | Validate `Recipient` matches the exact ACS URL for this tenant |
| IdP spoofing | Strict trust store per tenant — only accept assertions from the configured IdP entity ID |

### ASCII Sequence Diagram

```
  User          Browser           IdP (Okta)         GGID (SP)
   │               │                  │                    │
   │  (1) Login at │                  │                    │
   │  IdP dashboard│                  │                    │
   │─ ─ ─ ─ ─ ─ ─►│                  │                    │
   │               │  GET /dashboard  │                    │
   │               │─ ─ ─ ─ ─ ─ ─ ─ ─►│                    │
   │               │                  │                    │
   │               │                  │ (2) User already   │
   │               │                  │     has session    │
   │               │                  │                    │
   │  (3) Click    │                  │                    │
   │  GGID app     │                  │                    │
   │─ ─ ─ ─ ─ ─ ─►│                  │                    │
   │               │  Click app icon  │                    │
   │               │─ ─ ─ ─ ─ ─ ─ ─ ─►│                    │
   │               │                  │                    │
   │               │                  │ (4) Gen assertion  │
   │               │                  │     No AuthnReq    │
   │               │                  │     No InResponseTo│
   │               │                  │     Sign w/ IdP    │
   │               │                  │     key            │
   │               │                  │                    │
   │               │  (5) POST form   │                    │
   │               │  to GGID ACS     │                    │
   │               │◄─ ─ ─ ─ ─ ─ ─ ─ ─│                    │
   │               │                  │                    │
   │               │  (6) POST /saml/{tenant_id}/acs       │
   │               │  SAMLResponse=<base64>                │
   │               │─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─►│
   │               │                  │                    │
   │               │                  │ (7) Determine      │
   │               │                  │     tenant from    │
   │               │                  │     URL path       │
   │               │                  │                    │
   │               │                  │ (8) Validate:      │
   │               │                  │  - Issuer match ✓  │
   │               │                  │  - Signature ✓     │
   │               │                  │  - Conditions ✓    │
   │               │                  │  - Audience match ✓│
   │               │                  │  - Recipient match ✓│
   │               │                  │  - No InResponseTo │
   │               │                  │    (allowed if     │
   │               │                  │     IdP-init enabled)│
   │               │                  │  - Not replayed ✓  │
   │               │                  │                    │
   │               │                  │ (9) Map attrs      │
   │               │                  │     Create session │
   │               │                  │                    │
   │               │  302 Redirect    │                    │
   │               │  Set-Cookie      │                    │
   │               │◄─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│
   │               │                  │                    │
   │  (10) Dashboard│                 │                    │
   │◄─ ─ ─ ─ ─ ─ ─│                  │                    │
```

---

## 6. Attribute Mapping

### 6.1 The Problem

Every IdP sends user attributes differently. There is **no universal standard** for
attribute names across IdPs:

| Attribute | Okta | Azure AD | ADFS | Google Workspace |
|-----------|------|----------|------|------------------|
| Email | `email` | `http://schemas.xmlsoap.org/ws/2005/05/claims/emailaddress` | `http://schemas.xmlsoap.org/ws/2005/05/claims/emailaddress` | `email` |
| First name | `firstName` | `http://schemas.xmlsoap.org/ws/2005/05/claims/givenname` | `givenname` | `firstName` |
| Last name | `lastName` | `http://schemas.xmlsoap.org/ws/2005/05/claims/surname` | `surname` | `lastName` |
| Groups | `groups` | `http://schemas.microsoft.com/ws/2008/06/identity/claims/groups` | `memberOf` | — |
| Department | `department` | `http://schemas.xmlsoap.org/ws/2005/05/claims/department` | `department` | — |

Additionally, some IdPs use **Claim URIs** (long XML namespace strings), while
others use short names. GGID must normalize these to its internal user model.

### 6.2 GGID's Internal User Model

GGID's `Identity` service has a standard user representation:

```go
type User struct {
    ID          uuid.UUID `json:"id"`
    TenantID    uuid.UUID `json:"tenant_id"`
    Username    string    `json:"username"`
    Email       string    `json:"email"`
    FirstName   string    `json:"first_name"`
    LastName    string    `json:"last_name"`
    DisplayName string    `json:"display_name"`
    Status      string    `json:"status"`
    // ...
}
```

### 6.3 Per-Tenant Attribute Map Configuration

Each tenant configures a mapping from IdP attribute names to GGID field names:

```json
{
  "email":        "http://schemas.xmlsoap.org/ws/2005/05/claims/emailaddress",
  "first_name":   "http://schemas.xmlsoap.org/ws/2005/05/claims/givenname",
  "last_name":    "http://schemas.xmlsoap.org/ws/2005/05/claims/surname",
  "display_name": "http://schemas.xmlsoap.org/ws/2005/05/claims/name",
  "groups":       "http://schemas.xmlsoap.org/claims/Group",
  "department":   "http://schemas.xmlsoap.org/ws/2005/05/claims/department"
}
```

The **keys** are GGID internal field names. The **values** are the IdP-specific
attribute names found in the assertion's `AttributeStatement`.

### 6.4 NameID Format

The NameID element in the assertion subject identifies the user. Common formats:

| Format URI | Meaning | Example |
|------------|---------|---------|
| `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress` | Email address | `alice@acme.com` |
| `urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified` | Opaque identifier | `abc123def456` |
| `urn:oasis:names:tc:SAML:2.0:nameid-format:persistent` | Persistent opaque ID | `a1b2c3d4` |
| `urn:oasis:names:tc:SAML:2.0:nameid-format:transient` | One-time ID | `xyz789` |
| `urn:oasis:names:tc:SAML:1.1:nameid-format:WindowsDomainQualifiedName` | Domain\user | `ACME\alice` |

For GGID, the recommended NameID format is `emailAddress` — it provides a stable,
human-readable identifier that can be used for user lookup. However, some tenants
may prefer `persistent` opaque identifiers for privacy reasons.

The `name_id_format` field in the tenant's `IdPConfig` tells GGID what to expect.

### 6.5 Attribute Extraction Implementation

GGID's existing `pkg/saml/assertion.go` already provides the primitives:

```go
// ExtractAttributes returns all attributes as a map
attrs := saml.ExtractAttributes(assertion)
// attrs["email"] = ["alice@acme.com"]
// attrs["groups"] = ["engineers", "admins"]

// GetAttribute returns the first value of a named attribute
email := saml.GetAttribute(assertion, "email")
// "alice@acme.com"
```

The multi-tenant SAML service applies the attribute map:

```go
func ApplyAttributeMap(
    assertion *saml.SAMLAssertion,
    attrMap map[string]string,
) map[string]string {
    result := make(map[string]string)

    // Always map NameID as the primary identifier
    result["name_id"] = assertion.Subject.NameID

    // Apply the per-tenant mapping
    for ggidField, idpAttrName := range attrMap {
        if val := saml.GetAttribute(assertion, idpAttrName); val != "" {
            result[ggidField] = val
        }
    }

    return result
}
```

### 6.6 Multi-Valued Attributes

Groups and roles are typically multi-valued:

```xml
<saml:Attribute Name="groups">
  <saml:AttributeValue>engineers</saml:AttributeValue>
  <saml:AttributeValue>admins</saml:AttributeValue>
  <saml:AttributeValue>security-team</saml:AttributeValue>
</saml:Attribute>
```

GGID's `ExtractAttributes` returns these as `[]string`:

```go
attrs := saml.ExtractAttributes(assertion)
groups := attrs["http://schemas.xmlsoap.org/claims/Group"]
// ["engineers", "admins", "security-team"]
```

### 6.7 Default Attribute Maps per IdP

GGID can ship pre-configured attribute maps for common IdPs:

```go
var DefaultAttrMaps = map[string]map[string]string{
    "okta": {
        "email":      "email",
        "first_name": "firstName",
        "last_name":  "lastName",
        "groups":     "groups",
    },
    "azure_ad": {
        "email":      "http://schemas.xmlsoap.org/ws/2005/05/claims/emailaddress",
        "first_name": "http://schemas.xmlsoap.org/ws/2005/05/claims/givenname",
        "last_name":  "http://schemas.xmlsoap.org/ws/2005/05/claims/surname",
        "groups":     "http://schemas.microsoft.com/ws/2008/06/identity/claims/groups",
    },
    "adfs": {
        "email":      "http://schemas.xmlsoap.org/ws/2005/05/claims/emailaddress",
        "first_name": "http://schemas.xmlsoap.org/ws/2005/05/claims/givenname",
        "last_name":  "http://schemas.xmlsoap.org/ws/2005/05/claims/surname",
        "groups":     "http://schemas.xmlsoap.org/claims/Group",
    },
    "google": {
        "email": "email",
        "first_name": "firstName",
        "last_name": "lastName",
    },
}
```

---

## 7. Just-in-Time (JIT) Provisioning

### 7.1 Concept

JIT provisioning automatically creates a local GGID user account when a user
authenticates via SAML for the first time. Without JIT, an administrator must
pre-create every user account before they can SSO — which defeats much of the
purpose of federated identity.

### 7.2 JIT Flow

```
1. User authenticates at IdP
2. IdP sends SAML assertion to GGID
3. GGID validates assertion
4. GGID extracts NameID (e.g., email)
5. GGID looks up user by (tenant_id, email or NameID)
6. IF user exists:
     → Update attributes from assertion (if configured)
     → Link external identity (if not already linked)
7. IF user does NOT exist AND jit_enabled = true:
     → Create new user with attributes from assertion
     → Assign default role
     → Link external identity
8. IF user does NOT exist AND jit_enabled = false:
     → Reject login: "Account not found. Contact your administrator."
```

### 7.3 JIT Configuration

Per-tenant JIT settings:

```json
{
  "jit_enabled": true,
  "jit_default_role": "member",
  "jit_allowed_domains": ["acme.com", "acme-corp.com"],
  "jit_update_on_login": true,
  "jit_link_external_id": true
}
```

| Setting | Description | Default |
|---------|-------------|---------|
| `jit_enabled` | Whether to auto-create users | `false` |
| `jit_default_role` | Role assigned to JIT-created users | `"member"` |
| `jit_allowed_domains` | Only create users with emails from these domains | `[]` (all) |
| `jit_update_on_login` | Update local attrs from assertion on each login | `true` |
| `jit_link_external_id` | Store NameID as external identity link | `true` |

### 7.4 Security: Domain Validation

A critical security control: **validate the user's email domain** before JIT-creating
the account. Without domain validation, anyone who has an account on the tenant's
IdP (including guest accounts, test accounts, or compromised accounts) could get a
GGID account with access to the tenant's data.

```
Allowed domains: ["acme.com"]

NameID: alice@acme.com        → ✓ Domain matches, JIT allowed
NameID: bob@gmail.com          → ✗ Domain doesn't match, JIT rejected
NameID: charlie@acme-corp.com  → ✗ Not in allowed list, JIT rejected
```

**Domain ownership verification** (recommended for production):

Before enabling JIT for a domain, verify that the tenant actually owns it:
1. GGID generates a random verification token.
2. Tenant adds a DNS TXT record: `_ggid-verify.acme.com TXT "ggid-verify=<token>"`.
3. GGID checks the DNS record.
4. If it matches, the domain is verified and added to `jit_allowed_domains`.

### 7.5 External Identity Linking

GGID's `Identity` service already supports external identities via
`FindExternalIdentity` and `LinkExternalIdentity` (used by the social login flow
in `auth_service.go`). The SAML JIT flow reuses this:

```go
// Check if external identity already exists
link, err := s.identityClient.FindExternalIdentity(
    ctx, tenantID, "saml", nameID,
)
if err == nil && link != nil {
    // User already linked — just update and login
    user := s.identityClient.GetUser(ctx, link.UserID)
    return s.loginExistingUser(ctx, user, assertion)
}

// JIT provisioning
if config.JITEnabled && domainAllowed(email, config.JITAllowedDomains) {
    user, err := s.identityClient.CreateUserFromSocial(
        ctx, tenantID,
        deriveUsername(nameID),
        email,
        name,
        "saml",
        nameID,
        metadata,
    )
    if err != nil {
        return nil, fmt.Errorf("jit provisioning failed: %w", err)
    }

    // Assign default role
    s.assignRole(ctx, tenantID, user.ID, config.JITDefaultRole)

    return s.createSessionForUser(ctx, user)
}

return nil, fmt.Errorf("account not found and JIT is disabled")
```

### 7.6 JIT vs. SCIM

JIT and SCIM (System for Cross-domain Identity Management) serve complementary
purposes:

| Feature | JIT Provisioning | SCIM Provisioning |
|---------|-----------------|-------------------|
| Trigger | User's first SAML login | Admin creates/updates/deletes user in IdP |
| Timing | Just-in-time (lazy) | Pre-provisioned (eager) |
| User creation | GGID creates from assertion | IdP pushes user to GGID via SCIM API |
| Deactivation | Manual or session expiry | IdP pushes deactivation immediately |
| Attribute sync | On each login | On any IdP change |

**Recommendation**: Use both. SCIM for proactive provisioning/deprovisioning,
JIT as a fallback for users not yet provisioned via SCIM. GGID's SCIM 2.0 skeleton
(`services/identity/`) provides the API surface for SCIM.

---

## 8. GGID Implementation Design

### 8.1 Data Model

#### Table: `saml_configs`

Stores per-tenant SAML SP and IdP configuration.

```sql
-- Migration: 000004_create_saml_configs.up.sql

CREATE TABLE IF NOT EXISTS saml_configs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                VARCHAR(255) NOT NULL,           -- human-readable config name

    -- Service Provider (GGID) configuration
    sp_entity_id        TEXT NOT NULL,                    -- e.g. https://ggid.example.com/saml/{tenant_id}
    sp_acs_url          TEXT NOT NULL,                    -- Assertion Consumer Service URL
    sp_slo_url          TEXT,                             -- Single Logout URL (nullable)
    sp_cert_pem         TEXT NOT NULL,                    -- SP signing certificate (PEM)
    sp_key_pem          TEXT NOT NULL,                    -- SP private key (PEM, encrypted at rest)
    sp_metadata_signed  BOOLEAN NOT NULL DEFAULT TRUE,    -- Sign SP metadata?

    -- Identity Provider configuration
    idp_entity_id       TEXT NOT NULL,                    -- e.g. http://www.okta.com/exk1abc
    idp_sso_url         TEXT NOT NULL,                    -- IdP SSO endpoint
    idp_slo_url         TEXT,                             -- IdP SLO endpoint (nullable)
    idp_cert_pem        TEXT NOT NULL,                    -- IdP signing certificate (PEM)
    idp_cert_pem_old    TEXT,                             -- Previous cert (during rotation)
    idp_metadata_url    TEXT,                             -- IdP metadata URL (for auto-refresh)
    idp_metadata_xml    TEXT,                             -- Cached IdP metadata XML
    idp_metadata_refreshed TIMESTAMPTZ,                   -- Last metadata refresh time

    -- Protocol settings
    name_id_format      VARCHAR(255) NOT NULL DEFAULT
        'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress',
    authn_request_signed BOOLEAN NOT NULL DEFAULT TRUE,
    want_assertions_signed BOOLEAN NOT NULL DEFAULT TRUE,
    allow_idp_initiated BOOLEAN NOT NULL DEFAULT FALSE,   -- Allow unsolicited assertions?
    clock_skew_seconds  INT NOT NULL DEFAULT 60,          -- Tolerance for time validation

    -- Attribute mapping (JSON)
    attr_map            JSONB NOT NULL DEFAULT '{}',      -- {"email": "http://...emailaddress", ...}

    -- JIT provisioning
    jit_enabled         BOOLEAN NOT NULL DEFAULT FALSE,
    jit_default_role    VARCHAR(100) NOT NULL DEFAULT 'member',
    jit_allowed_domains JSONB NOT NULL DEFAULT '[]',      -- ["acme.com", "acme-corp.com"]
    jit_update_on_login BOOLEAN NOT NULL DEFAULT TRUE,

    -- Lifecycle
    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT uq_saml_tenant_idp_entity UNIQUE (tenant_id, idp_entity_id)
);

-- Indexes
CREATE INDEX idx_saml_configs_tenant ON saml_configs(tenant_id) WHERE enabled = TRUE;
CREATE INDEX idx_saml_configs_sp_entity ON saml_configs(sp_entity_id);
CREATE INDEX idx_saml_configs_idp_entity ON saml_configs(idp_entity_id);

COMMENT ON TABLE saml_configs IS 'Per-tenant SAML 2.0 federation configuration';
COMMENT ON COLUMN saml_configs.sp_key_pem IS 'SP private key — encrypted at rest with AES-256-GCM';
```

#### Table: `saml_assertion_log`

Tracks consumed assertion IDs for replay protection.

```sql
-- Migration: 000005_create_saml_assertion_log.up.sql

CREATE TABLE IF NOT EXISTS saml_assertion_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    assertion_id    VARCHAR(255) NOT NULL,        -- The Assertion ID attribute
    request_id      VARCHAR(255),                 -- The AuthnRequest ID (InResponseTo)
    name_id         TEXT NOT NULL,                -- The NameID from the assertion
    idp_entity_id   TEXT NOT NULL,                -- Issuer
    ip_address      INET,                         -- Request source IP
    user_agent      TEXT,                         -- Browser UA
    consumed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL          -- TTL = assertion NotOnOrAfter + grace
);

-- Index for replay checking
CREATE UNIQUE INDEX uq_assertion_log_tenant_id ON saml_assertion_log(tenant_id, assertion_id);
CREATE INDEX idx_assertion_log_expires ON saml_assertion_log(expires_at);

COMMENT ON TABLE saml_assertion_log IS 'SAML assertion replay protection (auto-purged after TTL)';
```

#### Table: `saml_idp_metadata_cache`

Stores parsed IdP metadata for efficient lookup and certificate rotation tracking.

```sql
-- Migration: 000006_create_saml_idp_metadata.up.sql

CREATE TABLE IF NOT EXISTS saml_idp_metadata (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    entity_id       TEXT NOT NULL,                 -- IdP entity ID
    metadata_url    TEXT NOT NULL,                 -- Source URL
    metadata_xml    TEXT NOT NULL,                 -- Raw metadata XML
    metadata_hash   VARCHAR(64) NOT NULL,          -- SHA-256 of metadata (for change detection)
    sso_url         TEXT NOT NULL,                 -- Extracted SSO endpoint
    slo_url         TEXT,                          -- Extracted SLO endpoint
    certificates    JSONB NOT NULL DEFAULT '[]',   -- ["cert1-pem", "cert2-pem"] (multi-key)
    cert_expires_at TIMESTAMPTZ,                   -- Earliest expiry across certs
    fetched_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_refresh_at TIMESTAMPTZ NOT NULL,          -- Scheduled refresh time
    signature_valid BOOLEAN NOT NULL DEFAULT FALSE,

    CONSTRAINT uq_idp_metadata_tenant_entity UNIQUE (tenant_id, entity_id)
);

CREATE INDEX idx_idp_metadata_refresh ON saml_idp_metadata(next_refresh_at)
    WHERE next_refresh_at <= NOW();

COMMENT ON TABLE saml_idp_metadata IS 'Cached SAML IdP metadata with multi-key certificate support';
```

### 8.2 SAML Service

The multi-tenant SAML service coordinates all SAML operations. It uses GGID's
existing `pkg/saml/` package for assertion parsing and `pkg/tenant/` for tenant
context.

```go
package service

import (
    "context"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
    "encoding/xml"
    "fmt"
    "time"

    "github.com/ggid/ggid/pkg/saml"
    "github.com/ggid/ggid/pkg/tenant"
    "github.com/google/uuid"
)

// SAMLConfig represents the per-tenant SAML configuration loaded from DB.
type SAMLConfig struct {
    TenantID           uuid.UUID
    SPEntityID         string
    SPACSURL           string
    SPSLOURL           string
    SPCertPEM          string
    SPKeyPEM           string
    SPMetadataSigned   bool

    IDPEntityID        string
    IDPSSOURL          string
    IDPSLOURL          string
    IDPCertPEM         string
    IDPCertPEMOld      string
    IDPMetadataURL     string

    NameIDFormat       string
    AuthnRequestSigned bool
    WantAssertionsSigned bool
    AllowIDPInitiated  bool
    ClockSkewSeconds    int

    AttrMap            map[string]string
    JITEnabled         bool
    JITDefaultRole     string
    JITAllowedDomains  []string
    JITUpdateOnLogin   bool
}

// MultiTenantSAMLService coordinates per-tenant SAML operations.
type MultiTenantSAMLService struct {
    repo          SAMLConfigRepository
    identity      IdentityClient
    sessionMgr    SessionManager
    assertionLog  AssertionLogStore
    keyResolver   KeyResolver
    metadataCache MetadataCache
}

// SAMLConfigRepository provides database access to SAML configuration.
type SAMLConfigRepository interface {
    GetByTenant(ctx context.Context, tenantID uuid.UUID) (*SAMLConfig, error)
    GetBySPEntityID(ctx context.Context, entityID string) (*SAMLConfig, error)
    GetByIDPEntityID(ctx context.Context, entityID string) (*SAMLConfig, error)
    Save(ctx context.Context, config *SAMLConfig) error
    UpdateIDPCert(ctx context.Context, tenantID uuid.UUID, newCert string) error
}

// AssertionLogStore tracks consumed assertion IDs for replay protection.
type AssertionLogStore interface {
    IsConsumed(ctx context.Context, tenantID uuid.UUID, assertionID string) (bool, error)
    MarkConsumed(ctx context.Context, tenantID uuid.UUID, assertionID string, expiresAt time.Time) error
    PurgeExpired(ctx context.Context) error
}

// KeyResolver loads SP signing keys (possibly from HSM/KMS).
type KeyResolver interface {
    GetSPPrivateKey(ctx context.Context, tenantID uuid.UUID) (*rsa.PrivateKey, error)
    GetSPCertificate(ctx context.Context, tenantID uuid.UUID) (*x509.Certificate, error)
    GetIDPCertificates(ctx context.Context, tenantID uuid.UUID) ([]*x509.Certificate, error)
}

// NewMultiTenantSAMLService creates a new SAML service.
func NewMultiTenantSAMLService(
    repo SAMLConfigRepository,
    identity IdentityClient,
    sessionMgr SessionManager,
    assertionLog AssertionLogStore,
    keyResolver KeyResolver,
    metadataCache MetadataCache,
) *MultiTenantSAMLService {
    return &MultiTenantSAMLService{
        repo:          repo,
        identity:      identity,
        sessionMgr:    sessionMgr,
        assertionLog:  assertionLog,
        keyResolver:   keyResolver,
        metadataCache: metadataCache,
    }
}

// GetConfig retrieves the SAML configuration for a tenant.
func (s *MultiTenantSAMLService) GetConfig(
    ctx context.Context,
    tenantID uuid.UUID,
) (*SAMLConfig, error) {
    config, err := s.repo.GetByTenant(ctx, tenantID)
    if err != nil {
        return nil, fmt.Errorf("get saml config for tenant %s: %w", tenantID, err)
    }
    if !config.isEnabled() {
        return nil, fmt.Errorf("saml is not enabled for tenant %s", tenantID)
    }
    return config, nil
}

// GenerateAuthnRequest creates a signed AuthnRequest for SP-initiated SSO.
func (s *MultiTenantSAMLService) GenerateAuthnRequest(
    ctx context.Context,
    tenantID uuid.UUID,
) (*AuthnRequestResult, error) {
    config, err := s.GetConfig(ctx, tenantID)
    if err != nil {
        return nil, err
    }

    requestID := generateRequestID()
    issueInstant := time.Now().UTC()

    // Build the AuthnRequest XML
    authnRequest := buildAuthnRequestXML(
        requestID,
        config.IDPSSOURL,
        config.SPACSURL,
        config.SPEntityID,
        config.NameIDFormat,
        issueInstant,
    )

    // Sign if configured
    var signature string
    if config.AuthnRequestSigned {
        spKey, err := s.keyResolver.GetSPPrivateKey(ctx, tenantID)
        if err != nil {
            return nil, fmt.Errorf("load SP signing key: %w", err)
        }
        signature = signXML(authnRequest, spKey)
    }

    // Store request ID for InResponseTo matching
    s.assertionLog.StoreRequestID(ctx, tenantID, requestID, issueInstant.Add(10*time.Minute))

    // Build the redirect URL (HTTP-Redirect binding)
    redirectURL := buildRedirectURL(
        config.IDPSSOURL,
        authnRequest,
        signature,
        config.AuthnRequestSigned,
    )

    return &AuthnRequestResult{
        RequestID:   requestID,
        RedirectURL: redirectURL,
        RelayState:  generateRelayState(tenantID),
    }, nil
}

// ProcessAssertion validates and processes a SAML assertion at the ACS endpoint.
func (s *MultiTenantSAMLService) ProcessAssertion(
    ctx context.Context,
    tenantID uuid.UUID,
    samlResponseXML []byte,
    relayState string,
) (*SAMLLoginResult, error) {
    config, err := s.GetConfig(ctx, tenantID)
    if err != nil {
        return nil, err
    }

    // Step 1: Parse the SAML Response wrapper
    response, err := saml.ParseResponse(samlResponseXML)
    if err != nil {
        return nil, fmt.Errorf("parse saml response: %w", err)
    }

    // Step 2: Validate status
    if response.Status.StatusCode.Value != saml.StatusCodeSuccess {
        return nil, fmt.Errorf("idp returned error status: %s",
            response.Status.StatusCode.Value)
    }

    // Step 3: Validate issuer
    if response.Issuer != config.IDPEntityID {
        return nil, fmt.Errorf("issuer mismatch: expected %s, got %s",
            config.IDPEntityID, response.Issuer)
    }

    // Step 4: Validate destination
    if response.Destination != config.SPACSURL {
        return nil, fmt.Errorf("destination mismatch: expected %s, got %s",
            config.SPACSURL, response.Destination)
    }

    // Step 5: Check InResponseTo (for SP-initiated flows)
    if response.InResponseTo != "" {
        // SP-initiated: verify we sent this request
        valid, err := s.assertionLog.ValidateRequestID(ctx, tenantID, response.InResponseTo)
        if err != nil || !valid {
            return nil, fmt.Errorf("invalid InResponseTo: %s", response.InResponseTo)
        }
    } else {
        // IdP-initiated: only allowed if explicitly enabled
        if !config.AllowIDPInitiated {
            return nil, fmt.Errorf("unsolicited assertion rejected (IdP-initiated SSO disabled)")
        }
    }

    // Step 6: Parse the assertion
    assertion, err := saml.ParseAssertion(response.AssertionXML)
    if err != nil {
        return nil, fmt.Errorf("parse assertion: %w", err)
    }

    // Step 7: Replay check
    consumed, err := s.assertionLog.IsConsumed(ctx, tenantID, assertion.ID)
    if err != nil {
        return nil, fmt.Errorf("replay check failed: %w", err)
    }
    if consumed {
        return nil, fmt.Errorf("assertion replay detected: ID=%s", assertion.ID)
    }

    // Step 8: Validate conditions (with configurable clock skew)
    if err := assertion.ValidateConditionsWithSkew(
        time.Duration(config.ClockSkewSeconds) * time.Second,
    ); err != nil {
        return nil, fmt.Errorf("condition validation failed: %w", err)
    }

    // Step 9: Validate audience
    if !containsAudience(assertion, config.SPEntityID) {
        return nil, fmt.Errorf("audience mismatch: expected %s", config.SPEntityID)
    }

    // Step 10: Verify signature
    idpCerts, err := s.keyResolver.GetIDPCertificates(ctx, tenantID)
    if err != nil {
        return nil, fmt.Errorf("load IdP certificates: %w", err)
    }
    if err := saml.VerifyAssertionSignature(assertion, idpCerts); err != nil {
        return nil, fmt.Errorf("signature verification failed: %w", err)
    }

    // Step 11: Mark assertion as consumed (replay protection)
    expiresAt := parseNotOnOrAfter(assertion)
    if err := s.assertionLog.MarkConsumed(ctx, tenantID, assertion.ID, expiresAt); err != nil {
        return nil, fmt.Errorf("mark assertion consumed: %w", err)
    }

    // Step 12: Map attributes
    userAttrs := s.applyAttributeMap(assertion, config.AttrMap)

    // Step 13: Find or create user (JIT)
    user, err := s.findOrCreateUser(ctx, config, assertion, userAttrs)
    if err != nil {
        return nil, err
    }

    // Step 14: Update user attributes if configured
    if config.JITUpdateOnLogin {
        if err := s.identity.UpdateUserAttrs(ctx, config.TenantID, user.ID, userAttrs); err != nil {
            // Non-fatal — log but continue
        }
    }

    // Step 15: Create session
    session, err := s.sessionMgr.CreateSession(ctx, user, "saml")
    if err != nil {
        return nil, fmt.Errorf("create session: %w", err)
    }

    return &SAMLLoginResult{
        User:        user,
        Session:     session,
        RedirectURL: decodeRelayState(relayState),
    }, nil
}

// GetSPMetadata generates and returns the SP metadata XML for a tenant.
func (s *MultiTenantSAMLService) GetSPMetadata(
    ctx context.Context,
    tenantID uuid.UUID,
) ([]byte, error) {
    config, err := s.GetConfig(ctx, tenantID)
    if err != nil {
        return nil, err
    }

    cert, err := s.keyResolver.GetSPCertificate(ctx, tenantID)
    if err != nil {
        return nil, fmt.Errorf("load SP certificate: %w", err)
    }

    metadata := buildSPMetadataXML(
        config.SPEntityID,
        config.SPACSURL,
        config.SPSLOURL,
        cert,
        config.AuthnRequestSigned,
        config.WantAssertionsSigned,
        config.NameIDFormat,
    )

    // Sign metadata if configured
    if config.SPMetadataSigned {
        spKey, err := s.keyResolver.GetSPPrivateKey(ctx, tenantID)
        if err != nil {
            return nil, fmt.Errorf("load SP key for metadata signing: %w", err)
        }
        metadata = signXML(metadata, spKey)
    }

    return metadata, nil
}

// RefreshIDPMetadata fetches and parses IdP metadata from the configured URL.
func (s *MultiTenantSAMLService) RefreshIDPMetadata(
    ctx context.Context,
    tenantID uuid.UUID,
) error {
    config, err := s.GetConfig(ctx, tenantID)
    if err != nil {
        return err
    }
    if config.IDPMetadataURL == "" {
        return nil // No metadata URL configured
    }

    // Fetch metadata over HTTPS
    metadataXML, err := s.metadataCache.Fetch(ctx, config.IDPMetadataURL)
    if err != nil {
        return fmt.Errorf("fetch idp metadata: %w", err)
    }

    // Parse metadata
    parsed, err := saml.ParseIDPMetadata(metadataXML)
    if err != nil {
        return fmt.Errorf("parse idp metadata: %w", err)
    }

    // Validate entity ID matches
    if parsed.EntityID != config.IDPEntityID {
        return fmt.Errorf("metadata entity ID mismatch: expected %s, got %s",
            config.IDPEntityID, parsed.EntityID)
    }

    // Update certificates (multi-key support)
    newCerts := parsed.SigningCertificates
    if len(newCerts) > 0 {
        // Store all certs — keep old ones during grace period
        if err := s.repo.UpdateIDPCerts(ctx, tenantID, newCerts); err != nil {
            return fmt.Errorf("update idp certs: %w", err)
        }
    }

    // Update SSO URL if changed
    if parsed.SSOURL != "" && parsed.SSOURL != config.IDPSSOURL {
        if err := s.repo.UpdateSSOURL(ctx, tenantID, parsed.SSOURL); err != nil {
            return fmt.Errorf("update sso url: %w", err)
        }
    }

    // Cache the metadata
    return s.metadataCache.Store(ctx, tenantID, parsed)
}
```

### 8.3 HTTP Routes

SAML endpoints are registered on the auth service's HTTP router. These routes
must be accessible **without** JWT authentication (they're browser redirect
targets). However, tenant resolution is required.

```go
package server

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
)

// RegisterSAMLRoutes sets up the SAML federation endpoints.
// These routes are NOT behind JWT auth — they're browser redirect targets.
func (s *Server) RegisterSAMLRoutes(r chi.Router) {
    r.Route("/saml/{tenant_id}", func(r chi.Router) {
        r.Use(s.tenantResolutionMiddleware)

        // SP Metadata — public, returns XML
        r.Get("/metadata", s.handleSPMetadata)

        // Initiate SP-initiated SSO — redirects to IdP
        r.Get("/login", s.handleSAMLLogin)

        // Assertion Consumer Service — receives POST from IdP
        r.Post("/acs", s.handleACS)
        r.Get("/acs", s.handleACS) // some IdPs use GET

        // Single Logout
        r.Post("/slo", s.handleSLO)
        r.Get("/slo", s.handleSLO)

        // IdP metadata proxy (optional — for debugging)
        r.Get("/metadata/idp", s.handleIDPMetadata)
    })
}

// tenantResolutionMiddleware extracts tenant_id from the URL path.
func (s *Server) tenantResolutionMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantIDStr := chi.URLParam(r, "tenant_id")
        tenantID, err := uuid.Parse(tenantIDStr)
        if err != nil {
            http.Error(w, "invalid tenant_id", http.StatusBadRequest)
            return
        }

        // Attach tenant context to request
        ctx := tenant.WithContext(r.Context(), &tenant.Context{
            TenantID: tenantID,
        })
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// handleSPMetadata returns the SP metadata XML for IdP configuration.
func (s *Server) handleSPMetadata(w http.ResponseWriter, r *http.Request) {
    tc, err := tenant.FromContext(r.Context())
    if err != nil {
        http.Error(w, "tenant context required", http.StatusBadRequest)
        return
    }

    metadata, err := s.samlService.GetSPMetadata(r.Context(), tc.TenantID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/xml")
    w.Write(metadata)
}

// handleSAMLLogin initiates SP-initiated SSO by redirecting to the IdP.
func (s *Server) handleSAMLLogin(w http.ResponseWriter, r *http.Request) {
    tc, err := tenant.FromContext(r.Context())
    if err != nil {
        http.Error(w, "tenant context required", http.StatusBadRequest)
        return
    }

    result, err := s.samlService.GenerateAuthnRequest(r.Context(), tc.TenantID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, result.RedirectURL, http.StatusFound)
}

// handleACS processes the SAML assertion posted by the IdP.
func (s *Server) handleACS(w http.ResponseWriter, r *http.Request) {
    tc, err := tenant.FromContext(r.Context())
    if err != nil {
        http.Error(w, "tenant context required", http.StatusBadRequest)
        return
    }

    samlResponse := r.FormValue("SAMLResponse")
    relayState := r.FormValue("RelayState")

    if samlResponse == "" {
        http.Error(w, "missing SAMLResponse", http.StatusBadRequest)
        return
    }

    decoded, err := base64Decode(samlResponse)
    if err != nil {
        http.Error(w, "invalid SAMLResponse encoding", http.StatusBadRequest)
        return
    }

    result, err := s.samlService.ProcessAssertion(
        r.Context(), tc.TenantID, decoded, relayState,
    )
    if err != nil {
        // Log the error for debugging
        s.logger.Error("SAML ACS error", "tenant", tc.TenantID, "err", err)
        http.Error(w, "SAML authentication failed", http.StatusUnauthorized)
        return
    }

    // Set session cookie
    s.setSessionCookie(w, result.Session)

    // Redirect to RelayState or default
    redirectURL := result.RedirectURL
    if redirectURL == "" {
        redirectURL = "/dashboard"
    }
    http.Redirect(w, r, redirectURL, http.StatusFound)
}
```

**Route summary:**

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/saml/{tenant_id}/metadata` | None | Returns SP metadata XML |
| GET | `/saml/{tenant_id}/login` | None | Initiates SP-initiated SSO (302 to IdP) |
| POST | `/saml/{tenant_id}/acs` | None | Assertion Consumer Service |
| GET | `/saml/{tenant_id}/acs` | None | ACS (GET variant, some IdPs use this) |
| POST | `/saml/{tenant_id}/slo` | Session | Single Logout receiver |
| GET | `/saml/{tenant_id}/slo` | Session | SLO receiver (GET variant) |
| GET | `/saml/{tenant_id}/metadata/idp` | Admin | Proxies/returns cached IdP metadata |

### 8.4 Certificate Management

```go
package service

import (
    "context"
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/pem"
    "fmt"
    "math/big"
    "time"

    "github.com/google/uuid"
)

// CertificateManager handles SP certificate generation, rotation, and IdP
// certificate refresh.
type CertificateManager struct {
    repo      SAMLConfigRepository
    secretKey []byte // AES-256 key for encrypting private keys at rest
}

// GenerateSPKeyPair creates a new RSA key pair and self-signed certificate
// for a tenant's SP.
func (cm *CertificateManager) GenerateSPKeyPair(
    ctx context.Context,
    tenantID uuid.UUID,
    entityID string,
) (*SPKeyPair, error) {
    // Generate RSA 2048-bit key
    privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        return nil, fmt.Errorf("generate rsa key: %w", err)
    }

    // Create self-signed certificate (valid 1 year)
    template := &x509.Certificate{
        SerialNumber:          big.NewInt(time.Now().UnixNano()),
        Subject:               pkix.Name{CommonName: entityID},
        NotBefore:             time.Now().Add(-time.Hour),
        NotAfter:              time.Now().AddDate(1, 0, 0),
        KeyUsage:              x509.KeyUsageDigitalSignature,
        ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
        BasicConstraintsValid: true,
    }

    certDER, err := x509.CreateCertificate(
        rand.Reader, template, template, &privateKey.PublicKey, privateKey,
    )
    if err != nil {
        return nil, fmt.Errorf("create certificate: %w", err)
    }

    // Encode to PEM
    keyPEM := pem.EncodeToMemory(&pem.Block{
        Type:  "RSA PRIVATE KEY",
        Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
    })
    certPEM := pem.EncodeToMemory(&pem.Block{
        Type:  "CERTIFICATE",
        Bytes: certDER,
    })

    // Encrypt private key at rest
    encryptedKey, err := encryptAESGCM(cm.secretKey, keyPEM)
    if err != nil {
        return nil, fmt.Errorf("encrypt private key: %w", err)
    }

    return &SPKeyPair{
        PrivateKeyPEM:     encryptedKey, // encrypted
        CertificatePEM:    string(certPEM),
        CertificateExpiry: template.NotAfter,
    }, nil
}

// RotateSPCertificate performs a zero-downtime SP certificate rotation.
//
// Phase 1: Generate new key pair
// Phase 2: Store alongside existing key (metadata publishes both)
// Phase 3: Wait for propagation
// Phase 4: Switch to new key for signing
// Phase 5: Remove old key
func (cm *CertificateManager) RotateSPCertificate(
    ctx context.Context,
    tenantID uuid.UUID,
) error {
    config, err := cm.repo.GetByTenant(ctx, tenantID)
    if err != nil {
        return err
    }

    // Phase 1: Generate new key pair
    newKeyPair, err := cm.GenerateSPKeyPair(ctx, tenantID, config.SPEntityID)
    if err != nil {
        return fmt.Errorf("generate new key pair: %w", err)
    }

    // Phase 2: Store alongside existing (both certs valid)
    if err := cm.repo.SetPendingSPKey(ctx, tenantID, newKeyPair); err != nil {
        return fmt.Errorf("store pending key: %w", err)
    }

    // Metadata now publishes BOTH old and new certificates
    // IdPs that refresh metadata will see both

    // Phase 3-5 happen asynchronously via a scheduled job:
    //   - After grace period (configurable, default 48h), promote the new key
    //   - Start signing with new key
    //   - Remove old key from metadata

    return nil
}

// PromotePendingKey switches the signing key to the pending key.
// Called by a scheduled job after the grace period.
func (cm *CertificateManager) PromotePendingKey(
    ctx context.Context,
    tenantID uuid.UUID,
) error {
    config, err := cm.repo.GetByTenant(ctx, tenantID)
    if err != nil {
        return err
    }

    if config.PendingSPKey == nil {
        return nil // No pending key
    }

    // Archive old key (keep for grace period, then purge)
    if err := cm.repo.ArchiveSPKey(ctx, tenantID, config.SPCertPEM); err != nil {
        return err
    }

    // Promote pending key to active
    if err := cm.repo.PromoteSPKey(ctx, tenantID); err != nil {
        return err
    }

    return nil
}

// RefreshIDPCertificates fetches IdP metadata and updates stored certificates.
// Supports multi-key: both old and new certs are kept during rotation.
func (cm *CertificateManager) RefreshIDPCertificates(
    ctx context.Context,
    tenantID uuid.UUID,
    metadataURL string,
) error {
    // Fetch metadata XML over HTTPS
    metadataXML, err := fetchHTTPS(metadataURL)
    if err != nil {
        return fmt.Errorf("fetch metadata: %w", err)
    }

    // Parse certificates from metadata
    certs, err := parseCertificatesFromMetadata(metadataXML)
    if err != nil {
        return fmt.Errorf("parse certificates: %w", err)
    }

    // Get currently stored certificates
    config, err := cm.repo.GetByTenant(ctx, tenantID)
    if err != nil {
        return err
    }

    currentCerts := []string{config.IDPCertPEM}
    if config.IDPCertPEMOld != "" {
        currentCerts = append(currentCerts, config.IDPCertPEMOld)
    }

    // Merge: union of current and new certs (multi-key support)
    allCerts := mergeCertLists(currentCerts, certs)

    // Update in database
    if err := cm.repo.UpdateIDPCerts(ctx, tenantID, allCerts); err != nil {
        return fmt.Errorf("update idp certs: %w", err)
    }

    // Schedule removal of old certs not in latest metadata
    staleCerts := findStaleCerts(currentCerts, certs)
    if len(staleCerts) > 0 {
        // Schedule removal after grace period (7 days)
        cm.scheduleStaleCertRemoval(tenantID, staleCerts, 7*24*time.Hour)
    }

    return nil
}

// SPKeyPair holds a generated key pair.
type SPKeyPair struct {
    PrivateKeyPEM     []byte // encrypted at rest
    CertificatePEM    string
    CertificateExpiry time.Time
}
```

### 8.5 Go Implementation: AuthnRequest Generation

```go
package service

import (
    "bytes"
    "compress/flate"
    "crypto/rsa"
    "crypto/sha256"
    "encoding/base64"
    "encoding/xml"
    "fmt"
    "net/url"
    "time"

    "github.com/google/uuid"
)

// AuthnRequest represents the SAML AuthnRequest XML structure.
type samlpAuthnRequest struct {
    XMLName                  xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol AuthnRequest"`
    ID                       string   `xml:"ID,attr"`
    Version                  string   `xml:"Version,attr"`
    IssueInstant             string   `xml:"IssueInstant,attr"`
    Destination              string   `xml:"Destination,attr"`
    ProtocolBinding          string   `xml:"ProtocolBinding,attr"`
    AssertionConsumerServiceURL string `xml:"AssertionConsumerServiceURL,attr"`
    Issuer                   samlIssuer `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
    NameIDPolicy             samlNameIDPolicy `xml:"NameIDPolicy"`
}

type samlIssuer struct {
    Value string `xml:",chardata"`
}

type samlNameIDPolicy struct {
    AllowCreate bool   `xml:"AllowCreate,attr"`
    Format      string `xml:"Format,attr"`
}

// buildAuthnRequestXML constructs the AuthnRequest XML string.
func buildAuthnRequestXML(
    requestID string,
    idpSSOURL string,
    acsURL string,
    spEntityID string,
    nameIDFormat string,
    issueInstant time.Time,
) []byte {
    req := samlpAuthnRequest{
        ID:                       requestID,
        Version:                  "2.0",
        IssueInstant:             issueInstant.Format(time.RFC3339),
        Destination:              idpSSOURL,
        ProtocolBinding:          "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
        AssertionConsumerServiceURL: acsURL,
        Issuer: samlIssuer{Value: spEntityID},
        NameIDPolicy: samlNameIDPolicy{
            AllowCreate: true,
            Format:      nameIDFormat,
        },
    }

    xmlBytes, _ := xml.MarshalIndent(req, "", "  ")
    return xmlBytes
}

// generateRequestID creates a unique SAML request identifier.
func generateRequestID() string {
    return "_" + uuid.New().String()
}

// buildRedirectURL creates the HTTP-Redirect binding URL.
//
// Steps:
// 1. DEFLATE the XML (raw, no header)
// 2. Base64 encode
// 3. URL-encode
// 4. Append as ?SAMLRequest=<encoded>
// 5. If signed: append &SigAlg=<algorithm>&Signature=<signature>
func buildRedirectURL(
    idpSSOURL string,
    authnRequestXML []byte,
    signature string,
    signed bool,
) string {
    // Step 1: DEFLATE
    var deflated bytes.Buffer
    writer, _ := flate.NewWriter(&deflated, flate.DefaultCompression)
    writer.Write(authnRequestXML)
    writer.Close()

    // Step 2: Base64 encode
    encoded := base64.StdEncoding.EncodeToString(deflated.Bytes())

    // Step 3: URL-encode
    params := url.Values{}
    params.Set("SAMLRequest", encoded)

    if signed {
        params.Set("SigAlg", "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256")
        params.Set("Signature", signature)
    }

    // Step 4: Build URL
    separator := "&"
    if !contains(idpSSOURL, "?") {
        separator = "?"
    }

    return idpSSOURL + separator + params.Encode()
}

// signXML creates an XML Digital Signature using RSA-SHA256.
func signXML(xmlBytes []byte, key *rsa.PrivateKey) string {
    // In production, use a proper XML-Signature library:
    //   github.com/russellhaering/go-saml or github.com/crewjam/saml
    //
    // This simplified version signs the raw bytes for illustration.
    h := sha256.Sum256(xmlBytes)
    signature, _ := rsa.SignPKCS1v15(nil, key, 0, h[:])
    return base64.StdEncoding.EncodeToString(signature)
}
```

### 8.6 Go Implementation: Assertion Validation

GGID's existing `pkg/saml/assertion.go` provides the foundation. The multi-tenant
service extends it:

```go
package service

import (
    "crypto/x509"
    "encoding/pem"
    "fmt"
    "strings"
    "time"

    "github.com/ggid/ggid/pkg/saml"
)

// ValidateAssertion performs comprehensive assertion validation.
// This is the multi-tenant-aware wrapper around pkg/saml primitives.
func ValidateAssertion(
    assertion *saml.SAMLAssertion,
    config *SAMLConfig,
    idpCerts []*x509.Certificate,
    clockSkew time.Duration,
) error {
    // 1. Validate issuer
    if assertion.Issuer != config.IDPEntityID {
        return fmt.Errorf("issuer mismatch: expected %q, got %q",
            config.IDPEntityID, assertion.Issuer)
    }

    // 2. Validate conditions with clock skew
    if err := validateConditionsWithSkew(assertion, clockSkew); err != nil {
        return fmt.Errorf("conditions: %w", err)
    }

    // 3. Validate audience
    if err := validateAudience(assertion, config.SPEntityID); err != nil {
        return fmt.Errorf("audience: %w", err)
    }

    // 4. Validate recipient
    if err := validateRecipient(assertion, config.SPACSURL); err != nil {
        return fmt.Errorf("recipient: %w", err)
    }

    // 5. Verify signature (try each trusted certificate)
    if config.WantAssertionsSigned {
        if err := verifyAssertionWithCerts(assertion, idpCerts); err != nil {
            return fmt.Errorf("signature: %w", err)
        }
    }

    return nil
}

// validateConditionsWithSkew checks NotBefore/NotOnOrAfter with tolerance.
func validateConditionsWithSkew(assertion *saml.SAMLAssertion, skew time.Duration) error {
    now := time.Now().UTC()

    if assertion.Conditions.NotBefore != "" {
        notBefore, err := time.Parse(time.RFC3339, assertion.Conditions.NotBefore)
        if err == nil && now.Before(notBefore.Add(-skew)) {
            return fmt.Errorf("assertion not yet valid (NotBefore=%s, skew=%v)",
                assertion.Conditions.NotBefore, skew)
        }
    }

    if assertion.Conditions.NotOnOrAfter != "" {
        notOnOrAfter, err := time.Parse(time.RFC3339, assertion.Conditions.NotOnOrAfter)
        if err == nil && !now.Before(notOnOrAfter.Add(skew)) {
            return fmt.Errorf("assertion expired (NotOnOrAfter=%s, skew=%v)",
                assertion.Conditions.NotOnOrAfter, skew)
        }
    }

    return nil
}

// validateAudience checks that the assertion's audience matches the SP entity ID.
// Note: The current pkg/saml assertion struct doesn't parse AudienceRestriction.
// This needs to be added. For now, we check the raw XML.
func validateAudience(assertion *saml.SAMLAssertion, expectedAudience string) error {
    // Check raw XML for Audience element
    rawStr := string(assertion.RawXML)
    if !strings.Contains(rawStr, "<Audience>"+expectedAudience+"</Audience>") {
        return fmt.Errorf("audience %q not found in assertion", expectedAudience)
    }
    return nil
}

// validateRecipient checks the SubjectConfirmation Recipient attribute.
func validateRecipient(assertion *saml.SAMLAssertion, expectedRecipient string) error {
    rawStr := string(assertion.RawXML)
    if !strings.Contains(rawStr, "Recipient=\""+expectedRecipient+"\"") {
        return fmt.Errorf("recipient %q not found in assertion", expectedRecipient)
    }
    return nil
}

// verifyAssertionWithCerts tries each certificate until one verifies.
func verifyAssertionWithCerts(
    assertion *saml.SAMLAssertion,
    certs []*x509.Certificate,
) error {
    if len(certs) == 0 {
        return fmt.Errorf("no trusted certificates provided")
    }

    var lastErr error
    for _, cert := range certs {
        if err := saml.ValidateSignature(assertion, cert); err == nil {
            return nil // Successfully verified
        } else {
            lastErr = err
        }
    }

    return fmt.Errorf("assertion did not verify against any trusted certificate: %w", lastErr)
}

// parsePEMCertificates parses one or more PEM-encoded certificates.
func parsePEMCertificates(pemStrs ...string) ([]*x509.Certificate, error) {
    var certs []*x509.Certificate

    for _, pemStr := range pemStrs {
        if pemStr == "" {
            continue
        }
        block, _ := pem.Decode([]byte(pemStr))
        if block == nil {
            continue
        }
        cert, err := x509.ParseCertificate(block.Bytes)
        if err != nil {
            continue
        }
        certs = append(certs, cert)
    }

    if len(certs) == 0 {
        return nil, fmt.Errorf("no valid certificates parsed")
    }

    return certs, nil
}
```

### 8.7 Go Implementation: JIT User Creation

```go
package service

import (
    "context"
    "fmt"
    "strings"

    "github.com/ggid/ggid/pkg/saml"
    "github.com/ggid/ggid/pkg/tenant"
    "github.com/google/uuid"
)

// findOrCreateUser handles user lookup and JIT provisioning.
func (s *MultiTenantSAMLService) findOrCreateUser(
    ctx context.Context,
    config *SAMLConfig,
    assertion *saml.SAMLAssertion,
    attrs map[string]string,
) (*User, error) {
    tc, err := tenant.FromContext(ctx)
    if err != nil {
        return nil, fmt.Errorf("tenant context required: %w", err)
    }

    nameID := assertion.Subject.NameID
    email := attrs["email"]
    if email == "" {
        email = nameID // Fallback: use NameID as email
    }

    // Step 1: Check for linked external identity
    link, err := s.identity.FindExternalIdentity(ctx, tc.TenantID, "saml", nameID)
    if err == nil && link != nil {
        // User exists — return it
        user, err := s.identity.GetUser(ctx, tc.TenantID, link.UserID)
        if err != nil {
            return nil, fmt.Errorf("get linked user: %w", err)
        }
        return user, nil
    }

    // Step 2: Check by email (in case identity was created differently)
    if email != "" {
        existingUser, err := s.identity.GetUserByEmail(ctx, tc.TenantID, email)
        if err == nil && existingUser != nil {
            // Link the SAML identity to existing user
            metadata := map[string]any{
                "provider":    "saml",
                "name_id":     nameID,
                "idp_entity":  config.IDPEntityID,
                "attributes":  attrs,
            }
            if err := s.identity.LinkExternalIdentity(
                ctx, tc.TenantID, existingUser.ID, "saml", nameID, metadata,
            ); err != nil {
                // Non-fatal: log but continue
            }
            return existingUser, nil
        }
    }

    // Step 3: JIT provisioning
    if !config.JITEnabled {
        return nil, fmt.Errorf("account not found for %s and JIT provisioning is disabled", email)
    }

    // Step 4: Validate domain
    if !isDomainAllowed(email, config.JITAllowedDomains) {
        return nil, fmt.Errorf("email domain not allowed for JIT: %s", extractDomain(email))
    }

    // Step 5: Create user
    username := deriveUsername(nameID, email)
    displayName := attrs["display_name"]
    if displayName == "" {
        displayName = attrs["first_name"] + " " + attrs["last_name"]
    }

    metadata := map[string]any{
        "provider":      "saml",
        "name_id":       nameID,
        "idp_entity":    config.IDPEntityID,
        "jit_created":   true,
        "attributes":    attrs,
    }

    newUser, err := s.identity.CreateUserFromSocial(
        ctx, tc.TenantID, username, email, displayName, "saml", nameID, metadata,
    )
    if err != nil {
        return nil, fmt.Errorf("jit create user: %w", err)
    }

    // Step 6: Assign default role
    if err := s.identity.AssignRole(ctx, tc.TenantID, newUser.ID, config.JITDefaultRole); err != nil {
        // Non-fatal — user created but no role assigned
    }

    return newUser, nil
}

// isDomainAllowed checks if the email's domain is in the allowed list.
func isDomainAllowed(email string, allowedDomains []string) bool {
    if len(allowedDomains) == 0 {
        return true // No restriction configured
    }

    domain := extractDomain(email)
    for _, allowed := range allowedDomains {
        if strings.EqualFold(domain, allowed) {
            return true
        }
    }
    return false
}

// extractDomain gets the domain part of an email.
func extractDomain(email string) string {
    parts := strings.SplitN(email, "@", 2)
    if len(parts) != 2 {
        return ""
    }
    return strings.ToLower(parts[1])
}

// deriveUsername creates a username from NameID or email.
func deriveUsername(nameID, email string) string {
    if email != "" {
        // Use email prefix
        parts := strings.SplitN(email, "@", 2)
        if len(parts) == 2 {
            return truncate(parts[0], 60)
        }
    }
    // Fallback: use NameID (truncated)
    return truncate(nameID, 60)
}

func truncate(s string, max int) string {
    if len(s) <= max {
        return s
    }
    return s[:max]
}
```

### 8.8 Replay Protection Implementation

GGID's existing test code (`sp_flow_test.go`) already has a working assertion ID
cache. The production version uses Redis for distributed replay protection:

```go
package service

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
)

// RedisAssertionLogStore implements AssertionLogStore using Redis.
// Keys expire automatically based on the assertion's NotOnOrAfter + grace period.
type RedisAssertionLogStore struct {
    rdb *redis.Client
}

// NewRedisAssertionLogStore creates a new Redis-backed assertion log.
func NewRedisAssertionLogStore(rdb *redis.Client) *RedisAssertionLogStore {
    return &RedisAssertionLogStore{rdb: rdb}
}

// IsConsumed checks if an assertion ID has already been used.
func (s *RedisAssertionLogStore) IsConsumed(
    ctx context.Context,
    tenantID uuid.UUID,
    assertionID string,
) (bool, error) {
    key := fmt.Sprintf("saml:assertion:%s:%s", tenantID, assertionID)
    n, err := s.rdb.Exists(ctx, key).Result()
    if err != nil {
        return false, err
    }
    return n > 0, nil
}

// MarkConsumed records an assertion ID as consumed with a TTL.
func (s *RedisAssertionLogStore) MarkConsumed(
    ctx context.Context,
    tenantID uuid.UUID,
    assertionID string,
    expiresAt time.Time,
) error {
    key := fmt.Sprintf("saml:assertion:%s:%s", tenantID, assertionID)

    // TTL = time until expiry + 5 minute grace
    ttl := time.Until(expiresAt) + 5*time.Minute
    if ttl < 1*time.Minute {
        ttl = 1*time.Minute // Minimum TTL
    }

    // Use SET NX to ensure atomicity (only one writer wins)
    ok, err := s.rdb.SetNX(ctx, key, "consumed", ttl).Result()
    if err != nil {
        return err
    }
    if !ok {
        return fmt.Errorf("assertion already consumed (race condition): %s", assertionID)
    }
    return nil
}

// StoreRequestID stores an AuthnRequest ID for InResponseTo matching.
func (s *RedisAssertionLogStore) StoreRequestID(
    ctx context.Context,
    tenantID uuid.UUID,
    requestID string,
    expiresAt time.Time,
) error {
    key := fmt.Sprintf("saml:request:%s:%s", tenantID, requestID)
    ttl := time.Until(expiresAt)
    if ttl < 1*time.Minute {
        ttl = 5 * time.Minute
    }
    return s.rdb.Set(ctx, key, "pending", ttl).Err()
}

// ValidateRequestID checks if a stored request ID exists (for InResponseTo).
func (s *RedisAssertionLogStore) ValidateRequestID(
    ctx context.Context,
    tenantID uuid.UUID,
    requestID string,
) (bool, error) {
    key := fmt.Sprintf("saml:request:%s:%s", tenantID, requestID)
    n, err := s.rdb.Exists(ctx, key).Result()
    if err != nil {
        return false, err
    }
    if n > 0 {
        // Delete after use (one-time)
        s.rdb.Del(ctx, key)
    }
    return n > 0, nil
}
```

---

## 9. Security Considerations

### 9.1 XML Signature Wrapping (XSW) Attacks

**What it is:** An attacker takes a signed SAML assertion, wraps it inside
additional XML elements, and injects a different (unsigned) assertion that the
parser reads instead. The signature is still valid (it covers the original
assertion), but the application uses the attacker-controlled assertion.

**Example attack:**
```xml
<!-- Attacker constructs this -->
<Response>
  <!-- Original signed assertion (moved here) -->
  <Assertion ID="_original">
    <ds:Signature>...valid signature...</ds:Signature>
    <Subject><NameID>legitimate@corp.com</NameID></Subject>
  </Assertion>
  <!-- Attacker-injected unsigned assertion (parsed by naive parser) -->
  <Assertion ID="_injected">
    <Subject><NameID>admin@corp.com</NameID></Subject>
    <AttributeStatement>
      <Attribute Name="groups"><AttributeValue>superadmin</AttributeValue></Attribute>
    </AttributeStatement>
  </Assertion>
</Response>
```

A naive XML parser that processes the first `<Assertion>` element it encounters
will use `_injected` (the attacker's), while signature verification passes on
`_original` (the legitimate one).

**Prevention:**

1. **Verify that the signed element is the same element you extract data from.**
   After signature verification, extract user data from the **signed** assertion,
   not from a separate parse pass.

2. **Use a battle-tested SAML library** that handles XSW prevention:
   - `github.com/crewjam/saml` — includes XSW detection
   - `github.com/russellhaering/go-saml` — canonicalization-aware verification

3. **Validate the assertion is a direct child of the Response** (not nested in
   an Object element or other wrapper).

4. **Verify the signature reference URI** matches the assertion ID. The
   `<ds:Reference URI="#_original">` must point to the assertion you're using.

GGID's current `pkg/saml/assertion.go` notes this limitation:
```go
// In production, this would use github.com/russellhaering/go-saml
// or github.com/crewjam/saml to verify the XML-Signature.
// For now, we verify the assertion contains a Signature element.
```

**Action item:** Replace the simplified signature check with `crewjam/saml` or
`russellhaering/go-saml` for production use.

### 9.2 Replay Attacks

**What it is:** An attacker captures a valid SAML assertion (e.g., via network
sniffing, proxy logging, or browser cache) and resubmits it to gain access.

**Prevention:**

1. **Track consumed assertion IDs.** Store each assertion ID in Redis (TTL =
   `NotOnOrAfter` + grace period). Reject any assertion whose ID has been seen.

2. **Validate `InResponseTo`** for SP-initiated flows. The response must reference
   a request GGID actually sent.

3. **Use short assertion validity windows.** Work with tenants to configure short
   `NotBefore`/`NotOnOrAfter` windows (typically 5 minutes).

4. **Require TLS everywhere.** Assertions should never traverse plaintext HTTP.

GGID's existing test code (`sp_flow_test.go`) demonstrates the assertion ID
cache pattern. The production implementation uses Redis for distributed,
TTL-based replay protection (see §8.8).

### 9.3 Certificate Spoofing

**What it is:** An attacker presents a forged certificate to make GGID accept
assertions from an illegitimate IdP.

**Prevention:**

1. **Strict per-tenant trust store.** Each tenant has exactly the certificates
   GGID has configured for that tenant. No global trust store.

2. **Never auto-trust metadata certificates.** When refreshing IdP metadata,
   do NOT automatically replace the stored certificate. Instead, add new
   certificates alongside existing ones (multi-key) and require admin approval
   for cert changes outside of metadata refresh.

3. **Verify metadata source.** Only fetch IdP metadata over HTTPS with certificate
   pinning or known-good CA verification. Never fetch over HTTP.

4. **Log all certificate changes.** Every time a stored IdP certificate changes,
   log the event with the old and new certificate fingerprints.

### 9.4 Tenant Isolation

**Critical invariant:** A user authenticating via Tenant A's SAML config must
never access Tenant B's data.

**Prevention:**

1. **URL-path enforcement.** The ACS URL includes `{tenant_id}`. Assertions
   posted to `/saml/tenant-a/acs` are always processed in Tenant A's context.

2. **Audience validation.** The assertion's `Audience` must match the tenant's
   SP entity ID. An assertion issued for Tenant A's SP entity ID will fail
   Tenant B's audience check.

3. **Issuer validation.** The assertion's `Issuer` must match the tenant's
   configured IdP entity ID.

4. **Database Row-Level Security (RLS).** GGID uses PostgreSQL RLS to enforce
   tenant isolation at the database level. Even if an application bug allows
   cross-tenant context, RLS prevents data leakage.

5. **Session scoping.** Sessions created from SAML login are scoped to the
   tenant. The JWT includes the `tenant_id` claim, validated on every request.

### 9.5 Metadata Poisoning

**What it is:** An attacker tricks GGID into fetching malicious IdP metadata
(e.g., via DNS spoofing, BGP hijacking, or compromising the IdP's metadata
endpoint), which could redirect authentication or inject attacker-controlled
certificates.

**Prevention:**

1. **HTTPS-only metadata fetching.** Reject HTTP metadata URLs.

2. **Signed metadata.** If the IdP signs its metadata, verify the signature
   against the previously-known signing key before trusting the new metadata.

3. **Certificate fingerprint pinning.** Store the expected certificate
   fingerprint. Alert (don't auto-update) if the fingerprint changes unexpectedly.

4. **Metadata change alerts.** Notify the tenant admin when metadata changes
   (new certificate, new SSO URL, etc.) so they can verify the change is
   legitimate.

5. **Rate-limit metadata refresh.** Don't allow metadata to change more
   frequently than expected (e.g., not more than once per hour).

### 9.6 Clock Skew

SAML assertions have strict time windows (`NotBefore`, `NotOnOrAfter`). If the
GGID server's clock differs from the IdP's clock, valid assertions may be
rejected.

**Prevention:**

1. **Configurable clock skew tolerance.** Each tenant can configure
   `clock_skew_seconds` (default: 60 seconds). The validation accepts assertions
   within `NotBefore - skew` to `NotOnOrAfter + skew`.

2. **NTP synchronization.** All GGID servers must use NTP to keep clocks
   synchronized.

3. **Monitor for skew issues.** Log assertion validation failures due to time
   windows. If many failures correlate with time issues, investigate clock
   sync.

GGID's `pkg/saml/assertion.go` currently has a hardcoded 60-second tolerance.
The multi-tenant service makes this configurable per-tenant.

### 9.7 RelayState Security

`RelayState` is an opaque parameter passed through the SAML flow. GGID uses it
to carry the return URL after SSO.

**Prevention:**

1. **Only use RelayState for internal state.** Never trust RelayState as a
   security-sensitive parameter. It's user-controllable.

2. **Validate redirect targets.** When redirecting based on RelayState, ensure
   the URL is a relative path or an allowed domain. Prevent open redirect
   vulnerabilities:
   ```go
   func safeRedirect(relayState string) string {
       if relayState == "" || !strings.HasPrefix(relayState, "/") {
           return "/dashboard"
       }
       return relayState
   }
   ```

3. **Sign or encrypt RelayState** (optional, for additional security). GGID can
   HMAC-sign the RelayState value and verify it on return.

### 9.8 Security Checklist

| # | Check | Implemented |
|---|-------|-------------|
| 1 | Assertions are cryptographically signed by IdP | Required |
| 2 | Signature verified against correct tenant's IdP cert | Design |
| 3 | XSW attack prevention (same-element extraction) | TODO |
| 4 | Assertion ID replay tracking (Redis TTL) | Design |
| 5 | InResponseTo validation (SP-initiated) | Design |
| 6 | Audience restriction validated | Design |
| 7 | Recipient validated | Design |
| 8 | Conditions time window checked | Partial (pkg/saml) |
| 9 | Configurable clock skew | Design |
| 10 | Per-tenant strict trust store | Design |
| 11 | HTTPS-only metadata fetching | Design |
| 12 | Tenant isolation (URL + audience + RLS) | Design |
| 13 | RelayState open-redirect prevention | Design |
| 14 | RelayState HMAC-signed | Future |
| 15 | Multi-key certificate support | Design |
| 16 | Certificate change alerting | Future |
| 17 | Assertion consumed within TTL window | Design |
| 18 | TLS for all SAML endpoints | Required |
| 19 | Rate limiting on ACS endpoint | Design |
| 20 | Audit logging of all SAML events | Design |

---

## 10. Comparison with Other Multi-Tenant SAML Implementations

### 10.1 Auth0

Auth0 provides SAML federation through "Enterprise Connections." Each connection
represents a link to one IdP.

| Feature | Auth0 | GGID (planned) |
|---------|-------|----------------|
| Per-tenant IdP config | Yes, via Connections API | Yes, via saml_configs |
| Metadata exchange | Auto-fetch IdP metadata | Yes |
| SP-initiated SSO | Yes | Yes |
| IdP-initiated SSO | Yes (toggleable) | Yes (toggleable) |
| JIT provisioning | Yes, with rules | Yes, with domain validation |
| Attribute mapping | Via Rules pipeline | Via attr_map JSON |
| Multi-key cert support | Yes | Yes |
| Certificate rotation | Automated + alerts | Scheduled + multi-key |
| SAML logout (SLO) | Yes | Phase 4 |

**Key difference:** Auth0 uses a Rules pipeline (JavaScript) for attribute
mapping and transformations. GGID uses a simpler declarative JSON map, which
is more predictable and easier to audit.

### 10.2 Keycloak

Keycloak organizes tenants as "Realms." Each realm is fully isolated with its
own SAML identity providers.

| Feature | Keycloak | GGID (planned) |
|---------|----------|----------------|
| Tenant model | Realms (one per tenant) | Shared DB with RLS |
| Per-realm SAML IdPs | Yes | Yes |
| Metadata exchange | Yes, import/export | Yes |
| SP-initiated SSO | Yes | Yes |
| IdP-initiated SSO | Yes | Yes |
| JIT provisioning | Yes ("first broker login") | Yes |
| Attribute mapping | Per-IdP mappers | Per-tenant attr_map |
| Multi-key cert support | Yes | Yes |
| Certificate rotation | Manual UI | Scheduled + API |
| SAML logout (SLO) | Yes | Phase 4 |
| Admin UI | Full admin console | Console (planned) |

**Key difference:** Keycloak's realm model gives each tenant a separate
configuration namespace, which provides strong isolation but complicates
cross-tenant features. GGID's shared-database-with-RLS model allows
cross-tenant queries when needed while maintaining data isolation.

### 10.3 WorkOS

WorkOS is a managed SaaS API for enterprise SSO, including SAML.

| Feature | WorkOS | GGID (planned) |
|---------|--------|----------------|
| Model | Managed service (API) | Self-hosted |
| Per-tenant connections | Yes, via Connection API | Yes |
| Metadata exchange | Auto-fetch | Yes |
| SP-initiated SSO | Yes | Yes |
| IdP-initiated SSO | Yes | Yes |
| JIT provisioning | Yes | Yes |
| Attribute mapping | Via dashboard | Via attr_map JSON |
| Certificate management | Fully managed | Semi-automated |
| SAML logout (SLO) | Yes | Phase 4 |
| Pricing | Per-user / per-connection | Open source (free) |

**Key difference:** WorkOS is a fully managed, hosted service. GGID is
self-hosted, giving full control over data residency and compliance. WorkOS
abstracts away certificate management entirely; GGID requires some operational
involvement for cert rotation.

### 10.4 Ory (Kratos/Hydra)

Ory does not have native SAML support.

| Feature | Ory | GGID (planned) |
|---------|------|----------------|
| SAML support | None (requires custom) | Full |
| OIDC federation | Yes (via Hydra) | Yes |
| Self-hosted | Yes | Yes |
| Workaround for SAML | Use a SAML-to-OIDC bridge | N/A |

**Key difference:** Ory users who need SAML must build a custom bridge (typically
using a reverse proxy that converts SAML to OIDC). GGID provides native SAML,
eliminating this complexity.

### 10.5 Feature Comparison Matrix

| Feature | Auth0 | Keycloak | WorkOS | Ory | **GGID** |
|---------|-------|----------|--------|-----|----------|
| Open source | No | Yes | No | Yes | **Yes** |
| Self-hosted | No | Yes | No | Yes | **Yes** |
| Native SAML | Yes | Yes | Yes | No | **Yes** |
| Multi-tenant | Yes | Realms | Yes | Manual | **Yes (RLS)** |
| SP-init SSO | Yes | Yes | Yes | No | **Yes** |
| IdP-init SSO | Yes | Yes | Yes | No | **Yes** |
| JIT provisioning | Yes | Yes | Yes | No | **Yes** |
| Dynamic metadata | Yes | Yes | Yes | No | **Yes** |
| Cert auto-rotation | Yes | Manual | Yes | No | **Semi-auto** |
| Attribute mapping | Rules (JS) | Mappers | Dashboard | N/A | **JSON map** |
| SLO | Yes | Yes | Yes | No | **Phase 4** |
| SCIM | Yes | Yes | Yes | No | **Skeleton** |

---

## 11. GGID Roadmap

### Phase 1: SP-Initiated SSO MVP (Weeks 1-3)

**Goal:** A single tenant can configure SAML SSO and users can authenticate via
their IdP.

| Task | Effort | Description |
|------|--------|-------------|
| Database migrations | 2 days | `saml_configs`, `saml_assertion_log` tables |
| SAML service core | 3 days | `MultiTenantSAMLService` with `GenerateAuthnRequest`, `ProcessAssertion` |
| HTTP routes | 1 day | `/saml/{tenant_id}/login`, `/acs`, `/metadata` |
| AuthnRequest generation | 2 days | XML construction, HTTP-Redirect binding, signing |
| Assertion validation | 3 days | Signature, conditions, audience, recipient, InResponseTo |
| Replay protection (Redis) | 1 day | Assertion ID tracking with TTL |
| Integration with pkg/saml | 1 day | Wire up existing assertion parsing |
| Integration tests | 2 days | Mock IdP, end-to-end SP-initiated flow |
| **Total** | **~15 days** | |

**Deliverable:** Working SP-initiated SSO with per-tenant config.

### Phase 2: IdP-Initiated SSO + JIT Provisioning (Weeks 4-5)

| Task | Effort | Description |
|------|--------|-------------|
| IdP-initiated support | 2 days | Unsolicited assertion handling, config toggle |
| JIT provisioning | 3 days | User creation, domain validation, role assignment |
| Attribute mapping | 2 days | Per-tenant attr_map, default maps per IdP |
| External identity linking | 1 day | Reuse `FindExternalIdentity`/`LinkExternalIdentity` |
| Admin API | 2 days | CRUD for SAML configs |
| Integration tests | 2 days | IdP-initiated flow, JIT scenarios |
| **Total** | **~12 days** | |

**Deliverable:** Full SSO with automatic user provisioning.

### Phase 3: Dynamic Metadata Exchange + Certificate Management (Weeks 6-7)

| Task | Effort | Description |
|------|--------|-------------|
| SP metadata endpoint | 1 day | Generate signed SP metadata XML |
| IdP metadata fetcher | 2 days | HTTPS fetch, XML parse, cert extraction |
| Metadata cache table | 1 day | `saml_idp_metadata_cache` table |
| Multi-key cert support | 2 days | Accept multiple IdP certs simultaneously |
| Cert rotation scheduler | 2 days | Cron job for metadata refresh, stale cert cleanup |
| Certificate manager | 2 days | SP key generation, rotation workflow |
| Alerting | 1 day | Notify on cert changes, expiring certs |
| **Total** | **~11 days** | |

**Deliverable:** Zero-downtime certificate rotation and automatic metadata refresh.

### Phase 4: SAML Logout (SLO) + Admin UI (Weeks 8-10)

| Task | Effort | Description |
|------|--------|-------------|
| SLO (SP-initiated) | 3 days | Generate LogoutRequest, process LogoutResponse |
| SLO (IdP-initiated) | 2 days | Process unsolicited LogoutRequest from IdP |
| Session invalidation | 1 day | Revoke sessions for logged-out users |
| Console: SAML config page | 3 days | Admin UI for configuring SAML per tenant |
| Console: Attribute mapping UI | 2 days | Visual attribute mapping editor |
| Console: Metadata viewer | 1 day | View SP/IdP metadata XML |
| Console: Cert management | 2 days | Upload, rotate, view certificates |
| Console: SAML test tool | 2 days | "Test SAML connection" button |
| E2E tests | 2 days | Full SLO flow, admin UI tests |
| **Total** | **~18 days** | |

**Deliverable:** Complete SAML federation with admin console.

### Total Effort

| Phase | Duration | Cumulative |
|-------|----------|------------|
| Phase 1: SP-initiated MVP | ~3 weeks | 3 weeks |
| Phase 2: IdP-initiated + JIT | ~2 weeks | 5 weeks |
| Phase 3: Metadata + Cert mgmt | ~2 weeks | 7 weeks |
| Phase 4: SLO + Admin UI | ~3 weeks | 10 weeks |
| **Total** | **~10 weeks** | |

### Future Enhancements (Post-Phase 4)

- **Encrypted assertions** — support `<EncryptedAssertion>` with SP decryption key
- **SAML artifact binding** — back-channel artifact resolution
- **SAML attribute query** — SP requests attributes directly from IdP
- **Multi-IdP per tenant** — a tenant can federate with multiple IdPs
- **IdP discovery service** — user selects their IdP from a list
- **SCIM deprovisioning** — IdP pushes user deactivation via SCIM 2.0 API
- **SAML metadata aggregation** — support SAML metadata aggregate files (InCommon)
- **CAEP/RISC integration** — receive continuous access evaluation protocol events

---

## Appendix: SAML XML Examples

### A. Complete AuthnRequest (HTTP-Redirect encoded)

**Decoded XML:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<samlp:AuthnRequest
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="_a1b2c3d4-e5f6-7890-abcd-ef1234567890"
    Version="2.0"
    IssueInstant="2024-01-15T10:30:00.000Z"
    Destination="https://acme.okta.com/app/ggid/exk1abc/sso/saml"
    ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
    AssertionConsumerServiceURL="https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001/acs">
  <saml:Issuer
      xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">
      https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001
  </saml:Issuer>
  <samlp:NameIDPolicy
      AllowCreate="true"
      Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"/>
  <samlp:RequestedAuthnContext Comparison="minimum">
    <saml:AuthnContextClassRef>
      urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
    </saml:AuthnContextClassRef>
  </samlp:RequestedAuthnContext>
</samlp:AuthnRequest>
```

**Encoded for HTTP-Redirect:**
```
https://acme.okta.com/app/ggid/exk1abc/sso/saml
  ?SAMLRequest=eJzFkN1qhDAQhF8l%2B...  (base64-deflate-encoded)
  &RelayState=https%3A%2F%2Facme.ggid.com%2Fdashboard
  &SigAlg=http%3A%2F%2Fwww.w3.org%2F2001%2F04%2Fxmldsig-more%23rsa-sha256
  &Signature=signature_base64...
```

### B. Complete SAML Response with Signed Assertion

```xml
<?xml version="1.0" encoding="UTF-8"?>
<samlp:Response
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="_response-xyz789"
    InResponseTo="_a1b2c3d4-e5f6-7890-abcd-ef1234567890"
    Version="2.0"
    IssueInstant="2024-01-15T10:30:05.123Z"
    Destination="https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001/acs">
  <saml:Issuer
      xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">
      http://www.okta.com/exk1abc
  </saml:Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo>
      <ds:CanonicalizationMethod
          Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
      <ds:SignatureMethod
          Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
      <ds:Reference URI="#_assertion-001">
        <ds:Transforms>
          <ds:Transform
              Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
          <ds:Transform
              Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
        </ds:Transforms>
        <ds:DigestMethod
            Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
        <ds:DigestValue>abc123digestValue...==</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>
      xyz789signatureValue...base64...
    </ds:SignatureValue>
    <ds:KeyInfo>
      <ds:X509Data>
        <ds:X509Certificate>
          MIIDXTCCAkWgAwIBAgIJALmVVuDWu4NYMA0GCSqGSIb3DQEBCwUAMIGZMQswCQYD
          ...certificate bytes...
        </ds:X509Certificate>
      </ds:X509Data>
    </ds:KeyInfo>
  </ds:Signature>
  <samlp:Status>
    <samlp:StatusCode
        Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </samlp:Status>
  <saml:Assertion
      xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
      ID="_assertion-001"
      Version="2.0"
      IssueInstant="2024-01-15T10:30:05.000Z">
    <saml:Issuer>http://www.okta.com/exk1abc</saml:Issuer>
    <saml:Subject>
      <saml:NameID
          Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">
        alice.anderson@acme.com
      </saml:NameID>
      <saml:SubjectConfirmation
          Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
        <saml:SubjectConfirmationData
            InResponseTo="_a1b2c3d4-e5f6-7890-abcd-ef1234567890"
            NotOnOrAfter="2024-01-15T10:35:05.000Z"
            Recipient="https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001/acs"/>
      </saml:SubjectConfirmation>
    </saml:Subject>
    <saml:Conditions
        NotBefore="2024-01-15T10:30:05.000Z"
        NotOnOrAfter="2024-01-15T10:35:05.000Z">
      <saml:AudienceRestriction>
        <saml:Audience>
          https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001
        </saml:Audience>
      </saml:AudienceRestriction>
    </saml:Conditions>
    <saml:AuthnStatement
        AuthnInstant="2024-01-15T10:30:04.000Z"
        SessionIndex="_session-abc123"
        SessionNotOnOrAfter="2024-01-15T18:30:05.000Z">
      <saml:AuthnContext>
        <saml:AuthnContextClassRef>
          urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
        </saml:AuthnContextClassRef>
      </saml:AuthnContext>
    </saml:AuthnStatement>
    <saml:AttributeStatement>
      <saml:Attribute Name="email"
          NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
        <saml:AttributeValue>alice.anderson@acme.com</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="firstName"
          NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
        <saml:AttributeValue>Alice</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="lastName"
          NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
        <saml:AttributeValue>Anderson</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="groups"
          NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
        <saml:AttributeValue>engineering</saml:AttributeValue>
        <saml:AttributeValue>platform-admins</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="department"
          NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
        <saml:AttributeValue>Engineering</saml:AttributeValue>
      </saml:Attribute>
    </saml:AttributeStatement>
  </saml:Assertion>
</samlp:Response>
```

### C. SP Metadata XML

```xml
<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor
    xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001">
  <SPSSODescriptor
      protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"
      AuthnRequestsSigned="true"
      WantAssertionsSigned="true">
    <KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>
            MIIDXTCCAkWgAwIBAgIJALmVVuDWu4NYMA0GCSqGSIb3DQEBCwUAMIGZMQswCQYD
            VQQGEwJVUzELMAkGA1UECAwCQ0ExFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28x
            ...
          </ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <SingleLogoutService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001/slo"/>
    <NameIDFormat>
      urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress
    </NameIDFormat>
    <AssertionConsumerService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001/acs"
        index="0"
        isDefault="true"/>
    <AssertionConsumerService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Artifact"
        Location="https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001/acs"
        index="1"/>
  </SPSSODescriptor>
</EntityDescriptor>
```

### D. IdP Metadata XML (Okta example)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor
    xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="http://www.okta.com/exk1abc">
  <IDPSSODescriptor
      protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>
            MIIDpDCCAoygAwIBAgIGAXL...idp-signing-cert...
          </ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <NameIDFormat>
      urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress
    </NameIDFormat>
    <SingleSignOnService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://acme.okta.com/app/ggid/exk1abc/sso/saml"/>
    <SingleSignOnService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://acme.okta.com/app/ggid/exk1abc/sso/saml"/>
    <SingleLogoutService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://acme.okta.com/app/ggid/exk1abc/slo/saml"/>
  </IDPSSODescriptor>
</EntityDescriptor>
```

### E. SAML LogoutRequest (SP-initiated SLO)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<samlp:LogoutRequest
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="_logout-req-001"
    Version="2.0"
    IssueInstant="2024-01-15T12:00:00.000Z"
    Destination="https://acme.okta.com/app/ggid/exk1abc/slo/saml">
  <saml:Issuer
      xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">
      https://ggid.example.com/saml/00000000-0000-0000-0000-000000000001
  </saml:Issuer>
  <saml:NameID
      Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">
      alice.anderson@acme.com
  </saml:NameID>
  <samlp:SessionIndex>_session-abc123</samlp:SessionIndex>
</samlp:LogoutRequest>
```

---

## References

### Standards

- **SAML 2.0 Core**: OASIS, "Assertions and Protocols for the OASIS Security
  Assertion Markup Language (SAML) V2.0", 2005.
  https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf

- **SAML 2.0 Bindings**: OASIS, "Bindings for the OASIS Security Assertion
  Markup Language (SAML) V2.0", 2005.
  https://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf

- **SAML 2.0 Profiles**: OASIS, "Profiles for the OASIS Security Assertion
  Markup Language (SAML) V2.0", 2005.
  https://docs.oasis-open.org/security/saml/v2.0/saml-profiles-2.0-os.pdf

- **SAML 2.0 Metadata**: OASIS, "Metadata for the OASIS Security Assertion
  Markup Language (SAML) V2.0", 2005.
  https://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf

- **XML Signature**: W3C, "XML Signature Syntax and Processing Version 1.1",
  2013. https://www.w3.org/TR/xmldsig-core1/

- **XML Encryption**: W3C, "XML Encryption Syntax and Processing Version 1.1",
  2013. https://www.w3.org/TR/xmlenc-core1/

### Security Research

- **Somorovsky et al.**, "On Breaking SAML: Be Whoever You Want to Be", USENIX
  Security 2012. Describes XML Signature Wrapping attacks against SAML.
  https://arxiv.org/pdf/1401.7483

- **IBM Security Intelligence**, "XML Signature Wrapping" — explanation of XSW
  attack vectors and prevention.
  https://www.ibm.com/think/topics/xml-signature-wrapping

### Multi-Tenant Identity Architecture

- **Microsoft Azure Architecture Center**, "Architectural approaches for
  identity in multitenant solutions" — federation, tenant isolation,
  authentication delegation patterns.
  https://learn.microsoft.com/en-us/azure/architecture/guide/multitenant/approaches/identity

- **Microsoft Azure Architecture Center**, "Architectural considerations for
  identity in a multitenant solution" — SSO, tenant resolution, isolation
  strategies.
  https://learn.microsoft.com/en-us/azure/architecture/guide/multitenant/considerations/identity

### Certificate Management

- **Ping Identity**, "Best Practices: PingFederate SAML Signing Certificates" —
  self-signed vs CA-signed, rotation strategies.
  https://docs.pingidentity.com/solution-guides/best_practice_guides/htg_best_practice_pf_saml_signing_cert.html

- **Citrix Cloud**, "Update the Service Provider SAML Signing Certificate" —
  dual-cert rotation workflow, advertisement phase.
  https://docs.citrix.com/en-us/citrix-cloud/citrix-cloud-management/identity-access-management/saml-service-provider-signing-certificate.html

### Go SAML Libraries

- **crewjam/saml**: https://github.com/crewjam/saml — full SAML implementation
  with XSW prevention
- **russellhaering/go-saml**: https://github.com/russellhaering/go-saml —
  canonicalization-aware XML signature verification
- **mattermost/xml-signer**: https://github.com/mattermost/xml-signer —
  focused XML digital signature library

### GGID Internal References

- `pkg/saml/assertion.go` — assertion parsing, condition validation,
  signature element checking, attribute extraction
- `pkg/saml/sp_flow_test.go` — ACS flow tests, replay protection patterns,
  multi-valued attribute handling
- `pkg/tenant/tenant.go` — multi-tenant context propagation with
  `TenantID`, `IsolationLevel`, `Settings`
- `services/auth/internal/service/idp_federation.go` — `IdPConfig` struct
  modeling SAML/OIDC IdP federation configuration
- `services/auth/internal/service/auth_service.go` — `SocialLogin` pattern
  for external identity linking (reusable for SAML JIT)
- `services/auth/migrations/` — existing migration patterns (credentials,
  sessions, refresh tokens, MFA devices)

---

*Document version: 1.0 — Comprehensive multi-tenant SAML federation design for GGID.*
