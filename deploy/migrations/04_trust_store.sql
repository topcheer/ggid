-- Migration 04: Trust Store and Certificate Management
-- Stores trusted CA certificates and managed certificates (TLS, signing, JWT)

CREATE TABLE IF NOT EXISTS trusted_ca_certs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    name         TEXT NOT NULL,
    fingerprint  TEXT NOT NULL UNIQUE,
    subject      TEXT NOT NULL DEFAULT '',
    issuer       TEXT NOT NULL DEFAULT '',
    pem_data     TEXT NOT NULL,
    expiry_date  TIMESTAMPTZ NOT NULL,
    uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by  TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_trusted_ca_certs_tenant ON trusted_ca_certs(tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_trusted_ca_certs_fp ON trusted_ca_certs(tenant_id, fingerprint);

CREATE TABLE IF NOT EXISTS certificates (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    name         TEXT NOT NULL,
    type         TEXT NOT NULL DEFAULT 'TLS', -- TLS, signing, JWT
    issuer       TEXT NOT NULL DEFAULT '',
    fingerprint  TEXT NOT NULL,
    pem_data     TEXT NOT NULL,
    key_pem_data TEXT NOT NULL DEFAULT '',
    expiry_date  TIMESTAMPTZ NOT NULL,
    auto_renew   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_certificates_tenant ON certificates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_certificates_type ON certificates(tenant_id, type);
CREATE INDEX IF NOT EXISTS idx_certificates_expiry ON certificates(expiry_date);

CREATE TABLE IF NOT EXISTS mtls_config (
    tenant_id              UUID PRIMARY KEY DEFAULT '00000000-0000-0000-0000-000000000001',
    require_mtls           BOOLEAN NOT NULL DEFAULT FALSE,
    per_client_cert_binding BOOLEAN NOT NULL DEFAULT FALSE,
    revocation_check       TEXT NOT NULL DEFAULT 'none', -- none, CRL, OCSP, both
    allow_self_signed      BOOLEAN NOT NULL DEFAULT FALSE,
    fallback_to_bearer     BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO mtls_config (tenant_id) VALUES ('00000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id) DO NOTHING;
