# Identity Analytics Guide

## Overview

Identity analytics transforms raw identity and access data into actionable insights for security, compliance, and operational decision-making. This guide covers analytics use cases, data sources, analysis methods, dashboard design, reporting cadence, and how GGID supports identity analytics.

## Analytics Use Cases

### Access Pattern Analysis

Understand how users interact with systems to detect anomalies and optimize access policies.

- **Usage frequency**: Identify active vs dormant accounts and entitlements
- **Access time patterns**: Detect off-hours access that may indicate compromise
- **Geographic patterns**: Flag access from unusual locations or impossible travel
- **Resource access patterns**: Identify frequently and rarely accessed resources
- **Privilege escalation tracking**: Monitor changes in access levels over time
- **Service account usage**: Track API key and service account activity

**Key questions answered**:
- Which entitlements are actively used vs unused (for access review optimization)?
- Are there accounts accessing resources outside their normal pattern?
- What is the peak access time and are we provisioned for it?

### Anomaly Detection

Identify deviations from established baselines that may indicate security incidents.

- **Failed authentication spikes**: Brute force or credential stuffing attacks
- **Unusual MFA challenges**: MFA fatigue attacks or unauthorized access attempts
- **Privileged account anomalies**: Unexpected admin actions or access to sensitive resources
- **Bulk operations**: Mass exports, bulk deletions, or bulk permission changes
- **Dormant account activation**: Long-inactive accounts suddenly used
- **Configuration drift**: Unauthorized changes to security settings

**Detection methods**:
- Statistical: Z-score, IQR-based outlier detection
- ML-based: Isolation forest, autoencoder reconstruction error
- Rule-based: Velocity rules, geo-velocity rules, time-based rules
- Behavioral: User and entity behavior analytics (UEBA)

### Risk Trend Analysis

Track risk metrics over time to measure security posture improvement or degradation.

- **Risk score trends**: Aggregate risk scores by tenant, department, or user cohort
- **Threat landscape**: Track attack types, success rates, and mitigation effectiveness
- **Vulnerability exposure**: Monitor open vulnerabilities and patch velocity
- **Compliance drift**: Track compliance score changes and exception aging
- **User risk distribution**: Histogram of individual user risk scores
- **Incident correlation**: Correlate identity events with security incidents

### Compliance Gap Analysis

Identify and quantify gaps between current state and compliance requirements.

- **Access certification coverage**: % of users with current access reviews
- **Policy compliance**: % of access decisions conforming to policy
- **Segregation of duties violations**: Count and severity of SoD conflicts
- **Orphaned accounts**: Users with access but no active employment
- **Excessive privileges**: Users with more access than role requires
- **Audit trail completeness**: % of critical actions with complete audit records
- **Exception aging**: Open compliance exceptions by age and severity

## Data Sources

### Audit Events

The primary data source for identity analytics.

| Event Type | Analytics Value |
|-----------|-----------------|
| Authentication success/failure | Anomaly detection, brute force detection |
| Authorization decisions | Policy effectiveness, access pattern analysis |
| User provisioning/deprovisioning | Lifecycle tracking, orphaned account detection |
| Role assignment/removal | Privilege escalation tracking, SoD analysis |
| Configuration changes | Drift detection, compliance monitoring |
| Data access (read/write/delete) | Data exfiltration detection, usage patterns |
| Session start/end | Session duration analysis, concurrent session detection |
| MFA challenge/response | MFA fatigue detection, enrollment tracking |

### Authentication Logs

- **Login events**: Timestamp, user, IP, geo, user agent, success/failure, MFA type
- **Token issuance**: Token type, scope, audience, expiration, client ID
- **Token validation**: Validation results, claims, scope checks
- **Session events**: Session ID, start, end, duration, concurrent sessions
- **MFA events**: Challenge type, response, success/failure, device info

### Policy Decision Logs

- **Policy evaluations**: Policy ID, subject, resource, action, decision, obligations
- **ABAC attribute values**: Attribute names and values used in evaluation
- **Rule hit counts**: Which rules fired and how often
- **Denial reasons**: Why access was denied (policy, SoD, risk, time)
- **Policy version**: Which policy version was evaluated

### Session Data

- **Active sessions**: User, tenant, device, IP, start time, last activity
- **Session duration**: Distribution of session lengths
- **Concurrent sessions**: Users with multiple active sessions
- **Session anomalies**: Session hijacking indicators, impossible travel
- **Token refresh patterns**: Refresh frequency, scope expansion

### External Data Sources

