using System.Net;
using System.Text;
using System.Text.Json;
using GGID.SDK;
using GGID.SDK.Models;
using GGID.SDK.Middleware;
using Xunit;

namespace GGID.SDK.Tests;

/// <summary>
/// Unit tests for GGIDClient, JWT verification, RBAC, and ABAC.
/// Uses a custom HttpMessageHandler to mock API responses.
/// </summary>
public class ClientTest
{
    /// <summary>
    /// Create a mock HttpClient that returns canned JSON for all requests.
    /// </summary>
    private static HttpClient MockHttp(string jsonResponse, int statusCode = 200)
    {
        var handler = new MockHandler(jsonResponse, statusCode);
        return new HttpClient(handler) { Timeout = TimeSpan.FromSeconds(5) };
    }

    [Fact]
    public void Constructor_SetsBaseUrlAndTenantId()
    {
        var client = new GGIDClient("https://ggid.example.com/", "tenant-123");
        Assert.NotNull(client);
    }

    [Fact]
    public async Task LoginAsync_ReturnsTokenResponse()
    {
        var json = JsonSerializer.Serialize(new
        {
            access_token = "jwt-abc",
            refresh_token = "r-xyz",
            token_type = "Bearer",
            expires_in = 3600,
        });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var tokens = await ggid.LoginAsync("admin", "pass");

        Assert.Equal("jwt-abc", tokens.AccessToken);
        Assert.Equal("r-xyz", tokens.RefreshToken);
        Assert.Equal(3600, tokens.ExpiresIn);
    }

    [Fact]
    public async Task GetUserInfoAsync_ReturnsUserInfo()
    {
        var json = JsonSerializer.Serialize(new
        {
            sub = "user-1",
            name = "Alice",
            email = "alice@test.com",
            email_verified = true,
        });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var info = await ggid.GetUserInfoAsync("token");

        Assert.Equal("user-1", info.Sub);
        Assert.Equal("Alice", info.Name);
        Assert.Equal("alice@test.com", info.Email);
        Assert.True(info.EmailVerified);
    }

    [Fact]
    public async Task GetDiscoveryAsync_ReturnsDiscoveryConfig()
    {
        var json = JsonSerializer.Serialize(new
        {
            issuer = "https://ggid.example.com",
            authorization_endpoint = "https://ggid.example.com/oauth/authorize",
            token_endpoint = "https://ggid.example.com/api/v1/oauth/token",
            jwks_uri = "https://ggid.example.com/oauth/jwks",
        });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var discovery = await ggid.GetDiscoveryAsync();

        Assert.Equal("https://ggid.example.com", discovery.Issuer);
        Assert.Contains("authorize", discovery.AuthorizationEndpoint);
    }

    [Fact]
    public async Task GetJwksAsync_ReturnsJwks()
    {
        var json = JsonSerializer.Serialize(new
        {
            keys = new[] { new { kty = "RSA", kid = "key-1", use = "sig", alg = "RS256", n = "abc", e = "AQAB" } },
        });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var jwks = await ggid.GetJwksAsync();

        Assert.Single(jwks.Keys);
        Assert.Equal("key-1", jwks.Keys[0].Kid);
        Assert.Equal("RSA", jwks.Keys[0].Kty);
    }

    [Fact]
    public async Task CheckPermissionAsync_ReturnsTrue()
    {
        var json = JsonSerializer.Serialize(new { allowed = true, reason = "role permits read" });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var allowed = await ggid.CheckPermissionAsync("token", "products", "read");

        Assert.True(allowed);
    }

    [Fact]
    public async Task CheckPermissionAsync_ReturnsFalse()
    {
        var json = JsonSerializer.Serialize(new { allowed = false, reason = "insufficient permissions" });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var allowed = await ggid.CheckPermissionAsync("token", "products", "delete");

        Assert.False(allowed);
    }

