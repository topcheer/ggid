-- KB-368: Dynamic RBAC — role_route_permissions table
CREATE TABLE IF NOT EXISTS role_route_permissions (
    role_id          UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    route_prefix     TEXT NOT NULL,
    permission_level TEXT NOT NULL DEFAULT 'read',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, route_prefix)
);

CREATE INDEX IF NOT EXISTS idx_rrp_role ON role_route_permissions (role_id);

-- Seed: platform:admin gets all routes with admin level
INSERT INTO role_route_permissions (role_id, route_prefix, permission_level)
SELECT id, route, level FROM (
    VALUES
    ('platform:admin', '/dashboard', 'admin'),
    ('platform:admin', '/users', 'admin'),
    ('platform:admin', '/roles', 'admin'),
    ('platform:admin', '/policies', 'admin'),
    ('platform:admin', '/audit', 'admin'),
    ('platform:admin', '/security', 'admin'),
    ('platform:admin', '/settings', 'admin'),
    ('platform:admin', '/admin', 'admin'),
    ('platform:admin', '/oauth', 'admin'),
    ('platform:admin', '/sessions', 'admin'),
    ('platform:admin', '/profile', 'admin')
) AS seed(role_key, route, level)
JOIN roles ON roles.key = seed.role_key
ON CONFLICT DO NOTHING;

-- Seed: tenant:admin gets tenant-level routes
INSERT INTO role_route_permissions (role_id, route_prefix, permission_level)
SELECT id, route, level FROM (
    VALUES
    ('tenant:admin', '/dashboard', 'admin'),
    ('tenant:admin', '/users', 'admin'),
    ('tenant:admin', '/roles', 'read'),
    ('tenant:admin', '/policies', 'admin'),
    ('tenant:admin', '/audit', 'read'),
    ('tenant:admin', '/security', 'read'),
    ('tenant:admin', '/settings', 'admin'),
    ('tenant:admin', '/profile', 'admin'),
    ('tenant:admin', '/sessions', 'admin')
) AS seed(role_key, route, level)
JOIN roles ON roles.key = seed.role_key
ON CONFLICT DO NOTHING;

-- Seed: tenant:auditor gets read-only routes
INSERT INTO role_route_permissions (role_id, route_prefix, permission_level)
SELECT id, route, level FROM (
    VALUES
    ('tenant:auditor', '/dashboard', 'read'),
    ('tenant:auditor', '/audit', 'read'),
    ('tenant:auditor', '/security', 'read'),
    ('tenant:auditor', '/profile', 'read')
) AS seed(role_key, route, level)
JOIN roles ON roles.key = seed.role_key
ON CONFLICT DO NOTHING;

-- Seed: user:self gets minimal routes
INSERT INTO role_route_permissions (role_id, route_prefix, permission_level)
SELECT id, route, level FROM (
    VALUES
    ('user:self', '/dashboard', 'read'),
    ('user:self', '/profile', 'read'),
    ('user:self', '/sessions', 'read')
) AS seed(role_key, route, level)
JOIN roles ON roles.key = seed.role_key
ON CONFLICT DO NOTHING;
