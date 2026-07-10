# SAML 2.0 SP-Initiated SSO: Protocol Design

> **Scope:** Protocol-level details of SP-initiated SSO — AuthnRequest construction,
> NameIDPolicy, RelayState, ACS processing, attribute mapping, and SP vs IdP-initiated
> comparison. For multi-tenant certificate management and per-tenant configuration, see
> [`multi-tenant-saml.md`](./multi-tenant-saml.md).

---

## 1. SP-Initiated SSO Overview

In **SP-initiated SSO**, the user accesses a protected application at the Service
Provider (SP) first. The SP generates an `<AuthnRequest>`, redirects the user to the
Identity Provider (IdP), and waits for a signed `<Response>` containing one or more
assertions posted back to its Assertion Consumer Service (ACS) endpoint.

**SP-initiated is the recommended flow** because:

- The IdP receives an explicit request identifying the requesting SP.
- `RelayState` carries SP context (original URL, tenant ID) through the round trip.
- `InResponseTo` binds the response to a known request, mitigating unsolicited-assertion
  attacks.
- The SP controls the timing — it can decide which IdP to redirect to (important for
  multi-tenant deployments with multiple IdPs per tenant).

**IdP-initiated SSO** (unsolicited response): the user starts at the IdP portal, which
posts an assertion to the SP without a prior `AuthnRequest`. This is lower-security
because there is no request binding, no `RelayState`, and no `InResponseTo` to validate.

### ASCII Sequence Diagram

```
  User Agent          Service Provider          Identity Provider
      |                      |                         |
      |  1. GET /app         |                         |
      |--------------------- >|                        |
      |                      |                         |
      |  2. 302 Redirect     |                         |
      |     Location: IdP?SAMLRequest=…&RelayState=…   |
      |<---------------------|                         |
      |                      |                         |
      |  3. GET /sso?SAMLRequest=…&RelayState=…        |
      |------------------------------------------------ >|
      |                      |                         |
      |  4. Login page (credentials / MFA)             |
      |<-------------------------------------------------|
      |  5. POST credentials                           |
      |------------------------------------------------ >|
      |                      |                         |
      |  6. 200 Auto-POST form (SAMLResponse + RelayState)|
      |     action = SP ACS URL                         |
      |<-------------------------------------------------|
      |                      |                         |
      |  7. POST /acs SAMLResponse=… RelayState=…       |
      |--------------------- >|                        |
      |                      |                         |
      |                      | 8. Verify signature,    |
      |                      |    validate conditions, |
      |                      |    create session       |
      |                      |                         |
      |  9. 302 Redirect to RelayState target          |
      |<---------------------|                         |
```

---

## 2. AuthnRequest Construction

### XML Structure

```xml
<samlp:AuthnRequest
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="_a1b2c3d4e5f6"
    Version="2.0"
    IssueInstant="2024-01-15T10:30:00Z"
    Destination="https://idp.example.com/sso"
    ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
    AssertionConsumerServiceURL="https://sp.example.com/acs">
  <saml:Issuer>https://sp.example.com/metadata</saml:Issuer>
  <samlp:NameIDPolicy
      Format="urn:oasis:names:tc:SAML:2.0:nameid-format:transient"
      AllowCreate="true"/>
  <samlp:RequestedAuthnContext Comparison="minimum">
    <saml:AuthnContextClassRef>
      urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
    </saml:AuthnContextClassRef>
  </samlp:RequestedAuthnContext>
</samlp:AuthnRequest>
```

| Attribute / Element          | Required | Description                                     |
|------------------------------|----------|-------------------------------------------------|
| `ID`                         | Yes      | Unique request identifier (used for `InResponseTo`) |
| `Version`                    | Yes      | Always `"2.0"`                                  |
| `IssueInstant`               | Yes      | UTC timestamp in RFC 3339                       |
| `Destination`                | Yes      | IdP SSO service URL                             |
| `AssertionConsumerServiceURL`| Optional | SP endpoint that receives the response          |
| `ProtocolBinding`            | Optional | Binding for response (HTTP-POST is standard)     |
| `saml:Issuer`                | Yes      | SP entity ID                                    |
| `samlp:NameIDPolicy`         | Optional | Format and `AllowCreate` flags                  |
| `samlp:RequestedAuthnContext`| Optional | Minimum auth strength (password, MFA, etc.)     |

