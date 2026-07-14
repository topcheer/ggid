using System.Text.Json;
using GGID.SDK.Models;

namespace GGID.SDK;

/// <summary>/// RBAC operations: permission checking, role assignment, and role/permission listing.
/// </summary>
public static class RBAC
{
    /// <summary>
    /// Check if the token's user can perform an action on a resource.
    /// One-line authorization: <code>var allowed = await client.CheckPermissionAsync(token, "products", "read");</code>
    /// </summary>
    public static async Task<bool> CheckPermissionAsync(this GGIDClient client, string token, string resource, string action, CancellationToken ct = default)
    {
        var body = new { resource, action };
        var result = await client.PostAsync<PolicyResult>("/api/v1/policies/check", body, token, ct);
        return result?.Allowed ?? false;
    }

    /// <summary>
    /// Check permission and return full policy result.
    /// </summary>
    public static async Task<PolicyResult> CheckPermissionResultAsync(this GGIDClient client, string token, string resource, string action, CancellationToken ct = default)
    {
        var body = new { resource, action };
        return await client.PostAsync<PolicyResult>("/api/v1/policies/check", body, token, ct);
    }

    /// <summary>
    /// Assign a role to a user.
    /// </summary>
    public static async Task AssignRoleAsync(this GGIDClient client, string token, string userId, string roleId, CancellationToken ct = default)
    {
        var body = new { user_id = userId, role_id = roleId };
        await client.PostAsync<object>("/api/v1/roles/assign", body, token, ct);
    }

    /// <summary>
    /// Revoke a role from a user.
    /// </summary>
    public static async Task RevokeRoleAsync(this GGIDClient client, string token, string userId, string roleId, CancellationToken ct = default)
    {
        var body = new { user_id = userId, role_id = roleId };
        await client.DeleteWithBodyAsync("/api/v1/roles/revoke", body, token, ct);
    }

    /// <summary>
    /// Get all roles assigned to a user.
    /// </summary>
    public static async Task<List<Role>> GetUserRolesAsync(this GGIDClient client, string token, string userId, CancellationToken ct = default)
    {
        var data = await client.GetAsync<JsonElement>($"/api/v1/users/{userId}/roles", token, ct);
        if (data.ValueKind == JsonValueKind.Array)
            return data.Deserialize<List<Role>>() ?? new();
        if (data.TryGetProperty("roles", out var rolesEl))
            return rolesEl.Deserialize<List<Role>>() ?? new();
        return new();
    }

    /// <summary>
    /// List all roles in the tenant.
    /// </summary>
    public static async Task<List<Role>> ListRolesAsync(this GGIDClient client, string token, CancellationToken ct = default)
    {
        var data = await client.GetAsync<JsonElement>("/api/v1/roles", token, ct);
        if (data.ValueKind == JsonValueKind.Array)
            return data.Deserialize<List<Role>>() ?? new();
        if (data.TryGetProperty("roles", out var rolesEl))
            return rolesEl.Deserialize<List<Role>>() ?? new();
        return new();
    }

    /// <summary>
    /// List all available permissions.
    /// </summary>
    public static async Task<List<Permission>> ListPermissionsAsync(this GGIDClient client, string token, CancellationToken ct = default)
    {
        var data = await client.GetAsync<JsonElement>("/api/v1/permissions", token, ct);
        if (data.ValueKind == JsonValueKind.Array)
            return data.Deserialize<List<Permission>>() ?? new();
        if (data.TryGetProperty("permissions", out var permsEl))
            return permsEl.Deserialize<List<Permission>>() ?? new();
        return new();
    }

    /// <summary>
    /// Create a new role.
    /// </summary>
    public static async Task<Role> CreateRoleAsync(this GGIDClient client, string token, string name, string key, string? description = null, CancellationToken ct = default)
    {
        var body = new { name, key, description };
        return await client.PostAsync<Role>("/api/v1/roles", body, token, ct);
    }

    /// <summary>
    /// Delete a role by ID.
    /// </summary>
    public static async Task DeleteRoleAsync(this GGIDClient client, string token, string roleId, CancellationToken ct = default)
    {
        await client.DeleteAsync($"/api/v1/roles/{roleId}", token, ct);
    }
}
