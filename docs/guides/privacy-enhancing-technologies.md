# Privacy-Enhancing Technologies

Differential privacy, homomorphic encryption, secure multi-party computation, data minimization, pseudonymization vs anonymization, cookieless tracking, and consent-bound access.

## Techniques

### Differential Privacy

Add controlled noise to analytics queries so individual records can't be identified:

```go
func privateCount(actual int, epsilon float64) int {
    // Laplace mechanism
    noise := laplaceSample(0, 1/epsilon)
    return max(0, actual + int(noise))
}

// E.g., "How many users in Engineering?" → returns 142 ± 5
// Individual user's department membership is protected
```

| Parameter | Effect |
|-----------|--------|
| ε (epsilon) | Smaller = more privacy, less accuracy |
| Typical ε | 0.1-1.0 for sensitive data |

### Pseudonymization vs Anonymization

| Approach | Reversible? | GDPR Status |
|----------|------------|-------------|
| Pseudonymization (hash + salt) | Yes (with key) | Still PII |
| Anonymization (irreversible) | No | Not PII |
| K-anonymity | No | Not PII (if k≥5) |

### Data Minimization Patterns

```yaml
data_minimization:
  - scope: openid → release only: sub
  - scope: profile → release only: display_name, locale
  - scope: email → release only: email, email_verified
  # Never release: password_hash, mfa_secret, recovery_codes
```

### Consent-Bound Data Access

```go
func accessData(ctx context.Context, userID, purpose string) (Data, error) {
    consent := consentStore.Get(userID, purpose)
    if !consent.Granted || consent.Expired() {
        return nil, ErrConsentRequired
    }
    audit.Log("data.accessed", map[string]interface{}{
        "user_id": userID,
        "purpose": purpose,
        "consent_id": consent.ID,
    })
    return fetchData(userID, purpose), nil
}
```

## See Also

- [Privacy by Design](privacy-by-design.md)
- [Zero-Knowledge Proof Identity](zero-knowledge-proof-identity.md)
- [Consent Management Design](consent-management-design.md)
