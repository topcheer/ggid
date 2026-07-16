-- Migration 10: ITDR (Identity Threat Detection & Response) tables
-- Detection engine rules and detections for real-time threat detection.

CREATE TABLE IF NOT EXISTS itdr_rules (
    id          TEXT NOT NULL,
    tenant_id   UUID NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    severity    TEXT,
    threshold   JSONB NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, tenant_id)
);

CREATE TABLE IF NOT EXISTS itdr_detections (
    id           UUID PRIMARY KEY,
    tenant_id    UUID NOT NULL,
    rule_id      TEXT NOT NULL,
    actor_id     UUID,
    severity     TEXT NOT NULL,
    title        TEXT NOT NULL,
    detail       JSONB NOT NULL DEFAULT '{}',
    event_ids    UUID[] NOT NULL DEFAULT '{}',
    status       TEXT NOT NULL DEFAULT 'new',
    hit_count    INT NOT NULL DEFAULT 1,
    detected_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_itdr_det_tenant_time ON itdr_detections (tenant_id, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_itdr_det_status ON itdr_detections (tenant_id, status, severity);
