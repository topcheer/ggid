# Identity Threat Detection & Response (ITDR) — GGID Gap Analysis

*Research date: 2026-07-12*

## Overview

ITDR is a top 2025 Gartner security trend. It focuses on detecting and responding to identity-based attacks that bypass traditional security controls:

- **Credential compromise**: brute force, credential stuffing, password spray
- **Privilege escalation**: exploitation of misconfigured roles/permissions
- **Lateral movement**: using stolen tokens to move between systems
- **Identity supply chain attacks**: compromised IdP, SAML token theft
- **Golden ticket / SAML relay**: forged authentication attacks

Key vendors: Microsoft Defender for Identity, Sophos ITDR, CrowdStrike Falcon Identity Protection, Oort (Cisco).

## GGID Current State: PARTIAL

**Implemented:**
- [x] Credential stuffing detection
- [x] Password spray detection
- [x] Impossible travel detection
- [x] Session hijacking detection
- [x] Anomaly detection engine
- [x] Risk scoring (user + session)
- [x] Threat feed (suspicious events)
- [x] Token reuse detection
- [x] Cross-system correlation

**Missing (Critical Gaps):**
- [ ] Lateral movement detection (token-based)
- [ ] Privilege escalation monitoring (role change anomalies)
- [ ] Identity supply chain integrity (IdP tamper detection)
- [ ] Golden ticket detection (forged SAML/Kerberos)
- [ ] Honeytoken/decoy account alerts
- [ ] ITDR incident response playbooks (automated)
- [ ] MITRE ATT&CK mapping for identity attacks
- [ ] Real-time behavioral analytics baseline
- [ ] Identity attack kill chain visualization

## Gap Analysis

### P0: Lateral Movement Detection (Backend)
- Track token usage across services
- Alert when single token used from multiple IPs/devices simultaneously
- Detect token passed between services in unusual pattern
- `GET /api/v1/audit/itdr/lateral-movement` endpoint

### P1: Privilege Escalation Monitoring (Backend)
- Real-time monitoring of role changes
- Alert on: self-granting privileges, rapid role accumulation, off-hours changes
- Compare against approved change windows
- `GET /api/v1/audit/itdr/privilege-escalation` endpoint

### P1: MITRE ATT&CK Mapping (Backend + Frontend)
- Map detected threats to MITRE ATT&CK techniques
- T1078 Valid Accounts, T1098 Account Manipulation, T1556 Modify Auth Process
- Display in threat dashboard
- `GET /api/v1/audit/itdr/attack-techniques` endpoint

### P2: Honeytoken Accounts (Backend)
- Create decoy accounts that trigger alerts when used
- Monitor for credential leaks in breach databases
- `POST /api/v1/audit/itdr/honeytokens` CRUD

### P2: ITDR Incident Playbooks (Backend)
- Predefined automated response: isolate session, revoke tokens, force MFA, notify SOC
- `POST /api/v1/audit/itdr/playbooks/{id}/execute`

## Backlog Items Generated
- [ ] **P0** Backend: Lateral movement detection (services/audit/)
- [ ] **P1** Backend: Privilege escalation monitoring (services/audit/)
- [ ] **P1** Backend: MITRE ATT&CK mapping (services/audit/)
- [ ] **P1** Frontend: ITDR dashboard with attack kill chain (console/src/)
- [ ] **P2** Backend: Honeytoken/decoy account system (services/)
- [ ] **P2** Backend: ITDR incident playbooks (services/audit/)
