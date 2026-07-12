# Identity Proofing Guide

Document verification, biometric liveness, KBA, phone/email verification, per-risk-level requirements, and provider integration.

## Overview

Identity proofing verifies that a person is who they claim to be before granting access. GGID supports tiered proofing aligned with NIST 800-63 IAL (Identity Assurance Level).

## IAL Tiers (NIST 800-63)

| Level | Name | Requirement | Example |
|-------|------|-------------|---------|
| IAL1 | Self-asserted | User claims identity, no verification | Social sign-up |
| IAL2 | Remote proofing | Government ID + selfie match | Banking onboarding |
| IAL3 | In-person proofing | Physical presence + biometric | High-security gov |

## Verification Methods

### Email Verification

```bash
POST /api/v1/identity/verify/email
{"email": "user@corp.com"}
# → Sends 6-digit code, TTL 10 min

POST /api/v1/identity/verify/email/confirm
{"email": "user@corp.com", "code": "123456"}
# → 200 {"verified": true}
```

| Parameter | Value |
|-----------|-------|
| Code length | 6 digits |
| TTL | 10 minutes |
| Max attempts | 5 |
| Rate limit | 3/hour per email |

### Phone Verification (SMS OTP)

```bash
POST /api/v1/identity/verify/phone
{"phone": "+15550100"}
# → Sends 6-digit SMS code

POST /api/v1/identity/verify/phone/confirm
{"phone": "+15550100", "code": "123456"}
```

SMS is NOT considered strong verification (SIM swapping risk). Use for IAL1 only.

### Document Verification

Upload government ID → provider verifies authenticity:

```bash
POST /api/v1/identity/verify/document
{
  "provider": "onfido",  // or "jumio", "persona"
  "document_type": "passport",  // passport, drivers_license, id_card
  "document_image": "base64...",
  "selfie_image": "base64...",
  "tenant_id": "uuid"
}
# → 202 {"verification_id": "ver-abc", "status": "processing"}

# Poll for result
GET /api/v1/identity/verify/document/ver-abc
# → {"status": "approved", "confidence": 0.98, "details": {...}}
```

### Biometric Liveness

```bash
POST /api/v1/identity/verify/liveness
{
  "provider": "jumio",
  "session_token": "...",
  "challenge_type": "movement"  // movement, face_match
}
```

Detects:
- Static photo spoofing (printed photo)
- Screen replay (video of a video)
- 3D mask
- Deepfake

### Knowledge-Based Authentication (KBA)

```bash
POST /api/v1/identity/verify/kba
{
  "user_data": {
    "full_name": "Jane Doe",
    "address": "123 Main St",
    "ssn_last4": "1234"
  }
}
# → {"questions": [
#     {"id":"q1","question":"Which of these streets have you lived on?","choices":[...]},
#     {"id":"q2","question":"What model was your 2019 vehicle?","choices":[...]}
#   ]}

POST /api/v1/identity/verify/kba/answer
{"answers": [{"id":"q1","answer":"Oak St"},{"id":"q2","answer":"Honda Civic"}]}
# → {"status": "passed", "score": 4/5}
```

KBA is weak — answers can be found via public records. Use only as supplementary factor.

## Per-Risk-Level Requirements

| Risk Tier | Required Proofing | Use Case |
|-----------|------------------|----------|
| Low (IAL1) | Email verification | Free app signup |
| Medium (IAL2-Lite) | Email + Phone + ID upload | Marketplace seller |
| High (IAL2) | ID + Selfie match + Liveness | Financial services |
| Critical (IAL2+) | ID + Liveness + KBA + Address proof | Crypto exchange |
| Max (IAL3) | In-person verification | Government, defense |

```yaml
proofing_rules:
  - tier: low
    required: [email]
    provider: internal
    
  - tier: medium
    required: [email, phone, document]
    document_provider: onfido
    
  - tier: high
    required: [email, phone, document, liveness]
    document_provider: jumio
    liveness_provider: jumio
    
  - tier: critical
    required: [email, phone, document, liveness, kba]
    document_provider: persona
    kba_provider: experian
```

## Provider Integration

### Onfido

```bash
POST /api/v1/identity/verify/provider/onfido
{
  "first_name": "Jane",
  "last_name": "Doe",
  "email": "jane@corp.com",
  "document": {"type": "passport", "file": "base64..."},
  "face": {"variant": "video", "file": "base64..."}
}
# → Onfido webhook calls back with result
```

### Jumio

```bash
POST /api/v1/identity/verify/provider/jumio
{
  "customer_internal_ref": "user-uuid",
  "workflow": "DOCUMENT+FACE",
  "redirect_url": "https://app.example.com/verify/complete"
}
# → {"redirect_url": "https://..."}  // User completes in Jumio hosted flow
```

### Persona

```bash
POST /api/v1/identity/verify/provider/persona
{
  "inquiry_type": "government-id",
  "template_id": "tmpl_abc",
  "reference_id": "user-uuid"
}
# → Hosted inquiry URL
```

| Provider | Strengths | Cost |
|----------|-----------|------|
| Onfido | API-first, good docs | $$ |
| Jumio | Liveness expertise | $$$ |
| Persona | Custom workflows | $$ |

## Verification Lifecycle

```
Initiated → Processing → Result (approved/rejected/needs_review)
                                    │
                            ├── Approved → Mark user verified (IAL2)
                            ├── Rejected → User can retry (max 3)
                            └── Needs Review → Manual admin review
```

### Admin Review Queue

```bash
GET /api/v1/admin/identity/review-queue
# → Pending verifications needing manual review

POST /api/v1/admin/identity/review/{verification_id}
{"decision": "approve", "notes": "Document verified manually"}
```

## Audit Trail

```json
{
  "event": "identity.proofing.completed",
  "user_id": "uuid",
  "tier": "IAL2",
  "methods": ["email", "phone", "document", "liveness"],
  "provider": "jumio",
  "result": "approved",
  "confidence": 0.98,
  "timestamp": "2025-01-15T10:30:00Z"
}
```

Retention: 7 years (compliance requirement for KYC).

## See Also

- [Passwordless Auth Architecture](passwordless-auth-architecture.md)
- [MFA Architecture](mfa-architecture.md)
- [Privacy by Design](privacy-by-design.md)
- [Compliance Framework Mapping](compliance-framework-mapping.md)
