-- KB-337: Compliance framework mappings (SOC2/ISO27001/CCM)
CREATE TABLE IF NOT EXISTS compliance_mappings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    framework      TEXT NOT NULL,
    trust_category TEXT NOT NULL DEFAULT '',
    control_id     TEXT NOT NULL,
    control_name   TEXT NOT NULL,
    ggid_feature   TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'covered',
    evidence_query TEXT NOT NULL DEFAULT '',
    ccm_control_id TEXT NOT NULL DEFAULT '',
    description    TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (framework, control_id)
);

CREATE INDEX IF NOT EXISTS idx_compliance_framework
    ON compliance_mappings (framework, control_id);
CREATE INDEX IF NOT EXISTS idx_compliance_tenant
    ON compliance_mappings (tenant_id, framework);

-- Seed SOC2 Type II mappings (5 trust principles)
INSERT INTO compliance_mappings (framework, trust_category, control_id, control_name, ggid_feature, status, evidence_query, ccm_control_id, description) VALUES
('soc2', 'Security',         'CC6.1', 'Logical and Physical Access Controls',     'Auth + MFA + Password Policy',  'covered', 'SELECT count(*) FROM users WHERE mfa_enabled = true',                  'password_policy_compliance',  'GGID enforces MFA, password complexity, and RBAC'),
('soc2', 'Security',         'CC6.2', 'User Authentication Credentials',          'Auth + Password History',       'covered', 'SELECT count(*) FROM password_history WHERE created_at > now() - interval ''7 days''', 'password_history_check',     'Password rotation, breach check, pepper rotation'),
('soc2', 'Security',         'CC6.3', 'Authorization Controls for Access',        'RBAC + ABAC + Policy Engine',   'covered', 'SELECT count(*) FROM role_assignments WHERE active = true',            'rbac_coverage',               'Role-based + attribute-based access control via policy service'),
('soc2', 'Security',         'CC6.6', 'Logical Access Security Measures',         'JWT + Session Timeout',         'covered', 'SELECT count(*) FROM sessions WHERE expires_at > now()',               'session_timeout_compliance',  'JWT-based session management with configurable timeout'),
('soc2', 'Availability',     'A1.1',  'System Monitoring and Health',             'Health Check + Backup',         'covered', 'SELECT count(*) FROM health_checks WHERE status = ''healthy'' AND checked_at > now() - interval ''1 hour''', 'system_availability',       'Kubernetes health probes, database backups, Redis HA'),
('soc2', 'Availability',     'A1.2', 'Environmental Protections',                'Rate Limiting + DDoS',          'partial', 'SELECT count(*) FROM rate_limit_events WHERE blocked = true',          'rate_limit_active',           'Per-tenant rate limiting and IP-based throttling'),
('soc2', 'Processing Integrity', 'PI1.1', 'Audit Chain Integrity',               'Hash Chain Audit Logging',      'covered', 'SELECT count(*) FROM audit_events WHERE hash_verified = true',         'audit_integrity',             'Tamper-evident hash chain for all audit events'),
('soc2', 'Processing Integrity', 'PI1.2', 'Error Handling',                       'Error Tracking + SIEM',         'partial', 'SELECT count(*) FROM error_logs WHERE resolved = true',                'error_handling',              'Structured error logging with SIEM integration'),
('soc2', 'Confidentiality',  'C1.1',  'Data Confidentiality Measures',            'RLS + Encryption at Rest',      'covered', 'SELECT count(*) FROM information_schema.tables WHERE table_name LIKE ''%rls%''', 'encryption_at_rest',      'Row-level security, field-level encryption, TLS in transit'),
('soc2', 'Confidentiality',  'C1.2', 'Data Transmission and Disposal',           'TLS 1.3 + Key Rotation',        'covered', 'SELECT count(*) FROM jwt_key_rotations WHERE rotated_at > now() - interval ''90 days''', 'key_rotation',          'TLS 1.3 everywhere, automatic JWT key rotation every 90 days'),
('soc2', 'Privacy',          'P1.1',  'Privacy Notice and Consent',               'GDPR Consent + DSR',            'partial', 'SELECT count(*) FROM dsr_requests WHERE status = ''completed''',       'gdpr_compliance',             'GDPR data subject rights, consent tracking, data minimization'),
('soc2', 'Privacy',          'P2.1',  'Data Retention and Disposal',              'Retention Policies + GDPR',     'partial', 'SELECT count(*) FROM retention_policies WHERE active = true',          'data_retention',              'Configurable retention periods, automated data deletion'),
('soc2', 'Security',         'CC7.1', 'System Operations Monitoring',             'Metrics + Grafana + Alerts',    'covered', 'SELECT count(*) FROM prometheus_metrics WHERE scrape_ok = true',       'monitoring_coverage',         'Full Prometheus metrics, Grafana dashboards, alerting rules'),
('soc2', 'Security',         'CC7.2', 'Anomaly Detection',                        'ITDR + UEBA + SIEM',            'covered', 'SELECT count(*) FROM itdr_detections WHERE severity IN (''high'',''critical'')', 'anomaly_detection',     'AI-driven threat detection, behavioral analytics, SIEM feed'),
('soc2', 'Security',         'CC8.1', 'Change Management',                        'GitOps + Migration Tools',      'partial', 'SELECT count(*) FROM schema_migrations ORDER BY version DESC LIMIT 1', 'change_management',      'Versioned migrations, CI/CD pipeline, audit trail for changes')
ON CONFLICT (framework, control_id) DO NOTHING;

