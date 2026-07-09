-- Reverse of 000001_initial_schema.up.sql
DROP TABLE IF EXISTS email_verification_tokens;
DROP TABLE IF EXISTS user_external_identities;
DROP TABLE IF EXISTS user_emails;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS update_updated_at() CASCADE;
DROP FUNCTION IF EXISTS sync_primary_email() CASCADE;
