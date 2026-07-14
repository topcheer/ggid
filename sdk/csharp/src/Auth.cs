using System.IdentityModel.Tokens.Jwt;
using System.Net.Http.Json;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using GGID.SDK.Models;
using Microsoft.IdentityModel.Tokens;

namespace GGID.SDK;

/// <summary>
/// Verifies RS256 JWTs against GGID's JWKS endpoint.
/// Caches signing keys with a 5-minute TTL.
/// </summary>
public class JwtVerifier
{
    private readonly string _jwksUrl;
    private readonly HttpClient _http;
    private readonly string _tenantId;
    private JsonWebKeySet? _cachedKeys;
    private DateTime _cachedAt;
    private readonly TimeSpan _ttl = TimeSpan.FromMinutes(5);
    private readonly SemaphoreSlim _lock = new(1, 1);

    public JwtVerifier(string jwksUrl, HttpClient http, string tenantId)
    {
        _jwksUrl = jwksUrl;
        _http = http;
        _tenantId = tenantId;
    }

    /// <summary>
    /// Verify a JWT and return its claims.
    /// </summary>
    public async Task<Claims> VerifyAsync(string token, CancellationToken ct = default)
    {
        if (string.IsNullOrWhiteSpace(token))
            throw new InvalidTokenException("token is empty");

        var handler = new JwtSecurityTokenHandler();
        if (!handler.CanReadToken(token))
            throw new InvalidTokenException("invalid JWT format");

        var jwt = handler.ReadJwtToken(token);

        // Check expiration with 60s clock skew tolerance
        if (jwt.ValidTo != DateTime.MinValue && jwt.ValidTo < DateTime.UtcNow.AddSeconds(-60))
            throw new TokenExpiredException();

        // Fetch and cache JWKS keys
        var keys = await GetKeysAsync(ct: ct);
        if (keys == null || keys.Count == 0)
            throw new InvalidTokenException("no signing keys available");

        // Try to validate with matching kid
        var validationParams = new TokenValidationParameters
        {
            ValidateIssuer = false,
            ValidateAudience = false,
            ValidateLifetime = false, // already checked above
            ValidateIssuerSigningKey = true,
            IssuerSigningKeys = keys,
            ClockSkew = TimeSpan.FromSeconds(60),
        };

        try
        {
            handler.ValidateToken(token, validationParams, out _);
        }
        catch (SecurityTokenSignatureKeyNotFoundException)
        {
            // Key not found — refresh cache and retry once
            _cachedKeys = null;
            keys = await GetKeysAsync(forceRefresh: true, ct: ct);
            validationParams.IssuerSigningKeys = keys;
            handler.ValidateToken(token, validationParams, out _);
        }
        catch (SecurityTokenException ex)
        {
            throw new InvalidTokenException($"signature validation failed: {ex.Message}");
        }

        // Extract claims into our model
        var roles = jwt.Claims
            .Where(c => c.Type == "roles" || c.Type == "role")
            .Select(c => c.Value)
            .ToList();

        var scope = jwt.Claims.FirstOrDefault(c => c.Type == "scope")?.Value;
        var sub = jwt.Claims.FirstOrDefault(c => c.Type == JwtRegisteredClaimNames.Sub)?.Value
                  ?? jwt.Claims.FirstOrDefault(c => c.Type == "user_id")?.Value;
        var tenantId = jwt.Claims.FirstOrDefault(c => c.Type == "tenant_id")?.Value;
        var email = jwt.Claims.FirstOrDefault(c => c.Type == JwtRegisteredClaimNames.Email)?.Value;
        var name = jwt.Claims.FirstOrDefault(c => c.Type == JwtRegisteredClaimNames.Name)?.Value;

        return new Claims
        {
            UserId = sub,
            TenantId = tenantId,
            Roles = roles,
            Scope = scope,
            Exp = jwt.ValidTo != DateTime.MinValue ? new DateTimeOffset(jwt.ValidTo).ToUnixTimeSeconds() : 0,
            Iat = jwt.ValidFrom != DateTime.MinValue ? new DateTimeOffset(jwt.ValidFrom).ToUnixTimeSeconds() : 0,
            Iss = jwt.Issuer,
            Email = email,
            Name = name,
        };
    }

    private async Task<List<SecurityKey>> GetKeysAsync(bool forceRefresh = false, CancellationToken ct = default)
    {
        if (!forceRefresh && _cachedKeys != null && DateTime.UtcNow - _cachedAt < _ttl)
            return _cachedKeys.GetSigningKeys().ToList();

        await _lock.WaitAsync(ct);
        try
        {
            if (!forceRefresh && _cachedKeys != null && DateTime.UtcNow - _cachedAt < _ttl)
                return _cachedKeys.GetSigningKeys().ToList();

            var req = new HttpRequestMessage(HttpMethod.Get, _jwksUrl);
            req.Headers.Add("X-Tenant-ID", _tenantId);
            var resp = await _http.SendAsync(req, ct);
            resp.EnsureSuccessStatusCode();
            var json = await resp.Content.ReadAsStringAsync(ct);
            _cachedKeys = new JsonWebKeySet(json);
            _cachedAt = DateTime.UtcNow;
            return _cachedKeys.GetSigningKeys().ToList();
        }
        finally
        {
            _lock.Release();
        }
    }
}