### Signing

The AuthnRequest **SHOULD** be signed (SAML Core section 3.4.1.4). The signing method
differs by binding:

- **HTTP-Redirect:** the signature is computed over a deflated query string containing
  `SAMLRequest`, `RelayState`, `SigAlg`, and passed as a `Signature` query parameter.
  The raw XML itself is NOT signed — only the redirect URL is.
- **HTTP-POST:** the `<ds:Signature>` element is embedded directly in the XML before
  base64 encoding and form-POSTing.

### Go Code (GGID Pattern)

GGID's `pkg/saml/sp.go` provides `BuildAuthnRequest` and `EncodeForRedirect`:

```go
sp := &saml.ServiceProvider{
    EntityID:            "https://sp.example.com/metadata",
    ACSURL:              "https://sp.example.com/acs",
    WantAssertionsSigned: true,
}

// Build the AuthnRequest
req := saml.BuildAuthnRequest(sp, "https://idp.example.com/sso")

// Encode for HTTP-Redirect binding (DEFLATE + base64)
encoded, err := req.EncodeForRedirect()
// → "eJzVkd1ygjAUx8/X…"

// Construct redirect URL
redirectURL := fmt.Sprintf("https://idp.example.com/sso?SAMLRequest=%s", url.QueryEscape(encoded))
```

Internally, `EncodeForRedirect` calls `Marshal()` → `deflate()` (raw RFC 1951) →
base64. The `deflate` function delegates to `flateCompress` in `flate_compress.go`.

**Gap:** GGID's `BuildAuthnRequest` does not sign the redirect URL. For signed
HTTP-Redirect, you must compute a separate query-string signature and append
`&SigAlg=…&Signature=…`.

---

## 3. NameIDPolicy

### Format Options

| Format URI                              | Behavior                                     | Use Case                         |
|-----------------------------------------|----------------------------------------------|----------------------------------|
| `…:1.1:nameid-format:unspecified`       | IdP chooses any format                       | No preference                    |
| `…:1.1:nameid-format:emailAddress`      | User's email address as NameID               | Simple federation                |
| `…:2.0:nameid-format:persistent`        | Opaque, stable identifier per SP-user pair   | **Recommended** — repeatable     |
| `…:2.0:nameid-format:transient`         | Random, single-use identifier per session    | Privacy-preserving, per-session  |

### AllowCreate

- **`AllowCreate="true"`** — the IdP may provision a new user at the SP if one doesn't
  exist. This enables **JIT (Just-In-Time) provisioning**: the SP creates or updates a
  local user record from the assertion attributes at login time.
- **`AllowCreate="false"`** — the IdP must only return a NameID for an existing user.
  If no user matches, the IdP returns an error response.

### Security Considerations

- **Persistent** identifiers are opaque and SP-specific, preventing cross-SP correlation.
  The same user at two SPs receives different persistent NameIDs. This is ideal when the
  SP needs to link repeat logins without exposing the user's real identity.
- **Transient** identifiers are regenerated each session, providing maximum privacy. The
  SP must maintain an in-memory session-to-user mapping (since the NameID changes on
  every login). GGID defaults to transient (`NameIDFormatTransient` in `BuildAuthnRequest`).
- **EmailAddress** is the least private — it's a stable, human-readable identifier that
  enables cross-SP tracking. Use only for simple internal deployments.

---

## 4. RelayState

`RelayState` is an opaque value that the SP includes in the redirect to the IdP. The IdP
must return it **unchanged** in the response. Its primary purpose is to carry SP-side
context through the SSO round trip.

### Typical Usage

| Context Type     | Example Value                         |
|------------------|---------------------------------------|
| Original URL     | `L2Rhc2hib2FyZC9yZXBvcnRz` (base64 of `/dashboard/reports`) |
| Tenant ID        | `dGxhbnQtMTIz` (base64 of `tenant-123`)  |
| App context      | `eyJhcHAiOiJociJ9` (base64 of JSON)     |

### Security

- **Open redirect prevention:** validate RelayState against an allow-list of paths or
  verify it decodes to an internal relative URL. Never blindly redirect to an arbitrary
  value from the POST body.
