# Anomaly Detection Guide

This guide covers detection methods, signal types, feature engineering, real-time vs batch detection, alerting thresholds, false positive reduction, ML model lifecycle, and GGID's anomaly detection implementation.

## Overview

Anomaly detection identifies patterns that deviate from expected behavior, indicating potential security threats, insider risks, or system issues. GGID uses a combination of statistical, ML, rule-based, and threshold-based detection methods.

## Detection Methods

### 1. Statistical

Uses statistical baselines to detect outliers:

| Method | Description | Use Case |
|---|---|---|
| Z-score | Deviations from mean (>3 sigma) | Login frequency |
| IQR | Outliers outside interquartile range | Session duration |
| Moving average | Deviation from rolling average | API call volume |
| EWMA | Exponentially weighted moving average | Error rate trends |

```go
func zScore(value float64, history []float64) float64 {
    mean := avg(history)
    stdDev := stdDev(history)
    if stdDev == 0 {
        return 0
    }
    return (value - mean) / stdDev
}

func isAnomaly(value float64, history []float64) bool {
    return math.Abs(zScore(value, history)) > 3.0  // 3-sigma
}
```

### 2. Machine Learning

| Algorithm | Type | Use Case |
|---|---|---|
| Isolation Forest | Unsupervised | Multi-dimensional anomalies |
| One-Class SVM | Unsupervised | Boundary detection |
| Autoencoder | Deep learning | Complex pattern reconstruction |
| LSTM | Deep learning | Time-series prediction |
| Random Forest | Supervised | Classification with labels |

### 3. Rule-Based

Explicit rules for known attack patterns:

```yaml
anomaly_rules:
  - name: "impossible_travel"
    condition: "geo_velocity > 1000 km/h"
    severity: "high"
    action: "block"
  - name: "unusual_login_time"
    condition: "hour < 6 OR hour > 22"
    severity: "medium"
    action: "challenge"
  - name: "mass_failed_logins"
    condition: "failed_logins_5min > 20"
    severity: "high"
    action: "rate_limit"
  - name: "bulk_data_export"
    condition: "records_exported_1h > 10000"
    severity: "high"
    action: "alert"
```

### 4. Threshold-Based

Simple threshold checks:

| Signal | Threshold | Action |
|---|---|---|
| Failed logins per user | >10/hour | Lockout |
| Failed logins per IP | >50/hour | IP block |
| API calls per user | >1000/min | Rate limit |
| Data accessed per session | >10GB | Alert |
| Concurrent sessions | >5 | Alert |
| Admin actions per hour | >50 | Alert |

## Signal Types

### Login Signals

| Signal | Description | Baseline |
|---|---|---|
| Login frequency | Logins per day per user | 1-3/day normal |
| Login time | Hour of day | Business hours normal |
| Login location | IP geolocation | Usual locations |
| Device fingerprint | Device hash | Known devices |
| Failed attempts | Consecutive failures | 0-2 normal |
| Login method | Password, MFA, SSO | Usual method |

### Transaction Signals

| Signal | Description |
|---|---|
| Transaction volume | Number of operations per time window |
| Transaction value | Monetary value (if applicable) |
| Transaction type | Types of operations (create/delete/modify) |
| Transaction velocity | Speed of sequential operations |

### Access Pattern Signals

| Signal | Description |
|---|---|
| Resource access pattern | Which resources accessed |
| API endpoint pattern | Which endpoints called |
| Data access pattern | What data queried/exported |
| Permission usage | Which permissions exercised |
| Unusual resource access | Accessing resources not normally used |

### Data Exfiltration Signals

| Signal | Description | Threshold |
|---|---|---|
| Large data export | Bulk data retrieval | >1000 records |
| Off-hours data access | Data access at unusual times | After 10 PM |
| New destination | Data sent to new endpoint | Never-seen-before |
| Compression before export | Data compressed (zip/gzip) | Any compression |
| Multiple format export | Export in many formats | >3 formats in session |

## Feature Engineering

### Feature Extraction

