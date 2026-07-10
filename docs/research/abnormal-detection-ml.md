# ML-Based Anomaly Detection for GGID IAM

> **Status:** Research / Design
> **Priority:** P2 (rule-based detection first, ML layered on top)
> **Dependencies:** `pkg/audit` event pipeline, NATS JetStream, credential-theft-defense.md rule framework

---

## 1. Overview

Rule-based detection is precise but limited: "if `login_failed > 5` in 10 min, alert."
Rules catch *known* patterns but miss novel or subtle deviations.

ML-based anomaly detection learns what *normal* looks like per user/tenant and flags
deviations. It is **adaptive** — retraining on recent data means the model evolves
as behavior changes. It finds **unknown attack patterns** no human wrote a rule for.

| IAM Use Case | Description |
|---|---|
| Account takeover | Stolen credentials from new device/location |
| Insider threat | Legitimate user escalating privileges or exfiltrating |
| Bot / credential stuffing | Automated high-volume logins, rotating IPs |
| Privilege abuse | Role assignment outside normal admin pattern |
| Session hijacking | Mid-session IP/geo shift, impossible travel |

GGID's `audit.Event` (NATS `audit.events`) already captures `ActorType`, `Action`,
`Result`, `IPAddress`, `UserAgent`, `Metadata`, `CreatedAt` — sufficient raw signal.

---

## 2. Feature Engineering

Each `audit.Event` is transformed into a numeric feature vector before scoring.

### Temporal Features

| Feature | Type | Source | Notes |
|---|---|---|---|
| `hour_of_day` | int (0-23) | `CreatedAt` | Login at 03:00 = unusual |
| `day_of_week` | int (0-6) | `CreatedAt` | Weekend access = unusual |
| `time_since_last_login` | float (s) | delta from prior event, same ActorID | Long gap → stale session |
| `login_frequency` | float/day | count over 24h window | Baseline deviation via z-score |

### Network Features

| Feature | Type | Source | Notes |
|---|---|---|---|
| `geo_distance` | float (km) | delta from last known geo | Impossible-travel detection |
| `asn` | int | GeoIP DB | Datacenter ASN → bot signal |
| `ip_reputation` | float (0-1) | threat intel feed | Known bad IP score |
| `new_ip_ratio` | float (0-1) | distinct IPs / total in 24h | High ratio = IP rotation |

### Device Features

| Feature | Type | Source | Notes |
|---|---|---|---|
| `device_fingerprint` | hash | `UserAgent` + browser attrs | Stable per device |
| `is_new_device` | bool→0/1 | fingerprint history per user | First sighting = risk |
| `os_family` | onehot | parsed from UA | Windows/macOS/Linux/iOS/Android |

### User Features

| Feature | Type | Source | Notes |
|---|---|---|---|
| `user_role` | onehot | identity service lookup | Admin baseline differs |
| `failed_attempts_24h` | int | count `result=failure` events | High → brute-force |
| `mfa_enrolled` | bool→0/1 | identity service | Behavioral difference |
| `account_age_days` | float | created_at lookup | New accounts differ |

### Feature Vector Construction

```go
// FeatureExtractor transforms an audit.Event into a numeric feature vector.
type FeatureExtractor struct {
    geoIP    *GeoIPLookup
    threatDB *ThreatIntelDB
    userSvc  UserMetaLookup  // role, mfa_enrolled, account_age
}

// Extract produces a fixed-length float64 vector for model scoring.
func (fe *FeatureExtractor) Extract(e audit.Event) FeatureVector {
    fv := FeatureVector{}

    // Temporal
    fv.Add(float64(e.CreatedAt.Hour()))
    fv.Add(float64(e.CreatedAt.Weekday()))
    fv.Add(float64(fe.timeSinceLastLogin(e.ActorID, e.CreatedAt)))

    // Network
    geo := fe.geoIP.Lookup(e.IPAddress)
    fv.Add(geo.DistanceKm)             // geo_distance
    fv.Add(geo.ASN)                    // asn
    fv.Add(fe.threatDB.Score(e.IPAddress)) // ip_reputation
    fv.Add(fe.newIPRatio(e.ActorID))   // last 24h

    // Device
    fp := fingerprint(e.UserAgent)
    fv.AddBool(fe.isNewDevice(e.ActorID, fp))
    fv.AddOneHot(parsedOS(e.UserAgent))  // 5-dim one-hot

    // User
    meta := fe.userSvc.Lookup(e.ActorID, e.TenantID)
    fv.AddOneHot(meta.Role)
    fv.Add(float64(meta.FailedAttempts24h))
    fv.AddBool(meta.MFAEnrolled)
    fv.Add(meta.AccountAgeDays)

    return fv
}
```

