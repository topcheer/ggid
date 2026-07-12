# Identity Correlation Guide

This guide covers identity graph concepts, correlation signals, accuracy vs privacy tradeoffs, graph database approach, real-time correlation for fraud detection, and GGID's implementation.

## Identity Graph Concept

### What is an Identity Graph?

An identity graph maps a single real-world identity across multiple identity providers, devices, sessions, and contexts:

```
         ┌─────────────┐
         │ Real Identity│
         │  "John Doe"  │
         └──────┬───────┘
                │
    ┌───────────┼───────────┐
    │           │           │
┌───▼───┐ ┌────▼────┐ ┌────▼────┐
│ LDAP  │ │ Google  │ │  SAML   │
│ uid:  │ │ sub:    │ │ NameID: │
│ jdoe  │ │ 123456  │ │ j@ex.com│
└───┬───┘ └────┬────┘ └────┬────┘
    │          │           │
    └──────┬───┘───────────┘
           │
    ┌──────┼──────┐
    │      │      │
┌───▼──┐┌──▼───┐┌──▼───┐
│MacBook││iPhone││iPad  │
│fp:abc ││fp:def││fp:ghi│
└───────┘└──────┘└──────┘
```

### Graph Nodes

| Node Type | Examples |
|---|---|
| User | LDAP uid, Google sub, SAML NameID |
| Device | Fingerprint hash, hardware ID |
| Session | Session ID, token JTI |
| Credential | Email, phone, biometric template |
| Network | IP address, ASN, geolocation |

## Correlation Signals

### Signal Types

| Signal | Strength | Privacy | Example |
|---|---|---|---|
| Email address | High | PII | Same email across IdPs |
| Phone number | High | PII | Same phone for 2FA |
| Device fingerprint | Medium | Medium | Same browser fingerprint |
| IP address | Low | Low | Same IP range |
| Behavioral pattern | Medium | Low | Same login time, navigation |
| Biometric | Very High | Very High | Same face/fingerprint |
| Name | Low | PII | Same full name |

### Signal Weighting

```go
type CorrelationEngine struct {
    weights map[SignalType]float64
}

func DefaultWeights() map[SignalType]float64 {
    return map[SignalType]float64{
        SignalEmail:       0.9,
        SignalPhone:       0.8,
        SignalBiometric:   1.0,
        SignalDevice:      0.6,
        SignalBehavioral:  0.4,
        SignalIP:          0.2,
        SignalName:        0.3,
    }
}

func (e *CorrelationEngine) Score(signals []Signal) float64 {
    totalWeight := 0.0
    matchedWeight := 0.0
    for _, s := range signals {
        w := e.weights[s.Type]
        totalWeight += w
        if s.Matched {
            matchedWeight += w
        }
    }
    if totalWeight == 0 {
        return 0
    }
    return matchedWeight / totalWeight
}
```

## Correlation Accuracy vs Privacy

### Tradeoff

| Approach | Accuracy | Privacy | Use Case |
|---|---|---|---|
| Full correlation | High | Low | Fraud detection |
| Probabilistic only | Medium | High | Risk scoring |
| Pseudonymous | Medium | High | Analytics |
| No correlation | N/A | Maximum | Privacy-first |

### Privacy-Preserving Techniques

1. **Bloom filters** — Probabilistic matching without storing raw values
2. **Differential privacy** — Add noise to prevent re-identification
3. **Secure multi-party computation** — Correlate without sharing raw data
4. **Tokenization** — Replace identifiers with tokens, correlate on tokens

```go
func bloomFilterMatch(value string, filter *bloom.Filter) bool {
    return filter.Test([]byte(value))
}

func addToBloomFilter(value string, filter *bloom.Filter) {
    filter.Add([]byte(value))
}
```

## Graph Database Approach

### Graph Model

```
(User:jdoe) -[:SAME_EMAIL]-> (User:j@ex.com)
(User:jdoe) -[:SAME_DEVICE]-> (Device:fp-abc)
(User:jdoe) -[:LOGGED_IN_FROM]-> (IP:192.168.1.50)
(Device:fp-abc) -[:USED_BY]-> (User:jane)  // Shared device!
```