```go
type UserFeatures struct {
    // Temporal features
    AvgLoginHour     float64  // Average login hour
    LoginHourVar     float64  // Variance in login hour
    DaysSinceLastLog float64  // Days since last login

    // Geographic features
    HomeCountry      string   // Most common country
    CountryCount     int      // Unique countries
    AvgGeoVelocity   float64  // Average km/h between logins

    // Device features
    DeviceCount      int      // Unique devices
    PrimaryDevice    string   // Most used device
    NewDeviceRate    float64  // Fraction of new devices

    // Behavioral features
    AvgSessionLen    float64  // Average session duration
    APIPerSession    float64  // Average API calls per session
    FailedLoginRate  float64  // Fraction of failed logins
    AdminActionRate  float64  // Fraction of admin actions

    // Access pattern features
    UniqueResources  int      // Unique resources accessed
    ReadToWriteRatio float64  // Read vs write operations
}
```

### Feature Vectors

```go
func buildFeatureVector(userID string, history []*AuditEvent) []float64 {
    features := UserFeatures{
        AvgLoginHour:    avgLoginHour(history),
        LoginHourVar:    loginHourVariance(history),
        DaysSinceLastLog: daysSinceLastLogin(history),
        CountryCount:    uniqueCountries(history),
        AvgGeoVelocity:  avgGeoVelocity(history),
        DeviceCount:     uniqueDevices(history),
        NewDeviceRate:   newDeviceRate(history),
        AvgSessionLen:   avgSessionLength(history),
        APIPerSession:   avgAPIPerSession(history),
        FailedLoginRate: failedLoginRate(history),
        AdminActionRate: adminActionRate(history),
        UniqueResources: uniqueResources(history),
        ReadToWriteRatio: readWriteRatio(history),
    }
    return featuresToVector(features)
}
```

## Real-Time vs Batch Detection

### Real-Time

Processes events as they occur, with sub-second latency:

| Aspect | Value |
|---|---|
| Latency | <100ms |
| Methods | Rule-based, threshold, statistical |
| Use Case | Block/challenge at request time |
| Limitation | Can't use complex ML models |

### Batch

Processes events in batches, with minutes-to-hours latency:

| Aspect | Value |
|---|---|
| Latency | 5-60 minutes |
| Methods | ML, complex statistical, pattern matching |
| Use Case | Detect subtle anomalies, train models |
| Limitation | Too slow for real-time blocking |

### Hybrid (GGID Approach)

```yaml
anomaly_detection:
  real_time:
    enabled: true
    methods: ["rule_based", "threshold", "statistical"]
    response_time: "<100ms"
    actions: ["block", "challenge", "rate_limit"]
  batch:
    enabled: true
    interval: "15m"
    methods: ["ml_isolation_forest", "pattern_matching"]
    actions: ["alert", "flag_for_review", "update_baseline"]
```

## Alerting Thresholds

### Severity Levels

| Severity | Score Range | Response | Notification |
|---|---|---|---|
| Low | 0.3-0.5 | Log | None |
| Medium | 0.5-0.7 | Alert security team | Slack channel |
| High | 0.7-0.85 | Alert + investigate | Email + Slack + PagerDuty |
| Critical | 0.85-1.0 | Auto-respond + escalate | All channels + CISO |

### Alert Configuration

```yaml
alerting:
  thresholds:
    low: 0.3
    medium: 0.5
    high: 0.7
    critical: 0.85
  channels:
    low:
      - "audit_log"
    medium:
      - "audit_log"
      - "slack:#security-alerts"
    high:
      - "audit_log"
      - "slack:#security-alerts"
      - "email:security-team@example.com"
      - "pagerduty:security"
    critical:
      - "audit_log"
      - "slack:#security-alerts"
      - "email:security-team@example.com"
      - "pagerduty:security-critical"
      - "email:ciso@example.com"
```

## False Positive Reduction

### Techniques

| Technique | Description | Impact |
|---|---|---|
| Multi-signal correlation | Require 2+ signals | Reduces FP by 60% |
| User baseline adaptation | Learn per-user patterns | Reduces FP by 40% |
| Confidence scoring | Weight by signal confidence | Reduces FP by 30% |
| Feedback loop | Learn from analyst decisions | Reduces FP over time |
| Whitelisting | Exclude known-good patterns | Eliminates known FPs |
| Time-window correlation | Correlate within time window | Reduces noise |

