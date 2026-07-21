-- WORM (Write Once Read Many) protection for audit_events.
-- Blocks UPDATE/DELETE unless the caller explicitly sets the session GUC
--   SET LOCAL app.allow_audit_mutation = 'on'
-- inside a transaction. Authorized paths that do this:
--   - retention cleanup (DeleteOlderThan)
--   - GDPR erasure (gdpr/forget handler)
-- Hash chain columns make any other mutation detectable via tamper-check.

CREATE OR REPLACE FUNCTION audit_events_worm_guard() RETURNS trigger AS $$
BEGIN
    IF current_setting('app.allow_audit_mutation', true) IS DISTINCT FROM 'on' THEN
        RAISE EXCEPTION 'audit_events is WORM-protected: % denied (set app.allow_audit_mutation=on for authorized maintenance)', TG_OP
            USING ERRCODE = 'raise_exception';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_events_worm ON audit_events;
CREATE TRIGGER audit_events_worm
    BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_worm_guard();

COMMENT ON FUNCTION audit_events_worm_guard() IS 'WORM guard: audit_events rows are immutable unless app.allow_audit_mutation=on is set in the transaction';
