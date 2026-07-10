# Audit Log Compliance Requirements for IAM Systems

> **Focus**: Exactly what each compliance framework requires from audit logs —
> mandatory event types, retention periods, breach notification timelines, PII
> handling in audit data, and a concrete gap analysis of GGID's current audit
> infrastructure.
>
> **Companion docs** (do not duplicate):
> - `audit-log-compliance.md` — high-level framework overview, event taxonomy,
>   hash chain sketch, SIEM integration roadmap.
> - `audit-tampering-detection.md` — cryptographic tamper-evidence: hash chains,
>   Merkle trees, WORM storage, RFC 3161, external commitment.

---

## 1. PCI-DSS v4.0 Audit Requirements

PCI-DSS v4.0 **Requirement 10** ("Log and Monitor All Access to System
Components and Cardholder Data") is the most prescriptive audit logging
standard. It moved from "best practice" to **mandatory** in the v4.0.1 update
(March 2025).

### 1.1 What Must Be Logged (Req. 10.2)

| Requirement | Mandatory Events |
|---|---|
| **10.2.1** | All individual user accesses to cardholder data |
| **10.2.2** | All actions taken by root or administrative privileges |
| **10.2.3** | All changes to audit trails and audit configuration |
| **10.2.4** | All failed authentication attempts |
| **10.2.5** | All changes to identification and authentication credentials (password resets, MFA enrollment) |
| **10.2.6** | All initialization of new audit logs and stopping/starting existing logs |
| **10.2.7** | All creation and deletion of system-level objects |

### 1.2 Retention (Req. 10.5)

- **10.5.1**: Retain audit history for at least **12 months online** (immediately
  available for analysis).
- **10.5.2**: Retain at least **1 additional year offline** (2 years total
  minimum), available within 72 hours if requested by acquirer or PCI Forensic
  Investigator (PFI).

### 1.3 Daily Log Review (Req. 10.4)

- **10.4.1**: Automated mechanisms must identify anomalies or suspicious
  activity **at least daily**.
- **10.4.2**: Logs must be reviewed at least **daily** for anomalies. Automated
  SIEM with daily alert summaries satisfies this if a human reviews the alerts.

### 1.4 PCI-Compliant Audit Event

```go
// PCIAuditEvent extends GGID's standard audit.Event with PCI-specific fields.
type PCIAuditEvent struct {
	EventID          string            `json:"event_id"`
	Timestamp        time.Time         `json:"timestamp"`
	TenantID         string            `json:"tenant_id"`
	ActorID          string            `json:"actor_id"`
	ActorType        string            `json:"actor_type"` // user | admin | system
	Action           string            `json:"action"`
	Result           string            `json:"result"` // success | failure | denied
	SourceIP         string            `json:"source_ip"`
	ResourceAccessed string            `json:"resource_accessed"`
	// PCI 10.2.1 — every access to cardholder data must be tagged.
	CardholderDataAccess bool `json:"cardholder_data_access"`
	// PCI 10.2.2 — flag root/admin actions for priority review.
	IsAdminAction bool `json:"is_admin_action"`
	// PCI 10.2.3 — changes to audit configuration must be flagged.
	AuditConfigChange bool `json:"audit_config_change"`
	// PCI 10.2.7 — system-level object creation/deletion.
	SystemObjectMutation bool `json:"system_object_mutation"`
	// Chain hash for tamper-evidence (see tampering-detection doc).
	PrevHash string            `json:"prev_hash"`
	Hash     string            `json:"hash"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// EmitPCIEvent publishes a PCI-DSS-compliant audit event to GGID's NATS stream.
func EmitPCIEvent(ctx context.Context, pub *audit.Publisher, e PCIAuditEvent) error {
	if e.EventID == "" {
		e.EventID = uuid.NewString()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	// Serialize the payload for hash chain computation.
	payload := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		e.EventID, e.Timestamp.UTC().Format(time.RFC3339Nano),
		e.TenantID, e.ActorID, e.Action, e.Result)
	h := hmac.New(sha256.New, hmacKey)
	h.Write([]byte(payload + e.PrevHash))
	e.Hash = hex.EncodeToString(h.Sum(nil))

	return pub.Publish(ctx, audit.Event{
		ID:           uuid.MustParse(e.EventID),
		TenantID:     uuid.MustParse(e.TenantID),
		ActorType:    e.ActorType,
		ActorID:      uuid.MustParse(e.ActorID),
		Action:       e.Action,
		Result:       e.Result,
		IPAddress:    e.SourceIP,
		CreatedAt:    e.Timestamp,
		Metadata: map[string]any{
			"pci_cardholder_data_access": e.CardholderDataAccess,
			"pci_admin_action":           e.IsAdminAction,
			"pci_audit_config_change":    e.AuditConfigChange,
			"pci_system_object_mutation": e.SystemObjectMutation,
			"pci_hash":                   e.Hash,
			"pci_prev_hash":              e.PrevHash,
		},
	})
}
```

### 1.5 PCI Gap in GGID

GGID's `CleanupOldEvents` defaults to **90 days** — far below PCI's 12-month
minimum. The NATS stream retains events for only **72 hours** (`MaxAge: 72h` in
`publisher.go`), meaning any event not consumed within 72 hours is lost.

---

## 2. GDPR Breach Notification

GDPR Articles 33 and 34 impose strict timelines for breach reporting. Audit
logs are the primary forensic evidence used to determine scope, timeline, and
notification obligations.

### 2.1 Article 33 — Notify Supervisory Authority (72 Hours)

| Requirement | Detail |
|---|---|
| **Timeline** | Within **72 hours** of becoming aware of the breach |
| **Who** | The data controller to the competent supervisory authority (e.g., ICO, CNIL, DPB) |
| **Content** | Nature of breach, categories and approximate number of data subjects, likely consequences, measures taken |
| **Exemption** | Breach unlikely to result in risk to rights and freedoms of natural persons |

### 2.2 Article 34 — Notify Data Subjects (Without Undue Delay)

| Requirement | Detail |
|---|---|
| **Timeline** | Without **undue delay** when breach is likely to result in **high risk** |
| **Content** | Description of nature, DPO contact point, likely consequences, safeguards |
| **Exemption** | Data encrypted/anonymized, subsequent measures ensure high risk no longer likely, disproportionate effort (then public communication instead) |

### 2.3 What Constitutes a Breach

- **Confidentiality breach**: Unauthorized disclosure of or access to personal
  data (exfiltration, misconfigured S3 bucket, shared credentials).
- **Integrity breach**: Unauthorized alteration of personal data (SQL injection
  UPDATE, ransomware).
- **Availability breach**: Accidental or unlawful destruction or loss of access
  to personal data (ransomware lockout, DB deletion, cloud outage).

### 2.4 Audit Log Requirements for Breach Evidence

GDPR Article 33(5) requires the controller to document breaches — the audit
trail is the evidence. Logs must support:

1. **Detection timestamp** — when the breach was first identified.
2. **Scope determination** — which records, users, tenants were affected.
3. **Root cause analysis** — the initial access vector.
4. **Timeline reconstruction** — from initial compromise to containment.
5. **Data subject enumeration** — exact list of affected individuals.

### 2.5 Breach Detection Alerting

```go
// BreachDetector monitors audit events for indicators of a security breach.
type BreachDetector struct {
	pub     *audit.Publisher
	notifier BreachNotifier
	// Thresholds
	failedLoginThreshold   int           // e.g., 50 in 10 min → brute force
	failedLoginWindow      time.Duration
	massExportThreshold    int           // e.g., >100 records/min
	adminConfigChangeAlert bool
}

type BreachAlert struct {
	BreachID     string    `json:"breach_id"`
	DetectedAt   time.Time `json:"detected_at"`
	TenantID     string    `json:"tenant_id"`
	BreachType   string    `json:"breach_type"` // brute_force | mass_export | privilege_abuse | config_tamper
	Severity     string    `json:"severity"`    // low | medium | high | critical
	Description  string    `json:"description"`
	AffectedUsers []string `json:"affected_users"`
	Evidence     []string  `json:"evidence"` // audit event IDs
	NotificationDeadline time.Time `json:"notification_deadline"`
}

// DetectBreach checks recent audit events for breach indicators.
// Called periodically (every 5 minutes) or triggered by real-time event stream.
func (d *BreachDetector) DetectBreach(ctx context.Context, tenantID uuid.UUID) ([]BreachAlert, error) {
	var alerts []BreachAlert
	since := time.Now().UTC().Add(-d.failedLoginWindow)

	// Query failed logins in the detection window.
	events, _, err := d.auditSvc.ListEvents(ctx, domain.ListFilter{
		TenantID:  tenantID,
		Action:    "user.login",
		Result:    domain.ResultFailure,
		StartTime: &since,
	}, 1, 500)
	if err != nil {
		return nil, err
	}

	// Group by IP to detect brute-force patterns.
	ipCounts := make(map[string]int)
	for _, e := range events {
		ipCounts[e.IPAddress]++
	}
	for ip, count := range ipCounts {
		if count >= d.failedLoginThreshold {
			deadline := time.Now().UTC().Add(72 * time.Hour)
			alerts = append(alerts, BreachAlert{
				BreachID:             uuid.NewString(),
				DetectedAt:           time.Now().UTC(),
				TenantID:             tenantID.String(),
				BreachType:           "brute_force",
				Severity:             "high",
				Description:          fmt.Sprintf("%d failed logins from %s in %v", count, ip, d.failedLoginWindow),
				NotificationDeadline: deadline,
			})
		}
	}

	// Emit breach-detected audit event for Article 33(5) documentation.
	for i := range alerts {
		d.pub.Publish(ctx, audit.Event{
			TenantID:  tenantID,
			ActorType: "system",
			Action:    "security.breach_detected",
			Result:    "success",
			Metadata: map[string]any{
				"breach_id":             alerts[i].BreachID,
				"breach_type":           alerts[i].BreachType,
				"gdpr_art33_deadline":   alerts[i].NotificationDeadline.Format(time.RFC3339),
			},
		})
	}

	return alerts, nil
}
```

---

## 3. SOX Change Management Logs

Sarbanes-Oxley Section 404 requires management to assess and auditors to attest
to the effectiveness of **Internal Control over Financial Reporting (ICFR)**.
IT general controls (ITGCs) supporting financial systems must demonstrate
controlled change management.

### 3.1 Required Audit Trail Elements

For every production change to a financial-impacting system, SOX auditors
require evidence of the full chain:

| Stage | Required Evidence | Audit Field |
|---|---|---|
| **Request** | Ticket/CR with business justification | `change_ticket_id` |
| **Approval** | Approver identity and timestamp (separate from requester) | `approver_id`, `approved_at` |
| **Execution** | Who deployed and when | `deployed_by`, `deployed_at` |
| **What changed** | Code commit(s), config diff | `commit_sha`, `config_diff` |
| **Verification** | Post-deployment test results | `verification_status` |

### 3.2 Segregation of Duties (SoD)

SOX ITGCs require that the person who **develops** code is not the same person
who **approves** or **deploys** it. The audit trail must demonstrate this
separation by recording distinct `developer_id`, `approver_id`, and
`deployer_id` values for each change.

### 3.3 Change Management Audit Event

```go
// ChangeManagementEvent records a SOX-compliant change management trail.
type ChangeManagementEvent struct {
	EventID        string    `json:"event_id"`
	Timestamp      time.Time `json:"timestamp"`
	TenantID       string    `json:"tenant_id"`

	// Request linkage
	ChangeTicketID string `json:"change_ticket_id"`
	ChangeRequester string `json:"change_requester"` // who filed the ticket

	// Approval (SoD: approver ≠ requester ≠ deployer)
	ApproverID   string    `json:"approver_id"`
	ApprovedAt   time.Time `json:"approved_at"`

	// Execution
	DeployerID   string    `json:"deployer_id"`
	DeployedAt   time.Time `json:"deployed_at"`
	Environment  string    `json:"environment"` // production | staging

	// What changed
	CommitSHA    string `json:"commit_sha"`
	ServiceName  string `json:"service_name"`
	ConfigDiff   string `json:"config_diff,omitempty"`

	// Verification
	PostDeployVerified   bool   `json:"post_deploy_verified"`
	VerificationMethod   string `json:"verification_method"` // automated_test | manual_check | smoke_test
}

// SoDViolation checks for segregation of duties violations.
// Returns true if any of the three roles overlap.
func (e ChangeManagementEvent) SoDViolation() bool {
	return e.ChangeRequester == e.ApproverID ||
		e.ChangeRequester == e.DeployerID ||
		e.ApproverID == e.DeployerID
}

// EmitChangeManagementEvent publishes a SOX-compliant change event.
func EmitChangeManagementEvent(ctx context.Context, pub *audit.Publisher, e ChangeManagementEvent) error {
	if e.SoDViolation() {
		// Emit SoD violation alert — this is a SOX finding.
		_ = pub.Publish(ctx, audit.Event{
			TenantID:  uuid.MustParse(e.TenantID),
			ActorType: "system",
			Action:    "compliance.sod_violation",
			Result:    "denied",
			Metadata: map[string]any{
				"change_ticket_id": e.ChangeTicketID,
				"requester":        e.ChangeRequester,
				"approver":         e.ApproverID,
				"deployer":         e.DeployerID,
				"sox_control":      "ITGC-CM-001",
			},
		})
		return fmt.Errorf("segregation of duties violation: overlapping roles in change %s", e.ChangeTicketID)
	}

	return pub.Publish(ctx, audit.Event{
		TenantID:     uuid.MustParse(e.TenantID),
		ActorType:    "user",
		ActorID:      uuid.MustParse(e.DeployerID),
		Action:       "change.deploy",
		Result:       "success",
		ResourceType: "service",
		ResourceName: e.ServiceName,
		Metadata: map[string]any{
			"change_ticket_id":    e.ChangeTicketID,
			"change_requester":    e.ChangeRequester,
			"approver_id":         e.ApproverID,
			"approved_at":         e.ApprovedAt.Format(time.RFC3339),
			"commit_sha":          e.CommitSHA,
			"environment":         e.Environment,
			"post_deploy_verified": e.PostDeployVerified,
			"sod_compliant":       true,
		},
	})
}
```

### 3.4 SOX Gap in GGID

GGID currently has no change management audit events. The audit taxonomy
supports `user.login`, `role.assign`, etc., but there is no `change.deploy`
action or change-ticket linkage. GGID's CI/CD pipeline (GitHub Actions) has
commit history but does not feed deployment events into the audit stream.

---

## 4. ISO 27001:2022 A.12.4 Logging and Monitoring

ISO 27001:2022 Annex A.8 (renumbered from A.12 in the 2013 version) covers
logging. The 2022 update consolidated controls but the logging requirements
remain functionally identical.

### 4.1 Key Controls

| Control | Requirement |
|---|---|
| **A.8.15 (formerly A.12.4.1)** | Event logging: record user activities, exceptions, faults, and information security events |
| **A.8.15 (formerly A.12.4.2)** | Protection of log information: tamper-proofing, access controls, integrity monitoring |
| **A.8.17 (formerly A.12.4.3)** | Administrator and operator logs: log all privileged operations |
| **A.8.15** | Clock synchronization: all systems must use synchronized time sources (NTP/PTP) |
| **A.5.30 (formerly A.12.4.4)** | Information security event logging is integrated with incident management |

### 4.2 What to Log

- **User activity**: login/logout, resource access, privilege use, data exports.
- **Exceptions and faults**: application errors, system crashes, failed transactions.
- **Information security events**: authentication failures, authorization denials,
  intrusion detection alerts, malware detections.
- **Administrator/root activity**: every privileged command, every configuration
  change, every access to sensitive data by an admin.

### 4.3 Clock Synchronization

ISO 27001 requires all systems generating audit events to share a common,
authoritative time source. GGID's audit `CreatedAt` field uses `time.Now()` in
the publisher — if publishers run on different machines with unsynchronized
clocks, event ordering becomes unreliable. The Audit Service should override
`CreatedAt` with server-side timestamp on receipt (it partially does this in
`processMessage` if `CreatedAt.IsZero()`).

```go
// ISO27001Logger wraps the audit publisher to enforce ISO 27001 requirements:
//   1. Clock synchronization (server-side timestamp override)
//   2. Privileged operation flagging
//   3. Log integrity (hash chain)
type ISO27001Logger struct {
	pub      *audit.Publisher
	clock    func() time.Time // injectable for testing
	prevHash atomic.Value     // string
	hmacKey  []byte
}

// Log enforces ISO 27001 A.8.15 requirements.
func (l *ISO27001Logger) Log(ctx context.Context, e audit.Event) error {
	// ISO A.8.15 — clock synchronization: override client timestamp.
	e.CreatedAt = l.clock().UTC()

	// Hash chain for log protection (A.8.15 tamper-proofing).
	prev := l.prevHash.Load().(string)
	payload := fmt.Sprintf("%s|%s|%s|%s|%s", e.ID, e.CreatedAt.Format(time.RFC3339Nano),
		e.TenantID, e.ActorID, e.Action)
	h := hmac.New(sha256.New, l.hmacKey)
	h.Write([]byte(payload + prev))
	currHash := hex.EncodeToString(h.Sum(nil))
	l.prevHash.Store(currHash)

	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata["iso27001_hash"] = currHash
	e.Metadata["iso27001_prev_hash"] = prev

	// ISO A.8.17 — flag privileged operations for priority review.
	if e.ActorType == "user" && (strings.Contains(e.Action, "role.") ||
		strings.Contains(e.Action, "config.") ||
		strings.Contains(e.Action, "policy.")) {
		e.Metadata["iso27001_privileged_op"] = true
	}

	return l.pub.Publish(ctx, e)
}
```

### 4.4 ISO 27001 Gap in GGID

- The `Hash` field exists in the domain model but is **never populated or
  stored** — the `Insert` query does not include it.
- No NTP enforcement or clock-drift detection.
- No privileged-operation tagging — admin actions look identical to user actions.

---

## 5. HIPAA §164.312 Audit Controls

HIPAA Security Rule §164.312(b) requires: "Implement hardware, software, and/or
procedural mechanisms that record and examine activity in systems that contain
or use electronic protected health information (ePHI)."

### 5.1 What Must Be Logged

| Category | Required Events |
|---|---|
| **ePHI access** | Every read, write, update, or deletion of electronic Protected Health Information |
| **User authentication** | Login success/failure, session start/end, MFA challenge |
| **Administrative actions** | User provisioning, role changes, audit config changes |
| **Data disclosures** | Any sharing, export, or transmission of ePHI |
| **System access** | All access to systems containing ePHI (including by administrators) |

### 5.2 Retention

- **HIPAA §164.316(b)(2)**: Documentation (including audit logs) must be
  retained for a minimum of **6 years** from creation or last effective date.
- State laws may require longer: Texas (10 years), New York (6 years adult /
  6 years + minority for minors).

### 5.3 Breach Notification (HIPAA §164.404)

| Requirement | Timeline |
|---|---|
| Notify affected individuals | **60 days** from discovery |
| Notify HHS Secretary | **60 days** (for breaches affecting 500+ individuals) |
| Notify media (500+ residents) | **60 days** from discovery |
| Breach of <500 individuals | Annual report to HHS within 60 days of end of calendar year |

### 5.4 HIPAA Audit Event

```go
// HIPAAAuditEvent records HIPAA-compliant audit events for ePHI access.
type HIPAAAuditEvent struct {
	EventID    string    `json:"event_id"`
	Timestamp  time.Time `json:"timestamp"`
	TenantID   string    `json:"tenant_id"`

	// Who
	UserID       string `json:"user_id"`
	UserRole     string `json:"user_role"` // physician | nurse | admin | technician
	UserName     string `json:"user_name"`
	SourceIP     string `json:"source_ip"`
	WorkstationID string `json:"workstation_id"`

	// What
	Action       string `json:"action"` // ephi.read | ephi.write | ephi.delete | ephi.export
	PatientID    string `json:"patient_id"`    // pseudonymized MRN
	RecordType   string `json:"record_type"`   // lab_result | medication | diagnosis | imaging
	ResourceID   string `json:"resource_id"`

	// Why (minimum necessary standard)
	Purpose      string `json:"purpose"` // treatment | payment | operations | research

	// Outcome
	Result       string `json:"result"`
}

// EmitHIPAAEvent publishes a HIPAA-compliant audit event.
func EmitHIPAAEvent(ctx context.Context, pub *audit.Publisher, e HIPAAAuditEvent) error {
	if e.EventID == "" {
		e.EventID = uuid.NewString()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	// HIPAA requires pseudonymization of patient identifiers in logs.
	// Use HMAC-SHA256(patientID, key) to create a deterministic pseudonym.
	pseudonym := pseudonymizePatientID(e.PatientID)

	return pub.Publish(ctx, audit.Event{
		ID:           uuid.MustParse(e.EventID),
		TenantID:     uuid.MustParse(e.TenantID),
		ActorType:    "user",
		ActorID:      uuid.MustParse(e.UserID),
		ActorName:    e.UserName,
		Action:       e.Action,
		ResourceType: e.RecordType,
		ResourceID:   uuid.MustParse(e.ResourceID),
		Result:       e.Result,
		IPAddress:    e.SourceIP,
		CreatedAt:    e.Timestamp,
		Metadata: map[string]any{
			"hipaa_user_role":      e.UserRole,
			"hipaa_workstation_id": e.WorkstationID,
			"hipaa_patient_pseudonym": pseudonym,
			"hipaa_purpose":        e.Purpose,
			"hipaa_minimum_necessary": true,
		},
	})
}

func pseudonymizePatientID(patientID string) string {
	h := hmac.New(sha256.New, patientPseudonymKey)
	h.Write([]byte(patientID))
	return "pat_" + hex.EncodeToString(h.Sum(nil))[:16]
}
```

### 5.5 HIPAA Gap in GGID

GGID has no patient/clinical data model and no minimum-necessary logging. The
audit taxonomy has no `ephi.*` actions. However, GGID's IAM audit events
(authentication, role changes) partially satisfy HIPAA audit control
requirements for access management.

---

## 6. SOC 2 Type II Logging Requirements

SOC 2 audits assess controls against the AICPA Trust Service Criteria (TSC).
The primary criteria relevant to audit logging are:

### 6.1 Applicable Trust Service Criteria

| TSC | Control Area | Audit Logging Implication |
|---|---|---|
| **CC7.1** | Detection and monitoring | Consistent logging of all security-relevant events |
| **CC7.2** | Anomaly detection | Automated monitoring with alerting for anomalies |
| **CC7.3** | Incident response | Evidence trail for security incidents |
| **A1.2** | Environmental protection | System monitoring and capacity logging |
| **C1.1** | Confidentiality | Access control and data access logging |

### 6.2 What Auditors Examine

1. **Consistent logging coverage**: Do all critical paths generate audit events?
   Auditors sample transactions and verify corresponding log entries.
2. **Tamper protection**: Can a user modify or delete logs? Auditors verify
   access controls and integrity mechanisms.
3. **Time-series integrity**: Are timestamps reliable? Is there evidence of
   clock synchronization?
4. **Access control on logs**: Who can read, query, or export audit data?
   Auditors verify that log access is restricted to authorized security personnel.
5. **Retention enforcement**: Are logs retained per policy? Auditors verify
   retention configuration and test deletion controls.
6. **Monitoring evidence**: Is there evidence of regular log review? Auditors
   look for daily review logs, alert acknowledgment records, and incident tickets.

### 6.3 Evidence Samples

SOC 2 auditors typically request:
- **Population listing**: A complete list of all audit events for a selected
  period (e.g., one month). GGID's `ListEvents` API provides this.
- **Sample testing**: Randomly select 25-60 events and trace each to its
  triggering system action (forward tracing) and from action to log (backward
  tracing).
- **Access review**: Who queried the audit API during the audit period? GGID's
  audit-on-audit (logging access to audit endpoints) is currently **not
  implemented**.
- **Tamper test**: Attempt to modify or delete a log entry via the API and
  verify it fails. GGID's API has no DELETE endpoint — good — but direct DB
  access is uncontrolled.

### 6.4 Gap Remediation

| Gap | Remediation | Priority |
|---|---|---|
| No audit-on-audit (who queried the audit API) | Log all audit query API calls as audit events | High |
| No access control on audit API beyond tenant_id | Add role-based access: only `audit:reader` role can query | High |
| No tamper-evidence (hash field unused) | Implement hash chain (see tampering-detection doc) | Critical |
| No evidence of daily review | Implement daily audit summary report + acknowledgment workflow | Medium |

---

## 7. Log Retention Matrix

### 7.1 Per-Framework Retention Requirements

| Framework | Online (Hot) | Archive (Cold) | Total Minimum | Reference |
|---|---|---|---|---|
| **PCI-DSS v4.0** | 12 months | 12 months | **2 years** | Req. 10.5.1–10.5.2 |
| **SOX** | Not specified | Not specified | **7 years** | SEC Rule 17a-4 / Sarbanes-Oxley §802 |
| **HIPAA** | Not specified | Not specified | **6 years** | §164.316(b)(2) |
| **ISO 27001** | Per policy | Per policy | **1–5 years** (typical) | A.8.15 (organization-defined) |
| **SOC 2** | 12 months | Per policy | **12 months min** | CC7.1 (auditor expectation) |
| **GDPR** | Not specified | Not specified | **As long as needed** | Art. 5(1)(e) storage limitation |
| **CCPA/CPRA** | Not specified | Not specified | **As long as needed** | §1798.100(d) |

### 7.2 Multi-Framework Strategy

When subject to multiple frameworks, the **longest retention period wins**.
For a system subject to PCI + SOX + HIPAA + GDPR:

```
Minimum retention = max(2y, 7y, 6y, "as needed") = 7 years (SOX)
```

### 7.3 Tiered Storage Architecture

```
Hot (0–30 days)       → PostgreSQL (SSD, queryable via API)
Warm (30 days–1 year)  → PostgreSQL on cheaper storage OR Parquet on S3
Cold (1–3 years)       → S3/GCS compressed Parquet, query via Athena
Archive (3–7+ years)   → Glacier/Archive tier, queryable within 72h (PCI req)
```

### 7.4 Retention Policy Enforcement

```go
// RetentionPolicy enforces multi-framework audit log retention.
type RetentionPolicy struct {
	HotRetentionDays   int // e.g., 90 (GGID default)
	WarmRetentionDays  int // e.g., 365
	ColdRetentionDays  int // e.g., 1095 (3 years)
	ArchiveRetentionDays int // e.g., 2555 (7 years, SOX)
}

// DefaultMultiFrameworkPolicy returns a retention policy satisfying the
// strictest requirements across PCI, SOX, HIPAA, ISO, and SOC 2.
func DefaultMultiFrameworkPolicy() RetentionPolicy {
	return RetentionPolicy{
		HotRetentionDays:     90,    // PostgreSQL SSD
		WarmRetentionDays:    365,   // PostgreSQL HDD or Parquet
		ColdRetentionDays:    1095,  // S3 Parquet (3 years)
		ArchiveRetentionDays: 2555,  // Glacier (7 years, SOX max)
	}
}

// EnforceRetention moves events through the storage tiers.
func (p RetentionPolicy) EnforceRetention(ctx context.Context, svc *AuditService) error {
	now := time.Now().UTC()

	// Tier 1→2: Move events older than HotRetentionDays to warm storage.
	hotCutoff := now.AddDate(0, 0, -p.HotRetentionDays)
	if _, err := svc.ArchiveToWarm(ctx, hotCutoff); err != nil {
		return fmt.Errorf("hot→warm archive: %w", err)
	}

	// Tier 2→3: Move events older than WarmRetentionDays to cold storage.
	warmCutoff := now.AddDate(0, 0, -p.WarmRetentionDays)
	if _, err := svc.ArchiveToCold(ctx, warmCutoff); err != nil {
		return fmt.Errorf("warm→cold archive: %w", err)
	}

	// Tier 3→4: Move events older than ColdRetentionDays to archive storage.
	coldCutoff := now.AddDate(0, 0, -p.ColdRetentionDays)
	if _, err := svc.ArchiveToGlacier(ctx, coldCutoff); err != nil {
		return fmt.Errorf("cold→archive: %w", err)
	}

	// Permanent deletion only after ArchiveRetentionDays (SOX 7-year max).
	archiveCutoff := now.AddDate(0, 0, -p.ArchiveRetentionDays)
	if _, err := svc.DeleteFromArchive(ctx, archiveCutoff); err != nil {
		return fmt.Errorf("archive purge: %w", err)
	}

	return nil
}
```

### 7.5 Current GGID Retention Gap

GGID's `CleanupOldEvents` uses a **flat 90-day retention** with no tiering:
```go
// services/audit/internal/service/audit_service.go
func (s *AuditService) CleanupOldEvents(ctx context.Context, retentionDays int) (int64, error) {
    if retentionDays <= 0 {
        retentionDays = 90 // default 90 days
    }
    before := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
    return s.repo.DeleteOlderThan(ctx, before)
}
```
This violates PCI (12 months), SOX (7 years), and HIPAA (6 years).

---

## 8. PII in Audit Logs

Audit logs themselves contain personal data: IP addresses, user IDs, email
addresses, usernames, user agents. Under GDPR, audit logs are subject to the
same data protection principles as any other personal data.

### 8.1 PII Categories in GGID Audit Events

| Field | PII Type | GDPR Risk |
|---|---|---|
| `ActorID` | Pseudonymous identifier | Low (but linkable) |
| `ActorName` | Direct identifier (name/email) | High |
| `IPAddress` | Online identifier | Medium (Art. 4(1) includes IP) |
| `UserAgent` | May contain device fingerprints | Low–Medium |
| `Metadata` (arbitrary) | May contain email, phone, etc. | Variable |

### 8.2 GDPR Obligations for Audit Logs

1. **Lawful basis**: Process audit data under Art. 6(1)(c) — compliance with
   legal obligation (logging is legally required by PCI, SOX, etc.).
2. **Storage limitation**: Retain only as long as the legal obligation requires.
3. **Data minimization**: Log only what is necessary for compliance.
4. **Data subject rights**: Audit logs may be exempt from access/erasure rights
   when processing is for compliance (Art. 17(3)(b)–(e) exemptions), but the
   controller must document the basis for refusal.
5. **Security**: Audit logs must be encrypted at rest (Art. 32).

### 8.3 Pseudonymization

Replace direct identifiers with pseudonyms before logging:

```go
// PIISafeLogger wraps the audit publisher to pseudonymize PII fields.
type PIISafeLogger struct {
	pub        *audit.Publisher
	pseudonymizer *Pseudonymizer
}

// Pseudonymizer maps real identifiers to stable pseudonyms using HMAC-SHA256.
type Pseudonymizer struct {
	key []byte
}

func NewPseudonymizer(key []byte) *Pseudonymizer {
	return &Pseudonymizer{key: key}
}

// Pseudonymize produces a deterministic, irreversible pseudonym.
func (p *Pseudonymizer) Pseudonymize(identifier string) string {
	if identifier == "" {
		return ""
	}
	h := hmac.New(sha256.New, p.key)
	h.Write([]byte(identifier))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// PseudonymizeIP masks the last octet of IPv4 / last 80 bits of IPv6.
func (p *Pseudonymizer) PseudonymizeIP(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return p.Pseudonymize(ip) // fallback to HMAC
	}
	if parsed.To4() != nil {
		// IPv4: zero last octet (192.168.1.42 → 192.168.1.0)
		return parsed.Mask(net.CIDRMask(24, 32)).String()
	}
	// IPv6: zero last 80 bits
	return parsed.Mask(net.CIDRMask(48, 128)).String()
}

// LogPIISafe publishes an audit event with PII pseudonymized.
func (l *PIISafeLogger) LogPIISafe(ctx context.Context, e audit.Event) error {
	// Pseudonymize actor ID and name.
	if e.ActorID != uuid.Nil {
		e.ActorName = "user_" + l.pseudonymizer.Pseudonymize(e.ActorID.String())[:8]
	}
	// Pseudonymize IP address.
	if e.IPAddress != "" {
		e.IPAddress = l.pseudonymizer.PseudonymizeIP(e.IPAddress)
	}
	// Scrub PII from metadata.
	if e.Metadata != nil {
		e.Metadata = l.scrubMetadata(e.Metadata)
	}
	return l.pub.Publish(ctx, e)
}

func (l *PIISafeLogger) scrubMetadata(m map[string]any) map[string]any {
	scrubbed := make(map[string]any, len(m))
	piiKeys := map[string]bool{
		"email": true, "phone": true, "address": true,
		"ssn": true, "dob": true, "name": true,
	}
	for k, v := range m {
		lk := strings.ToLower(k)
		if piiKeys[lk] || strings.Contains(lk, "email") || strings.Contains(lk, "phone") {
			scrubbed[k] = "[REDACTED]"
		} else {
			scrubbed[k] = v
		}
	}
	return scrubbed
}
```

### 8.4 Access Control for Audit Queries

Audit logs should be accessible only to authorized security/compliance roles.
GGID currently filters by `tenant_id` but has **no role-based access control**
on the audit query API. Any authenticated user within a tenant can query all
audit events for that tenant.

### 8.5 Encryption at Rest

GDPR Art. 32 requires "encryption of personal data" where appropriate. GGID's
audit events are stored in PostgreSQL without column-level encryption or TDE.
The `Metadata` JSONB field may contain PII in cleartext. Recommendations:
- Enable PostgreSQL TDE (pgcrypto `pgp_sym_encrypt` for sensitive metadata).
- Use a KMS-managed key for envelope encryption of archived audit data (S3 SSE-KMS).

---

## 9. GGID Audit Compliance Matrix

### 9.1 Current Implementation Summary

Based on review of `pkg/audit/publisher.go`, `services/audit/internal/`:

| Capability | Status | Details |
|---|---|---|
| Event capture (who/what/when/where) | **Implemented** | `Event` struct captures actor, action, timestamp, IP |
| Asynchronous publishing (NATS) | **Implemented** | `Publisher.Publish` via JetStream, fire-and-forget async |
| Persistence (PostgreSQL) | **Implemented** | Monthly range partitions, indexed by tenant/actor/action |
| Query API (gRPC + REST) | **Implemented** | `ListEvents`, `GetEvent` with filtering and pagination |
| Analytics/stats | **Implemented** | `GetStats` — 24h aggregates, failed login counts |
| Hash chain / tamper-evidence | **Stub only** | `Hash` field in domain model but never populated or stored |
| Retention enforcement | **Partial** | `CleanupOldEvents` with flat 90-day default, no tiering |
| PII pseudonymization | **Not implemented** | Raw actor names, IPs stored in cleartext |
| Encryption at rest | **Not implemented** | No TDE, no column encryption, metadata in plaintext JSONB |
| Breach detection | **Not implemented** | No anomaly detection, no alerting |
| Change management audit | **Not implemented** | No `change.deploy` events, no ticket linkage |
| Audit-on-audit | **Not implemented** | Audit API calls are not themselves audited |
| NTP/clock synchronization | **Not enforced** | Client-side `time.Now()`, no drift detection |
| Access control on audit API | **Partial** | Tenant-scoped but no role check |

### 9.2 Compliance Mapping

| Requirement | PCI-DSS | SOX | ISO 27001 | HIPAA | SOC 2 | GDPR |
|---|---|---|---|---|---|---|
| Auth event logging | PASS | PASS | PASS | PASS | PASS | PASS |
| Admin action logging | PARTIAL | — | FAIL | FAIL | PARTIAL | — |
| Config change logging | PARTIAL | FAIL | PARTIAL | PARTIAL | PARTIAL | — |
| Tamper-evidence | FAIL | FAIL | FAIL | FAIL | FAIL | FAIL |
| 12-month retention | FAIL | — | PARTIAL | — | FAIL | PASS |
| 7-year retention | — | FAIL | — | — | — | — |
| PII pseudonymization | — | — | — | — | — | FAIL |
| Encryption at rest | PARTIAL | — | — | FAIL | — | FAIL |
| Breach detection | FAIL | — | FAIL | FAIL | FAIL | FAIL |
| Change management trail | — | FAIL | — | — | — | — |
| NTP synchronization | PARTIAL | — | FAIL | — | PARTIAL | — |

**Legend**: PASS = fully compliant · PARTIAL = structure exists but incomplete · FAIL = not implemented

### 9.3 Key Observation

GGID's audit infrastructure captures the right **events** (authentication, role
changes, resource access) but lacks the **compliance enforcement** layer:
tamper-evidence, retention, pseudonymization, and breach detection. The `Hash`
field's existence without implementation is particularly misleading — it implies
tamper-evidence that does not exist.

---

## 10. Gap Analysis and Recommendations

### 10.1 Prioritized Action Items

| Priority | Gap | Framework(s) | Effort | Recommendation |
|---|---|---|---|---|
| **P0 — Critical** | Hash chain not implemented (field exists, never used) | PCI, SOX, ISO, HIPAA, SOC 2 | **3–5 days** | Wire HMAC hash chain in consumer: compute hash on persist, store in `hash` column, add `verify_chain` API endpoint |
| **P0 — Critical** | Retention violates all frameworks (90-day flat) | PCI (12mo), SOX (7yr), HIPAA (6yr) | **2–3 days** | Replace flat `CleanupOldEvents` with tiered `RetentionPolicy`; add S3 archival for cold/archive tiers; set default to 2555 days (SOX max) |
| **P1 — High** | No PII pseudonymization in audit logs | GDPR | **2 days** | Wrap `Publisher.Publish` with `PIISafeLogger`; pseudonymize `ActorName`, `IPAddress`, and PII in metadata |
| **P1 — High** | No access control on audit query API | SOC 2, HIPAA, ISO 27001 | **1 day** | Require `audit:reader` scope on the `ListEvents`/`GetEvent` gRPC and REST endpoints; add audit-on-audit logging for all queries |
| **P2 — Medium** | No breach detection/alerting | GDPR Art. 33, PCI 10.4 | **3–4 days** | Implement `BreachDetector` with rules for brute force, mass export, config tamper; emit `security.breach_detected` events; integrate with notification service |
| **P2 — Medium** | No change management audit trail | SOX §404 | **2 days** | Add `change.deploy` action type; wire CI/CD pipeline to emit deployment events with ticket, approver, commit SHA, SoD check |

### 10.2 Implementation Sequencing

```
Phase 1 (Week 1–2): P0 items
  ├── Wire hash chain into consumer and repository
  ├── Implement tiered retention policy
  └── Add migration for `hash` and `prev_hash` columns

Phase 2 (Week 3): P1 items
  ├── PII-safe logger wrapper
  ├── Audit API RBAC enforcement
  └── Audit-on-audit logging

Phase 3 (Week 4–5): P2 items
  ├── Breach detection rules engine
  ├── Notification integration (email/Slack/IM)
  └── Change management event pipeline (CI/CD integration)
```

### 10.3 Effort Summary

| Phase | Items | Estimated Effort | Compliance Risk Mitigated |
|---|---|---|---|
| Phase 1 | Hash chain + retention | 5–8 days | All frameworks (tamper-evidence + retention) |
| Phase 2 | PII + RBAC | 3 days | GDPR + SOC 2 + HIPAA (privacy + access control) |
| Phase 3 | Breach + change mgmt | 5–6 days | GDPR Art. 33 + SOX §404 (detection + ITGC) |
| **Total** | | **13–17 days** | Full multi-framework compliance coverage |

### 10.4 Long-Term Recommendations

1. **RFC 3161 timestamping**: Integrate trusted timestamping for legally
   defensible evidence (see `audit-tampering-detection.md` §6).
2. **WORM storage**: Implement append-only storage at the database level (REVOKE
   DELETE/UPDATE on `audit_events` for the application role).
3. **SIEM integration**: Export audit events to Splunk/Elastic/Sumo Logic via
   Syslog or Kafka for external correlation and retention.
4. **Quarterly compliance review**: Automate a compliance dashboard showing
   event coverage, retention health, hash chain integrity, and access patterns.
5. **Legal hold capability**: Implement litigation hold that suspends retention
   deletion for specific tenants/events when legal proceedings are anticipated.

---

*This document provides compliance-specific audit requirements. For
cryptographic tamper-evidence implementation details, see
`audit-tampering-detection.md`. For a high-level framework overview, see
`audit-log-compliance.md`.*
