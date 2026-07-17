-- 022_federation_entities.sql
-- Federation Hub: trust chain registry + assertion transform rules.

CREATE TABLE IF NOT EXISTS federation_entities (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    entity_id       TEXT NOT NULL,
    entity_name     TEXT NOT NULL,
    entity_type     TEXT NOT NULL DEFAULT 'idp',
    protocol        TEXT NOT NULL DEFAULT 'saml',
    metadata_url    TEXT,
    issuer          TEXT,
    trust_level     TEXT NOT NULL DEFAULT 'pending',
    trust_direction TEXT NOT NULL DEFAULT 'inbound',
    certificates    JSONB NOT NULL DEFAULT '[]',
    jwks_url        TEXT,
    expires_at      TIMESTAMPTZ,
    last_checked    TIMESTAMPTZ,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, entity_id, protocol)
);
CREATE INDEX IF NOT EXISTS idx_fed_entities_type ON federation_entities(tenant_id, entity_type, enabled);
CREATE INDEX IF NOT EXISTS idx_fed_entities_protocol ON federation_entities(tenant_id, protocol, enabled);

CREATE TABLE IF NOT EXISTS assertion_transform_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            TEXT NOT NULL,
    source_protocol TEXT NOT NULL,
    target_protocol TEXT NOT NULL,
    transform_type  TEXT NOT NULL,
    claim_mappings  JSONB NOT NULL DEFAULT '{}',
    claim_filters   JSONB NOT NULL DEFAULT '[]',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_transform_rules_tenant ON assertion_transform_rules(tenant_id, enabled);

CREATE TABLE IF NOT EXISTS federation_email_routes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    email_domain    TEXT NOT NULL,
    entity_id       TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, email_domain)
);
CREATE INDEX IF NOT EXISTS idx_email_routes_domain ON federation_email_routes(tenant_id, email_domain);

COMMENT ON TABLE federation_entities IS 'Federation Hub trust chain registry';
COMMENT ON TABLE assertion_transform_rules IS 'Cross-protocol assertion transformation rules';
