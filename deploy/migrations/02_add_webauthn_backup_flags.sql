-- WA-1: Add WebAuthn backup_eligible, backup_state, user_verified, attestation_type, aaguid columns
-- These fields are derived from the authenticator data flags during registration.

ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS backup_eligible BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS backup_state BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS user_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS attestation_type VARCHAR(64) NOT NULL DEFAULT 'none';
ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS aaguid BYTEA;
