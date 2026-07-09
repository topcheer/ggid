-- Remove seeded roles and permissions for default tenant.
DELETE FROM role_permissions
    WHERE role_id IN (SELECT id FROM roles WHERE tenant_id = '00000000-0000-0000-0000-000000000001');

DELETE FROM permissions WHERE tenant_id = '00000000-0000-0000-0000-000000000001' AND system_perm = TRUE;
DELETE FROM roles WHERE tenant_id = '00000000-0000-0000-0000-000000000001' AND system_role = TRUE;