- **Max length:** the SAML spec recommends 80 bytes. Longer values may be truncated or
  rejected by some IdPs. Use server-side session storage keyed by a short random token if
  you need to carry more data.
- **Integrity:** since RelayState is echoed by the IdP, it is not tamper-proof. Do not
  put security-critical data in it without signing (HMAC) or use server-side session state.

### GGID Gap

GGID's `EncodeForRedirect()` does not accept or append a `RelayState` parameter.
The redirect URL construction is left to the caller, which currently has no RelayState
support.

---

## 5. Assertion Consumer Service (ACS)

### Processing Steps

```
1. Receive SAMLResponse via HTTP-POST (form field "SAMLResponse")
2. Base64-decode → raw XML
3. Verify XML signature (Response-level and/or Assertion-level)
4. Validate conditions:
   - NotBefore / NotOnOrAfter (with clock skew tolerance)
   - AudienceRestriction (must contain this SP's entity ID)
   - Recipient (must match ACS URL)
5. Check InResponseTo matches the original AuthnRequest ID (replay prevention)
6. Extract NameID and AttributeStatement
7. Map attributes to local user (lookup or JIT provisioning)
8. Create session, issue application cookie/token
9. Redirect to RelayState target (or default landing page)
```

### Error Handling

| Condition               | HTTP Status | Action                                       |
|-------------------------|-------------|----------------------------------------------|
| Signature validation fail | 401       | Reject, log security event, show error page |
| Assertion expired        | 401       | Check `NotOnOrAfter` ± 60s clock skew       |
| Audience mismatch        | 403       | Assertion not meant for this SP              |
| Replay (duplicate ID)   | 409       | Track assertion ID in cache; reject if seen |
| Missing InResponseTo    | 401       | Reject unsolicited (unless IdP-initiated allowed) |

### Replay Prevention

Each assertion has a unique `ID`. The ACS endpoint must track seen assertion IDs in a
time-bounded store (Redis, with TTL matching the assertion validity window). GGID's
`VerifySignedAssertion` does NOT include replay checking — this must be added at the
handler layer.

---

## 6. Attribute Mapping

### Common IdP Attributes

| Claim (ADFS/Azure)                                        | Typical Field |
|-----------------------------------------------------------|---------------|
| `http://schemas.xmlsoap.org/ws/2005/05/claims/emailaddress` | email         |
| `http://schemas.xmlsoap.org/ws/2005/05/claims/givenname`   | first_name    |
| `http://schemas.xmlsoap.org/ws/2005/05/claims/surname`     | last_name     |
| `http://schemas.xmlsoap.org/claims/Group`                  | groups        |
| `http://schemas.xmlsoap.org/ws/2005/05/claims/upn`         | username      |

Okta and OneLogin use shorter names: `email`, `firstName`, `lastName`, `groups`.

### Mapping Configuration

```json
{
  "email":      "http://schemas.xmlsoap.org/ws/2005/05/claims/emailaddress",
  "first_name": "http://schemas.xmlsoap.org/ws/2005/05/claims/givenname",
  "last_name":  "http://schemas.xmlsoap.org/ws/2005/05/claims/surname",
  "groups":     "http://schemas.xmlsoap.org/claims/Group"
}
```

### Go Code (GGID Pattern)

GGID provides `ExtractAttributes` and `GetAttribute` in `assertion.go`:

```go
assertion, err := saml.VerifySignedAssertion(rawXML, idpCert)
attrs := saml.ExtractAttributes(assertion)

// Manual mapping
email := saml.GetAttribute(assertion, "http://schemas.xmlsoap.org/ws/2005/05/claims/emailaddress")
```

**Gap:** There is no configurable `AttributeMapper` struct. Attribute names are
hard-coded or must be looked up by the caller. A proper mapper would accept a
JSON config and return a typed user object.

---

## 7. SP-Initiated vs IdP-Initiated Comparison

