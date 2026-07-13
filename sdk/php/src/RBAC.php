<?php
declare(strict_types=1);

namespace Ggid\Sdk;

/**
 * RBAC trait — provides role-based access control methods.
 *
 * Intended to be used by GGIDClient.
 */
trait RBAC
{
    /**
     * Check if the current user has permission for resource+action.
     *
     * @return PermissionCheckResult
     */
    public function checkPermission(string $token, string $resource, string $action): PermissionCheckResult
    {
        $data = $this->request('GET', '/api/v1/policies/check', null, $token, [
            'resource' => $resource,
            'action' => $action,
        ]);
        return PermissionCheckResult::fromArray($data);
    }

    /**
     * Assign a role to a user.
     */
    public function assignRole(string $token, string $userId, string $roleId): array
    {
        return $this->request('POST', "/api/v1/policies/roles/{$roleId}/users/{$userId}", [
            'user_id' => $userId,
            'role_id' => $roleId,
        ], $token);
    }

    /**
     * Revoke a role from a user.
     */
    public function revokeRole(string $token, string $userId, string $roleId): void
    {
        $this->request('DELETE', "/api/v1/policies/roles/{$roleId}/users/{$userId}", null, $token);
    }

    /**
     * Get all roles assigned to a user.
     *
     * @return Role[]
     */
    public function getUserRoles(string $token, string $userId): array
    {
        $data = $this->request('GET', "/api/v1/policies/users/{$userId}/roles", null, $token);
        $roles = [];
        foreach ($data as $item) {
            if (is_array($item)) {
                $roles[] = Role::fromArray($item);
            }
        }
        return $roles;
    }

    /**
     * List all roles in the tenant.
     *
     * @return Role[]
     */
    public function listRoles(string $token): array
    {
        $data = $this->request('GET', '/api/v1/roles', null, $token);
        $roles = [];
        $items = $data['roles'] ?? $data['data'] ?? $data;
        if (!is_array($items)) {
            return $roles;
        }
        foreach ($items as $item) {
            if (is_array($item)) {
                $roles[] = Role::fromArray($item);
            }
        }
        return $roles;
    }

    /**
     * List all permissions (permission tree).
     *
     * @return Permission[]
     */
    public function listPermissions(string $token): array
    {
        $data = $this->request('GET', '/api/v1/policies/permissions/tree', null, $token);
        $permissions = [];
        if (!is_array($data)) {
            return $permissions;
        }
        $items = $data['permissions'] ?? $data['data'] ?? $data;
        foreach ($items as $item) {
            if (is_array($item)) {
                $permissions[] = Permission::fromArray($item);
            }
        }
        return $permissions;
    }
}
