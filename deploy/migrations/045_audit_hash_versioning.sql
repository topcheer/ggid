-- P2-6: Audit hash chain secret versioning
-- Tracks which secret version was used to hash each event, enabling key
-- rotation without invalidating old events.
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS hash_secret_version INT DEFAULT 0;
COMMENT ON COLUMN audit_events.hash_secret_version IS 'Version of the HMAC secret used for this event hash (0 = pre-versioning)';