### Cypher Query Example

```cypher
// Find all identities correlated with a given user
MATCH (u:User {id: 'jdoe'})-[:CORRELATED*1..3]-(related)
RETURN related.id, related.type, count(*) as strength
ORDER BY strength DESC

// Find suspicious correlations (same device, different users)
MATCH (d:Device)<-[:USED]-(u1:User), (d)<-[:USED]-(u2:User)
WHERE u1 <> u2
RETURN d.id, u1.id, u2.id
```

### Graph Schema

```yaml
identity_graph:
  nodes:
    - label: User
      properties: [id, tenant_id, source, created_at]
    - label: Device
      properties: [fingerprint, type, first_seen, last_seen]
    - label: Credential
      properties: [type, value_hash, verified]
    - label: Network
      properties: [ip, asn, country, city]
  edges:
    - type: CORRELATED
      properties: [signal, weight, timestamp]
    - type: USED
      properties: [first_seen, last_seen, count]
    - type: LOGGED_IN_FROM
      properties: [timestamp, success]
```

## Real-Time Correlation for Fraud Detection

### Fraud Detection Signals

| Signal | Fraud Indicator |
|---|---|
| Same device, different accounts | Account sharing or takeover |
| Same IP, many accounts | Bot/credential stuffing |
| Impossible travel | Account compromise |
| New device + password change | Takeover attempt |
| Same biometric, different accounts | Identity fraud |

### Real-Time Pipeline

```
Login Event → Extract Signals → Query Graph → Score → Decision
                  ↓                ↓          ↓
              Add to graph    Find matches   Allow/Challenge/Block
```

### Implementation

```go
func (e *CorrelationEngine) EvaluateLogin(userID string, ctx *LoginContext) *FraudAssessment {
    assessment := &FraudAssessment{}
    
    // Extract signals from login context
    signals := e.extractSignals(ctx)
    
    // Query graph for correlations
    correlations := e.graph.FindCorrelations(userID, signals)
    
    // Check for red flags
    for _, corr := range correlations {
        switch corr.Type {
        case "shared_device":
            if corr.DifferentTenants {
                assessment.RiskScore += 0.3
                assessment.Flags = append(assessment.Flags, "shared_device_cross_tenant")
            }
        case "impossible_travel":
            assessment.RiskScore += 0.4
            assessment.Flags = append(assessment.Flags, "impossible_travel")
        case "credential_reuse":
            assessment.RiskScore += 0.2
            assessment.Flags = append(assessment.Flags, "credential_reuse")
        }
    }
    
    // Update graph with new signals
    e.graph.AddSignals(userID, signals)
    
    return assessment
}
```

## GGID Identity Correlation

### Configuration

```yaml
identity_correlation:
  enabled: true
  graph:
    backend: "neo4j"  # or "memory" for small deployments
    url: "bolt://neo4j:7687"
  signals:
    email: true
    phone: true
    device: true
    ip: true
    behavioral: false  # Privacy concern, off by default
  scoring:
    threshold_correlated: 0.7
    threshold_suspicious: 0.85
  privacy:
    bloom_filter: false
    hash_signals: true
    retention: 90d
  fraud_detection:
    real_time: true
    alert_on: ["shared_device_cross_tenant", "impossible_travel"]
```

## Best Practices

1. **Hash sensitive signals** — Don't store raw emails/phones in graph
2. **Set correlation thresholds** — Not all matches indicate same identity
3. **Respect privacy** — Minimize data used for correlation
4. **Audit correlation access** — Track who queries the identity graph
5. **Set retention limits** — Don't keep correlation data forever
6. **Use for risk, not identity** — Correlation informs risk, doesn't merge identities
7. **Handle shared devices** — Don't assume same device = same person
8. **Consider privacy regulations** — GDPR requires justification for correlation
9. **Test with known patterns** — Verify correlation accuracy
10. **Human review for high-risk** — Don't auto-act on correlation alone