    [Fact]
    public async Task ListRolesAsync_ReturnsRoles()
    {
        var json = JsonSerializer.Serialize(new[]
        {
            new { id = "r1", name = "Admin", key = "admin", system_role = true },
            new { id = "r2", name = "User", key = "user", system_role = false },
        });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var roles = await ggid.ListRolesAsync("token");

        Assert.Equal(2, roles.Count);
        Assert.Equal("Admin", roles[0].Name);
        Assert.True(roles[0].SystemRole);
    }

    [Fact]
    public async Task ListPermissionsAsync_ReturnsPermissions()
    {
        var json = JsonSerializer.Serialize(new[]
        {
            new { id = "p1", name = "Read Products", resource = "products", action = "read" },
            new { id = "p2", name = "Write Products", resource = "products", action = "write" },
        });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var perms = await ggid.ListPermissionsAsync("token");

        Assert.Equal(2, perms.Count);
        Assert.Equal("products", perms[0].Resource);
        Assert.Equal("read", perms[0].Action);
    }

    [Fact]
    public async Task AssignRoleAsync_Succeeds()
    {
        var http = MockHttp("{}");
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        await ggid.AssignRoleAsync("token", "user-1", "role-1");
    }

    [Fact]
    public async Task GetUserRolesAsync_ReturnsRoles()
    {
        var json = JsonSerializer.Serialize(new[]
        {
            new { id = "r1", name = "Admin", key = "admin", system_role = true },
        });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var roles = await ggid.GetUserRolesAsync("token", "user-1");

        Assert.Single(roles);
        Assert.Equal("admin", roles[0].Key);
    }

    [Fact]
    public async Task ListUsersAsync_ReturnsUsers()
    {
        var json = JsonSerializer.Serialize(new
        {
            users = new[]
            {
                new { id = "u1", username = "admin", email = "admin@test.com", status = "active" },
            },
        });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var users = await ggid.ListUsersAsync("token");

        Assert.Single(users);
        Assert.Equal("admin", users[0].Username);
    }

    [Fact]
    public async Task GetAuthorizeUrl_BuildsCorrectUrl()
    {
        var ggid = new GGIDClient("https://ggid.example.com", "tenant-1");
        var url = ggid.GetAuthorizeUrl("client-1", "https://app.example.com/callback", "openid profile", "state123");

        Assert.Contains("client_id=client-1", url);
        Assert.Contains("response_type=code", url);
        Assert.Contains("state=state123", url);
        Assert.StartsWith("https://ggid.example.com/oauth/authorize?", url);
    }

    [Fact]
    public async Task ExchangeCodeAsync_ReturnsTokens()
    {
        var json = JsonSerializer.Serialize(new
        {
            access_token = "at-xyz",
            refresh_token = "rt-abc",
            token_type = "Bearer",
            expires_in = 3600,
        });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var tokens = await ggid.ExchangeCodeAsync("code123", "https://app.example.com/callback", "client-1", "secret");

        Assert.Equal("at-xyz", tokens.AccessToken);
        Assert.Equal("Bearer", tokens.TokenType);
    }

    [Fact]
    public async Task CheckPolicyAsync_ReturnsPolicyResult()
    {
        var json = JsonSerializer.Serialize(new
        {
            allowed = true,
            reason = "ABAC policy matched",
            matched_rules = new[] { "rule-1" },
        });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var req = new PolicyCheckRequest
        {
            Subject = "user-1",
            Resource = "documents",
            Action = "read",
            Context = new() { ["department"] = "finance" },
        };
        var result = await ggid.CheckPolicyAsync("token", req);

        Assert.True(result.Allowed);
        Assert.Contains("rule-1", result.MatchedRules);
    }

    [Fact]
    public async Task APIError_ThrowsGGIDException()
    {
        var json = JsonSerializer.Serialize(new { error = "forbidden", detail = "insufficient role" });
        var http = MockHttp(json, 403);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var ex = await Assert.ThrowsAsync<GGIDException>(() => ggid.ListRolesAsync("token"));
        Assert.Equal(403, ex.StatusCode);
        Assert.Contains("insufficient role", ex.Message);
    }