- **HR system**: Employment status, department, manager, role changes
- **Asset inventory**: Resource metadata, classification, owner
- **Vulnerability scanner**: CVE data, exposure metrics
- **Threat intelligence**: Known malicious IPs, domains, indicators
- **Network logs**: VPN logs, firewall logs, DNS queries
- **SIEM**: Correlated security events from other systems

## Analysis Methods

### Descriptive Analytics

Answers "What happened?" by summarizing historical data.

- **Summary statistics**: Counts, rates, averages, percentiles
- **Time series**: Event counts over time (hourly, daily, weekly)
- **Top-N reports**: Most active users, most accessed resources, most denied actions
- **Distribution analysis**: User risk score distribution, session duration histogram
- **Trend visualization**: Line charts, area charts for temporal patterns

**Example metrics**:
- Daily authentication success rate: 99.2%
- Top 10 denied resources by count
- Average session duration: 4.5 hours
- MFA enrollment rate: 94.3%

### Diagnostic Analytics

Answers "Why did it happen?" by investigating root causes.

- **Drill-down analysis**: From aggregate anomaly to specific events
- **Correlation analysis**: Relationships between events (e.g., config change followed by access spike)
- **Comparative analysis**: Compare current period vs previous period, or one tenant vs another
- **Root cause investigation**: Trace event chains to identify initiating cause
- **Factor analysis**: Identify contributing factors to an outcome

**Example investigation**:
- Anomaly: Failed logins increased 300% on Tuesday
- Drill-down: 85% of failures from IP range 203.0.113.0/24
- Correlation: No config changes; new external integration deployed Monday
- Root cause: Credential stuffing from compromised external service

### Predictive Analytics

Answers "What will happen?" using statistical models and machine learning.

- **Risk scoring**: Predict user risk scores based on behavior patterns
- **Anomaly prediction**: Forecast expected behavior ranges for comparison
- **Capacity planning**: Predict peak authentication loads for scaling
- **Churn prediction**: Identify users likely to abandon MFA enrollment
- **Attack prediction**: Predict likely attack vectors based on threat intelligence

**Models**:
- Time series forecasting: ARIMA, Prophet for load prediction
- Classification: Random forest, gradient boosting for risk classification
- Anomaly detection: Isolation forest, LOF, autoencoders
- Survival analysis: Time-to-deprovisioning for inactive accounts

### Prescriptive Analytics

Answers "What should we do?" by recommending actions.

- **Access recommendations**: Suggest entitlement removal for unused access
- **Policy recommendations**: Suggest policy changes based on denial patterns
- **Risk mitigation**: Recommend actions to reduce individual or aggregate risk
- **Resource allocation**: Recommend where to focus review efforts
- **Automated actions**: Trigger automated responses (e.g., force MFA re-enrollment)

**Example recommendations**:
- User jdoe has 15 unused entitlements, recommend removal in next access review
- Policy P-003 denies 40% of marketing team access, review policy scope
- Risk score for tenant T-007 increased 20%, recommend security audit
- 500 dormant accounts detected, initiate deprovisioning workflow

## Dashboard Design

### Executive Dashboard

For CISO and leadership - high-level posture and trends.

| Widget | Type | Refresh |
|--------|------|---------|
| Overall risk score | Gauge | Daily |
| Authentication success rate | KPI card | Daily |
| Active threats | Alert list | Real-time |
| Compliance posture | Stacked bar | Weekly |
| Risk trend (30-day) | Line chart | Daily |
| Top 5 risk tenants | Table | Daily |
| Incident summary | Table | Real-time |

### Security Operations Dashboard

For SOC analysts - operational monitoring and investigation.

| Widget | Type | Refresh |
|--------|------|---------|
| Authentication events (real-time) | Stream | Real-time |
| Failed auth heatmap | Heatmap | 5 min |
| Anomaly alerts | Alert queue | Real-time |
| Active sessions by geo | Map | 5 min |
| Top source IPs (failed) | Bar chart | 15 min |
| MFA challenge rate | Gauge | 15 min |
| Risk score distribution | Histogram | Hourly |
| Session duration distribution | Histogram | Hourly |

### Compliance Dashboard

For compliance officers - regulatory and audit posture.

| Widget | Type | Refresh |
|--------|------|---------|
| Access review status | Progress bars | Daily |
| SoD violation count | KPI card | Daily |
| Orphaned accounts | Table | Daily |
| Policy compliance rate | Gauge | Daily |
| Exception aging | Stacked bar | Weekly |
| Audit trail completeness | Gauge | Daily |
| Certification coverage by tenant | Table | Weekly |

### Access Review Dashboard

For managers - certification and review workflow.

| Widget | Type | Refresh |
|--------|------|---------|
| Pending reviews | Task list | Daily |
| Entitlement usage | Usage bars | On-demand |
| Unused access (30/60/90 days) | Table | On-demand |
| SoD conflicts for user | Alert list | On-demand |
| Review history | Timeline | On-demand |
| Peer access comparison | Table | On-demand |

