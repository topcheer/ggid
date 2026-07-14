using System.Text.Json.Serialization;

namespace GGID.SDK.Models;

/// <summary>
/// JWT claims extracted from a verified access token.
/// </summary>
public record Claims
{
    [JsonPropertyName("sub")] public string? UserId { get; init; }
    [JsonPropertyName("tenant_id")] public string? TenantId { get; init; }
    [JsonPropertyName("roles")] public List<string> Roles { get; init; } = new();
    [JsonPropertyName("scope")] public string? Scope { get; init; }
    [JsonPropertyName("exp")] public long Exp { get; init; }
    [JsonPropertyName("iat")] public long Iat { get; init; }
    [JsonPropertyName("iss")] public string? Iss { get; init; }
    [JsonPropertyName("email")] public string? Email { get; init; }
    [JsonPropertyName("name")] public string? Name { get; init; }
}

/// <summary>
/// OpenID Connect UserInfo response.
/// </summary>
public record UserInfo
{
    [JsonPropertyName("sub")] public string? Sub { get; init; }
    [JsonPropertyName("name")] public string? Name { get; init; }
    [JsonPropertyName("email")] public string? Email { get; init; }
    [JsonPropertyName("email_verified")] public bool EmailVerified { get; init; }
    [JsonPropertyName("preferred_username")] public string? PreferredUsername { get; init; }
    [JsonPropertyName("picture")] public string? Picture { get; init; }
    [JsonPropertyName("locale")] public string? Locale { get; init; }
    [JsonPropertyName("updated_at")] public long UpdatedAt { get; init; }
}

/// <summary>
/// OAuth 2.0 token response from login or token exchange.
/// </summary>
public record TokenResponse
{
    [JsonPropertyName("access_token")] public string AccessToken { get; init; } = "";
    [JsonPropertyName("refresh_token")] public string? RefreshToken { get; init; }
    [JsonPropertyName("id_token")] public string? IdToken { get; init; }
    [JsonPropertyName("token_type")] public string TokenType { get; init; } = "Bearer";
    [JsonPropertyName("expires_in")] public int ExpiresIn { get; init; }
}

/// <summary>
/// A GGID role.
/// </summary>
public record Role
{
    [JsonPropertyName("id")] public string Id { get; init; } = "";
    [JsonPropertyName("name")] public string Name { get; init; } = "";
    [JsonPropertyName("key")] public string Key { get; init; } = "";
    [JsonPropertyName("description")] public string? Description { get; init; }
    [JsonPropertyName("system_role")] public bool SystemRole { get; init; }
}

/// <summary>
/// A GGID permission entry.
/// </summary>
public record Permission
{
    [JsonPropertyName("id")] public string Id { get; init; } = "";
    [JsonPropertyName("name")] public string Name { get; init; } = "";
    [JsonPropertyName("resource")] public string Resource { get; init; } = "";
    [JsonPropertyName("action")] public string Action { get; init; } = "";
    [JsonPropertyName("description")] public string? Description { get; init; }
}

/// <summary>
/// Result of a permission/policy check.
/// </summary>
public record PolicyResult
{
    [JsonPropertyName("allowed")] public bool Allowed { get; init; }
    [JsonPropertyName("reason")] public string? Reason { get; init; }
    [JsonPropertyName("matched_rules")] public List<string> MatchedRules { get; init; } = new();
}

/// <summary>
/// ABAC policy check request.
/// </summary>
public record PolicyCheckRequest
{
    [JsonPropertyName("subject")] public string Subject { get; init; } = "";
    [JsonPropertyName("resource")] public string Resource { get; init; } = "";
    [JsonPropertyName("action")] public string Action { get; init; } = "";
    [JsonPropertyName("context")] public Dictionary<string, string> Context { get; init; } = new();
}

/// <summary>
/// ABAC condition for attribute-based evaluation.
/// </summary>
public record AbacCondition
{
    [JsonPropertyName("field")] public string Field { get; init; } = "";
    [JsonPropertyName("operator")] public string Operator { get; init; } = "";
    [JsonPropertyName("value")] public string Value { get; init; } = "";
}

/// <summary>
/// ABAC evaluation request.
/// </summary>
public record AbacEvalRequest
{
    [JsonPropertyName("action")] public string Action { get; init; } = "";
    [JsonPropertyName("resource")] public string Resource { get; init; } = "";
    [JsonPropertyName("conditions")] public List<AbacCondition> Conditions { get; init; } = new();
}

/// <summary>
/// ABAC evaluation result.
/// </summary>
public record AbacEvalResult
{
    [JsonPropertyName("matched")] public bool Matched { get; init; }
    [JsonPropertyName("matched_rules")] public List<string> MatchedRules { get; init; } = new();
}

/// <summary>
/// OpenID Connect discovery document.
/// </summary>
public record DiscoveryConfig
{
    [JsonPropertyName("issuer")] public string? Issuer { get; init; }
    [JsonPropertyName("authorization_endpoint")] public string? AuthorizationEndpoint { get; init; }
    [JsonPropertyName("token_endpoint")] public string? TokenEndpoint { get; init; }
    [JsonPropertyName("userinfo_endpoint")] public string? UserInfoEndpoint { get; init; }
    [JsonPropertyName("jwks_uri")] public string? JwksUri { get; init; }
    [JsonPropertyName("revocation_endpoint")] public string? RevocationEndpoint { get; init; }
    [JsonPropertyName("scopes_supported")] public List<string> ScopesSupported { get; init; } = new();
}

/// <summary>
/// JSON Web Key Set for JWT signature verification.
/// </summary>
public record Jwks
{
    [JsonPropertyName("keys")] public List<Jwk> Keys { get; init; } = new();
}

/// <summary>
/// Single JSON Web Key.
/// </summary>
public record Jwk
{
    [JsonPropertyName("kty")] public string? Kty { get; init; }
    [JsonPropertyName("kid")] public string? Kid { get; init; }
    [JsonPropertyName("use")] public string? Use { get; init; }
    [JsonPropertyName("alg")] public string? Alg { get; init; }
    [JsonPropertyName("n")] public string? N { get; init; }
    [JsonPropertyName("e")] public string? E { get; init; }
    [JsonPropertyName("x")] public string? X { get; init; }
    [JsonPropertyName("y")] public string? Y { get; init; }
    [JsonPropertyName("crv")] public string? Crv { get; init; }
}

/// <summary>
/// Represents a GGID user.
/// </summary>
public record User
{
    [JsonPropertyName("id")] public string Id { get; init; } = "";
    [JsonPropertyName("username")] public string Username { get; init; } = "";
    [JsonPropertyName("email")] public string Email { get; init; } = "";
    [JsonPropertyName("status")] public string Status { get; init; } = "";
    [JsonPropertyName("display_name")] public string? DisplayName { get; init; }
    [JsonPropertyName("created_at")] public string? CreatedAt { get; init; }
}

/// <summary>
/// Custom exception for GGID API errors.
/// </summary>
public class GGIDException : Exception
{
    public int StatusCode { get; }

    public GGIDException(int statusCode, string message) : base(message)
    {
        StatusCode = statusCode;
    }
}

/// <summary>
/// Exception thrown when JWT verification fails.
/// </summary>
public class InvalidTokenException : Exception
{
    public InvalidTokenException(string message) : base(message) { }
}

/// <summary>
/// Exception thrown when a token has expired.
/// </summary>
public class TokenExpiredException : Exception
{
    public TokenExpiredException() : base("token expired") { }
}