---

## 3. Unsupervised Models

Unsupervised models learn normal patterns without labeled attack data — the
**practical starting point** since most organizations lack labeled attacks.

### Isolation Forest (Recommended First Model)

**Algorithm:** randomly partition feature space. Anomalies are *few and different*,
so they isolate with shorter path lengths. Average path length across trees → score.

- **Train on:** 30-90 days of historical audit events
- **Detect:** new events with high anomaly score (short isolation path)
- **Pros:** no labels needed, handles high-dim + mixed features, O(n log n) training,
  sub-ms inference
- **Cons:** cold start (needs training data), concept drift (retrain weekly)

```go
// IsolationForestScorer wraps a trained Isolation Forest model.
type IsolationForestScorer struct {
    trees       []*IsolationTree
    threshold   float64  // anomaly score cutoff (e.g. 0.65)
}

func (s *IsolationForestScorer) Score(fv FeatureVector) AnomalyScore {
    var avgPath float64
    for _, t := range s.trees {
        avgPath += float64(t.pathLength(fv))
    }
    avgPath /= float64(len(s.trees))
    // Normalize: shorter path → higher anomaly score
    score := normalizeScore(avgPath, len(fv.values))
    return AnomalyScore{Value: score, IsAnomaly: score > s.threshold}
}
```

### One-Class SVM

Learns a boundary enclosing "normal" behavior in kernel space. Events outside = anomaly.
- **Pros:** mathematically principled, well-studied
- **Cons:** kernel selection sensitive, O(n^2) training → doesn't scale past ~50k samples

### Autoencoder (Neural Network)