### Design Principles

1. **Right information, right audience**: Tailor widgets to the consumer's role
2. **Actionable**: Every widget should answer a question or enable a decision
3. **Contextual**: Show trends, not just snapshots - include historical context
4. **Drill-down capable**: Allow users to navigate from summary to detail
5. **Real-time where needed**: Operational dashboards need real-time; strategic dashboards don't
6. **Mobile-responsive**: Ensure dashboards are accessible on mobile devices
7. **Exportable**: Support CSV/PDF export for reporting and audit

## Reporting Cadence

| Report | Frequency | Content | Audience |
|--------|-----------|---------|----------|
| Real-time alerts | Event-driven | Anomaly, threshold breach, incident | SOC |
| Daily security summary | Daily | Auth stats, anomalies, incidents | SecOps lead |
| Weekly risk report | Weekly | Risk trends, top threats, mitigation status | CISO |
| Monthly compliance report | Monthly | Compliance metrics, exceptions, audit status | Compliance officer |
| Quarterly executive summary | Quarterly | Strategic posture, investment recommendations | C-suite, Board |
| Annual identity posture | Annual | Full-year trends, maturity assessment, roadmap | All stakeholders |

## GGID Identity Analytics

### Data Collection

GGID collects identity analytics data through its audit service:

- **Audit events**: All authentication, authorization, and administrative events
- **Session events**: Session lifecycle with device, IP, and geo context
- **Policy decisions**: Full ABAC/RBAC evaluation context and results
- **Risk events**: Risk score calculations and contributing factors
- **Configuration changes**: All security configuration modifications

### Analytics Capabilities

| Capability | Implementation |
|-----------|----------------|
| Real-time monitoring | Audit event stream via NATS JetStream |
| Anomaly detection | Risk engine with behavioral analysis |
| Access pattern analysis | Audit query API with aggregation support |
| Compliance reporting | Scheduled reports + on-demand queries |
| Dashboard | Admin Console with role-based dashboards |
| SIEM integration | Syslog/CEF forwarder for external SIEM |
| API access | REST API for custom analytics integration |

### Audit Query API

```
GET /api/v1/audit/events?tenant_id={id}&start={ts}&end={ts}&event_type={type}
GET /api/v1/audit/events/aggregate?group_by=user&metric=count&event_type=auth_failed
GET /api/v1/audit/export?format=csv&start={ts}&end={ts}
```

### Risk Engine Integration

GGID's risk engine computes real-time risk scores based on:
- Authentication behavior (success/failure patterns, MFA usage)
- Access patterns (resource access frequency, unusual access)
- Session behavior (duration, concurrency, geo)
- External factors (threat intelligence, vulnerability status)

Risk scores feed into:
- Adaptive authentication (step-up MFA for high-risk sessions)
- Access decisions (deny high-risk access)
- Analytics dashboards (risk trend visualization)
- Alerting (threshold-based alerts)

### Console Analytics

The GGID Admin Console provides built-in analytics dashboards:

- **Dashboard page**: Overview metrics, risk gauge, recent events
- **Audit page**: Event search with filtering and export
- **Security Center**: Risk trends, anomaly alerts, threat indicators
- **Reports page**: Scheduled report generation and download

### External Analytics Integration

- **SIEM**: Forward audit events to Splunk, ELK, QRadar, Sentinel
- **Data warehouse**: Export audit data to Snowflake, BigQuery, Redshift for custom analytics
- **Business intelligence**: Connect Grafana, Tableau, Power BI via REST API
- **Custom analytics**: Use the REST API to build custom analytics pipelines

## Best Practices

1. **Define clear use cases**: Start with specific questions before building analytics
2. **Ensure data quality**: Garbage in, garbage out - validate event data completeness
3. **Start simple**: Begin with descriptive analytics, evolve to predictive
4. **Close the loop**: Analytics should drive action, not just observation
5. **Respect privacy**: Anonymize data where possible, especially for analytics exports
6. **Version your models**: Track model versions and retrain periodically
7. **Monitor your monitors**: Alert when analytics pipelines fail or degrade
8. **Context matters**: Always provide context (baselines, trends, comparisons)
9. **Make it actionable**: Every metric should have an associated action or decision
10. **Iterate**: Continuously refine metrics, models, and dashboards based on feedback

## See Also

- [Identity Threat Detection and Response](./identity-threat-detection-response.md)
- Audit Logging Guide
- Security Monitoring Guide
- Access Review Guide
- Risk Engine Guide
- SIEM Forwarder Guide