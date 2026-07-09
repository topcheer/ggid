-- Enable pgcrypto for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create enums
CREATE TYPE actor_type AS ENUM ('user', 'api_key', 'system', 'anonymous');
CREATE TYPE event_result AS ENUM ('success', 'failure', 'denied');

-- Audit events table with monthly range partitioning
CREATE TABLE audit_events (
    id            UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    actor_type    actor_type NOT NULL DEFAULT 'user',
    actor_id      UUID,
    actor_name    VARCHAR(200) DEFAULT '',
    action        VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) DEFAULT '',
    resource_id   UUID,
    resource_name VARCHAR(200) DEFAULT '',
    result        event_result NOT NULL DEFAULT 'success',
    ip_address    INET,
    user_agent    TEXT DEFAULT '',
    request_id    VARCHAR(64) DEFAULT '',
    metadata      JSONB DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create initial partitions for current and next month
-- These will be replaced by a partitioning function in production,
-- but for the skeleton we create a few static partitions.

-- Default partition (catch-all for events outside defined partitions)
CREATE TABLE audit_events_default PARTITION OF audit_events DEFAULT;

-- Indexes on the parent table (inherited by all partitions)
CREATE INDEX idx_audit_tenant_created ON audit_events (tenant_id, created_at DESC);
CREATE INDEX idx_audit_actor ON audit_events (actor_id, created_at DESC);
CREATE INDEX idx_audit_action ON audit_events (action, created_at DESC);
CREATE INDEX idx_audit_resource ON audit_events (resource_type, resource_id);
CREATE INDEX idx_audit_result ON audit_events (result);
CREATE INDEX idx_audit_request_id ON audit_events (request_id);