| Aspect             | SP-Initiated                           | IdP-Initiated                        |
|--------------------|----------------------------------------|--------------------------------------|
| Who starts         | User at SP application                 | User at IdP portal                   |
| AuthnRequest       | Yes — SP generates and sends           | No — unsolicited response            |
| RelayState         | Supported — carries SP context         | Not supported                        |
| InResponseTo       | Present — validated against request ID | Absent — cannot bind to a request    |
| Security           | Higher — request/response binding      | Lower — no request binding           |
| Replay protection  | Strong (InResponseTo + assertion ID)  | Weaker (assertion ID only)           |
| Use case           | App-first access                       | Portal/dashboard launch              |
| Recommendation     | **Default for all SPs**                | Allow only with strict validation    |

### Recommendation

Always default to SP-initiated. If IdP-initiated must be supported (e.g., an existing
corporate portal), enforce these mitigations:

1. Require signed assertions (not just signed responses).
2. Validate `AudienceRestriction` strictly.
3. Implement assertion ID replay tracking.
4. Restrict allowed NameID values to pre-provisioned accounts.

---

## 8. GGID Implementation Analysis

GGID's `pkg/saml/` package provides the following capabilities:

| Feature                         | Status      | Notes                                                       |
|---------------------------------|-------------|-------------------------------------------------------------|
| AuthnRequest construction       | **Present** | `BuildAuthnRequest()` in `sp.go` — transient NameID, POST binding |
| HTTP-Redirect encoding          | **Present** | `EncodeForRedirect()` — DEFLATE + base64                    |
| AuthnRequest signing            | **Gap**     | No query-string signature for HTTP-Redirect                 |
| SP metadata generation          | **Present** | `GenerateSPMetadata()` in `sp.go`                           |
| Assertion parsing               | **Present** | `ParseAssertion()` in `assertion.go`                        |
| XMLDSig verification            | **Present** | `VerifySignedAssertion()` + `VerifySignedAssertionWithDigest()` |
| RSA/ECDSA support               | **Present** | PKCS#1 v1.5 + ECDSA ASN.1 in `signed_assertion.go`          |
| Conditions validation           | **Partial** | Only `NotBefore`/`NotOnOrAfter`; no `AudienceRestriction`   |
| RelayState                      | **Gap**     | Not generated or validated                                  |
| InResponseTo validation         | **Gap**     | No request ID tracking                                      |
| Attribute mapping config        | **Gap**     | Manual `GetAttribute()` only; no mapper struct              |
| JIT provisioning                | **Gap**     | No user creation logic in SAML package                      |
| Assertion replay prevention     | **Gap**     | No ID tracking store                                        |
| IdP-initiated SSO               | **Gap**     | No unsolicited response handling                            |

### Key Architectural Notes

- `VerifySignedAssertion` performs full XMLDSig: parses `<ds:Signature>`, verifies
  `DigestValue`, verifies the cryptographic signature, and validates time conditions.
- `ValidateConditions()` has a 60-second clock skew tolerance (`-time.Minute`).
- The signature path handles both namespaced (`ds:`) and non-namespaced variants.
- `constantTimeEqual` prevents timing attacks on digest comparison.

---

## 9. Roadmap

### Phase 1: AuthnRequest Signing + RelayState (3-4 days)

- Add `RelayState` parameter to redirect URL construction.
- Implement HTTP-Redirect query-string signing (SigAlg + Signature parameters).
- Store `AuthnRequest.ID` in Redis for `InResponseTo` validation.
- Add configurable NameIDPolicy format (persistent vs transient).

### Phase 2: ACS Hardening (3-4 days)

- Add `AudienceRestriction` validation to `ValidateConditions()`.
- Implement assertion ID replay tracking (Redis SET with TTL).
- Add configurable clock-skew tolerance.
- Enforce `InResponseTo` matching against stored request IDs.

### Phase 3: Attribute Mapping + JIT Provisioning (3-4 days)

- Create `AttributeMapper` struct with JSON config: `IdPAttribute → LocalField`.
- Ship default mappings for Azure AD, Okta, ADFS, OneLogin.
- Add `JITProvisioner` that creates/updates local user from assertion attributes.
- Map IdP groups to local roles.

### Phase 4: IdP-Initiated Support (2-3 days)

- Accept unsolicited responses with strict validation only.
- Require signed assertions, strict audience check, replay tracking.
- Add admin toggle per SP to enable/disable IdP-initiated.

**Estimated total effort: ~2 weeks** for a production-grade SP-initiated SSO with
optional IdP-initiated support.