Compresses to bottleneck then reconstructs. Anomalies produce high reconstruction
error (model can't compress unusual inputs). LSTM variant captures temporal sequences.
- **Pros:** captures complex nonlinear/temporal patterns
- **Cons:** needs more data, GPU training, overkill for most IAM use cases

### Recommendation

| Model | When to Use | Training Cost | Inference Latency |
|---|---|---|---|
| **Isolation Forest** | Default — start here | O(n log n), minutes | <1ms |
| One-Class SVM | Small data (<10k events) | O(n^2), minutes | ~1ms |
| Autoencoder | Complex temporal patterns | Hours, GPU | ~5ms |

**Start with Isolation Forest.** Retrain weekly to handle concept drift. Implement
in Go via [gorgonia](https://gorgonia.org) or call a Python microservice
(scikit-learn) over gRPC.

---

## 4. Supervised Models

### When to Use

Supervised models classify events as attack/normal using **labeled** training data.
Use when you have incident history with confirmed attack labels.

### Algorithms

| Algorithm | Strengths | Best For |
|---|---|---|
| **Random Forest** | Robust, interpretable feature importance | General baseline |
| **Gradient Boosting (XGBoost)** | High accuracy, handles mixed features | Production classifier |
| **Logistic Regression** | Simple, interpretable coefficients | Explainable baseline |

### Label Sources

- **Historical incidents:** manually labeled attack events from SOC tickets
- **Synthetic data:** generate known attack patterns (credential stuffing bursts,
  privilege escalation sequences)
- **Threat intel:** cross-reference IPs/ASNs against known-bad feeds to auto-label

### Challenge: Class Imbalance

Attack data is ~0.1% of all events. Naive training → model predicts "normal" for
everything.

**Mitigations:**
- SMOTE (Synthetic Minority Over-sampling) to balance training set
- Class weights (`class_weight='balanced'`) to penalize false negatives
- Pre-filter with unsupervised anomaly detection, then classify only high-risk events

---

## 5. Model Lifecycle

### Training Pipeline

```
Audit Events (NATS) → Feature Store → Feature Engineering → Model Training
       ↓                  ↑                    ↓                   ↓
  PostgreSQL          30-90d window      FeatureVector[]      Model artifact
                                                                (.pkl / ONNX)
```

1. **Data collection:** audit events → feature store (PostgreSQL or ClickHouse)
2. **Feature engineering:** extract numeric features from raw events
3. **Training:** fit model on 30-90 day window of normal behavior
4. **Validation:** test on held-out data, measure precision/recall/F1
5. **Deployment:** model artifact → scoring service (load on startup or hot-reload)

### Scoring Pipeline

1. **Real-time:** audit event → feature extraction → model score
2. **Threshold:** `score > 0.65` → alert; `> 0.85` → step-up; `> 0.95` → block
3. **Latency target:** <50ms per event
4. **Batch:** daily re-score of all events (catches patterns real-time misses)

### Monitoring

- **Drift detection:** track feature distribution over time (KS test / PSI)
- **Alert rate:** spike = model degradation OR real attack surge
- **Retraining trigger:** weekly cron OR drift threshold breach
- **A/B testing:** shadow-score 10% traffic with candidate model before promoting

---

## 6. GGID Audit Pipeline Integration

### Architecture

```
  Services ──audit.events──► NATS JetStream ──persist──► Audit Svc (Postgres)
  (Auth...)     (existing)        │
                                 │ subscribe (fan-out)
                                 ▼
  Gateway ◄──block/step-up── ML Scoring Svc ◄──features── Feature Store
             (subscribes      (Go/Python)                    (ClickHouse)
              ml.anomaly.*)       │
                                  ▼ publish ml.anomaly.{tenant}
```

The ML scoring service subscribes to the existing `audit.events` subject — a
**fan-out** pattern where both the Audit Service (persistence) and ML service
(scoring) receive all events independently.

### Feature Store

- **ClickHouse** (recommended): columnar OLAP, fast time-window aggregations,
  sub-second queries over millions of events. Ideal for `login_frequency`,
  `new_ip_ratio`, `failed_attempts_24h`.
- **PostgreSQL** (fallback): works for smaller deployments; pre-compute aggregates
  via materialized views refreshed every 5 minutes.

| Feature Category | Source | Freshness |
|---|---|---|
| Materialized (aggregates) | Feature store | 5-min refresh |
| Real-time (current event) | `audit.Event` fields | Instant |

### Go Implementation

```go
// MLScorer scores a feature vector and returns an anomaly assessment.
type MLScorer interface {
    Score(fv FeatureVector) AnomalyScore
}

type AnomalyScore struct {
    Value     float64 // 0.0 (normal) to 1.0 (anomaly)
    IsAnomaly bool
    Severity  string  // low | medium | high | critical
}

// MLScoringService consumes audit events, scores them, and publishes alerts.
type MLScoringService struct {
    extractor FeatureExtractor
    scorer    MLScorer
    js        jetstream.JetStream
    threshold float64
}

func (s *MLScoringService) HandleEvent(e audit.Event) {
    fv := s.extractor.Extract(e)
    score := s.scorer.Score(fv)

    if score.IsAnomaly {
        alert := MLAnomalyAlert{
            EventID: e.ID, TenantID: e.TenantID, ActorID: e.ActorID,
            Action: e.Action, Score: score.Value, Severity: score.Severity,
        }
        subject := fmt.Sprintf("ml.anomaly.%s", e.TenantID)
        data, _ := json.Marshal(alert)
        s.js.PublishAsync(subject, data)
    }
}
```

### Python Microservice Alternative

```
Go MLScoringService  ──gRPC──►  Python Service (scikit-learn / XGBoost)
                                  │
                                  ▼
                              trained model (.pkl)
```

- **Pros:** richer ML ecosystem, faster training, battle-tested libraries
- **Cons:** additional service, gRPC ~1-2ms latency, Python dependency management
- **Recommendation:** train in Python, export model to ONNX, load in Go for inference
  (via `onnx-go`) to avoid the gRPC hop

---

## 7. Anomaly Types to Detect

| Anomaly Type | Key Features | Best Model | Response Action |
|---|---|---|---|
| Impossible travel | `geo_distance`, `time_since_last_login` | **Rule** (simpler) | Require step-up MFA |
| New device + new location | `is_new_device`, `geo_distance` | Rule + Isolation Forest | Step-up MFA |
| Unusual time access | `hour_of_day`, user baseline | **Isolation Forest** | Alert SOC |
| Volume spike (brute force) | `login_frequency` vs baseline | **Statistical** (z-score) | Rate limit / block IP |
| Bot / credential stuffing | `user_agent`, timing, `asn` | **Supervised** classifier | Block IP, CAPTCHA |
| Privilege anomaly | `role.assign` outside pattern | **Supervised** classifier | Deny + alert admin |
| Session from datacenter IP | `asn` (AWS/GCP/Azure ranges) | Rule + ML | Step-up MFA |
| Token replay | `request_id` reuse, geo shift | Rule | Invalidate session |

**Hybrid approach:** simple, deterministic rules for patterns we *know* (impossible
travel, known bad IPs). ML for patterns we *suspect* (unusual time, behavioral
deviation). Supervised classification once we have labeled data.

---

## 8. Practical Considerations

### Start Simple — Phased Rollout

| Phase | Approach | Effort | Value |
|---|---|---|---|
| 1 | Rule-based (from credential-theft-defense.md) | Low | Immediate |
| 2 | Statistical baselines (z-score on login frequency) | Low | Medium |
| 3 | Isolation Forest (first ML model) | Medium | High |
| 4 | Supervised classifier (needs labeled data) | High | Highest |

### False Positive Management

- **Target:** <5% FP rate — higher causes alert fatigue
- **Human-in-the-loop:** high-severity → SOC review before block; medium → step-up MFA
- **Feedback loop:** analyst labels FPs → feeds supervised model training

### Data Privacy

- Feature store may contain PII (IP, geo) → **encrypt at rest** (AES-256, per-tenant keys)
- **Audit model decisions:** every score and action logged as `ml.score`/`ml.alert` event

---

## 9. Roadmap

| Phase | Deliverable | Effort |
|---|---|---|
| **1** | Feature store schema + feature engineering pipeline | ~1 week |
| **2** | Rule-based detection + statistical baselines (z-score) | ~1 week |
| **3** | Isolation Forest model (unsupervised), trained on 30d window | ~2 weeks |
| **4** | Real-time scoring service + NATS `ml.anomaly.*` integration | ~1 week |
| **5** | Supervised classifier (Random Forest/XGBoost) with labeled data | ~3+ weeks |
| **6** | Model monitoring (drift detection), auto-retraining, A/B testing | Ongoing |

**Priority:** P2. Implement rule-based detection first (Phase 1-2, from
`credential-theft-defense.md`). ML scoring (Phase 3-4) adds adaptive detection
on top. Supervised models (Phase 5-6) require accumulated labeled incident data.

---

## References

- Liu et al., "Isolation-Based Anomaly Detection" (ACM TKS 2012)
- [Isolation Mechanisms Survey](https://arxiv.org/pdf/2403.10802) — 2024 survey
- [AI-Driven IAM Anomaly Detection](https://github.com/keyfive5/AI-Driven-IAM-Anomaly-Detection)
  — Isolation Forest + LSTM + Random Forest reference
- GGID `pkg/audit/publisher.go` — event struct, NATS publisher
- GGID `docs/research/credential-theft-defense.md` — rule-based foundation