-- Seed ISO 27001 Annex A mappings (key 10 controls)
INSERT INTO compliance_mappings (framework, trust_category, control_id, control_name, ggid_feature, status, evidence_query, ccm_control_id, description) VALUES
('iso27001', 'A.5',  'A.5.1',   'Policies for Information Security',        'Security Policy + Docs',        'covered', 'SELECT count(*) FROM security_policies WHERE active = true',          'policy_docs',                 'Documented information security policies and hardening guides'),
('iso27001', 'A.6',  'A.6.1.1', 'Information Security Roles and Responsibilities', 'RBAC + Admin Roles',       'covered', 'SELECT count(*) FROM role_assignments WHERE role_name LIKE ''%admin%''', 'admin_access',            'Role-based admin access with separation of duties checks'),
('iso27001', 'A.8',  'A.8.1.1', 'Inventory of Assets',                       'NHI Registry + Device Mgmt',    'partial', 'SELECT count(*) FROM non_human_identities WHERE active = true',        'asset_inventory',             'Non-human identity registry, device registration and tracking'),
('iso27001', 'A.8',  'A.8.2.1', 'Classification of Information',             'Data Classification + PII Tags','partial', 'SELECT count(*) FROM pii_fields WHERE classified = true',              'data_classification',         'PII field tagging, data classification metadata'),
('iso27001', 'A.9',  'A.9.1.1', 'Access Control Policy',                     'Policy Engine + ABAC',          'covered', 'SELECT count(*) FROM access_policies WHERE enabled = true',            'access_policy',               'Attribute-based access control policies with PDP engine'),
('iso27001', 'A.9',  'A.9.2.3', 'Management of Privileged Access Rights',    'PAM + Privilege Creep',         'partial', 'SELECT count(*) FROM privileged_sessions WHERE active = true',        'privileged_access',           'Privileged access management with session recording and creep detection'),
('iso27001', 'A.9',  'A.9.4.1', 'Information Access Restriction',            'RLS + Column-Level Security',   'covered', 'SELECT count(*) FROM rls_policies WHERE enabled = true',               'row_level_security',          'PostgreSQL row-level security with tenant isolation'),
('iso27001', 'A.12', 'A.12.4.1','Event Logging',                            'Audit Chain + SIEM',            'covered', 'SELECT count(*) FROM audit_events WHERE created_at > now() - interval ''1 hour''', 'audit_logging',         'Tamper-evident audit logging with hash chain verification'),
('iso27001', 'A.12', 'A.12.6.1','Management of Technical Vulnerabilities',  'Vuln Scanning + Dep Shield',    'partial', 'SELECT count(*) FROM dependency_scans WHERE critical = 0',            'vulnerability_management',    'Dependency scanning, secret detection, patch management'),
('iso27001', 'A.16', 'A.16.1.1','Incident Management',                      'ITDR + SoAR + Alerting',        'covered', 'SELECT count(*) FROM itdr_detections WHERE status = ''handled''',      'incident_response',           'Automated incident detection, SoAR playbook, webhook alerting')
ON CONFLICT (framework, control_id) DO NOTHING;
