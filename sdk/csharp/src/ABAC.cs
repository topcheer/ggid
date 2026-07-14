using GGID.SDK.Models;

namespace GGID.SDK;

/// <summary>
/// ABAC operations: attribute-based policy evaluation.
/// </summary>
public static class ABAC
{
    /// <summary>
    /// Evaluate an ABAC policy with conditions.
    /// </summary>
    public static async Task<AbacEvalResult> EvaluateAbacAsync(this GGIDClient client, string token, AbacEvalRequest request, CancellationToken ct = default)
    {
        return await client.PostAsync<AbacEvalResult>("/api/v1/policies/abac/evaluate", request, token, ct);
    }

    /// <summary>
    /// Check a policy with subject, resource, action, and context.
    /// </summary>
    public static async Task<PolicyResult> CheckPolicyAsync(this GGIDClient client, string token, PolicyCheckRequest request, CancellationToken ct = default)
    {
        return await client.PostAsync<PolicyResult>("/api/v1/policies/check", request, token, ct);
    }

    /// <summary>
    /// Convenience: check policy with simple parameters.
    /// </summary>
    public static async Task<bool> CheckPolicyAsync(this GGIDClient client, string token, string subject, string resource, string action, Dictionary<string, string>? context = null, CancellationToken ct = default)
    {
        var req = new PolicyCheckRequest
        {
            Subject = subject,
            Resource = resource,
            Action = action,
            Context = context ?? new(),
        };
        var result = await client.CheckPolicyAsync(token, req, ct);
        return result?.Allowed ?? false;
    }
}
