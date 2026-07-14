# Zero-Knowledge Proof Identity

zk-SNARKs for attribute verification without revealing PII, circuit design, ZKP scope integration with OIDC, privacy-preserving auth flows, and performance benchmarks.

## Overview

Zero-knowledge proofs (ZKP) allow users to prove they possess an attribute (age > 18, has credential) without revealing the attribute value. GGID integrates ZKP with OIDC for privacy-preserving authentication.

## Use Cases

| Proof | What's Proven | What's Hidden |
|-------|--------------|---------------|
| Age > 18 | Boolean: true | Exact age, DOB |
| Citizenship | In allowed country list | Exact country |
| Credit score > 700 | Boolean: true | Exact score |
| Has valid credential | Credential exists | Credential content |
| Income > threshold | Boolean: true | Exact income |

## ZKP Flow (OIDC Integration)

```
1. User authenticates normally (password + MFA)
2. Client requests ZKP scope: scope=openid zkp:age_over_18
3. GGID generates zk-SNARK proof:
   - Input (private): user.attributes.dob
   - Input (public): threshold (18), current date
   - Circuit: verify(current_year - dob_year >= 18)
4. Proof returned as claim in ID token
5. Verifier checks proof without seeing DOB
```

## Circuit Design

```
// Circom circuit: age verification
template AgeCheck() {
    signal private input dob_year;
    signal input threshold;
    signal input current_year;

    signal diff = current_year - dob_year;
    diff >= threshold;
}

component main = AgeCheck();
```

### Proof Generation

```go
func GenerateAgeProof(dobYear, threshold, currentYear int) (*ZKProof, error) {
    // 1. Compile circuit (pre-compiled, cached)
    circuit := loadCompiledCircuit("age_check")

    // 2. Prepare inputs
    inputs := map[string]interface{}{
        "dob_year":     dobYear,      // Private
        "threshold":    threshold,     // Public
        "current_year": currentYear,  // Public
    }

    // 3. Generate proof (witness + proof)
    proof, err := groth16.Prove(circuit, inputs)
    if err != nil { return nil, err }

    // 4. Return proof (no private data in output)
    return &ZKProof{
        Proof:     proof,
        PublicInputs: []string{strconv.Itoa(threshold), strconv.Itoa(currentYear)},
        CircuitID: "age_check_v1",
    }, nil
}
```

### Proof Verification

```go
func VerifyAgeProof(proof *ZKProof) (bool, error) {
    circuit := loadCompiledCircuit(proof.CircuitID)
    return groth16.Verify(circuit, proof.Proof, proof.PublicInputs)
}
```

## OIDC ZKP Scope

```bash
# Client requests age proof instead of actual DOB
GET /authorize?scope=openid+profile+zkp:age_over_18
```

### Token Response

```json
{
  "sub": "pairwise-uuid",
  "zkp": {
    "age_over_18": {
      "circuit": "age_check_v1",
      "proof": "0xabc123...",
      "public_inputs": ["18", "2025"],
      "verified": true
    }
  }
}
```

Verifier sees proof + verified=true. Never sees DOB.

## Privacy Benefits

| Approach | Data Exposed | Privacy |
|----------|-------------|---------|
| Standard claim (DOB) | Full birth date | None |
| Boolean claim (is_adult) | True/false | Medium (server knows) |
| ZKP claim | Only proof | High (even GGID doesn't store proof) |

## Performance

| Operation | Time | Size |
|-----------|------|------|
| Circuit compile | 5s (once, cached) | — |
| Proof generation | 50-200ms | ~1KB |
| Proof verification | 5-20ms | — |
| Trusted setup | 10min (once per circuit) | — |

## Supported Circuits

| Circuit ID | Inputs | Proof |
|-----------|--------|-------|
| `age_check_v1` | dob (private), threshold (public) | Age >= threshold |
| `citizenship_v1` | country (private), allowed_list (public) | In allowed list |
| `credential_valid_v1` | credential_hash (private), issuer (public) | Credential exists |
| `salary_range_v1` | income (private), min/max (public) | Within range |

## See Also

- [Privacy by Design](privacy-by-design.md)
- [Identity Proofing Guide](identity-proofing-guide.md)
- [Consent Management Design](consent-management-design.md)
- [Data Classification Implementation](data-classification-implementation.md)
