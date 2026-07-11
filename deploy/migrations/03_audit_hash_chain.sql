-- Add hash chain columns to audit_events for tamper detection
-- Each event stores its hash (HMAC-SHA256 of prev_hash + canonical event data)
-- and the previous event's hash for chain verification.

-- Add columns to the parent partitioned table
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS hash VARCHAR(64) DEFAULT '';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS prev_hash VARCHAR(64) DEFAULT '';

-- Index for chain verification queries
CREATE INDEX IF NOT EXISTS idx_audit_hash ON audit_events (hash) WHERE hash != '';
CREATE INDEX IF NOT EXISTS idx_audit_prev_hash ON audit_events (prev_hash) WHERE prev_hash != '';

COMMENT ON COLUMN audit_events.hash IS 'HMAC-SHA256 hash chain link for tamper detection';
COMMENT ON COLUMN audit_events.prev_hash IS 'Hash of the previous event in the chain (empty for genesis)';
