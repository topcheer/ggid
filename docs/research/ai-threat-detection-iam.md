# AI/ML-Based Threat Detection for IAM Systems

> **Status:** Research / Design
> **Priority:** P2 (rule-based detection first, ML layered on top)
> **Companion Document:** `docs/research/abnormal-detection-ml.md` (model internals, feature engineering, unsupervised/supervised algorithms)
> **Scope:** Real-time vs batch architecture, Go ML inference deployment, false positive management, integration with GGID's existing security infrastructure, explainable AI, and gap analysis of GGID's current detection capabilities.

---

## Table of Contents

1. [Beyond Rule-Based Detection](#1-beyond-rule-based-detection)
2. [Real-Time vs Batch Processing](#2-real-time-vs-batch-processing)
3. [Go ML Inference with ONNX Runtime](#3-go-ml-inference-with-onnx-runtime)
4. [Feature Store for IAM](#4-feature-store-for-iam)
5. [Streaming Feature Computation](#5-streaming-feature-computation)
6. [False Positive Management](#6-false-positive-management)
7. [Model Lifecycle for IAM](#7-model-lifecycle-for-iam)
8. [Integration with GGID Security Infrastructure](#8-integration-with-ggid-security-infrastructure)
9. [Anomaly Detection Specifics for IAM](#9-anomaly-detection-specifics-for-iam)
10. [Explainable AI for IAM](#10-explainable-ai-for-iam)
11. [GGID AI Detection Gap Analysis](#11-ggid-ai-detection-gap-analysis)
12. [Gap Analysis & Recommendations](#12-gap-analysis--recommendations)

---

## 1. Beyond Rule-Based Detection

### 1.1 The Fundamental Limits of Rules

GGID's current detection stack is entirely rule-based. Every check is a hard-coded
if/then: "if 5 failed logins in 15 minutes, lock the account" (`anomaly_detection.go`),
"if IP failure count >= 3 across users, flag brute force" (`risk_auth.go`), "if User-Agent
contains 'sqlmap', block" (`botdetect.go`). These are deterministic, explainable, and
fast — and they catch *known* attack patterns. But they have structural blind spots:

| Limitation | Rule-Based Behavior | What Gets Missed |
|---|---|---|
| **Static thresholds** | 5 failures → lock | An attacker doing 4 failures per 15-min window never triggers the rule. "Slow brute force" stays invisible. |
| **Per-dimension blindness** | Checks IP, user, device independently | An attacker rotating across 50 IPs with 3 attempts each (150 total) evades the per-IP threshold. The *pattern* (distributed low-volume attack) is invisible to per-dimension rules. |
| **No behavioral baselines** | "Night login = +10 risk" applied uniformly | A developer who routinely deploys at 2 AM triggers the same "anomalous hour" flag as an attacker. No personalization. |
| **No sequence awareness** | Each event scored independently | An attacker who logs in, changes MFA settings, creates an API key, and downloads the user list within 5 minutes looks like 4 normal events. The *sequence* is the attack. |
| **Manual maintenance** | Rules must be written for each known pattern | Zero-day attack patterns have no rule until a human writes one. Lag between attack emergence and rule deployment. |

### 1.2 What ML Catches That Rules Miss

#### Slow Brute Force (Low-and-Slow)

```
Attack pattern:     1 attempt per user per IP per 10 minutes
Volume per IP:      6/hour (below rate limit threshold)
Volume per user:    1/hour (below lockout threshold)
Total over 24h:     1,440 attempts across 240 IPs × 6 users
```

Rules see nothing — every individual counter stays below threshold. ML sees the
*aggregate pattern*: abnormal login success rate across a user population, geographic
clustering of IPs, timing regularity (automated tools have near-perfect intervals).

#### Credential Stuffing with Rotating IPs

```
Attack pattern:     Botnet tries leaked credentials from 10,000 IPs
Per-IP volume:      2-3 attempts (never triggers rate limiter)
Per-user volume:    1 attempt (never triggers lockout)
IP diversity:       High (residential proxy pool)
Detection signal:   Low success rate across high-IP-diversity, concentrated timeframe
```

The signal is not in any single dimension. It's the *combination*: high IP diversity +
low success rate + concentrated time window + leaked credential patterns. ML models
(Isolation Forest, supervised classifiers) can learn this multi-dimensional pattern.

#### Behavioral Anomalies

| Anomaly Type | Rule Can't Detect | How ML Helps |
|---|---|---|
| User accesses API endpoints they've never used before | Rules don't track per-user endpoint diversity | Sequence model learns each user's typical API surface |
| Admin grants roles outside their normal pattern (e.g., creates 50 admins at 3 AM) | No rule for "unusual admin behavior" | Model learns per-admin baselines for role operations |
| Session from datacenter ASN that user has never used | Rule checks datacenter ASNs generically, not per-user | Model flags deviation from user's historical ASN distribution |
| Token created with unusual scopes never requested before | Rules don't understand scope semantics | NLP/embedding model flags unusual scope combinations |

### 1.3 When ML Adds Value vs When Rules Suffice

```
┌──────────────────────────────────────────────────────────────────┐
│                   Detection Method Decision Framework              │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│   Known pattern + Single dimension  ──────────────►  RULE          │
│   (e.g., "5 failed logins → lock")                                 │
│                                                                    │
│   Known pattern + Multiple dimensions  ──────────►  RULE + SCORE    │
│   (e.g., impossible travel: geo + time)                            │
│                                                                    │
│   Known pattern + Sequence/timing  ─────────────►  RULE + TIMEOUT   │
│   (e.g., rapid role escalation in 5 min)                           │
│                                                                    │
│   Unknown pattern + Single user  ───────────────►  UNSUPERVISED ML  │
│   (e.g., unusual login time for THIS user)                         │
│                                                                    │
│   Unknown pattern + Population level  ──────────►  SUPERVISED ML    │
│   (e.g., credential stuffing across tenant)                        │
│                                                                    │
│   Novel attack + No historical data  ──────────►  ENSEMBLE + ALERT  │
│   (e.g., zero-day exploit pattern)                                 │
│                                                                    │
└──────────────────────────────────────────────────────────────────┘
```

**Rule-first principle:** Start with rules for what you know. Add ML for what you
suspect. Use supervised ML once you have labeled attack data. Never rely on ML alone
for blocking — always have rules as a safety net.

### 1.4 GGID's Current Detection Inventory

Based on source code review (`services/auth/internal/service/` and
`services/gateway/internal/middleware/`):

| Detection Capability | Implementation | File | Type |
|---|---|---|---|
| Failed login lockout | Redis sorted set, 5 attempts/15min | `anomaly_detection.go` | Rule |
| Geo anomaly | Haversine distance, 500km threshold | `anomaly_detection.go` | Rule |
| New device check | Redis SADD fingerprint set | `anomaly_detection.go` | Rule |
| Risk scoring (0-100) | Weighted scoring of 5 signals | `risk_auth.go` | Rule |
| Step-up MFA trigger | Risk score >= 30 | `risk_auth.go` | Rule |
| Brute-force detection | Multi-user per IP (>=3 users) | `risk_auth.go` | Rule |
| IP blocklist | Redis SET with TTL | `risk_auth.go` | Rule |
| Static bot detection | UA pattern matching (sqlmap, nikto...) | `botdetect.go` | Rule |
| Behavioral bot detection | Per-IP request rate threshold | `botdetect.go` | Rule |
| Fixed-window rate limit | Per-path, per-IP | `ratelimit.go` | Rule |
| Sliding-window rate limit | Redis Lua + sorted sets, per-tier | `sliding_ratelimit.go` | Rule |
| Token bucket limit | Per-tenant + IP | `token_bucket.go` | Rule |
| Adaptive rate limit | Latency-based QPS adjustment | `adaptive_geo_dedup.go` | Rule |
| Geo enrichment | IP prefix → country/city | `adaptive_geo_dedup.go` | Enrichment |

**Score: 14 rule-based capabilities, 0 ML-based capabilities.**

---

## 2. Real-Time vs Batch Processing

### 2.1 Two Detection Modes

Threat detection operates in two fundamentally different modes with different latency
budgets, data requirements, and action capabilities:

| Dimension | Real-Time (Inline) | Batch (Post-Hoc) |
|---|---|---|
| **When** | During the authentication request | After the event is logged |
| **Latency budget** | < 50ms (on the request critical path) | Minutes to hours |
| **Data available** | Current event + Redis-cached features | Full historical dataset |
| **Action** | Block, challenge (MFA), throttle | Alert, investigate, quarantine |
| **Model** | Lightweight (ONNX, < 1ms inference) | Complex (sequence models, graph analysis) |
| **Feature source** | Online feature store (Redis) | Offline feature store (Parquet/S3) |
| **False positive cost** | High — blocks legitimate users | Lower — just an alert to review |
| **Throughput** | Must match peak auth QPS | Can process at leisure |

### 2.2 Real-Time Inline Scoring

Real-time scoring happens *synchronously* within the auth request. The risk score
determines the response: allow, challenge (step-up MFA), or block.

```
User Login Request Flow (with ML scoring):

    Client ────POST /api/v1/auth/login────► Gateway
                                              │
                                    ┌─────────┴─────────┐
                                    │  Rate Limiter      │  < 1ms (rule)
                                    │  Bot Detect        │  < 1ms (rule)
                                    │  Geo Enrich        │  < 1ms
                                    └─────────┬─────────┘
                                              │
                                    ┌─────────┴─────────┐
                                    │  Feature Extract   │  ~5-10ms
                                    │  (Redis reads)     │  (pipelined)
                                    └─────────┬─────────┘
                                              │
                                    ┌─────────┴─────────┐
                                    │  ML Score (ONNX)   │  ~2-5ms
                                    │  Risk Assessment   │
                                    └─────────┬─────────┘
                                              │
                                    ┌─────────┴─────────┐
                                    │  Decision Engine   │  < 1ms
                                    │  (rule + ML)       │
                                    └─────────┬─────────┘
                                              │
                          ┌───────────────────┼───────────────────┐
                          ▼                   ▼                   ▼
                     ALLOW (score<30)    CHALLENGE (30-70)    BLOCK (score>70)
                     ────────────        ─────────────         ────────
                     Return JWT          Return MFA challenge  Return 403
                     Record success      Record challenge      Record block
```

### 2.3 Latency Budget Analysis

The total inline latency budget for ML scoring must fit within the auth request's
acceptable response time. For GGID, the auth service already has Redis lookups for
rate limiting and anomaly detection — ML scoring must add minimal overhead:

```
Auth request components (target: < 200ms total):
├── Gateway routing + TLS:          ~5ms
├── Rate limiter check:             ~2ms (Redis)
├── Bot detection:                  ~1ms (in-memory)
├── Credential verification:        ~20-50ms (bcrypt/argon2)
├── Feature extraction (ML):        ~5-15ms (Redis pipeline)  ← NEW
├── ML inference (ONNX):            ~2-5ms                    ← NEW
├── Risk decision:                  ~1ms
├── JWT generation:                 ~2ms
├── Audit event publish:            ~1ms (async NATS)
└── Network round-trip:             ~10-50ms
                                    ──────────
Total with ML:                      ~49-127ms  (was ~41-122ms)
ML overhead:                        ~8-20ms (7-16% increase)
```

The 8-20ms overhead is acceptable. The key insight: **feature extraction dominates**
(5-15ms), not model inference (2-5ms). Optimizing Redis reads via pipelining is more
impactful than model optimization.

### 2.4 Batch Post-Hoc Analysis

Batch analysis processes all events asynchronously, catching patterns that real-time
scoring misses due to limited context window:

```go
// BatchAnalyzer processes historical events to find patterns that
// real-time scoring cannot detect. Runs as a scheduled job.
type BatchAnalyzer struct {
    eventRepo   EventRepository     // reads from audit DB
    featureStore OfflineFeatureStore // reads from Parquet
    alertPub    AlertPublisher       // publishes to NATS
    window      time.Duration        // analysis window (e.g., 1h)
}

// Run executes a batch analysis cycle. Designed to be called by a
// cron scheduler every hour.
func (ba *BatchAnalyzer) Run(ctx context.Context) error {
    now := time.Now()
    since := now.Add(-ba.window)

    // 1. Fetch all events in the window.
    events, err := ba.eventRepo.GetEventsInRange(ctx, since, now)
    if err != nil {
        return fmt.Errorf("fetch events: %w", err)
    }

    // 2. Group by tenant and analyze patterns.
    byTenant := groupByTenant(events)
    for tenantID, tenantEvents := range byTenant {
        alerts := ba.analyzeTenantPatterns(ctx, tenantID, tenantEvents)
        for _, alert := range alerts {
            ba.alertPub.Publish(ctx, alert)
        }
    }

    // 3. Update offline feature baselines.
    ba.featureStore.UpdateBaselines(ctx, events)

    // 4. Check for model drift.
    ba.checkDrift(ctx, events)

    return nil
}

// analyzeTenantPatterns detects population-level anomalies that
// real-time scoring cannot see.
func (ba *BatchAnalyzer) analyzeTenantPatterns(ctx context.Context,
    tenantID uuid.UUID, events []audit.Event) []BatchAlert {

    var alerts []BatchAlert

    // Pattern 1: Credential stuffing burst — many users, low success rate,
    // high IP diversity, concentrated timeframe.
    loginEvents := filterByAction(events, "user.login")
    if stuffingAlert := ba.detectCredentialStuffing(loginEvents); stuffingAlert != nil {
        stuffingAlert.TenantID = tenantID
        alerts = append(alerts, *stuffingAlert)
    }

    // Pattern 2: Privilege escalation chain — rapid role changes by admin.
    adminEvents := filterByAction(events, "role.assign", "role.update")
    if escAlert := ba.detectPrivilegeEscalation(adminEvents); escAlert != nil {
        escAlert.TenantID = tenantID
        alerts = append(alerts, *escAlert)
    }

    // Pattern 3: Mass data export — abnormal volume of list/read operations.
    readEvents := filterByActionPrefix(events, "user.list", "audit.query")
    if exportAlert := ba.detectMassExport(readEvents); exportAlert != nil {
        exportAlert.TenantID = tenantID
        alerts = append(alerts, *exportAlert)
    }

    return alerts
}

// detectCredentialStuffing looks for: >50 distinct IPs, <2% success rate,
// >100 distinct users attempted, within 1 hour.
func (ba *BatchAnalyzer) detectCredentialStuffing(events []audit.Event) *BatchAlert {
    if len(events) < 100 {
        return nil
    }

    ipSet := make(map[string]struct{})
    userSet := make(map[string]struct{})
    successes := 0

    for _, e := range events {
        ipSet[e.IPAddress] = struct{}{}
        if e.ActorID != uuid.Nil {
            userSet[e.ActorID.String()] = struct{}{}
        }
        if e.Result == "success" {
            successes++
        }
    }

    successRate := float64(successes) / float64(len(events))
    if len(ipSet) > 50 && len(userSet) > 100 && successRate < 0.02 {
        return &BatchAlert{
            Type:        "credential_stuffing",
            Severity:    "high",
            Description: fmt.Sprintf(
                "Credential stuffing detected: %d IPs, %d users, %.1f%% success rate",
                len(ipSet), len(userSet), successRate*100,
            ),
            EventIDs: extractIDs(events),
        }
    }
    return nil
}
```

### 2.5 Go Concurrency for Streaming Feature Computation

The feature extraction phase benefits from Go's goroutine/channel model. Multiple
Redis lookups can be parallelized:

```go
// ParallelFeatureExtractor extracts features concurrently using goroutines.
// Each feature lookup is independent and runs in parallel.
type ParallelFeatureExtractor struct {
    rdb    *redis.Client
    timeout time.Duration
}

// ExtractConcurrent computes all features for a login event in parallel.
// Uses a worker pool pattern to limit Redis connection usage.
func (fe *ParallelFeatureExtractor) ExtractConcurrent(
    ctx context.Context, req FeatureRequest,
) (FeatureVector, error) {
    ctx, cancel := context.WithTimeout(ctx, fe.timeout)
    defer cancel()

    type result struct {
        name string
        val  float64
        err  error
    }

    results := make(chan result, 8)

    // Launch parallel feature lookups.
    go func() {
        val, err := fe.getUserLoginCount(ctx, req.UserID)
        results <- result{"login_count_24h", float64(val), err}
    }()

    go func() {
        val, err := fe.getIPVelocity(ctx, req.IP)
        results <- result{"ip_velocity_1h", val, err}
    }()

    go func() {
        val, err := fe.getDeviceCount(ctx, req.UserID)
        results <- result{"device_count", float64(val), err}
    }()

    go func() {
        val, err := fe.getFailedAttempts(ctx, req.UserID)
        results <- result{"failed_attempts_1h", float64(val), err}
    }()

    go func() {
        val, err := fe.getASNDiversity(ctx, req.UserID)
        results <- result{"asn_diversity_24h", val, err}
    }()

    go func() {
        val, err := fe.getTimeSinceLastLogin(ctx, req.UserID, req.IP)
        results <- result{"time_since_last_login", val, err}
    }()

    go func() {
        val, err := fe.getGeoVelocity(ctx, req.UserID)
        results <- result{"geo_velocity_kmh", val, err}
    }()

    go func() {
        val, err := fe.getUniqueIPCount(ctx, req.UserID)
        results <- result{"unique_ips_24h", float64(val), err}
    }()

    // Collect results with timeout.
    fv := NewFeatureVector(8)
    for i := 0; i < 8; i++ {
        select {
        case r := <-results:
            if r.err != nil {
                fv.Set(r.name, 0) // default to 0 on error
            } else {
                fv.Set(r.name, r.val)
            }
        case <-ctx.Done():
            return fv, ctx.Err()
        }
    }

    return fv, nil
}

// PipelinedFeatureExtractor uses Redis pipelining instead of goroutines.
// More efficient for many small lookups — single round trip.
type PipelinedFeatureExtractor struct {
    rdb *redis.Client
}

// ExtractPipelined computes all features in a single Redis pipeline.
// This is preferred over goroutines because it reduces Redis round trips.
func (fe *PipelinedFeatureExtractor) ExtractPipelined(
    ctx context.Context, req FeatureRequest,
) (FeatureVector, error) {
    pipe := fe.rdb.Pipeline()

    // Queue all commands (no network I/O yet).
    loginCountCmd := pipe.ZCard(ctx, fmt.Sprintf("feat:login:%s", req.UserID))
    ipVelocityCmd := pipe.ZCard(ctx, fmt.Sprintf("feat:ipvel:%s", req.IP))
    deviceCountCmd := pipe.SCard(ctx, fmt.Sprintf("feat:devices:%s", req.UserID))
    failedCmd := pipe.Get(ctx, fmt.Sprintf("feat:failed:%s", req.UserID))
    ipSetCmd := pipe.ZCard(ctx, fmt.Sprintf("feat:userips:%s", req.UserID))
    lastLoginCmd := pipe.ZRevRangeWithScores(
        ctx, fmt.Sprintf("feat:logins:%s", req.UserID), 0, 0,
    )

    // Execute all commands in one round trip.
    _, err := pipe.Exec(ctx)
    if err != nil && err != redis.Nil {
        return FeatureVector{}, err
    }

    // Build feature vector from results.
    fv := NewFeatureVector(6)
    fv.Set("login_count_24h", float64(loginCountCmd.Val()))
    fv.Set("ip_velocity_1h", float64(ipVelocityCmd.Val()))
    fv.Set("device_count", float64(deviceCountCmd.Val()))

    failedCount, _ := failedCmd.Int()
    fv.Set("failed_attempts_1h", float64(failedCount))
    fv.Set("unique_ips_24h", float64(ipSetCmd.Val()))

    if len(lastLoginCmd.Val()) > 0 {
        lastScore := lastLoginCmd.Val()[0].Score
        timeSince := time.Since(time.Unix(0, int64(lastScore)))
        fv.Set("time_since_last_login_s", timeSince.Seconds())
    } else {
        fv.Set("time_since_last_login_s", 999999) // never logged in
    }

    return fv, nil
}
```

**Recommendation:** Use the pipelined approach for production. It achieves the same
parallelism as goroutines but with a single Redis round trip, reducing network
overhead by ~80%.

### 2.6 Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        GGID AI Threat Detection Architecture              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────┐     ┌──────────┐     ┌─────────────┐     ┌─────────────┐   │
│  │ Client  │────►│  Gateway  │────►│  Auth Svc   │────►│  Identity   │   │
│  └─────────┘     └─────┬────┘     └──────┬──────┘     └─────────────┘   │
│                        │                  │                               │
│            ┌───────────┼───────────┐      │                               │
│            ▼           ▼           ▼      ▼                               │
│      ┌──────────┐ ┌─────────┐ ┌──────────────────┐                       │
│      │ Rate     │ │ Bot     │ │ Risk Scoring     │                       │
│      │ Limiter  │ │ Detect  │ │ Middleware       │                       │
│      │ (rules)  │ │ (rules) │ │ (rules + ML)     │                       │
│      └──────────┘ └─────────┘ └────────┬─────────┘                       │
│                                      │ │                                 │
│                        ┌─────────────┘ └──────────────┐                 │
│                        ▼                               ▼                 │
│              ┌──────────────┐              ┌────────────────┐            │
│              │ Feature      │              │ ONNX Inference │            │
│              │ Extractor    │              │ Engine (Go)    │            │
│              │ (pipelined)  │              │ ~2-5ms         │            │
│              └──────┬───────┘              └────────┬───────┘            │
│                     │                               │                     │
│                     ▼                               ▼                     │
│              ┌──────────────┐              ┌────────────────┐            │
│              │ Redis        │              │ Model Registry │            │
│              │ Online       │              │ (versioned     │            │
│              │ Feature Store│              │  ONNX files)   │            │
│              └──────────────┘              └────────────────┘            │
│                                                                          │
│  ════════════════════════════════════════════════════════════════════    │
│                        ASYNC / BATCH LAYER                               │
│  ════════════════════════════════════════════════════════════════════    │
│                                                                          │
│  ┌─────────────┐    ┌──────────────────┐    ┌──────────────────────┐    │
│  │ NATS        │───►│ Batch Analyzer   │───►│ Alert Publisher      │    │
│  │ audit.events│    │ (hourly cron)    │    │ (NATS ml.alerts.*)   │    │
│  └──────┬──────┘    └────────┬─────────┘    └──────────────────────┘    │
│         │                    │                                          │
│         ▼                    ▼                                          │
│  ┌─────────────┐    ┌──────────────────┐                               │
│  │ Audit DB    │    │ Offline Feature  │                               │
│  │ (Postgres)  │    │ Store            │                               │
│  └─────────────┘    │ (Parquet/S3)     │                               │
│                     │ 30-day baselines │                               │
│                     └──────────────────┘                               │
│                              │                                          │
│                              ▼                                          │
│                     ┌──────────────────┐                               │
│                     │ Model Training   │                               │
│                     │ (Python, weekly) │                               │
│                     │ sklearn → ONNX   │                               │
│                     └──────────────────┘                               │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Go ML Inference with ONNX Runtime

### 3.1 Why ONNX for Go

GGID is written in Go. Running ML inference natively in Go — without a Python
sidecar — eliminates the gRPC hop (1-2ms latency), avoids Python dependency
management, and simplifies deployment. ONNX (Open Neural Network Exchange) is the
industry-standard model interchange format: train in Python (sklearn, TensorFlow,
PyTorch), export to `.onnx`, load in any language.

**GGID's constraint:** The existing `abnormal-detection-ml.md` recommends
`github.com/yalue/onnxruntime_go` for Go ONNX inference. This section provides the
detailed implementation.

### 3.2 Model Export from Python (sklearn to ONNX)

```python
#!/usr/bin/env python3
"""
export_model.py — Train Isolation Forest on audit features, export to ONNX.

Usage:
    python3 export_model.py --input audit_features.parquet --output iam_model.onnx
"""

import argparse
import numpy as np
import pandas as pd
from sklearn.ensemble import IsolationForest
from skl2onnx import to_onnx
from skl2onnx.common.data_types import FloatTensorType

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True, help="Parquet file with features")
    parser.add_argument("--output", default="iam_model.onnx")
    parser.add_argument("--n-estimators", type=int, default=100)
    parser.add_argument("--contamination", type=float, default=0.05)
    args = parser.parse_args()

    # Load features (extracted from audit events).
    df = pd.read_parquet(args.input)
    feature_cols = [
        "hour_of_day", "day_of_week", "time_since_last_login_s",
        "login_count_24h", "failed_attempts_1h", "ip_velocity_1h",
        "device_count", "unique_ips_24h", "asn_diversity_24h",
        "geo_velocity_kmh", "is_new_device", "is_new_ip",
        "is_datacenter_asn", "ip_reputation_score",
    ]
    X = df[feature_cols].values.astype(np.float32)

    # Train Isolation Forest.
    model = IsolationForest(
        n_estimators=args.n_estimators,
        contamination=args.contamination,
        random_state=42,
        n_jobs=-1,
    )
    model.fit(X)
    print(f"Trained on {len(X)} samples, {len(feature_cols)} features")

    # Export to ONNX.
    initial_type = [("input", FloatTensorType([None, len(feature_cols)]))]
    onnx_model = to_onnx(model, initial_types=initial_type, target_opset=15)

    # Save.
    with open(args.output, "wb") as f:
        f.write(onnx_model.SerializeToString())

    print(f"Exported model to {args.output}")
    print(f"Feature order: {feature_cols}")

    # Verify with test data.
    test_input = X[:5]
    expected = model.predict(test_input)
    print(f"Expected predictions: {expected}")
    print(f"Sanity check: {-expected} (1=normal, -1=anomaly in sklearn)")

if __name__ == "__main__":
    main()
```

### 3.3 Go ONNX Inference Server

```go
// Package mlinfer provides ONNX Runtime-based ML inference for GGID.
// It loads pre-trained models exported from Python (sklearn → ONNX) and
// performs sub-millisecond scoring of feature vectors.
package mlinfer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yalue/onnxruntime_go"
)

// ModelConfig configures the ONNX inference engine.
type ModelConfig struct {
	ModelPath      string   // path to .onnx file
	InputName      string   // ONNX input tensor name (default: "input")
	OutputName     string   // ONNX output tensor name
	FeatureNames   []string // ordered feature names matching training
	Threshold      float64  // anomaly score threshold for decision
	NumThreads     int      // ONNX Runtime intra-op threads (0 = auto)
	WarmupRuns     int      // warmup inference calls on load
}

// DefaultModelConfig returns production defaults for Isolation Forest.
func DefaultModelConfig(modelPath string) ModelConfig {
	return ModelConfig{
		ModelPath:    modelPath,
		InputName:    "input",
		OutputName:   "output",
		FeatureNames: defaultFeatureOrder,
		Threshold:    0.65,
		NumThreads:   0,
		WarmupRuns:   10,
	}
}

var defaultFeatureOrder = []string{
	"hour_of_day", "day_of_week", "time_since_last_login_s",
	"login_count_24h", "failed_attempts_1h", "ip_velocity_1h",
	"device_count", "unique_ips_24h", "asn_diversity_24h",
	"geo_velocity_kmh", "is_new_device", "is_new_ip",
	"is_datacenter_asn", "ip_reputation_score",
}

// InferenceEngine wraps an ONNX Runtime session for model inference.
// Thread-safe; multiple goroutines can call Score concurrently.
type InferenceEngine struct {
	mu         sync.RWMutex
	session    *onnxruntime_go.DynamicAdvancedSession
	cfg        ModelConfig
	version    atomic.Value // string: model version tag
	inferences atomic.Uint64
	totalTime  atomic.Int64 // nanoseconds
}

// NewInferenceEngine creates and initializes an ONNX inference engine.
func NewInferenceEngine(cfg ModelConfig) (*InferenceEngine, error) {
	// Initialize ONNX Runtime.
	if err := onnxruntime_go.InitializeRuntime(); err != nil {
		return nil, fmt.Errorf("init ONNX runtime: %w", err)
	}

	// Set thread count.
	if cfg.NumThreads > 0 {
		onnxruntime_go.SetNumberOfInterOpThreads(cfg.NumThreads)
	}

	// Create session from file.
	session, err := onnxruntime_go.NewDynamicAdvancedSession(
		[]string{cfg.ModelPath},
		[]string{cfg.InputName},
		[]string{cfg.OutputName},
	)
	if err != nil {
		return nil, fmt.Errorf("create ONNX session: %w", err)
	}

	engine := &InferenceEngine{
		session: session,
		cfg:     cfg,
	}
	engine.version.Store("v1.0.0")

	// Warmup runs to stabilize performance.
	dummy := make([]float32, len(cfg.FeatureNames))
	for i := 0; i < cfg.WarmupRuns; i++ {
		_, _ = engine.scoreRaw(dummy)
	}

	return engine, nil
}

// RiskScore represents the ML model's assessment.
type RiskScore struct {
	Value      float64   `json:"value"`       // 0.0 (normal) to 1.0 (anomaly)
	IsAnomaly  bool      `json:"is_anomaly"`
	Severity   string    `json:"severity"`    // low | medium | high | critical
	ModelVer   string    `json:"model_version"`
	Latency    float64   `json:"latency_ms"`  // inference latency
	Features   []float32 `json:"features"`    // input feature vector
}

// Score evaluates a feature vector and returns a risk score.
// Thread-safe; safe to call from concurrent HTTP handlers.
func (e *InferenceEngine) Score(features []float32) (RiskScore, error) {
	if len(features) != len(e.cfg.FeatureNames) {
		return RiskScore{}, fmt.Errorf(
			"feature count mismatch: got %d, expected %d",
			len(features), len(e.cfg.FeatureNames),
		)
	}

	start := time.Now()
	rawScore, err := e.scoreRaw(features)
	latency := time.Since(start).Seconds() * 1000

	// Record metrics.
	e.inferences.Add(1)
	e.totalTime.Add(int64(time.Since(start)))

	if err != nil {
		return RiskScore{}, fmt.Errorf("inference failed: %w", err)
	}

	score := normalizeScore(rawScore)

	result := RiskScore{
		Value:    score,
		IsAnomaly: score >= e.cfg.Threshold,
		ModelVer: e.version.Load().(string),
		Latency:  latency,
		Features: features,
	}
	result.Severity = classifySeverity(score)

	return result, nil
}

// scoreRaw performs the actual ONNX inference call.
func (e *InferenceEngine) scoreRaw(features []float32) (float32, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Prepare input tensor: shape [1, num_features].
	inputShape := []int64{1, int64(len(features))}
	input := onnxruntime_go.NewTensor[float32](inputShape)
	defer input.Destroy()
	copy(input.GetData(), features)

	// Run inference.
	outputs, err := e.session.Run(
		[]onnxruntime_go.Value{input},
	)
	if err != nil {
		return 0, err
	}

	// Extract output (Isolation Forest scores: negative = anomaly).
	defer outputs[0].Destroy()
	outputTensor := outputs[0].(*onnxruntime_go.Tensor[float32])
	data := outputTensor.GetData()
	if len(data) == 0 {
		return 0, fmt.Errorf("empty model output")
	}

	return data[0], nil
}

// normalizeScore converts raw model output to 0.0-1.0 risk score.
// Isolation Forest returns decision_function: positive = normal, negative = anomaly.
// We map: raw_score < -0.5 → risk 1.0, raw_score > 0.5 → risk 0.0
func normalizeScore(raw float32) float64 {
	// For Isolation Forest: score_samples range roughly [-1, 0]
	// More negative = more anomalous.
	if raw >= 0 {
		return 0
	}
	// Linear mapping: -0.5 → 0.5, -1.0 → 1.0
	normalized := float64(-raw * 2)
	if normalized > 1.0 {
		normalized = 1.0
	}
	return normalized
}

func classifySeverity(score float64) string {
	switch {
	case score >= 0.90:
		return "critical"
	case score >= 0.75:
		return "high"
	case score >= 0.50:
		return "medium"
	default:
		return "low"
	}
}

// AvgLatencyMs returns the average inference latency since engine start.
func (e *InferenceEngine) AvgLatencyMs() float64 {
	count := e.inferences.Load()
	if count == 0 {
		return 0
	}
	totalNs := e.totalTime.Load()
	return float64(totalNs) / float64(count) / 1e6
}

// Close releases ONNX Runtime resources.
func (e *InferenceEngine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.session != nil {
		e.session.Destroy()
		e.session = nil
	}
}

// Version returns the current model version tag.
func (e *InferenceEngine) Version() string {
	return e.version.Load().(string)
}
```

### 3.4 Model Versioning and Hot-Swap

```go
// ModelRegistry manages versioned ONNX models with hot-swap capability.
// The current production model can be swapped without restarting the service.
type ModelRegistry struct {
	mu         sync.RWMutex
	active     *InferenceEngine
	candidates map[string]*InferenceEngine // shadow models for A/B testing
	modelDir   string
	cfg        ModelConfig
}

// NewModelRegistry creates a registry that loads models from modelDir.
func NewModelRegistry(modelDir string, cfg ModelConfig) (*ModelRegistry, error) {
	reg := &ModelRegistry{
		candidates: make(map[string]*InferenceEngine),
		modelDir:   modelDir,
		cfg:        cfg,
	}

	// Load initial model.
	latest, err := reg.findLatestModel()
	if err != nil {
		return nil, fmt.Errorf("no model found in %s: %w", modelDir, err)
	}

	engine, err := NewInferenceEngine(cfg)
	if err != nil {
		return nil, err
	}
	engine.version.Store(latest.version)
	reg.active = engine

	return reg, nil
}

// HotSwap atomically replaces the active model. Old model is closed after
// in-flight requests complete (graceful shutdown via RWMutex).
func (reg *ModelRegistry) HotSwap(newModelPath, version string) error {
	cfg := reg.cfg
	cfg.ModelPath = newModelPath

	engine, err := NewInferenceEngine(cfg)
	if err != nil {
		return fmt.Errorf("load new model: %w", err)
	}
	engine.version.Store(version)

	// Atomic swap under write lock.
	reg.mu.Lock()
	old := reg.active
	reg.active = engine
	reg.mu.Unlock()

	// Close old engine after swap (no in-flight requests possible
	// because Score holds RLock during inference).
	if old != nil {
		old.Close()
	}

	return nil
}

// Score delegates to the active model.
func (reg *ModelRegistry) Score(features []float32) (RiskScore, error) {
	reg.mu.RLock()
	engine := reg.active
	reg.mu.RUnlock()
	return engine.Score(features)
}

// modelFile represents a versioned model artifact.
type modelFile struct {
	path    string
	version string
	modTime time.Time
}

// findLatestModel scans the model directory for the most recent .onnx file.
func (reg *ModelRegistry) findLatestModel() (*modelFile, error) {
	// Implementation: scan modelDir for *.onnx files,
	// parse version from filename (e.g., iam_model_v2.1.0.onnx),
	// return the latest by modTime.
	return nil, fmt.Errorf("not implemented")
}
```

### 3.5 Inference Latency Benchmarks

Expected latencies for ONNX Runtime inference on ARM64 (Apple M2 / AWS Graviton):

| Model | Features | ONNX Inference | Feature Extract (Redis) | Total Overhead |
|---|---|---|---|---|
| Isolation Forest (100 trees) | 14 | 0.3-0.8ms | 3-8ms (pipelined) | ~4-9ms |
| Random Forest (200 trees) | 14 | 0.5-1.2ms | 3-8ms | ~4-9ms |
| XGBoost (500 trees) | 14 | 0.8-2.0ms | 3-8ms | ~4-10ms |
| Small neural net (3 layers) | 14 | 0.1-0.3ms | 3-8ms | ~3-8ms |
| Large neural net (LSTM) | 14×20 | 2-5ms | 3-8ms | ~5-13ms |

All models are well within the 50ms real-time budget. Feature extraction (Redis
pipelined reads) dominates total overhead.

```go
// BenchmarkInferenceEngine measures ONNX inference latency.
// Run: go test -bench=BenchmarkInference -benchtime=10000x
func BenchmarkInferenceEngine(b *testing.B) {
	engine, err := NewInferenceEngine(DefaultModelConfig("testdata/iam_model.onnx"))
	if err != nil {
		b.Skip("ONNX model not available")
	}
	defer engine.Close()

	features := make([]float32, 14)
	for i := range features {
		features[i] = 0.5
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Score(features)
	}
}

// Expected output:
// BenchmarkInferenceEngine-8    10000    0.42 ms/op    2.1 KB/op    15 allocs/op
```

---

## 4. Feature Store for IAM

### 4.1 Online vs Offline Feature Stores

A feature store bridges the gap between training and inference. The **online store**
serves features in real-time (Redis, sub-millisecond). The **offline store** stores
historical features for training and batch analysis (Parquet/S3, columnar queries).

| Property | Online Store (Redis) | Offline Store (Parquet/S3) |
|---|---|---|
| Latency | < 1ms | Seconds (batch query) |
| Use case | Real-time feature lookup at auth time | Training data, batch analysis |
| Data volume | Last 24-48h, per-user state | 30-90 days, all events |
| Consistency | Eventually consistent | Point-in-time correct |
| Storage cost | High (RAM) | Low (object storage) |
| Schema | Key-value (Redis sorted sets, hashes) | Columnar (Parquet) |
| Updates | Per-event (streaming) | Daily/hourly batch export |

### 4.2 Online Feature Store (Redis)

The online store maintains per-user and per-IP rolling windows for real-time feature
computation. GGID already uses Redis for rate limiting and anomaly detection — the
feature store extends the same Redis instance with additional key patterns.

```go
// Package featurestore provides online (Redis) and offline (Parquet) feature
// storage for ML-based threat detection.
package featurestore

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// OnlineFeatureStore provides sub-millisecond feature lookups via Redis.
// Features are stored as sorted sets (time-windowed counts), hashes
// (key-value state), and sets (unique collections).
type OnlineFeatureStore struct {
	rdb     redis.Cmdable
	keyPfx  string // prefix for all keys (e.g., "feat:")
	maxAge  time.Duration // max feature retention (24h)
}

// NewOnlineFeatureStore creates a Redis-backed online feature store.
func NewOnlineFeatureStore(rdb redis.Cmdable) *OnlineFeatureStore {
	return &OnlineFeatureStore{
		rdb:    rdb,
		keyPfx: "feat:",
		maxAge: 24 * time.Hour,
	}
}

// RecordLoginEvent updates all online features for a login event.
// This is called AFTER authentication (regardless of success/failure)
// to keep features current. Uses Redis pipeline for efficiency.
func (fs *OnlineFeatureStore) RecordLoginEvent(ctx context.Context, evt LoginEvent) error {
	now := evt.Timestamp
	nowNs := now.UnixNano()

	pipe := fs.rdb.Pipeline()
	defer pipe.Exec(ctx)

	// --- Per-user features ---

	// 1. Login timestamp history (sorted set by timestamp).
	// Used for: login frequency, time since last login, inter-login intervals.
	loginKey := fs.key("logins", evt.UserID)
	pipe.ZAdd(ctx, loginKey, redis.Z{
		Score:  float64(nowNs),
		Member: fmt.Sprintf("%d:%s", nowNs, evt.Result),
	})
	pipe.ZRemRangeByScore(ctx, loginKey, "0", fmt.Sprintf("%d", now.Add(-fs.maxAge).UnixNano()))
	pipe.Expire(ctx, loginKey, fs.maxAge+time.Hour)

	// 2. IP set per user (set of distinct IPs).
	// Used for: unique IP count, new IP detection, IP diversity.
	userIPKey := fs.key("userips", evt.UserID)
	pipe.SAdd(ctx, userIPKey, evt.IP)
	pipe.Expire(ctx, userIPKey, fs.maxAge)

	// 3. Device fingerprints per user.
	// Used for: device count, new device detection.
	deviceKey := fs.key("devices", evt.UserID)
	pipe.SAdd(ctx, deviceKey, evt.DeviceFingerprint)
	pipe.Expire(ctx, deviceKey, fs.maxAge)

	// 4. ASN history per user.
	// Used for: ASN diversity, new ASN detection.
	asnKey := fs.key("userasns", evt.UserID)
	pipe.SAdd(ctx, asnKey, evt.ASN)
	pipe.Expire(ctx, asnKey, fs.maxAge)

	// 5. Failed attempt counter (hash with rolling window).
	if evt.Result == "failure" {
		failKey := fs.key("failed", evt.UserID)
		pipe.Incr(ctx, failKey)
		pipe.Expire(ctx, failKey, time.Hour) // 1-hour window
	} else {
		// Clear failures on successful login.
		failKey := fs.key("failed", evt.UserID)
		pipe.Del(ctx, failKey)
	}

	// --- Per-IP features ---

	// 6. Login attempts from this IP (sorted set by timestamp).
	// Used for: IP velocity, distributed attack detection.
	ipVelKey := fs.key("ipvel", evt.IP)
	pipe.ZAdd(ctx, ipVelKey, redis.Z{
		Score:  float64(nowNs),
		Member: fmt.Sprintf("%d:%s", nowNs, evt.UserID),
	})
	pipe.ZRemRangeByScore(ctx, ipVelKey, "0", fmt.Sprintf("%d", now.Add(-time.Hour).UnixNano()))
	pipe.Expire(ctx, ipVelKey, 2*time.Hour)

	// 7. Users attempted from this IP (set).
	// Used for: multi-user brute force detection (credential stuffing).
	ipUsersKey := fs.key("ipusers", evt.IP)
	pipe.SAdd(ctx, ipUsersKey, evt.UserID)
	pipe.Expire(ctx, ipUsersKey, time.Hour)

	// --- Geo features ---

	// 8. Last known geo per user (hash: IP → "lat,lon").
	if evt.Latitude != 0 && evt.Longitude != 0 {
		geoKey := fs.key("usergeo", evt.UserID)
		pipe.HSet(ctx, geoKey, evt.IP, fmt.Sprintf("%.4f,%.4f", evt.Latitude, evt.Longitude))
		pipe.Expire(ctx, geoKey, fs.maxAge)
	}

	// 9. Last login timestamp per user (simple string for fast lookup).
	lastLoginKey := fs.key("lastlogin", evt.UserID)
	pipe.Set(ctx, lastLoginKey, nowNs, fs.maxAge)

	// 10. Last login timestamp per user-IP (for geographic velocity).
	lastIPKey := fs.key("lastiplogin", evt.UserID, evt.IP)
	pipe.Set(ctx, lastIPKey, nowNs, time.Hour)

	return nil
}

// GetOnlineFeatures retrieves the feature vector for a login request.
// Returns pre-computed features from Redis. Called at auth time.
func (fs *OnlineFeatureStore) GetOnlineFeatures(
	ctx context.Context, req FeatureRequest,
) (map[string]float64, error) {
	pipe := fs.rdb.Pipeline()

	// Queue all feature lookups.
	loginCard := pipe.ZCard(ctx, fs.key("logins", req.UserID))
	userIPCard := pipe.SCard(ctx, fs.key("userips", req.UserID))
	deviceCard := pipe.SCard(ctx, fs.key("devices", req.UserID))
	asnCard := pipe.SCard(ctx, fs.key("userasns", req.UserID))
	failGet := pipe.Get(ctx, fs.key("failed", req.UserID))
	ipVelCard := pipe.ZCard(ctx, fs.key("ipvel", req.IP))
	ipUsersCard := pipe.SCard(ctx, fs.key("ipusers", req.IP))
	lastLoginGet := pipe.Get(ctx, fs.key("lastlogin", req.UserID))
	userIPsIsMember := pipe.SIsMember(ctx, fs.key("userips", req.UserID), req.IP)
	deviceIsMember := pipe.SIsMember(ctx, fs.key("devices", req.UserID), req.DeviceFingerprint)
	asnIsMember := pipe.SIsMember(ctx, fs.key("userasns", req.UserID), req.ASN)

	// Execute pipeline (single round trip).
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("feature store pipeline: %w", err)
	}

	features := make(map[string]float64)

	// Temporal features.
	features["login_count_24h"] = float64(loginCard.Val())
	features["unique_ips_24h"] = float64(userIPCard.Val())
	features["device_count"] = float64(deviceCard.Val())
	features["asn_diversity_24h"] = float64(asnCard.Val())
	features["ip_velocity_1h"] = float64(ipVelCard.Val())
	features["ip_user_count_1h"] = float64(ipUsersCard.Val())

	// Failed attempts.
	if failed, err := failGet.Int(); err == nil {
		features["failed_attempts_1h"] = float64(failed)
	} else {
		features["failed_attempts_1h"] = 0
	}

	// Time since last login.
	if lastLoginStr, err := lastLoginGet.Result(); err == nil {
		if lastNs, err := strconv.ParseInt(lastLoginStr, 10, 64); err == nil {
			since := time.Since(time.Unix(0, lastNs))
			features["time_since_last_login_s"] = since.Seconds()
		}
	} else {
		features["time_since_last_login_s"] = 999999 // never logged in
	}

	// Binary features (new IP / new device / new ASN).
	features["is_new_ip"] = boolToFloat(!userIPsIsMember.Val())
	features["is_new_device"] = boolToFloat(!deviceIsMember.Val())
	features["is_new_asn"] = boolToFloat(!asnIsMember.Val())

	// Current event context (not from Redis, from request).
	features["hour_of_day"] = float64(time.Now().UTC().Hour())
	features["day_of_week"] = float64(time.Now().UTC().Weekday())

	return features, nil
}

// LoginEvent represents a login attempt for feature recording.
type LoginEvent struct {
	UserID            string
	IP                string
	DeviceFingerprint string
	ASN               string
	Latitude          float64
	Longitude         float64
	Result            string // success | failure
	Timestamp         time.Time
}

// FeatureRequest represents the context for a feature lookup.
type FeatureRequest struct {
	UserID            string
	IP                string
	DeviceFingerprint string
	ASN               string
}

func (fs *OnlineFeatureStore) key(parts ...string) string {
	s := fs.keyPfx
	for _, p := range parts {
		s += p + ":"
	}
	return s[:len(s)-1] // strip trailing colon
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
```

### 4.3 Offline Feature Store (Parquet)

The offline store accumulates features for model training and batch analysis. It
exports daily from the audit database into columnar Parquet files.

```go
// OfflineFeatureStore stores historical features in Parquet format for
// model training and batch analysis.
type OfflineFeatureStore struct {
	storagePath string // local path or S3 prefix
	tenantID    string
}

// ExportFromAudit exports audit events to Parquet with engineered features.
// Run daily as a batch job.
func (fs *OfflineFeatureStore) ExportFromAudit(
	ctx context.Context, eventRepo EventRepository, date time.Time,
) error {
	// 1. Fetch all events for the date.
	start := date.Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour)
	events, err := eventRepo.GetEventsInRange(ctx, start, end)
	if err != nil {
		return fmt.Errorf("fetch events: %w", err)
	}

	// 2. Compute per-user daily aggregates (peer group baselines).
	baselines := fs.computeBaselines(events)

	// 3. Write to Parquet file.
	parquetPath := fmt.Sprintf("%s/features_%s.parquet",
		fs.storagePath, start.Format("2006-01-02"))

	// In production, use a Parquet writer (e.g., xitongsys/parquet-go).
	// Each row = one event with its full feature vector + computed aggregates.
	_ = parquetPath // write logic omitted for brevity

	// 4. Update baseline files.
	baselinePath := fmt.Sprintf("%s/baselines_%s.json",
		fs.storagePath, start.Format("2006-01-02"))
	_ = baselinePath

	return nil
}

// Baseline represents a per-user statistical baseline for peer comparison.
type Baseline struct {
	UserID              string  `json:"user_id"`
	AvgLoginsPerDay     float64 `json:"avg_logins_per_day"`
	AvgLoginHour        float64 `json:"avg_login_hour"`
	StdDevLoginHour     float64 `json:"stddev_login_hour"`
	UniqueIPs           int     `json:"unique_ips"`
	UniqueDevices       int     `json:"unique_devices"`
	UniqueASNs          int     `json:"unique_asns"`
	CommonASNs          []string `json:"common_asns"`
	FailedLoginRate     float64 `json:"failed_login_rate"`
	WeekendAccessRatio  float64 `json:"weekend_access_ratio"`
}

// computeBaselines calculates 30-day rolling baselines for each user.
func (fs *OfflineFeatureStore) computeBaselines(events []audit.Event) map[string]Baseline {
	byUser := make(map[string][]audit.Event)
	for _, e := range events {
		if e.ActorType == "user" {
			byUser[e.ActorID.String()] = append(byUser[e.ActorID.String()], e)
		}
	}

	baselines := make(map[string]Baseline, len(byUser))
	for userID, userEvents := range byUser {
		b := Baseline{UserID: userID}

		// Average logins per day.
		days := make(map[string]int)
		hours := make([]float64, 0, len(userEvents))
		ipSet := make(map[string]struct{})
		deviceSet := make(map[string]struct{})
		asnSet := make(map[string]struct{})
		failures := 0
		weekend := 0

		for _, e := range userEvents {
			day := e.CreatedAt.Format("2006-01-02")
			days[day]++
			hours = append(hours, float64(e.CreatedAt.Hour()))
			ipSet[e.IPAddress] = struct{}{}
			if ua := e.UserAgent; ua != "" {
				deviceSet[ua] = struct{}{}
			}
			if e.Result == "failure" {
				failures++
			}
			if e.CreatedAt.Weekday() == time.Saturday || e.CreatedAt.Weekday() == time.Sunday {
				weekend++
			}
		}

		b.AvgLoginsPerDay = float64(len(userEvents)) / float64(max(len(days), 1))
		b.UniqueIPs = len(ipSet)
		b.UniqueDevices = len(deviceSet)
		b.UniqueASNs = len(asnSet)
		b.FailedLoginRate = float64(failures) / float64(max(len(userEvents), 1))
		b.WeekendAccessRatio = float64(weekend) / float64(max(len(userEvents), 1))

		if len(hours) > 0 {
			b.AvgLoginHour = mean(hours)
			b.StdDevLoginHour = stddev(hours, b.AvgLoginHour)
		}

		baselines[userID] = b
	}

	return baselines
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func stddev(xs []float64, m float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += (x - m) * (x - m)
	}
	return sqrtSafe(s / float64(len(xs)-1))
}

func sqrtSafe(x float64) float64 {
	if x < 0 {
		return 0
	}
	return mathSqrt(x)
}
```

---

## 5. Streaming Feature Computation

### 5.1 Features at Auth Time

Real-time feature computation extracts signals from the current login request and
combines them with historical context from the online feature store. Each feature
captures a different aspect of the request's risk profile.

### 5.2 Feature Categories

| Category | Features | Data Source | Computation |
|---|---|---|---|
| **Temporal** | time_since_last_login, login_frequency_zscore | Redis sorted sets | ZRANGEBYSCORE, ZCARD |
| **Geographic** | geo_velocity_kmh, impossible_travel | Redis hash (last geo) | Haversine + time delta |
| **Network** | ip_velocity_1h, ip_user_count_1h, asn_diversity | Redis sorted sets/sets | ZCARD, SCARD |
| **Device** | device_count, is_new_device | Redis set | SISMEMBER, SCARD |
| **Behavioral** | failed_attempts_1h, user_login_baseline_deviation | Redis counter + offline | GET, baseline comparison |
| **Threat intel** | ip_reputation, is_datacenter_asn | External API (cached) | GET (Redis cache) |

### 5.3 Real-Time Feature Pipeline with Redis Sorted Sets

```go
// Package streamfeat provides real-time feature computation using Redis
// sorted sets for time-windowed aggregations.
package streamfeat

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

// FeaturePipeline computes real-time features at auth time.
// It reads from the online feature store and enriches with computed
// features that require multiple data points.
type FeaturePipeline struct {
	rdb       redis.Cmdable
	geoLookup GeoLookup // IP → lat/lon/ASN
	maxWindow time.Duration
}

// GeoLookup provides IP geolocation data.
type GeoLookup interface {
	Lookup(ip string) (lat, lon float64, asn string, err error)
}

// ComputeFeatures produces the full feature vector for ML scoring.
// This is the main entry point called by the risk-scoring middleware.
func (fp *FeaturePipeline) ComputeFeatures(
	ctx context.Context, req ScoreRequest,
) ([]float32, error) {
	// Start with online store features (pipelined Redis reads).
	online := fp.getOnlineFeatures(ctx, req)

	// Compute derived features (require calculation).
	derived := fp.computeDerived(ctx, req, online)

	// Merge into ordered feature vector matching model input.
	vector := fp.toModelInput(online, derived)

	return vector, nil
}

// ScoreRequest contains the context for a scoring request.
type ScoreRequest struct {
	UserID            string
	IP                string
	UserAgent         string
	DeviceFingerprint string
	Timestamp         time.Time
}

// getOnlineFeatures retrieves pre-computed features from Redis.
func (fp *FeaturePipeline) getOnlineFeatures(
	ctx context.Context, req ScoreRequest,
) map[string]float64 {
	pipe := fp.rdb.Pipeline()

	// Sorted set lookups for time-windowed features.
	loginKey := fmt.Sprintf("feat:logins:%s", req.UserID)
	ipVelKey := fmt.Sprintf("feat:ipvel:%s", req.IP)

	cutoff24h := req.Timestamp.Add(-24 * time.Hour).UnixNano()
	cutoff1h := req.Timestamp.Add(-time.Hour).UnixNano()

	loginCount24h := pipe.ZCount(ctx, loginKey, fmt.Sprintf("%d", cutoff24h), "+inf")
	ipVelCount1h := pipe.ZCount(ctx, ipVelKey, fmt.Sprintf("%d", cutoff1h), "+inf")

	// Set lookups for cardinality features.
	userIPCard := pipe.SCard(ctx, fmt.Sprintf("feat:userips:%s", req.UserID))
	deviceCard := pipe.SCard(ctx, fmt.Sprintf("feat:devices:%s", req.UserID))
	asnCard := pipe.SCard(ctx, fmt.Sprintf("feat:userasns:%s", req.UserID))
	ipUsersCard := pipe.SCard(ctx, fmt.Sprintf("feat:ipusers:%s", req.IP))

	// Membership checks for "is new" features.
	isNewIP := pipe.SIsMember(ctx, fmt.Sprintf("feat:userips:%s", req.UserID), req.IP)
	isNewDevice := pipe.SIsMember(ctx, fmt.Sprintf("feat:devices:%s", req.UserID), req.DeviceFingerprint)

	// Counter lookups.
	failedCount := pipe.Get(ctx, fmt.Sprintf("feat:failed:%s", req.UserID))

	// Last login timestamp.
	lastLogin := pipe.Get(ctx, fmt.Sprintf("feat:lastlogin:%s", req.UserID))

	// Last known geo.
	lastGeo := pipe.HGetAll(ctx, fmt.Sprintf("feat:usergeo:%s", req.UserID))

	// Execute pipeline.
	_, _ = pipe.Exec(ctx)

	features := make(map[string]float64)
	features["login_count_24h"] = float64(loginCount24h.Val())
	features["ip_velocity_1h"] = float64(ipVelCount1h.Val())
	features["unique_ips_24h"] = float64(userIPCard.Val())
	features["device_count"] = float64(deviceCard.Val())
	features["asn_diversity_24h"] = float64(asnCard.Val())
	features["ip_user_count_1h"] = float64(ipUsersCard.Val())
	features["is_new_ip"] = boolFloat(!isNewIP.Val())
	features["is_new_device"] = boolFloat(!isNewDevice.Val())

	if n, err := failedCount.Int(); err == nil {
		features["failed_attempts_1h"] = float64(n)
	} else {
		features["failed_attempts_1h"] = 0
	}

	// Time since last login.
	if ts, err := lastLogin.Int64(); err == nil {
		features["time_since_last_login_s"] = time.Since(time.Unix(0, ts)).Seconds()
	} else {
		features["time_since_last_login_s"] = 999999
	}

	// Store geo for derived computation.
	features["_last_geo_entries"] = float64(len(lastGeo.Val()))

	return features
}

// computeDerived calculates features that require arithmetic or external lookups.
func (fp *FeaturePipeline) computeDerived(
	ctx context.Context, req ScoreRequest, online map[string]float64,
) map[string]float64 {
	derived := make(map[string]float64)

	// Geographic velocity (km/h): distance from last login / time delta.
	lat, lon, asn, _ := fp.geoLookup.Lookup(req.IP)
	derived["current_lat"] = lat
	derived["current_lon"] = lon
	derived["asn"] = hashASN(asn)
	derived["is_datacenter_asn"] = isDatacenterASN(asn)

	if online["time_since_last_login_s"] < 999999 {
		// Get last geo from Redis.
		lastGeoKey := fmt.Sprintf("feat:usergeo:%s", req.UserID)
		lastGeo := fp.rdb.HGetAll(ctx, lastGeoKey).Val()

		maxVel := 0.0
		for _, coords := range lastGeo {
			var lastLat, lastLon float64
			if _, err := fmt.Sscanf(coords, "%f,%f", &lastLat, &lastLon); err != nil {
				continue
			}
			dist := haversine(lat, lon, lastLat, lastLon)
			timeHours := online["time_since_last_login_s"] / 3600
			if timeHours > 0 {
				vel := dist / timeHours
				if vel > maxVel {
					maxVel = vel
				}
			}
		}

		// Cap at 2000 km/h (commercial jet speed) for normalization.
		derived["geo_velocity_kmh"] = math.Min(maxVel, 2000) / 2000

		// Impossible travel: > 900 km/h means faster than commercial flight.
		if maxVel > 900 {
			derived["impossible_travel"] = 1
		} else {
			derived["impossible_travel"] = 0
		}
	} else {
		derived["geo_velocity_kmh"] = 0
		derived["impossible_travel"] = 0
	}

	// Login frequency z-score: (current - baseline_mean) / baseline_stddev.
	// Requires offline baseline. For now, use simple threshold.
	if online["login_count_24h"] > 50 {
		derived["login_freq_zscore"] = 3.0 // high anomaly
	} else if online["login_count_24h"] > 20 {
		derived["login_freq_zscore"] = 1.5
	} else {
		derived["login_freq_zscore"] = 0
	}

	// Off-hours detection (2 AM - 5 AM UTC).
	hour := float64(req.Timestamp.UTC().Hour())
	derived["hour_of_day"] = hour / 23.0 // normalized 0-1
	if hour >= 2 && hour <= 5 {
		derived["off_hours"] = 1
	} else {
		derived["off_hours"] = 0
	}

	// Day of week (normalized).
	derived["day_of_week"] = float64(req.Timestamp.Weekday()) / 6.0

	return derived
}

// toModelInput assembles the final feature vector in model-expected order.
func (fp *FeaturePipeline) toModelInput(
	online, derived map[string]float64,
) []float32 {
	// Order MUST match training feature order.
	return []float32{
		float32(derived["hour_of_day"] * 23),         // hour_of_day
		float32(derived["day_of_week"] * 6),          // day_of_week
		float32(online["time_since_last_login_s"]),   // time_since_last_login_s
		float32(online["login_count_24h"]),           // login_count_24h
		float32(online["failed_attempts_1h"]),        // failed_attempts_1h
		float32(online["ip_velocity_1h"]),            // ip_velocity_1h
		float32(online["device_count"]),              // device_count
		float32(online["unique_ips_24h"]),            // unique_ips_24h
		float32(online["asn_diversity_24h"]),         // asn_diversity_24h
		float32(derived["geo_velocity_kmh"] * 2000),  // geo_velocity_kmh
		float32(online["is_new_device"]),             // is_new_device
		float32(online["is_new_ip"]),                 // is_new_ip
		float32(derived["is_datacenter_asn"]),        // is_datacenter_asn
		0.0, // ip_reputation_score (from threat intel, cached)
	}
}

// haversine computes great-circle distance in km.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	lat1r := lat1 * math.Pi / 180
	lat2r := lat2 * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1r)*math.Cos(lat2r)
	return r * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// isDatacenterASN checks if an ASN is a known cloud/datacenter provider.
func isDatacenterASN(asn string) float64 {
	datacenterASNs := map[string]bool{
		"AS16509": true, // AWS
		"AS15169": true, // Google
		"AS8075":  true, // Microsoft Azure
		"AS14618": true, // Amazon AES
		"AS4837":  true, // China Unico
	}
	if datacenterASNs[asn] {
		return 1
	}
	return 0
}

func hashASN(asn string) float64 {
	if asn == "" {
		return 0
	}
	hash := uint32(0)
	for _, c := range asn {
		hash = hash*31 + uint32(c)
	}
	return float64(hash % 10000) // bucket ASN into numeric
}

func boolFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
```

---

## 6. False Positive Management

### 6.1 Why ML Has False Positives

ML models fundamentally trade precision for recall. An Isolation Forest with
`contamination=0.05` will flag 5% of normal events as anomalous — that's by design.
The question is not "can we eliminate false positives?" but "can we manage them
effectively?"

| FP Cause | Description | Mitigation |
|---|---|---|
| Concept drift | User behavior changes over time (new role, travel) | Regular retraining, drift detection |
| Insufficient baseline | New users have no history — everything looks anomalous | Warmup period (minimum events before scoring) |
| Feature correlation | Model finds spurious correlations in noise | Feature selection, regularization |
| Threshold too aggressive | Cutoff set for maximum recall catches normal edge cases | Adaptive thresholds, shadow mode |
| Population variance | Some users are inherently more variable than others | Per-user baselines, user clustering |

### 6.2 Shadow Mode: Score Without Acting

Shadow mode is the **most important false positive management tool**. The model
scores every event in real-time, but the scores are logged — never acted upon.
This allows:

1. **Validating the model** against production traffic before enforcement.
2. **Measuring false positive rate** on real data.
3. **Comparing model versions** (A/B testing) without risk.
4. **Building analyst confidence** before flipping to enforcement.

```go
// ShadowScorer evaluates ML risk scores without taking enforcement action.
// All scores are logged for analysis. Use during model validation phase.
type ShadowScorer struct {
	scorer   RiskScorer  // the ML model
	pipeline FeaturePipeline
	logger   ShadowLogger
	enabled  atomic.Bool
}

// RiskScorer is the interface for ML scoring implementations.
type RiskScorer interface {
	Score(features []float32) (RiskScore, error)
}

// ShadowLogger persists shadow scores for analysis.
type ShadowLogger interface {
	LogShadowScore(ctx context.Context, entry ShadowEntry) error
}

// ShadowEntry represents a shadow-scored event.
type ShadowEntry struct {
	Timestamp    time.Time      `json:"timestamp"`
	TenantID     string         `json:"tenant_id"`
	UserID       string         `json:"user_id"`
	IP           string         `json:"ip"`
	RiskScore    float64        `json:"risk_score"`
	Severity     string         `json:"severity"`
	ModelVersion string         `json:"model_version"`
	Features     map[string]float64 `json:"features"`
	RuleDecision string         `json:"rule_decision"` // what the rule engine decided
	MLDecision   string         `json:"ml_decision"`   // what ML would decide
	Agree        bool           `json:"agree"`         // did rule and ML agree?
}

// Evaluate computes a risk score in shadow mode.
// The score is logged but NOT used for enforcement decisions.
func (ss *ShadowScorer) Evaluate(
	ctx context.Context, req ScoreRequest, ruleDecision string,
) (RiskScore, error) {
	if !ss.enabled.Load() {
		return RiskScore{Value: 0, Severity: "unknown"}, nil
	}

	// Compute features.
	features, err := ss.pipeline.ComputeFeatures(ctx, req)
	if err != nil {
		return RiskScore{}, fmt.Errorf("feature computation: %w", err)
	}

	// Score with ML model.
	score, err := ss.scorer.Score(features)
	if err != nil {
		return RiskScore{}, fmt.Errorf("ml scoring: %w", err)
	}

	// Determine what ML WOULD do.
	mlDecision := "allow"
	if score.IsAnomaly {
		switch score.Severity {
		case "critical":
			mlDecision = "block"
		case "high":
			mlDecision = "block"
		case "medium":
			mlDecision = "challenge"
		}
	}

	// Log for analysis.
	entry := ShadowEntry{
		Timestamp:    time.Now(),
		UserID:       req.UserID,
		IP:           req.IP,
		RiskScore:    score.Value,
		Severity:     score.Severity,
		ModelVersion: score.ModelVer,
		RuleDecision: ruleDecision,
		MLDecision:   mlDecision,
		Agree:        ruleDecision == mlDecision,
	}

	// Log asynchronously — don't block the request.
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = ss.logger.LogShadowScore(bgCtx, entry)
	}()

	return score, nil
}

// SetEnabled toggles shadow mode on/off.
func (ss *ShadowScorer) SetEnabled(enabled bool) {
	ss.enabled.Store(enabled)
}

// ShadowAnalyzer analyzes shadow scoring results to measure model quality
// before promoting to enforcement.
type ShadowAnalyzer struct {
	repo ShadowEntryRepository
}

// AnalysisReport summarizes shadow scoring performance.
type AnalysisReport struct {
	TotalEvents     int            `json:"total_events"`
	MLAnomalies     int            `json:"ml_anomalies"`
	MLAnomalyRate   float64        `json:"ml_anomaly_rate"`
	RuleBlocks      int            `json:"rule_blocks"`
	RuleBlockRate   float64        `json:"rule_block_rate"`
	AgreementRate   float64        `json:"agreement_rate"` // rule vs ML
	Disagreements   int            `json:"disagreements"`
	NewDetections   int            `json:"new_detections"` // ML caught, rule missed
	FalseBlockRisk  int            `json:"false_block_risk"` // ML would block, rule allowed
	SeverityBreakdown map[string]int `json:"severity_breakdown"`
	Recommendation  string         `json:"recommendation"`
}

// Analyze generates a report from shadow scoring data.
func (sa *ShadowAnalyzer) Analyze(ctx context.Context, since time.Time) (*AnalysisReport, error) {
	entries, err := sa.repo.GetSince(ctx, since)
	if err != nil {
		return nil, err
	}

	report := &AnalysisReport{
		SeverityBreakdown: make(map[string]int),
	}
	report.TotalEvents = len(entries)

	for _, e := range entries {
		if e.MLDecision != "allow" {
			report.MLAnomalies++
		}
		if e.RuleDecision == "block" {
			report.RuleBlocks++
		}
		if !e.Agree {
			report.Disagreements++
			if e.MLDecision == "block" && e.RuleDecision == "allow" {
				report.FalseBlockRisk++ // potential FP if promoted
			}
			if e.MLDecision != "allow" && e.RuleDecision == "allow" {
				report.NewDetections++ // potential TP that rules missed
			}
		}
		report.SeverityBreakdown[e.Severity]++
	}

	if report.TotalEvents > 0 {
		report.MLAnomalyRate = float64(report.MLAnomalies) / float64(report.TotalEvents)
		report.RuleBlockRate = float64(report.RuleBlocks) / float64(report.TotalEvents)
		report.AgreementRate = 1 - float64(report.Disagreements)/float64(report.TotalEvents)
	}

	// Generate recommendation.
	switch {
	case report.FalseBlockRisk > report.TotalEvents*0.01:
		report.Recommendation = "HOLD: >1% of events would be falsely blocked. Retune threshold."
	case report.AgreementRate > 0.95 && report.NewDetections > 10:
		report.Recommendation = "PROMOTE: high agreement with rules, catching new threats. Safe to enforce."
	case report.NewDetections > 50 && report.FalseBlockRisk < report.TotalEvents*0.005:
		report.Recommendation = "PROMOTE with caution: strong new detection rate, low false block risk."
	default:
		report.Recommendation = "CONTINUE: gather more shadow data before promotion."
	}

	return report, nil
}
```

### 6.3 Human-in-the-Loop Review

```go
// ReviewQueue manages analyst review of high-severity ML alerts.
// Analysts confirm (TP) or dismiss (FP), creating labeled data for retraining.
type ReviewQueue struct {
	repo    ReviewRepository
	pub     AlertPublisher
}

// ReviewItem represents a pending analyst review.
type ReviewItem struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	TenantID      string    `json:"tenant_id"`
	UserID        string    `json:"user_id"`
	IP            string    `json:"ip"`
	RiskScore     float64   `json:"risk_score"`
	Severity      string    `json:"severity"`
	Features      map[string]float64 `json:"features"`
	RuleSignals   []string  `json:"rule_signals"`
	ActionTaken   string    `json:"action_taken"` // allow | challenge | block
	Status        string    `json:"status"`       // pending | confirmed | dismissed
	ReviewedBy    string    `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	Note          string    `json:"note,omitempty"`
}

// SubmitForReview creates a review item for a high-severity alert.
func (rq *ReviewQueue) SubmitForReview(ctx context.Context, item ReviewItem) error {
	item.Status = "pending"
	item.ID = uuid.New().String()
	if err := rq.repo.Create(ctx, item); err != nil {
		return err
	}

	// Publish to review channel for SOC analysts.
	return rq.pub.Publish(ctx, Alert{
		Type:     "ml_review_required",
		Severity: item.Severity,
		Payload:  item,
	})
}

// Resolve marks a review item as confirmed (true positive) or dismissed (false positive).
// Confirmed items become labeled training data; dismissed items become negative labels.
func (rq *ReviewQueue) Resolve(
	ctx context.Context, itemID, reviewerID string,
	confirmed bool, note string,
) error {
	status := "confirmed"
	if !confirmed {
		status = "dismissed"
	}
	now := time.Now()
	return rq.repo.Update(ctx, itemID, ReviewUpdate{
		Status:     status,
		ReviewedBy: reviewerID,
		ReviewedAt: &now,
		Note:       note,
	})
}
```

### 6.4 Precision-Recall Tradeoff Tuning

```go
// ThresholdManager dynamically adjusts the anomaly threshold based on
// observed false positive rate. If FP rate exceeds target, threshold
// is raised (more conservative). If FP rate is below target and recall
// is low, threshold is lowered (more sensitive).
type ThresholdManager struct {
	mu          sync.Mutex
	current     float64
	targetFP    float64 // target false positive rate (e.g., 0.01)
	targetRecall float64 // target recall (e.g., 0.90)
	adjustRate  float64 // adjustment step size (e.g., 0.01)
	minThresh   float64
	maxThresh   float64
	windowSize  int     // events to consider for adjustment
	recentDecisions []bool // true = anomaly, false = normal (with FP labels)
}

// NewThresholdManager creates a manager with defaults.
func NewThresholdManager() *ThresholdManager {
	return &ThresholdManager{
		current:     0.65,
		targetFP:    0.01,
		targetRecall: 0.90,
		adjustRate:  0.01,
		minThresh:   0.30,
		maxThresh:   0.95,
		windowSize:  1000,
		recentDecisions: make([]bool, 0, 1000),
	}
}

// RecordFeedback records an analyst's label for a scored event.
// This drives threshold adjustment.
func (tm *ThresholdManager) RecordFeedback(wasAnomaly, isTruePositive bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Only track false positives (model said anomaly, analyst says normal).
	if wasAnomaly && !isTruePositive {
		tm.recentDecisions = append(tm.recentDecisions, false)
	} else if wasAnomaly && isTruePositive {
		tm.recentDecisions = append(tm.recentDecisions, true)
	}

	// Trim to window size.
	if len(tm.recentDecisions) > tm.windowSize {
		tm.recentDecisions = tm.recentDecisions[1:]
	}

	// Adjust if we have enough data.
	if len(tm.recentDecisions) >= 100 {
		tm.adjust()
	}
}

func (tm *ThresholdManager) adjust() {
	tpCount := 0
	fpCount := 0
	for _, tp := range tm.recentDecisions {
		if tp {
			tpCount++
		} else {
			fpCount++
		}
	}

	total := tpCount + fpCount
	if total == 0 {
		return
	}

	fpRate := float64(fpCount) / float64(total)
	recall := float64(tpCount) / float64(total) // simplified

	if fpRate > tm.targetFP {
		// Too many false positives — raise threshold (more conservative).
		tm.current += tm.adjustRate
		if tm.current > tm.maxThresh {
			tm.current = tm.maxThresh
		}
	} else if recall < tm.targetRecall && fpRate < tm.targetFP*0.5 {
		// Room to be more sensitive — lower threshold.
		tm.current -= tm.adjustRate
		if tm.current < tm.minThresh {
			tm.current = tm.minThresh
		}
	}
}

// Current returns the current anomaly threshold.
func (tm *ThresholdManager) Current() float64 {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.current
}
```

### 6.5 A/B Testing Model Versions

```go
// ABTestManager routes a percentage of traffic to a candidate model
// for comparison against the production model.
type ABTestManager struct {
	mu            sync.RWMutex
	production    RiskScorer
	candidate     RiskScorer
	trafficPct    int // percentage of traffic to route to candidate (0-100)
	logger        ABTestLogger
}

// ScoreWithAB scores using production model, and for a percentage of
// requests, also scores with the candidate model for comparison.
func (ab *ABTestManager) ScoreWithAB(
	ctx context.Context, features []float32, requestID string,
) (RiskScore, error) {
	// Always score with production.
	prodScore, err := ab.production.Score(features)
	if err != nil {
		return RiskScore{}, err
	}

	// Route a percentage to candidate for comparison.
	ab.mu.RLock()
	pct := ab.trafficPct
	candidate := ab.candidate
	ab.mu.RUnlock()

	if candidate != nil && pct > 0 && hashToInt(requestID)%100 < pct {
		go func() {
			candScore, err := candidate.Score(features)
			if err != nil {
				return
			}
			ab.logger.LogABResult(ABResult{
				RequestID:     requestID,
				ProdScore:     prodScore.Value,
				CandScore:     candScore.Value,
				ProdSeverity:  prodScore.Severity,
				CandSeverity:  candScore.Severity,
				ModelVersionP: prodScore.ModelVer,
				ModelVersionC: candScore.ModelVer,
			})
		}()
	}

	return prodScore, nil
}

// SetCandidate installs a new candidate model for A/B testing.
func (ab *ABTestManager) SetCandidate(scorer RiskScorer, trafficPct int) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.candidate = scorer
	ab.trafficPct = trafficPct
}

// PromoteCandidate promotes the candidate to production.
func (ab *ABTestManager) PromoteCandidate() {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.production = ab.candidate
	ab.candidate = nil
	ab.trafficPct = 0
}

func hashToInt(s string) int {
	h := uint32(2166136261)
	for _, c := range s {
		h ^= uint32(c)
		h *= 16777619
	}
	return int(h % 10000)
}
```

---

## 7. Model Lifecycle for IAM

### 7.1 End-to-End MLOps Pipeline

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      ML Model Lifecycle (MLOps)                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │ 1. DATA COLLECTION (Continuous)                                  │   │
│  │    Audit Events (NATS) → Feature Store → Offline Store (Parquet) │   │
│  │    Analyst labels → Labeled dataset                              │   │
│  └───────────────────────┬─────────────────────────────────────────┘   │
│                          │                                              │
│  ┌───────────────────────▼─────────────────────────────────────────┐   │
│  │ 2. TRAINING (Python, weekly cron or drift-triggered)             │   │
│  │    sklearn Isolation Forest / XGBoost                            │   │
│  │    30-90 day training window                                     │   │
│  │    SMOTE for class imbalance                                     │   │
│  │    → Model artifact (.onnx)                                      │   │
│  └───────────────────────┬─────────────────────────────────────────┘   │
│                          │                                              │
│  ┌───────────────────────▼─────────────────────────────────────────┐   │
│  │ 3. VALIDATION (Backtesting)                                      │   │
│  │    Score historical events with new model                        │   │
│  │    Measure precision/recall/F1 against labeled data              │   │
│  │    Compare to current model (must improve by >5%)                │   │
│  │    → Pass/fail decision                                          │   │
│  └───────────────────────┬─────────────────────────────────────────┘   │
│                          │ Pass                                         │
│  ┌───────────────────────▼─────────────────────────────────────────┐   │
│  │ 4. SHADOW DEPLOYMENT (1-2 weeks)                                 │   │
│  │    Score all events with new model                               │   │
│  │    Log scores, don't act on them                                 │   │
│  │    Analysts review discrepancies                                 │   │
│  │    → FP rate <1%, recall >80% required                          │   │
│  └───────────────────────┬─────────────────────────────────────────┘   │
│                          │ Pass                                         │
│  ┌───────────────────────▼─────────────────────────────────────────┐   │
│  │ 5. GRADUAL ROLLOUT (Canary)                                      │   │
│  │    5% traffic → 25% → 50% → 100%                                 │   │
│  │    Monitor FP rate, latency, alert volume                        │   │
│  │    Auto-rollback if metrics degrade                               │   │
│  └───────────────────────┬─────────────────────────────────────────┘   │
│                          │                                              │
│  ┌───────────────────────▼─────────────────────────────────────────┐   │
│  │ 6. PRODUCTION (Active)                                           │   │
│  │    Inline scoring at auth time                                   │   │
│  │    Real-time feature store (Redis)                               │   │
│  │    ONNX inference in Go (~2-5ms)                                 │   │
│  └───────────────────────┬─────────────────────────────────────────┘   │
│                          │                                              │
│  ┌───────────────────────▼─────────────────────────────────────────┐   │
│  │ 7. MONITORING (Continuous)                                       │   │
│  │    Feature drift detection (PSI / KS test)                       │   │
│  │    Alert rate monitoring (spike = model degradation or attack)   │   │
│  │    Latency monitoring (P99 < 5ms)                                │   │
│  │    Throughput monitoring                                         │   │
│  └───────────────────────┬─────────────────────────────────────────┘   │
│                          │ Drift detected?                               │
│                     ┌────┴────┐                                         │
│                     │  YES    │──► Trigger retraining (back to Step 2)  │
│                     └─────────┘                                         │
│                     ┌────┴────┐                                         │
│                     │  NO     │──► Scheduled retraining (weekly)        │
│                     └─────────┘                                         │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 7.2 Training Pipeline (Python)

```python
#!/usr/bin/env python3
"""
train_model.py — Full training pipeline for GGID IAM threat detection.

Produces: iam_model_v{version}.onnx
Schedules: Weekly cron OR triggered by drift detection alert.
"""

import json
import sys
import pandas as pd
import numpy as np
from datetime import datetime, timedelta
from sklearn.ensemble import IsolationForest
from sklearn.model_selection import train_test_split
from sklearn.metrics import classification_report, precision_recall_fscore_support
from skl2onnx import to_onnx
from skl2onnx.common.data_types import FloatTensorType

def load_training_data(parquet_path: str, days: int = 30):
    """Load features from offline store, last N days."""
    df = pd.read_parquet(parquet_path)
    cutoff = datetime.utcnow() - timedelta(days=days)
    df = df[df['timestamp'] >= cutoff]
    return df

def prepare_features(df: pd.DataFrame):
    """Select and normalize features for model input."""
    feature_cols = [
        "hour_of_day", "day_of_week", "time_since_last_login_s",
        "login_count_24h", "failed_attempts_1h", "ip_velocity_1h",
        "device_count", "unique_ips_24h", "asn_diversity_24h",
        "geo_velocity_kmh", "is_new_device", "is_new_ip",
        "is_datacenter_asn", "ip_reputation_score",
    ]
    X = df[feature_cols].values.astype(np.float32)

    # Normalize to [0, 1] per feature.
    for i in range(X.shape[1]):
        col_min, col_max = X[:, i].min(), X[:, i].max()
        if col_max > col_min:
            X[:, i] = (X[:, i] - col_min) / (col_max - col_min)

    return X, feature_cols

def train_isolation_forest(X: np.ndarray, contamination: float = 0.05):
    """Train Isolation Forest model."""
    model = IsolationForest(
        n_estimators=100,
        contamination=contamination,
        max_samples='auto',
        random_state=42,
        n_jobs=-1,
    )
    model.fit(X)
    return model

def backtest(model, X_test, y_test=None):
    """Backtest model on held-out data."""
    predictions = model.predict(X_test)  # 1 = normal, -1 = anomaly
    anomaly_scores = model.score_samples(X_test)

    # If we have labels, compute metrics.
    if y_test is not None:
        # Convert: sklearn uses 1=normal, -1=anomaly
        # Labels: 0=normal, 1=anomaly
        y_pred = (predictions == -1).astype(int)
        precision, recall, f1, _ = precision_recall_fscore_support(
            y_test, y_pred, average='binary', zero_division=0
        )
        print(f"Precision: {precision:.3f}, Recall: {recall:.3f}, F1: {f1:.3f}")
        print(classification_report(y_test, y_pred, zero_division=0))
        return precision, recall, f1

    # No labels — just report anomaly rate.
    anomaly_rate = (predictions == -1).mean()
    print(f"Anomaly rate: {anomaly_rate:.3f}")
    return None, None, anomaly_rate

def export_onnx(model, feature_names: list, output_path: str):
    """Export trained model to ONNX format."""
    initial_type = [("input", FloatTensorType([None, len(feature_names)]))]
    onnx_model = to_onnx(model, initial_types=initial_type, target_opset=15)

    with open(output_path, "wb") as f:
        f.write(onnx_model.SerializeToString())

    # Save metadata alongside model.
    metadata = {
        "version": datetime.utcnow().strftime("%Y%m%d_%H%M%S"),
        "feature_names": feature_names,
        "model_type": "IsolationForest",
        "n_estimators": model.n_estimators,
        "contamination": model.contamination,
        "exported_at": datetime.utcnow().isoformat(),
    }
    meta_path = output_path.replace(".onnx", "_meta.json")
    with open(meta_path, "w") as f:
        json.dump(metadata, f, indent=2)

    print(f"Exported model to {output_path}")
    print(f"Metadata to {meta_path}")

def main():
    parquet_path = sys.argv[1] if len(sys.argv) > 1 else "data/features.parquet"
    output_path = sys.argv[2] if len(sys.argv) > 2 else "models/iam_model_latest.onnx"

    # 1. Load data.
    print("Loading training data...")
    df = load_training_data(parquet_path, days=30)
    print(f"  {len(df)} events loaded")

    # 2. Prepare features.
    X, feature_names = prepare_features(df)
    print(f"  {X.shape[1]} features prepared")

    # 3. Split for backtesting.
    X_train, X_test = train_test_split(X, test_size=0.2, random_state=42)

    # 4. Train.
    print("Training Isolation Forest...")
    model = train_isolation_forest(X_train, contamination=0.05)

    # 5. Backtest.
    print("Backtesting...")
    backtest(model, X_test)

    # 6. Export.
    version = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
    versioned_output = output_path.replace("latest", f"v{version}")
    export_onnx(model, feature_names, versioned_output)

    # Also save as "latest" for hot-swap.
    export_onnx(model, feature_names, output_path)

if __name__ == "__main__":
    main()
```

### 7.3 Drift Detection in Go

```go
// DriftDetector monitors feature distributions for concept drift.
// If the live feature distribution diverges significantly from the
// training distribution, the model should be retrained.
type DriftDetector struct {
	mu          sync.Mutex
	baselines   map[string]*DistributionStats // per-feature training stats
	recent      map[string][]float64          // recent feature values
	maxRecent   int
	psiThreshold float64 // Population Stability Index threshold
}

// DistributionStats holds statistics for a feature dimension.
type DistributionStats struct {
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"stddev"`
	P10    float64 `json:"p10"`  // 10th percentile
	P50    float64 `json:"p50"`  // median
	P90    float64 `json:"p90"`  // 90th percentile
	Bins   []float64 `json:"bins"` // histogram bins for PSI
}

// NewDriftDetector creates a detector from training-time statistics.
func NewDriftDetector(baselines map[string]*DistributionStats) *DriftDetector {
	return &DriftDetector{
		baselines:    baselines,
		recent:       make(map[string][]float64),
		maxRecent:    1000,
		psiThreshold: 0.2, // PSI > 0.2 indicates significant drift
	}
}

// RecordFeature observes a feature value for drift monitoring.
func (dd *DriftDetector) RecordFeature(name string, value float64) {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	dd.recent[name] = append(dd.recent[name], value)
	if len(dd.recent[name]) > dd.maxRecent {
		dd.recent[name] = dd.recent[name][1:]
	}
}

// CheckDrift computes PSI for all features and returns drifted ones.
func (dd *DriftDetector) CheckDrift() []DriftResult {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	var results []DriftResult
	for name, baseline := range dd.baselines {
		recent, ok := dd.recent[name]
		if !ok || len(recent) < 100 {
			continue
		}

		psi := dd.computePSI(baseline, recent)
		results = append(results, DriftResult{
			Feature:   name,
			PSI:       psi,
			Drifted:   psi > dd.psiThreshold,
			SampleSize: len(recent),
		})
	}

	return results
}

// DriftResult holds drift detection results for one feature.
type DriftResult struct {
	Feature    string  `json:"feature"`
	PSI        float64 `json:"psi"`
	Drifted    bool    `json:"drifted"`
	SampleSize int     `json:"sample_size"`
}

// computePSI calculates Population Stability Index between baseline and recent.
// PSI < 0.1: no drift. 0.1-0.2: minor drift. > 0.2: significant drift.
func (dd *DriftDetector) computePSI(baseline *DistributionStats, recent []float64) float64 {
	numBins := 10

	// Create bins from baseline percentiles.
	minVal := baseline.P10 * 0.5 // extend below P10
	maxVal := baseline.P90 * 1.5 // extend above P90
	if maxVal <= minVal {
		return 0
	}

	binWidth := (maxVal - minVal) / float64(numBins)

	// Count baseline frequencies per bin.
	baselineCounts := make([]int, numBins+1) // +1 for overflow
	for _, b := range baseline.Bins {
		idx := int((b - minVal) / binWidth)
		if idx >= 0 && idx < numBins {
			baselineCounts[idx]++
		} else if b >= maxVal {
			baselineCounts[numBins]++
		}
	}

	// Count recent frequencies per bin.
	recentCounts := make([]int, numBins+1)
	for _, v := range recent {
		idx := int((v - minVal) / binWidth)
		if idx >= 0 && idx < numBins {
			recentCounts[idx]++
		} else if v >= maxVal {
			recentCounts[numBins]++
		}
	}

	// Compute PSI.
	totalBaseline := len(baseline.Bins)
	totalRecent := len(recent)
	if totalBaseline == 0 || totalRecent == 0 {
		return 0
	}

	var psi float64
	for i := 0; i <= numBins; i++ {
		pBaseline := float64(baselineCounts[i]) / float64(totalBaseline)
		pRecent := float64(recentCounts[i]) / float64(totalRecent)

		// Avoid log(0).
		if pBaseline < 1e-6 {
			pBaseline = 1e-6
		}
		if pRecent < 1e-6 {
			pRecent = 1e-6
		}

		psi += (pRecent - pBaseline) * mathLog(pRecent/pBaseline)
	}

	return psi
}
```

---

## 8. Integration with GGID Security Infrastructure

### 8.1 Integration Points

GGID's existing security stack provides multiple integration points for ML scoring:

```
┌──────────────────────────────────────────────────────────────────────┐
│                  ML Integration Points in GGID                       │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Request ──► Gateway                                                 │
│              ├── Rate Limiter ──┐                                    │
│              │   (rules)        ├──► ML Risk Score (enrichment)      │
│              ├── Bot Detect ────┘                                    │
│              │                                                      │
│              ├──► Auth Service                                      │
│              │    ├── AssessLoginRisk() ──► ML Score (enrichment)   │
│              │    ├── AssessLoginAnomaly() ──► ML Score (enrichment)│
│              │    ├── Step-up MFA ──► ML Score (trigger)             │
│              │    └── JWT generation ──► Risk level in JWT claim     │
│              │                                                      │
│              ├──► Audit Service                                     │
│              │    └── Event logging ──► ML score in Metadata         │
│              │                                                      │
│              └──► Response headers                                  │
│                   └── X-Risk-Score, X-Risk-Level                    │
│                                                                      │
│  Async (NATS):                                                       │
│    audit.events ──► Batch Analyzer ──► ml.alerts.{tenant}           │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 8.2 Risk-Aware Auth Middleware

The ML risk score enriches the existing risk assessment in `risk_auth.go`.
Rather than replacing the rule-based `AssessLoginRisk`, ML scoring adds a
supplementary signal:

```go
// RiskAwareAuthMiddleware integrates ML scoring into the authentication flow.
// It runs AFTER existing rule-based checks and enriches the risk assessment
// with ML-derived signals. ML never overrides rules — it only adds risk.
type RiskAwareAuthMiddleware struct {
	pipeline     *FeaturePipeline
	scorer       RiskScorer
	shadow       *ShadowScorer    // for validation phase
	store        *OnlineFeatureStore
	mode         ScoringMode      // shadow | advisory | enforcement
}

// ScoringMode controls how ML scores are used.
type ScoringMode int

const (
	// ModeShadow: score but never act (validation phase).
	ModeShadow ScoringMode = iota
	// ModeAdvisory: score and log, but only advise (not enforce).
	ModeAdvisory
	// ModeEnforcement: score and use for MFA/blocking decisions.
	ModeEnforcement
)

// AssessWithML combines rule-based risk assessment with ML scoring.
// This wraps the existing AssessLoginRisk function.
func (m *RiskAwareAuthMiddleware) AssessWithML(
	ctx context.Context,
	tenantID, userID uuid.UUID, ip, userAgent, deviceFP string,
) (*CombinedRiskAssessment, error) {

	// 1. Run existing rule-based assessment (from risk_auth.go).
	ruleAssessment := m.runRuleBasedAssessment(ctx, tenantID, userID, ip, userAgent)

	// 2. Compute ML features.
	req := ScoreRequest{
		UserID:            userID.String(),
		IP:                ip,
		UserAgent:         userAgent,
		DeviceFingerprint: deviceFP,
		Timestamp:         time.Now(),
	}

	features, err := m.pipeline.ComputeFeatures(ctx, req)
	if err != nil {
		// Feature computation failed — fall back to rules only.
		return &CombinedRiskAssessment{
			RuleAssessment: ruleAssessment,
			MLAvailable:    false,
		}, nil
	}

	// 3. Score with ML model.
	mlScore, err := m.scorer.Score(features)
	if err != nil {
		return &CombinedRiskAssessment{
			RuleAssessment: ruleAssessment,
			MLAvailable:    false,
		}, nil
	}

	// 4. Record features for online store (after scoring).
	_ = m.store.RecordLoginEvent(ctx, LoginEvent{
		UserID:            userID.String(),
		IP:                ip,
		DeviceFingerprint: deviceFP,
		Timestamp:         time.Now(),
	})

	// 5. Shadow mode: log but don't act.
	if m.mode == ModeShadow {
		_ = m.shadow.Evaluate(ctx, req, ruleAssessment.Level.String())
	}

	// 6. Combine assessments.
	combined := m.combine(ruleAssessment, mlScore)

	return combined, nil
}

// CombinedRiskAssessment merges rule-based and ML-based assessments.
type CombinedRiskAssessment struct {
	RuleAssessment  *RiskAssessment `json:"rule_assessment"`
	MLScore         RiskScore       `json:"ml_score"`
	MLAvailable     bool            `json:"ml_available"`
	CombinedLevel   RiskLevel       `json:"combined_level"`
	CombinedScore   int             `json:"combined_score"` // 0-100
	RequiresStepUp  bool            `json:"requires_step_up"`
	RequiresBlock   bool            `json:"requires_block"`
	DecisionFactors []string        `json:"decision_factors"`
}

// combine merges rule-based and ML assessments.
// Rules are the floor — ML can only increase risk, never decrease it.
func (m *RiskAwareAuthMiddleware) combine(
	rule *RiskAssessment, ml RiskScore,
) *CombinedRiskAssessment {
	result := &CombinedRiskAssessment{
		RuleAssessment: rule,
		MLScore:        ml,
		MLAvailable:    true,
	}

	// Start with rule-based score.
	result.CombinedScore = rule.Score

	// ML adds risk on top of rules (never reduces).
	mlContribution := int(ml.Value * 30) // ML can add up to 30 points
	result.CombinedScore += mlContribution

	if result.CombinedScore > 100 {
		result.CombinedScore = 100
	}

	// Determine combined level.
	switch {
	case result.CombinedScore >= 80:
		result.CombinedLevel = RiskLevelHigh
		result.RequiresBlock = m.mode == ModeEnforcement
	case result.CombinedScore >= 50:
		result.CombinedLevel = RiskLevelMedium
		result.RequiresStepUp = true
	default:
		result.CombinedLevel = RiskLevelLow
	}

	// Build decision factors for explainability.
	result.DecisionFactors = append(result.DecisionFactors, rule.Reasons...)
	if ml.IsAnomaly {
		result.DecisionFactors = append(result.DecisionFactors,
			fmt.Sprintf("ML anomaly score: %.2f (%s)", ml.Value, ml.Severity))
	}

	return result
}
```

### 8.3 Risk Score as gRPC Metadata

```go
// RiskMetadata provides risk score propagation via gRPC metadata.
// This allows the gateway to pass risk information to downstream services.
package riskmeta

import (
	"context"
	"strconv"

	"google.golang.org/grpc/metadata"
)

const (
	// Metadata keys for risk propagation.
	MDRiskScore  = "x-risk-score"
	MDRiskLevel  = "x-risk-level"
	MDMLScore    = "x-ml-score"
	MDMLVersion  = "x-ml-version"
	MDMLSeverity = "x-ml-severity"
	MDAnomaly    = "x-anomaly-detected"
)

// InjectRiskMetadata adds risk assessment to outgoing gRPC context.
func InjectRiskMetadata(ctx context.Context, assessment *CombinedRiskAssessment) context.Context {
	pairs := []string{
		MDRiskScore, strconv.Itoa(assessment.CombinedScore),
		MDRiskLevel, string(assessment.CombinedLevel),
	}
	if assessment.MLAvailable {
		pairs = append(pairs,
			MDMLScore, strconv.FormatFloat(assessment.MLScore.Value, 'f', 4, 64),
			MDMLVersion, assessment.MLScore.ModelVer,
			MDMLSeverity, assessment.MLScore.Severity,
		)
		if assessment.MLScore.IsAnomaly {
			pairs = append(pairs, MDAnomaly, "true")
		}
	}
	return metadata.AppendToOutgoingContext(ctx, pairs...)
}

// ExtractRiskMetadata reads risk assessment from incoming gRPC context.
func ExtractRiskMetadata(ctx context.Context) (RiskMeta, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return RiskMeta{}, false
	}

	meta := RiskMeta{}
	if vals := md.Get(MDRiskScore); len(vals) > 0 {
		meta.RuleScore, _ = strconv.Atoi(vals[0])
	}
	if vals := md.Get(MDRiskLevel); len(vals) > 0 {
		meta.RuleLevel = vals[0]
	}
	if vals := md.Get(MDMLScore); len(vals) > 0 {
		meta.MLScore, _ = strconv.ParseFloat(vals[0], 64)
		meta.MLAvailable = true
	}
	if vals := md.Get(MDMLSeverity); len(vals) > 0 {
		meta.MLSeverity = vals[0]
	}
	if vals := md.Get(MDAnomaly); len(vals) > 0 {
		meta.AnomalyDetected = vals[0] == "true"
	}
	return meta, true
}

// RiskMeta holds extracted risk metadata.
type RiskMeta struct {
	RuleScore        int     `json:"rule_score"`
	RuleLevel        string  `json:"rule_level"`
	MLScore          float64 `json:"ml_score"`
	MLSeverity       string  `json:"ml_severity"`
	MLAvailable      bool    `json:"ml_available"`
	AnomalyDetected  bool    `json:"anomaly_detected"`
}
```

### 8.4 Adaptive MFA Triggering

The ML score determines whether MFA is required. Low-risk logins skip MFA
(frictionless auth). High-risk logins require step-up:

```go
// AdaptiveMFA determines MFA requirements based on combined risk.
// This replaces the static "always require MFA" or "never require MFA"
// with a risk-based decision.
type AdaptiveMFA struct {
	config MFAConfig
}

type MFAConfig struct {
	// MFA is required when risk score exceeds this threshold.
	RequireMFAThreshold int

	// MFA type to use for step-up.
	StepUpMFAType string // totp | webauthn | sms

	// If true, low-risk logins skip MFA entirely (passwordless experience).
	AllowPasswordlessLowRisk bool

	// Maximum number of MFA challenges per session per day.
	MaxChallengesPerDay int
}

func DefaultMFAConfig() MFAConfig {
	return MFAConfig{
		RequireMFAThreshold:       40,
		StepUpMFAType:             "totp",
		AllowPasswordlessLowRisk:  false, // conservative: require MFA for all initially
		MaxChallengesPerDay:       5,
	}
}

// MFARequirement represents the MFA decision for a login attempt.
type MFARequirement struct {
	Required     bool   `json:"required"`
	Type         string `json:"type"` // none | totp | webauthn | sms
	Reason       string `json:"reason"`
	ChallengesUsed int  `json:"challenges_used_today"`
}

// Evaluate determines if MFA is needed for this login attempt.
func (amfa *AdaptiveMFA) Evaluate(
	assessment *CombinedRiskAssessment, challengesUsedToday int,
) MFARequirement {
	req := MFARequirement{
		Required:       true, // default: require MFA
		Type:           amfa.config.StepUpMFAType,
		ChallengesUsed: challengesUsedToday,
	}

	// Cap: if user has been challenged too many times today, allow without MFA
	// (assume they've proven identity already). This prevents MFA fatigue.
	if challengesUsedToday >= amfa.config.MaxChallengesPerDay {
		req.Required = false
		req.Type = "none"
		req.Reason = "MFA challenge cap reached for today"
		return req
	}

	// Risk-based decision.
	if assessment.CombinedScore >= 80 {
		// Critical risk: require WebAuthn (strongest factor).
		req.Required = true
		req.Type = "webauthn"
		req.Reason = fmt.Sprintf("Critical risk (score %d): hardware key required", assessment.CombinedScore)
		return req
	}

	if assessment.CombinedScore >= amfa.config.RequireMFAThreshold {
		// Elevated risk: require TOTP.
		req.Required = true
		req.Type = "totp"
		req.Reason = fmt.Sprintf("Elevated risk (score %d): step-up authentication required", assessment.CombinedScore)
		return req
	}

	// Low risk: skip MFA if configured.
	if amfa.config.AllowPasswordlessLowRisk && assessment.CombinedScore < 20 {
		req.Required = false
		req.Type = "none"
		req.Reason = "Low risk: frictionless authentication"
		return req
	}

	// Default: require MFA.
	req.Reason = "Standard MFA policy"
	return req
}
```

### 8.5 Audit Integration

Every ML score must be logged for compliance and model monitoring:

```go
// AuditMLDecision logs an ML scoring decision as an audit event.
// This creates a complete trail of all ML-based decisions for compliance.
func AuditMLDecision(publisher *audit.Publisher, assessment *CombinedRiskAssessment,
	tenantID, userID uuid.UUID, ip, action string) {

	event := audit.Event{
		TenantID:     tenantID,
		ActorType:    "system",
		ActorID:      userID,
		Action:       "ml.score",
		ResourceType: "auth_session",
		Result:       "evaluated",
		IPAddress:    ip,
		Metadata: map[string]any{
			"rule_score":      assessment.RuleAssessment.Score,
			"rule_level":      assessment.RuleAssessment.Level,
			"ml_available":    assessment.MLAvailable,
			"ml_score":        assessment.MLScore.Value,
			"ml_severity":     assessment.MLScore.Severity,
			"ml_model":        assessment.MLScore.ModelVer,
			"ml_latency_ms":   assessment.MLScore.Latency,
			"combined_score":  assessment.CombinedScore,
			"requires_mfa":    assessment.RequiresStepUp,
			"requires_block":  assessment.RequiresBlock,
			"decision_factors": assessment.DecisionFactors,
			"trigger_action":  action,
		},
		CreatedAt: time.Now(),
	}

	publisher.PublishAsync(event)
}
```

---

## 9. Anomaly Detection Specifics for IAM

### 9.1 Login Anomalies

| Anomaly Type | Feature Vector | Detection Method | Response |
|---|---|---|---|
| **Impossible travel** | `[geo_velocity_kmh > 900, time_since_last_login < 4h, geo_distance > 3000km]` | Rule + ML | Step-up MFA |
| **New device** | `[is_new_device=1, device_count > 3, login_count_24h normal]` | Rule + ML | Notify user |
| **New ASN** | `[is_new_asn=1, is_datacenter_asn=1, asn_diversity > 5]` | ML | Step-up MFA |
| **Off-hours login** | `[hour_of_day in [2-5], login_count_24h > baseline+3σ]` | ML (per-user baseline) | Alert SOC |
| **New geo country** | `[geo_country not in user_baseline, login_count_24h normal]` | Rule + ML | Step-up MFA |

### 9.2 Feature Vectors for Login Anomalies

```go
// LoginAnomalyFeatures constructs feature vectors specific to login anomaly detection.
// These vectors are designed for the Isolation Forest model.
type LoginAnomalyFeatures struct{}

// ImpossibleTravelVector: detects logins from geographically impossible locations.
// Example: login from New York at 10:00, then from Tokyo at 10:30.
func (laf LoginAnomalyFeatures) ImpossibleTravelVector(
	currentLat, currentLon float64,
	lastLat, lastLon float64,
	timeSinceLastLogin time.Duration,
) []float32 {
	distance := haversine(currentLat, currentLon, lastLat, lastLon)
	timeHours := timeSinceLastLogin.Hours()
	velocity := 0.0
	if timeHours > 0 {
		velocity = distance / timeHours
	}

	return []float32{
		float32(distance / 20000),              // normalized distance (0-1)
		float32(timeSinceLastLogin.Seconds()),   // time delta
		float32(math.Min(velocity, 2000) / 2000), // normalized velocity
		float32(boolFloat(velocity > 900)),       // impossible_travel flag
		float32(boolFloat(distance > 3000)),      // long_distance flag
		float32(boolFloat(timeHours < 4)),        // rapid succession
	}
}

// CredentialStuffingVector: detects distributed credential stuffing.
// Example: 1000 logins from 200 IPs across 100 users in 1 hour, <2% success.
func (laf LoginAnomalyFeatures) CredentialStuffingVector(
	ipCount, userCount, totalAttempts, successes int,
	timeWindowHours float64,
) []float32 {
	successRate := 0.0
	if totalAttempts > 0 {
		successRate = float64(successes) / float64(totalAttempts)
	}

	attemptsPerIP := 0.0
	if ipCount > 0 {
		attemptsPerIP = float64(totalAttempts) / float64(ipCount)
	}

	return []float32{
		float32(ipCount),                        // IP diversity
		float32(userCount),                      // user count
		float32(totalAttempts),                  // total volume
		float32(successRate),                    // success rate (low = stuffing)
		float32(attemptsPerIP),                  // attempts per IP (low = distributed)
		float32(timeWindowHours),               // time concentration
		float32(boolFloat(successRate < 0.02)), // low success flag
		float32(boolFloat(ipCount > 50)),       // high diversity flag
	}
}

// BruteForceVector: detects targeted brute force against a single account.
func (laf LoginAnomalyFeatures) BruteForceVector(
	failedAttempts int,
	timeWindowHours float64,
	distinctIPs int,
	distinctUserAgents int,
) []float32 {
	attemptsPerHour := 0.0
	if timeWindowHours > 0 {
		attemptsPerHour = float64(failedAttempts) / timeWindowHours
	}

	return []float32{
		float32(failedAttempts),               // failure count
		float32(attemptsPerHour),              // rate
		float32(distinctIPs),                  // IP count (1 = single source)
		float32(distinctUserAgents),           // UA diversity
		float32(boolFloat(failedAttempts > 20)), // sustained attempt flag
		float32(boolFloat(distinctIPs == 1)),  // single source flag
	}
}

// AccountTakeoverVector: detects signs of successful account takeover.
func (laf LoginAnomalyFeatures) AccountTakeoverVector(
	isNewDevice bool,
	isNewIP bool,
	isNewASN bool,
	isNewGeo bool,
	failedBeforeSuccess int,
	settingsChangedRecently bool,
) []float32 {
	return []float32{
		float32(boolFloat(isNewDevice)),
		float32(boolFloat(isNewIP)),
		float32(boolFloat(isNewASN)),
		float32(boolFloat(isNewGeo)),
		float32(float64(failedBeforeSuccess)),
		float32(boolFloat(settingsChangedRecently)),
		float32(boolFloat(failedBeforeSuccess > 3)), // suspicious pattern
		float32(boolFloat(isNewDevice && isNewIP && isNewGeo)), // triple new
	}
}
```

### 9.3 API Anomalies

API-level anomalies are detected through request pattern analysis:

```go
// APIAnomalyDetector detects abnormal API usage patterns.
type APIAnomalyDetector struct {
	rdb redis.Cmdable
}

// DetectMassExport identifies bulk data export patterns.
// Example: user queries /api/v1/users with pagination every 2 seconds,
// downloading the entire user directory.
func (ad *APIAnomalyDetector) DetectMassExport(
	ctx context.Context, userID, endpoint string,
) (bool, error) {
	key := fmt.Sprintf("feat:apiread:%s:%s", userID, endpoint)
	count, err := ad.rdb.ZCard(ctx, key).Result()
	if err != nil {
		return false, err
	}
	// > 100 reads of same endpoint in 10 minutes = suspicious.
	return count > 100, nil
}

// DetectUnusualEndpointAccess identifies API endpoints the user hasn't used before.
func (ad *APIAnomalyDetector) DetectUnusualEndpointAccess(
	ctx context.Context, userID, endpoint string,
) (bool, error) {
	key := fmt.Sprintf("feat:endpoints:%s", userID)
	isMember, err := ad.rdb.SIsMember(ctx, key, endpoint).Result()
	if err != nil {
		return false, err
	}
	return !isMember, nil // true if endpoint is new for this user
}

// DetectMassTokenCreation identifies abnormal OAuth token creation patterns.
func (ad *APIAnomalyDetector) DetectMassTokenCreation(
	ctx context.Context, userID string,
) (bool, error) {
	key := fmt.Sprintf("feat:tokens:%s", userID)
	count, err := ad.rdb.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return false, err
	}
	// > 10 tokens created in 1 hour = suspicious.
	return count > 10, nil
}

// APIAnomalyVector constructs a feature vector for API-level anomaly detection.
func (ad *APIAnomalyDetector) APIAnomalyVector(
	ctx context.Context, userID, endpoint, method string,
) []float32 {
	readCount, _ := ad.rdb.ZCard(ctx, fmt.Sprintf("feat:apiread:%s:%s", userID, endpoint)).Result()
	tokenCount, _ := ad.rdb.Get(ctx, fmt.Sprintf("feat:tokens:%s", userID)).Int()
	isNewEndpoint, _ := ad.DetectUnusualEndpointAccess(ctx, userID, endpoint)

	return []float32{
		float32(readCount),              // read count for this endpoint
		float32(tokenCount),             // tokens created recently
		float32(boolFloat(isNewEndpoint)), // new endpoint for this user
		float32(boolFloat(method == "GET")),  // read vs write
		float32(boolFloat(readCount > 100)),  // mass read flag
		float32(boolFloat(tokenCount > 10)),  // mass token flag
	}
}
```

### 9.4 Admin Anomalies

```go
// AdminAnomalyDetector detects abnormal administrative actions.
type AdminAnomalyDetector struct {
	rdb redis.Cmdable
}

// AdminAnomalyVector detects bulk role changes and off-hours config changes.
func (ad *AdminAnomalyDetector) AdminAnomalyVector(
	ctx context.Context, adminID string, action string,
) []float32 {
	now := time.Now()

	// Count admin actions in the last hour.
	recentActions, _ := ad.rdb.ZCard(ctx,
		fmt.Sprintf("feat:adminactions:%s", adminID)).Result()

	// Check if action is during off-hours (10 PM - 6 AM).
	hour := now.UTC().Hour()
	isOffHours := hour >= 22 || hour <= 6

	// Check for bulk role changes.
	roleChanges, _ := ad.rdb.Get(ctx,
		fmt.Sprintf("feat:rolechanges:%s", adminID)).Int()

	// Check for config changes (sensitive operations).
	isConfigChange := action == "config.update" || action == "policy.update"

	return []float32{
		float32(recentActions),                // admin action rate
		float32(roleChanges),                  // role change count
		float32(boolFloat(isOffHours)),        // off-hours flag
		float32(boolFloat(isConfigChange)),    // config change flag
		float32(boolFloat(recentActions > 20)), // high activity flag
		float32(boolFloat(roleChanges > 5)),   // bulk role change flag
	}
}
```

---

## 10. Explainable AI for IAM

### 10.1 Why Explainability Matters

Security operations center (SOC) analysts receive alerts from the ML system. If
an alert says "anomaly score: 0.87" with no explanation, the analyst cannot:

1. **Triage effectively** — is this a credential stuffing attack or a user traveling?
2. **Make decisions** — should they block the account or just increase monitoring?
3. **Provide feedback** — if the alert is a false positive, what was wrong?
4. **Build trust** — without understanding, analysts will ignore ML alerts entirely.

**Regulatory requirement:** GDPR Article 22 grants individuals the right to
"meaningful information about the logic involved" in automated decision-making.
If ML contributes to account lockout or access denial, the decision must be
explainable.

### 10.2 SHAP Values for Feature Attribution

SHAP (SHapley Additive exPlanations) assigns each feature an importance value for
a specific prediction. For tree-based models (Isolation Forest, XGBoost),
TreeSHAP provides exact Shapley values in polynomial time.

```go
// Explainer provides explainability for ML risk scores.
// It wraps SHAP value computation (exported from Python) and
// produces human-readable explanations.
type Explainer struct {
	// shapValues are pre-computed for tree-based models.
	// In production, these are loaded from a SHAP explainer file
	// exported alongside the ONNX model.
	featureNames  []string
	expectedValue float64 // base value (average model output)
}

// Explanation represents an explainable risk assessment.
type Explanation struct {
	RiskScore      float64           `json:"risk_score"`
	Severity       string            `json:"severity"`
	BaseScore      float64           `json:"base_score"` // average risk for population
	TopFeatures    []FeatureAttribution `json:"top_features"`
	Summary        string            `json:"summary"`
	Recommendation string            `json:"recommendation"`
}

// FeatureAttribution shows how much a single feature contributed to the risk.
type FeatureAttribution struct {
	Feature      string  `json:"feature"`
	Value        float64 `json:"value"`         // feature value
	Contribution float64 `json:"contribution"`  // SHAP value (positive = increased risk)
	Description  string  `json:"description"`   // human-readable
}

// Explain produces an explanation for a risk score.
// Uses pre-computed SHAP values loaded from the model artifact.
func (ex *Explainer) Explain(score RiskScore, features []float32, shapValues []float64) Explanation {
	// Pair features with their SHAP values.
	attributions := make([]FeatureAttribution, len(ex.featureNames))
	for i, name := range ex.featureNames {
		attributions[i] = FeatureAttribution{
			Feature:      name,
			Value:        float64(features[i]),
			Contribution: shapValues[i],
			Description:  ex.describeFeature(name, float64(features[i])),
		}
	}

	// Sort by absolute contribution (most impactful first).
	sortByAbsContribution(attributions)

	// Take top 5 features.
	top := attributions
	if len(top) > 5 {
		top = top[:5]
	}

	// Generate summary.
	summary := ex.generateSummary(score, top)

	// Generate recommendation.
	rec := ex.generateRecommendation(score.Severity, top)

	return Explanation{
		RiskScore:      score.Value,
		Severity:       score.Severity,
		BaseScore:      ex.expectedValue,
		TopFeatures:    top,
		Summary:        summary,
		Recommendation: rec,
	}
}

// describeFeature converts a technical feature into a human-readable description.
func (ex *Explainer) describeFeature(name string, value float64) string {
	switch name {
	case "geo_velocity_kmh":
		if value > 900 {
			return fmt.Sprintf("Impossible travel: %.0f km/h (faster than flight)", value)
		}
		return fmt.Sprintf("Geographic velocity: %.0f km/h", value)
	case "is_new_device":
		if value == 1 {
			return "Login from a previously unseen device"
		}
		return "Login from a known device"
	case "is_new_ip":
		if value == 1 {
			return "Login from a new IP address"
		}
		return "Login from a known IP address"
	case "failed_attempts_1h":
		return fmt.Sprintf("%d failed login attempts in the last hour", int(value))
	case "ip_velocity_1h":
		return fmt.Sprintf("%d login attempts from this IP in the last hour", int(value))
	case "is_datacenter_asn":
		if value == 1 {
			return "Login from a datacenter/cloud provider IP (common for bots/proxies)"
		}
		return "Login from a residential/business IP"
	case "time_since_last_login_s":
		hours := value / 3600
		return fmt.Sprintf("%.1f hours since last login", hours)
	case "off_hours":
		if value == 1 {
			return "Login during off-hours (2 AM - 5 AM UTC)"
		}
		return "Login during normal hours"
	case "login_count_24h":
		return fmt.Sprintf("%d logins in the last 24 hours", int(value))
	case "asn_diversity_24h":
		return fmt.Sprintf("%d distinct network providers used in 24h", int(value))
	default:
		return fmt.Sprintf("%s = %.2f", name, value)
	}
}

func (ex *Explainer) generateSummary(score RiskScore, top []FeatureAttribution) string {
	positiveFactors := []string{}
	for _, f := range top {
		if f.Contribution > 0.01 {
			positiveFactors = append(positiveFactors, f.Description)
		}
	}

	if len(positiveFactors) == 0 {
		return fmt.Sprintf("Risk score %.2f (%s): no significant risk factors detected", score.Value, score.Severity)
	}

	return fmt.Sprintf("Risk score %.2f (%s): %s", score.Value, score.Severity,
		strings.Join(positiveFactors, "; "))
}

func (ex *Explainer) generateRecommendation(severity string, top []FeatureAttribution) string {
	switch severity {
	case "critical":
		return "BLOCK: Immediate threat detected. Block the request and alert SOC."
	case "high":
		return "CHALLENGE: Require step-up MFA (WebAuthn preferred). Alert SOC for review."
	case "medium":
		return "CHALLENGE: Require step-up MFA (TOTP). Log for monitoring."
	default:
		return "ALLOW: Low risk. Proceed with normal authentication."
	}
}

func sortByAbsContribution(attrs []FeatureAttribution) {
	// Sort descending by absolute contribution value.
	for i := 1; i < len(attrs); i++ {
		for j := i; j > 0 && math.Abs(attrs[j].Contribution) > math.Abs(attrs[j-1].Contribution); j-- {
			attrs[j], attrs[j-1] = attrs[j-1], attrs[j]
		}
	}
}
```

### 10.3 Decision Path Logging

For tree-based models, the decision path (which tree nodes were traversed) provides
additional explainability:

```go
// DecisionPath logs the model's internal decision process for audit trail.
type DecisionPath struct {
	ModelVersion string         `json:"model_version"`
	FeatureVector []float32     `json:"feature_vector"`
	Score         float64       `json:"score"`
	Trees         []TreeDecision `json:"trees"`
	Timestamp     time.Time     `json:"timestamp"`
}

// TreeDecision represents one tree's vote in an ensemble.
type TreeDecision struct {
	TreeID     int     `json:"tree_id"`
	PathLength int     `json:"path_length"` // shorter = more anomalous
	Vote       float64 `json:"vote"`        // contribution to final score
}

// LogDecisionPath persists the model's decision process.
// This is critical for debugging false positives and regulatory compliance.
func LogDecisionPath(logger DecisionLogger, path DecisionPath) {
	// In production, this writes to a structured log or database
	// that SOC analysts can query when reviewing alerts.
	logger.Log(path)
}
```

### 10.4 Explainable Scoring Response

```go
// ExplainableScoreResponse is the API response format that includes
// both the risk score and its explanation. This is what SOC analysts
// see in their dashboard.
type ExplainableScoreResponse struct {
	RiskScore    float64               `json:"risk_score"`
	Severity     string                `json:"severity"`
	IsAnomaly    bool                  `json:"is_anomaly"`
	ModelVersion string                `json:"model_version"`
	Explanation  Explanation           `json:"explanation"`
	Timestamp    time.Time             `json:"timestamp"`
	TraceID      string                `json:"trace_id"`
}

// Example JSON output:
// {
//   "risk_score": 0.82,
//   "severity": "high",
//   "is_anomaly": true,
//   "model_version": "v20240115_080000",
//   "explanation": {
//     "risk_score": 0.82,
//     "severity": "high",
//     "base_score": 0.15,
//     "top_features": [
//       {
//         "feature": "geo_velocity_kmh",
//         "value": 1200,
//         "contribution": 0.35,
//         "description": "Impossible travel: 1200 km/h (faster than flight)"
//       },
//       {
//         "feature": "is_new_device",
//         "value": 1,
//         "contribution": 0.22,
//         "description": "Login from a previously unseen device"
//       },
//       {
//         "feature": "is_new_ip",
//         "value": 1,
//         "contribution": 0.15,
//         "description": "Login from a new IP address"
//       }
//     ],
//     "summary": "Risk score 0.82 (high): Impossible travel: 1200 km/h; Login from unseen device; Login from new IP",
//     "recommendation": "CHALLENGE: Require step-up MFA (WebAuthn preferred). Alert SOC for review."
//   },
//   "timestamp": "2024-01-15T08:32:00Z",
//   "trace_id": "req-abc-123"
// }
```

---

## 11. GGID AI Detection Gap Analysis

### 11.1 What GGID Currently Has

Based on source code review of `services/auth/internal/service/` and
`services/gateway/internal/middleware/`:

#### Auth Service: `anomaly_detection.go`

| Function | What It Does | Type | Gap |
|---|---|---|---|
| `RecordFailedLoginAnomaly` | Counts failed logins via Redis sorted set, locks after 5 | Rule | Fixed threshold (5). Slow brute force (4/window) undetectable. |
| `CheckGeoAnomaly` | Haversine distance vs known IPs, 500km threshold | Rule | Binary threshold. No velocity (impossible travel). No per-user baselines. |
| `CheckNewDevice` | SISMEMBER on Redis device set | Rule | Simple set membership. No device fingerprint strength scoring. |
| `AssessLoginAnomaly` | Combines lockout + geo + device checks | Rule | Static rules. No ML scoring. No sequence awareness. |
| `haversineDistance` | Geographic distance calculation | Utility | Works well. No velocity computation. |

#### Auth Service: `risk_auth.go`

| Function | What It Does | Type | Gap |
|---|---|---|---|
| `AssessLoginRisk` | Weighted scoring of 5 signals (0-100) | Rule | Fixed weights (+40 for failures, +15 for new IP, +10 for night, +15 for UA change, +30 for brute force). No per-user personalization. No learning. |
| `RecordSuccessfulLogin` | Stores known IP and UA in Redis | Rule | Simple set. No frequency tracking. No behavioral profiling. |
| `RecordFailedLoginAttempt` | Tracks per-IP failures and multi-user attempts | Rule | Counts only. No timing patterns. No IP rotation detection. |
| `BlockSuspiciousIP` / `IsIPBlocked` | Redis-based IP blocklist | Rule | Manual management. No automatic blocking from ML signals. |

#### Gateway Middleware: `botdetect.go`

| Function | What It Does | Type | Gap |
|---|---|---|---|
| `BotDetect` | Static UA pattern matching (sqlmap, nikto, etc.) | Rule | Only catches known tools. Zero-day tools pass undetected. |
| `BehavioralBotDetect` | Per-IP request rate thresholding | Rule | Fixed threshold. No timing regularity detection (bots have perfect intervals). No behavioral fingerprinting. |

#### Gateway Middleware: `ratelimit.go`, `sliding_ratelimit.go`, `token_bucket.go`

| Component | What It Does | Type | Gap |
|---|---|---|---|
| `RateLimiter` | Fixed-window per-path rate limiting | Rule | Fixed limits (5 login/min). No adaptive limits. No ML-based throttling. |
| `SlidingWindowLimiter` | Redis Lua sliding window, per-tier | Rule | Good implementation. But still just counting. No pattern detection. |
| `TenantBucketLimiter` | Token bucket per-tenant+IP | Rule | Good for API throttling. No ML integration. |
| `AdaptiveRateLimiter` | Latency-based QPS adjustment | Rule | Adaptive to backend health, not to threat patterns. |

#### Gateway Middleware: `adaptive_geo_dedup.go`

| Component | What It Does | Type | Gap |
|---|---|---|---|
| `AdaptiveRateLimiter` | Backend latency → QPS adjustment | Rule | Good for load protection. Not threat-aware. |
| `RequestDeduplicator` | Idempotency-key response caching | Rule | Correct but unrelated to threat detection. |
| `GeoEnricher` | IP prefix → country/city headers | Enrichment | Very basic (prefix matching). Needs MaxMind GeoIP2 for production. |

### 11.2 What GGID Is Missing

| Capability | Current State | ML Would Add | Priority |
|---|---|---|---|
| **Per-user behavioral baselines** | None | Login time/geo/device baselines per user | P1 |
| **Multi-dimensional pattern detection** | Per-dimension only (IP OR user OR device) | Cross-dimensional correlation (IP + time + device + ASN) | P1 |
| **Credential stuffing detection** | None | Population-level: high IP diversity + low success rate + concentrated time | P1 |
| **Impossible travel with velocity** | Binary geo distance (500km threshold) | Velocity computation + per-user geo baseline | P1 |
| **Adaptive MFA** | Static step-up trigger (score >= 30) | Risk-based: skip MFA for low-risk, WebAuthn for high-risk | P2 |
| **Shadow mode / model validation** | N/A | Score without acting to validate before enforcement | P2 |
| **Model monitoring / drift detection** | N/A | PSI/KS test on feature distributions | P2 |
| **Explainable AI** | None | SHAP values + human-readable explanations for SOC | P2 |
| **Batch post-hoc analysis** | None | Population-level pattern detection, privilege escalation chains | P3 |
| **Sequence detection** | None | Temporal patterns (login → role change → data export) | P3 |

### 11.3 Specific Code-Level Gaps

```go
// GAP 1: AssessLoginRisk uses FIXED thresholds, not per-user baselines.
// From risk_auth.go line 46:
if failCount >= 5 {  // ← FIXED: should be user-specific z-score
    assessment.Score += 40
}

// What ML would do:
// userBaseline := getUserBaseline(userID)
// zScore := (failCount - userBaseline.MeanFailures) / userBaseline.StdDevFailures
// if zScore > 2.5 {  // 2.5σ above normal for THIS user
//     assessment.Score += mlContribution
// }

// GAP 2: CheckGeoAnomaly uses a FIXED distance threshold for all users.
// From anomaly_detection.go line 17:
const anomalyKnownIPThreshold = 500.0 // km  ← FIXED for everyone

// What ML would do:
// Per-user geographic radius based on historical distribution.
// A traveling salesperson has a larger "normal" radius than a remote worker.

// GAP 3: BotDetect uses STATIC user-agent patterns.
// From botdetect.go lines 20-23:
var suspiciousPatterns = []string{
    "sqlmap", "nikto", "nmap", "masscan", ...
}
// What ML would do:
// Behavioral fingerprinting: request timing intervals, header order,
// TLS fingerprint (JA3), navigation patterns. These are much harder
// for attackers to spoof than User-Agent strings.

// GAP 4: No sequence/event-chain detection.
// Current: each event scored independently.
// What ML would do:
// LSTM/sequence model: login → password change → role escalation → data export
// = account takeover pattern, even if each individual event looks normal.
```

---

## 12. Gap Analysis & Recommendations

### 12.1 Current State Summary

GGID has a solid rule-based security foundation with 14 detection capabilities.
The gaps are not in *coverage* (the right signals are being checked) but in
*sophistication* (the checks are static, one-dimensional, and don't learn).

### 12.2 Recommended Action Items

| # | Action | Effort | Priority | Dependencies |
|---|---|---|---|---|
| 1 | **Build online feature store in Redis** — Extend existing Redis keys to store per-user login history, IP velocity, device sets, ASN sets. This is the foundation for all ML work. | ~1 week | P1 | None (uses existing Redis) |
| 2 | **Deploy ONNX inference engine in Go** — Use `onnxruntime_go` to load a trained Isolation Forest model. Start in shadow mode (score but don't act). | ~1 week | P1 | Action 1 |
| 3 | **Implement streaming feature pipeline** — Pipelined Redis reads at auth time, computing all features in a single round trip. Target: <10ms overhead. | ~1 week | P1 | Action 1 |
| 4 | **Train first model and validate in shadow mode** — Python pipeline to export Isolation Forest to ONNX. Run shadow scoring for 2 weeks. Measure FP rate. | ~2 weeks | P1 | Actions 1-3 |
| 5 | **Integrate ML score into AssessLoginRisk** — Enrich existing risk assessment with ML contribution. Start in advisory mode. Promote to enforcement after validation. | ~1 week | P2 | Actions 1-4 |
| 6 | **Implement explainable scoring** — SHAP value computation and human-readable explanations for SOC analysts. | ~1 week | P2 | Actions 1-5 |
| 7 | **Build drift detection and retraining pipeline** — PSI monitoring on feature distributions. Weekly retraining schedule. | ~1 week | P2 | Actions 1-5 |
| 8 | **Add batch post-hoc analysis** — Hourly batch job for population-level patterns (credential stuffing, privilege escalation chains). | ~1 week | P3 | Actions 1-2 |

### 12.3 Phased Implementation Roadmap

```
Phase 1 (Weeks 1-3): Foundation
├── Online feature store (Redis)
├── Streaming feature pipeline (pipelined reads)
├── ONNX inference engine (Go)
└── First model trained (Isolation Forest)

Phase 2 (Weeks 4-5): Shadow Mode
├── Deploy model in shadow mode
├── Log all scores for analysis
├── Analyst review of discrepancies
└── Measure FP rate, recall, latency

Phase 3 (Weeks 6-7): Advisory Mode
├── ML scores logged alongside rule decisions
├── SOC dashboard shows ML explanations
├── A/B testing framework
└── Drift detection

Phase 4 (Weeks 8-9): Enforcement
├── ML score contributes to risk assessment
├── Adaptive MFA triggering
├── Risk metadata propagation (gRPC)
└── Audit trail for all ML decisions

Phase 5 (Weeks 10+): Advanced
├── Batch post-hoc analysis
├── Sequence/event-chain detection
├── Supervised model (with labeled data)
└── Automated retraining pipeline
```

### 12.4 Success Metrics

| Metric | Target | Measurement |
|---|---|---|
| Feature extraction latency | < 10ms P99 | Redis pipeline timing |
| ONNX inference latency | < 5ms P99 | In-engine timing |
| Total ML overhead | < 20ms P99 | End-to-end auth timing |
| False positive rate (shadow) | < 1% of events | Shadow analysis report |
| New threats detected | > 0 that rules miss | Discrepancy analysis |
| Alert precision (enforcement) | > 90% | Analyst feedback labels |
| Model retraining cadence | Weekly + drift-triggered | Training log |
| SOC analyst trust | > 80% act on ML alerts | Survey / action rate |

### 12.5 Risk Considerations

| Risk | Mitigation |
|---|---|
| ML blocks legitimate users | Start in shadow mode. Never let ML *decrease* risk below rule level. |
| Model degradation over time | Drift detection + weekly retraining. Auto-rollback if metrics degrade. |
| Privacy: features contain PII | Encrypt at rest. Per-tenant isolation. Audit all feature access. |
| Attackers poison training data | Validate training data. Detect outliers in training set. Use robust models. |
| Explainability for compliance | SHAP values + decision path logging for every score. |
| Latency regression | ONNX inference is < 5ms. Feature extraction is the bottleneck — optimize Redis pipeline. |

---

## References

- [ONNX Runtime for Go](https://github.com/yalue/onnxruntime_go) — Go bindings for ONNX Runtime
- [skl2onnx](https://github.com/onnx/sklearn-onnx) — sklearn to ONNX model converter
- [SHAP (SHapley Additive exPlanations)](https://github.com/shap/shap) — Model explainability
- GGID `services/auth/internal/service/anomaly_detection.go` — existing rule-based anomaly detection
- GGID `services/auth/internal/service/risk_auth.go` — existing risk scoring and MFA triggering
- GGID `services/gateway/internal/middleware/botdetect.go` — existing bot detection
- GGID `services/gateway/internal/middleware/ratelimit.go` — existing rate limiting
- GGID `services/gateway/internal/middleware/sliding_ratelimit.go` — Redis sliding window limiter
- GGID `pkg/audit/publisher.go` — audit event struct and NATS publisher
- GGID `docs/research/abnormal-detection-ml.md` — companion document covering model internals
- [Population Stability Index (PSI)](https://www.listendata.com/2015/05/population-stability-index.html) — drift detection method
- [Feature Store Architecture](https://www.featurestore.org/) — online/offline feature store patterns
