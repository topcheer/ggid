-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create enums
CREATE TYPE scope_type AS ENUM ('global', 'organization', 'department', 'team', 'resource');
CREATE TYPE policy_effect AS ENUM ('allow', 'deny');
CREATE TYPE principal_type AS ENUM ('user', 'role', 'group');
