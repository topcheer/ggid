# Security Console Pages — User Guide

> Console path: `/security/*` | Features: F-44, F-45 + security suite

## Overview

The GGID Console security section provides ITDR (Identity Threat Detection & Response), access governance, and zero-trust management capabilities. This guide covers the key pages.

## Key Pages

### Ransomware Defense Center (`/security/ransomware-defense`)

Kill chain visualization based on MITRE ATT&CK framework for identity-based ransomware detection.

**5 Tabs:**

| Tab | Description |
|-----|-------------|
| Kill Chain | 5-stage attack visualization: Initial Access (TA0001) → Lateral Movement (TA0008) → Privilege Escalation (TA0004) → Data Exfiltration (TA0010) → Encryption (TA0040) |
| Detection Rules | Composite detection rules that correlate identity events across kill chain stages |
| Isolation Playbook | Automated response actions (disable account, revoke sessions, quarantine device) |
| Threat Heatmap | Geographic/temporal heatmap of detected threats |
| Incident Response | Open incidents with status tracking and resolution workflows |

**Attack Timeline:** Shows real-time kill chain events. Empty state: "No kill chain events detected. System is clean."

### ReBAC Permission Graph (`/security/rebac`)

Relationship-Based Access Control using a Zanzibar-style tuple engine.

**Features:**
- **Permission Check** — Test whether a subject has a relation on an object (namespace/object/relation/subject)
- **Tuples** — Manage permission tuples (subject → relation → object)
- **Statistics** — Live counts of subjects, objects, and tuples
- **Add Relation** — Create new permission tuples

**Supported namespaces:** `document`, `folder`, `resource`, `policy`

### Identity-Based DLP (`/security/dlp`)

Data Loss Prevention integrated with identity context.

**4 Tabs:**

| Tab | Description |
|-----|-------------|
| Policies | Create/manage DLP policies with regex triggers and action rules (block/mask/log) |
| Events | Real-time DLP violation events with user/resource/pattern/action |
| Risk Heatmap | Visualization of DLP violations by user, resource, and pattern severity |
| Policy Tester | Test policy patterns against sample data before deployment |

**Policy actions:** `block` (deny request), `mask` (redact sensitive data), `log` (record without blocking)

**Trigger scopes:** `api`, `query`, `export` — policies can target specific data access paths

### Audit Chain Explorer (`/security/audit-explorer`)

Hash chain verification and advanced audit search.

**5 Tabs:**

| Tab | Description |
|-----|-------------|
| Hash Chain | Verify audit log integrity via cryptographic hash chain (each block hashes the previous) |
| Search | Advanced search across audit events (user, action, resource, time range) |
| Timeline | User activity timeline reconstruction |
| Anomalies | Machine-learning detected anomalies in access patterns |
| Export | Evidence export for compliance (PDF/CSV with hash verification) |

**Chain Status:** Shows "Chain INTACT — N blocks verified" or alerts on tampering.

### Adaptive Auth Choreography (`/security/adaptive-auth`)

Risk-based dynamic authentication with signal weighting and step-up orchestration.

**5 Tabs:**

| Tab | Description |
|-----|-------------|
| Simulator | Interactive risk evaluation: enter context (IP, device trust, location, hour, failed attempts) and get risk score + recommended AAL |
| Risk Matrix | Heatmap of risk scores across signal combinations |
| AAL Thresholds | Configure Authenticator Assurance Level thresholds (AAL1/AAL2/AAL3) for different risk tiers |
| Step-Up Tree | Visualization of step-up authentication flow based on risk level |
| Signal Sources | Configure which signals (IP reputation, device trust, geo-velocity, behavior) contribute to risk scoring |

**Simulator inputs:** IP address, device trust (trusted/new/unmanaged), current/previous location, hour of day, failed login attempts.

### Secret Broker (`/security/secret-broker`)

See [Secret Broker Console Guide](secret-broker-console-guide.md) for full documentation.

### Threat Intelligence Hub (`/security/threat-intel`)

External threat intelligence feeds, IOC management, and ITDR correlation.

## Related Documentation

- [ITDR Guide](../guides/identity-threat-detection-response.md)
- [Threat Intel Design](../architecture/threat-intel-design.md)
- [Audit Hash Chain](audit-hash-chain.md)
- [Adaptive Authentication](adaptive-authentication.md)
- [Security Best Practices](../security-best-practices.md)
