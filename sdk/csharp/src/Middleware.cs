using System.IdentityModel.Tokens.Jwt;
using System.Security.Claims;
using GGID.SDK.Models;
using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.Builder;
using Microsoft.Extensions.DependencyInjection;

namespace GGID.SDK.Middleware;

/// <summary>
/// ASP.NET Core middleware for JWT authentication.
/// Public paths (login, register, healthz) bypass authentication.
/// </summary>
public class GGIDAuthMiddleware
{
    private readonly RequestDelegate _next;
    private static readonly HashSet<string> PublicPaths = new(StringComparer.OrdinalIgnoreCase)
    {
        "/", "/healthz", "/docs", "/api-docs", "/login", "/register",
    };

    public GGIDAuthMiddleware(RequestDelegate next) => _next = next;

    public async Task InvokeAsync(HttpContext context, GGIDClient ggid)
    {
        var path = context.Request.Path.Value ?? "";

        // Skip public paths and auth endpoints
        if (PublicPaths.Contains(path) ||
            path.StartsWith("/api/v1/auth/", StringComparison.OrdinalIgnoreCase) ||
            path.StartsWith("/oauth/", StringComparison.OrdinalIgnoreCase))
        {
            await _next(context);
            return;
        }

        // Extract Bearer token
        var authHeader = context.Request.Headers.Authorization.ToString();
        if (!authHeader.StartsWith("Bearer ", StringComparison.OrdinalIgnoreCase))
        {
            context.Response.StatusCode = 401;
            await context.Response.WriteAsJsonAsync(new { error = "missing bearer token" });
            return;
        }

        var token = authHeader[7..];

        try
        {
            var claims = await ggid.VerifyTokenAsync(token, context.RequestAborted);

            // Inject claims into HttpContext.Items
            context.Items["GGIDClaims"] = claims;

            // Also set on HttpContext.User for attribute compatibility
            var identity = new ClaimsIdentity(
                claims.Roles.Select(r => new Claim("role", r))
                    .Append(new Claim(ClaimTypes.NameIdentifier, claims.UserId ?? ""))
                    .Append(new Claim("tenant_id", claims.TenantId ?? "")),
                "GGID");
            if (claims.Email != null) identity.AddClaim(new Claim(ClaimTypes.Email, claims.Email));
            if (claims.Name != null) identity.AddClaim(new Claim(ClaimTypes.Name, claims.Name));
            context.User = new ClaimsPrincipal(identity);
        }
        catch (TokenExpiredException)
        {
            context.Response.StatusCode = 401;
            await context.Response.WriteAsJsonAsync(new { error = "token expired" });
            return;
        }
        catch (InvalidTokenException ex)
        {
            context.Response.StatusCode = 401;
            await context.Response.WriteAsJsonAsync(new { error = "invalid token", detail = ex.Message });
            return;
        }

        await _next(context);
    }
}

/// <summary>
/// Extension methods for registering GGID middleware.
/// </summary>
public static class GGIDMiddlewareExtensions
{
    /// <summary>
    /// Register GGIDClient as a singleton and add JWT auth middleware.
    /// </summary>
    public static IServiceCollection AddGGID(this IServiceCollection services, string baseUrl, string tenantId)
    {
        services.AddSingleton(new GGIDClient(baseUrl, tenantId).WithJwks());
        return services;
    }

    /// <summary>
    /// Use GGID JWT authentication middleware.
    /// </summary>
    public static IApplicationBuilder UseGGIDAuth(this IApplicationBuilder app)
    {
        return app.UseMiddleware<GGIDAuthMiddleware>();
    }
}

/// <summary>
/// Attribute to require authentication on a controller action or minimal API endpoint.
    /// Usage: <code>[Authorize]</code>
    /// </summary>
[AttributeUsage(AttributeTargets.Method | AttributeTargets.Class, AllowMultiple = false)]
public class AuthorizeAttribute : Attribute { }

/// <summary>
/// Attribute to require a specific permission.
    /// Usage: <code>[RequirePermission("products", "read")]</code>
    /// </summary>
[AttributeUsage(AttributeTargets.Method | AttributeTargets.Class, AllowMultiple = true)]
public class RequirePermissionAttribute : Attribute
{
    public string Resource { get; }
    public string Action { get; }

    public RequirePermissionAttribute(string resource, string action)
    {
        Resource = resource;
        Action = action;
    }
}

/// <summary>
/// Attribute to require a specific role.
    /// Usage: <code>[RequireRole("admin")]</code>
    /// </summary>
[AttributeUsage(AttributeTargets.Method | AttributeTargets.Class, AllowMultiple = true)]
public class RequireRoleAttribute : Attribute
{
    public string Role { get; }

    public RequireRoleAttribute(string role)
    {
        Role = role;
    }
}

/// <summary>
/// Helper to extract GGID claims from HttpContext.
    /// </summary>
public static class GGIDContextExtensions
{
    /// <summary>
    /// Get the verified GGID claims from the request context.
    /// </summary>
    public static Claims? GetGGIDClaims(this HttpContext context)
    {
        return context.Items.TryGetValue("GGIDClaims", out var c) ? c as Claims : null;
    }

    /// <summary>
    /// Get the Bearer token from the Authorization header.
    /// </summary>
    public static string? GetBearerToken(this HttpContext context)
    {
        var header = context.Request.Headers.Authorization.ToString();
        return header.StartsWith("Bearer ", StringComparison.OrdinalIgnoreCase) ? header[7..] : null;
    }
}
