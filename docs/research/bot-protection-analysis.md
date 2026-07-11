# Bot Protection Analysis

**Date**: 2025-07-11
**Status**: Code exists, wired, coverage analyzed

## What GGID Has

1. **botdetect.go** — BehavioralBotDetect middleware in gateway
   - Per-IP request rate tracking with configurable threshold + window
   - Background cleanup loop prevents memory leaks (fixed in 097f6a7)
   - **WIRED** into gateway Handler() chain (confirmed line 359 of router.go)

2. **Circuit breaker** — Per-backend circuit breaker (circuitbreaker.go)
   - Closed/Open/Half-open states
   - Prevents cascade failures from bot traffic

3. **Rate limiter** — Token bucket per-IP (token_bucket.go)
   - Tiered overrides (tier_ratelimit_test.go)
   - Per-tenant isolation

4. **Content-Type validator** — Blocks malformed bot requests (content_type_validator.go)

5. **Host header validation** — DNS rebinding defense (host_validation.go)

6. **Body size limit** — Prevents oversized payload attacks (bodysize.go)

## What Auth0/Keycloak Have That GGID Lacks

| Feature | Auth0 | Keycloak 26 | GGID |
|---|---|---|---|
| CAPTCHA/Turnstile | Built-in | Plugin | **Missing** |
| Device fingerprinting | Built-in | Plugin | **Missing** |
| ASN/IP reputation | Built-in | Plugin | **Missing** (research doc exists) |
| ML anomaly detection | Built-in | Plugin | **Missing** (research doc exists) |
| Bot score (0-100) | Built-in | Plugin | **Missing** |
| Progressive challenges | Built-in | Plugin | Rate limit only |

## Coverage of botdetect.go

- `NewBehavioralBotDetect`: 100%
- `Middleware`: 85%
- `cleanupLoop`: 33% (ticker loop needs integration test)
- `botRateStore` operations: 100%

## Recommendation

GGID's bot protection is functional for rate-limiting and circuit-breaking. For competitive parity, the priority additions are:

1. **Cloudflare Turnstile integration** — Drop-in CAPTCHA replacement, no user friction
2. **IP reputation feed** — Research doc at docs/research/ip-reputation-iam.md
3. **Progressive challenge** — JS challenge → CAPTCHA → block (three-tier response)

These are P2 items — the current rate limiter + circuit breaker provide baseline protection.