### Confidence Scoring

```go
func calculateConfidence(signals []AnomalySignal) float64 {
    if len(signals) == 0 {
        return 0
    }

    totalConfidence := 0.0
    for _, signal := range signals {
        // Each signal contributes weighted confidence
        totalConfidence += signal.Confidence * signal.Weight
    }

    // Multi-signal boost: more signals = higher confidence
    multiSignalBoost := math.Min(float64(len(signals))/3.0, 1.0) * 0.2

    return math.Min(totalConfidence + multiSignalBoost, 1.0)
}
```

### Feedback Loop

```go
func recordAnalystFeedback(alertID string, isTruePositive bool) {
    alert := getAlert(alertID)
    
    if isTruePositive {
        // Reinforce: similar patterns are more likely anomalies
        reinforceModel(alert.Signals, alert.Features)
    } else {
        // Suppress: similar patterns are likely normal
        suppressModel(alert.Signals, alert.Features)
        addWhitelistPattern(alert.Features)
    }
    
    audit.Log("anomaly_feedback", alertID, isTruePositive)
}
```

## ML Model Lifecycle

### 1. Training

```go
func trainAnomalyModel(trainingData [][]float64) *IsolationForest {
    model := NewIsolationForest(
        NumTrees(100),
        SampleSize(256),
        MaxFeatures(len(trainingData[0])),
    )
    model.Fit(trainingData)
    return model
}
```

### 2. Deployment

```yaml
ml:
  deployment:
    strategy: "canary"  # Deploy to 10% first
    canary_percentage: 10
    auto_rollback:
      fp_rate_threshold: 0.15  # Rollback if FP > 15%
      monitoring_period: 24h
```

### 3. Monitoring

```yaml
ml:
  monitoring:
    metrics:
      - "true_positive_rate"
      - "false_positive_rate"
      - "detection_latency"
      - "model_drift"
    drift_detection:
      enabled: true
      threshold: 0.1  # PSI > 0.1 indicates drift
      check_interval: 24h
```

### 4. Retraining

```yaml
ml:
  retraining:
    trigger:
      - "drift_detected"
      - "fp_rate > 0.10"
      - "schedule: monthly"
    min_samples: 10000
    validation_split: 0.2
```

## GGID Anomaly Detection

### Architecture

```
Audit Events → Feature Extraction → Real-Time Detection
                                → Batch Detection (15min)
                                → Alert Generation
                                → Response (block/challenge/alert)
```

### Configuration

```yaml
anomaly_detection:
  enabled: true
  real_time:
    enabled: true
    methods: ["rule_based", "threshold", "statistical"]
    response_time: "<100ms"
  batch:
    enabled: true
    interval: "15m"
    methods: ["isolation_forest", "pattern_matching"]
  signals:
    login: true
    transaction: true
    access_pattern: true
    data_exfiltration: true
  alerting:
    thresholds:
      low: 0.3
      medium: 0.5
      high: 0.7
      critical: 0.85
  false_positive_reduction:
    multi_signal: true
    min_signals: 2
    user_baseline: true
    feedback_loop: true
  ml:
    model: "isolation_forest"
    retraining_interval: "monthly"
    drift_detection: true
```

## Best Practices

1. **Layer detection methods** — Combine rule-based + statistical + ML
2. **Use real-time for blocking** — Fast methods for immediate response
3. **Use batch for investigation** — Complex models for subtle patterns
4. **Reduce false positives** — Multi-signal correlation + feedback loop
5. **Monitor model drift** — Retrain when patterns shift
6. **Start conservative** — Log-only mode before enforcing actions
7. **Alert by severity** — Don't page for low-severity anomalies
8. **Keep human in loop** — Auto-respond for critical, human review for high
9. **Audit detection events** — Log what was detected and what action was taken
10. **Regularly review rules** — Update thresholds and rules as patterns evolve