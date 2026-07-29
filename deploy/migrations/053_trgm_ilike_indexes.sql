-- Accelerate ILIKE '%term%' searches (leading wildcard) with trigram GIN indexes.
-- Covers identity ListUsers (username/email ILIKE) and ListGroups (display_name ILIKE).
-- B-tree indexes cannot help leading-wildcard patterns; pg_trgm can.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_users_username_trgm
    ON users USING gin (username gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_users_email_trgm
    ON users USING gin (email gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_scim_groups_display_name_trgm
    ON scim_groups USING gin (display_name gin_trgm_ops);
