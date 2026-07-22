-- R2-01: DB-backed alert rules for ITDR alerting
CREATE TABLE IF NOT EXISTS alert_rules (
    id TEXT PRIMARY KEY,
    tenant_id UUID,
    rule_name TEXT NOT NULL,
    pattern TEXT NOT NULL,          -- audit event action pattern (e.g., "auth.login.failed")
    threshold INT DEFAULT 5,        -- number of matches within window
    window_minutes INT DEFAULT 10,  -- sliding window
    severity TEXT DEFAULT 'high',   -- info, low, medium, high, critical
    webhook_url TEXT,               -- per-rule override (empty = use global)
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);
COMMENT ON TABLE alert_rules IS 'Configurable alert rules for ITDR real-time alerting';