    [Fact]
    public async Task VerifyTokenAsync_ThrowsOnEmptyToken()
    {
        var ggid = new GGIDClient("https://ggid.example.com", "tenant-1");
        await Assert.ThrowsAsync<InvalidTokenException>(() => ggid.VerifyTokenAsync(""));
    }

    [Fact]
    public async Task VerifyTokenAsync_ThrowsOnInvalidFormat()
    {
        var ggid = new GGIDClient("https://ggid.example.com", "tenant-1");
        await Assert.ThrowsAsync<InvalidTokenException>(() => ggid.VerifyTokenAsync("not-a-jwt"));
    }

    [Fact]
    public void AuthorizeAttribute_CanInstantiate()
    {
        var attr = new AuthorizeAttribute();
        Assert.NotNull(attr);
    }

    [Fact]
    public void RequirePermissionAttribute_SetsResourceAndAction()
    {
        var attr = new RequirePermissionAttribute("products", "read");
        Assert.Equal("products", attr.Resource);
        Assert.Equal("read", attr.Action);
    }

    [Fact]
    public void RequireRoleAttribute_SetsRole()
    {
        var attr = new RequireRoleAttribute("admin");
        Assert.Equal("admin", attr.Role);
    }

    [Fact]
    public async Task IntrospectTokenAsync_ReturnsActiveStatus()
    {
        var json = JsonSerializer.Serialize(new { active = true, sub = "user-1", exp = 1700000000 });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var result = await ggid.IntrospectTokenAsync("token", "client-1", "secret");

        Assert.True(result.GetProperty("active").GetBoolean());
        Assert.Equal("user-1", result.GetProperty("sub").GetString());
    }

    [Fact]
    public async Task ListWebhooksAsync_ReturnsWebhooks()
    {
        var json = JsonSerializer.Serialize(new[]
        {
            new { id = "wh-1", url = "https://example.com/hook", events = new[] { "user.created", "user.deleted" } },
            new { id = "wh-2", url = "https://example.com/hook2", events = new[] { "role.assigned" } },
        });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var webhooks = await ggid.ListWebhooksAsync("token");

        Assert.Equal(2, webhooks.GetArrayLength());
        Assert.Equal("wh-1", webhooks[0].GetProperty("id").GetString());
    }

    [Fact]
    public async Task CreateWebhookAsync_ReturnsCreatedWebhook()
    {
        var json = JsonSerializer.Serialize(new { id = "wh-3", url = "https://example.com/hook3", events = new[] { "user.created" } });
        var http = MockHttp(json);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        var result = await ggid.CreateWebhookAsync("token", "https://example.com/hook3", new List<string> { "user.created" });

        Assert.Equal("wh-3", result.GetProperty("id").GetString());
        Assert.Equal("https://example.com/hook3", result.GetProperty("url").GetString());
    }

    [Fact]
    public async Task DeleteWebhookAsync_Succeeds()
    {
        var http = MockHttp("{}", 200);
        var ggid = new GGIDClient("http://localhost:9999", "tenant-1", http);

        await ggid.DeleteWebhookAsync("token", "wh-1");
        // No exception means success
    }
}

/// <summary>
/// Simple mock HttpMessageHandler that returns a fixed JSON response.
/// </summary>
internal class MockHandler : HttpMessageHandler
{
    private readonly string _json;
    private readonly int _statusCode;

    public MockHandler(string json, int statusCode = 200)
    {
        _json = json;
        _statusCode = statusCode;
    }

    protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
    {
        var resp = new HttpResponseMessage((HttpStatusCode)_statusCode)
        {
            Content = new StringContent(_json, Encoding.UTF8, "application/json"),
        };
        return Task.FromResult(resp);
    }
}
