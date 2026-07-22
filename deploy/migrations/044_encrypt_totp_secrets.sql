-- P2-1: TOTP secret encryption at rest
-- mfa_devices.secret column now stores AES-256-GCM encrypted values (base64).
-- Existing plaintext secrets remain readable (DecryptTOTPSecret falls back
-- to plaintext when decryption fails). New secrets are encrypted on INSERT.
-- To migrate existing rows: set GGID_ENCRYPTION_KEY, then run a one-time
-- script that reads+re-writes each row (ReadAll → EncryptTOTPSecret → UPDATE).
ALTER TABLE mfa_devices ADD COLUMN IF NOT EXISTS secret_encrypted BOOLEAN DEFAULT false;
COMMENT ON COLUMN mfa_devices.secret_encrypted IS 'true if secret column is AES-256-GCM encrypted';
