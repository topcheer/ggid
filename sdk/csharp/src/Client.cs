using System.Net.Http.Json;
using System.Text;
using System.Text.Json;
using GGID.SDK.Models;

namespace GGID.SDK;

/// <summary>
/// Main GGID API client for authentication, RBAC, ABAC, and OAuth/OIDC.
/// </summary>
public class GGIDClient
{
    /// <summary>SDK version.</summary>
    public const string Version = "1.0.0";

    private readonly HttpClient _http;
    private readonly string _baseUrl;
    private readonly string _tenantId;
    private JwtVerifier? _verifier;

    /// <summary>
    /// Create a new GGIDClient.
    /// </summary>
    /// <param name="baseUrl">GGID gateway URL (e.g. https://ggid.iot2.win)</param>
    /// <param name="tenantId">Tenant UUID</param>
    /// <param name="httpClient">Optional custom HttpClient</param>
    public GGIDClient(string baseUrl, string tenantId, HttpClient? httpClient = null)
    {
        _baseUrl = baseUrl.TrimEnd('/');
        _tenantId = tenantId;
        _http = httpClient ?? new HttpClient { Timeout = TimeSpan.FromSeconds(30) };
    }

    /// <summary>
    /// Enable JWT verification via JWKS.
    /// </summary>
    public GGIDClient WithJwks(string? jwksUrl = null)
    {
        _verifier = new JwtVerifier(jwksUrl ?? $"{_baseUrl}/oauth/jwks", _http, _tenantId);
        return this;
    }

    // ── Authentication ──

    /// <summary>
    /// Login with username/password and receive tokens.
    /// </summary>
    public async Task<TokenResponse> LoginAsync(string username, string password, CancellationToken ct = default)
    {
        var body = new { username, password };
        return await PostAsync<TokenResponse>("/api/v1/auth/login", body, token: null, ct);
    }

    /// <summary>
    /// Register a new user account.
    /// </summary>
    public async Task<string> RegisterAsync(string username, string email, string password, string name, CancellationToken ct = default)
    {
        var body = new { username, email, password, name };
        var result = await PostAsync<Dictionary<string, JsonElement>>("/api/v1/auth/register", body, token: null, ct);
        return result.TryGetValue("user_id", out var uid) ? uid.GetString() ?? "" : "";
    }

    /// <summary>
    /// Refresh tokens using a refresh token.
    /// </summary>
    public async Task<TokenResponse> RefreshTokenAsync(string refreshToken, CancellationToken ct = default)
    {
        var body = new { refresh_token = refreshToken };
        return await PostAsync<TokenResponse>("/api/v1/auth/refresh", body, token: null, ct);
    }

    // ── JWT Verification ──

    /// <summary>
    /// Verify a JWT and return claims.
    /// </summary>
    public async Task<Claims> VerifyTokenAsync(string token, CancellationToken ct = default)
    {
        if (_verifier == null)
            _verifier = new JwtVerifier($"{_baseUrl}/oauth/jwks", _http, _tenantId);
        return await _verifier.VerifyAsync(token, ct);
    }

    // ── UserInfo ──

    /// <summary>
    /// Get user info for the given access token.
    /// </summary>
    public async Task<UserInfo> GetUserInfoAsync(string accessToken, CancellationToken ct = default)
    {
        return await GetAsync<UserInfo>("/oauth/userinfo", accessToken, ct);
    }

    // ── OAuth/OIDC ──

    /// <summary>
    /// Get OIDC discovery document.
    /// </summary>
    public async Task<DiscoveryConfig> GetDiscoveryAsync(CancellationToken ct = default)
    {
        return await GetAsync<DiscoveryConfig>("/.well-known/openid-configuration", token: null, ct);
    }

    /// <summary>
    /// Get JWKS for JWT verification.
    /// </summary>
    public async Task<Jwks> GetJwksAsync(CancellationToken ct = default)
    {
        return await GetAsync<Jwks>("/oauth/jwks", token: null, ct);
    }

    /// <summary>
    /// Build an authorization URL for the OAuth code flow.
    /// </summary>
    public string GetAuthorizeUrl(string clientId, string redirectUri, string? scope = null, string? state = null)
    {
        var query = new Dictionary<string, string>
        {
            ["client_id"] = clientId,
            ["redirect_uri"] = redirectUri,
            ["response_type"] = "code",
        };
        if (scope != null) query["scope"] = scope;
        if (state != null) query["state"] = state;

        var qs = string.Join("&", query.Select(kv => $"{kv.Key}={Uri.EscapeDataString(kv.Value)}"));
        return $"{_baseUrl}/oauth/authorize?{qs}";
    }

    /// <summary>
    /// Exchange an authorization code for tokens.
    /// </summary>
    public async Task<TokenResponse> ExchangeCodeAsync(string code, string redirectUri, string clientId, string clientSecret, CancellationToken ct = default)
    {
        var body = new Dictionary<string, string>
        {
            ["grant_type"] = "authorization_code",
            ["code"] = code,
            ["redirect_uri"] = redirectUri,
            ["client_id"] = clientId,
            ["client_secret"] = clientSecret,
        };
        return await PostFormAsync<TokenResponse>("/api/v1/oauth/token", body, ct);
    }

    /// <summary>
    /// Revoke a token (RFC 7009).
    /// </summary>
    public async Task RevokeTokenAsync(string token, CancellationToken ct = default)
    {
        var body = new { token };
        await PostAsync<object>("/api/v1/oauth/revoke", body, token: null, ct);
    }

    /// <summary>
    /// Introspect a token (RFC 7662). Returns active status, subject, expiry, etc.
    /// </summary>
    public async Task<JsonElement> IntrospectTokenAsync(string token, string? clientId = null, string? clientSecret = null, CancellationToken ct = default)
    {
        var body = new Dictionary<string, string> { ["token"] = token };
        if (clientId != null) body["client_id"] = clientId;
        if (clientSecret != null) body["client_secret"] = clientSecret;
        return await PostAsync<JsonElement>("/api/v1/oauth/introspect", body, token: null, ct);
    }

    // ── Webhooks ──

    /// <summary>
    /// List all webhooks in the tenant.
    /// </summary>
    public async Task<JsonElement> ListWebhooksAsync(string token, CancellationToken ct = default)
    {
        return await GetAsync<JsonElement>("/api/v1/webhooks", token, ct);
    }

    /// <summary>
    /// Create a new webhook.
    /// </summary>
    public async Task<JsonElement> CreateWebhookAsync(string token, string url, List<string> events, CancellationToken ct = default)
    {
        var body = new { url, events };
        return await PostAsync<JsonElement>("/api/v1/webhooks", body, token, ct);
    }

    /// <summary>
    /// Delete a webhook by ID.
    /// </summary>
    public async Task DeleteWebhookAsync(string token, string webhookId, CancellationToken ct = default)
    {
        await DeleteAsync($"/api/v1/webhooks/{webhookId}", token, ct);
    }

    // ── Agent Identity ──

    /// <summary>
    /// Register a new AI agent.
    /// </summary>
    public async Task<Agent> RegisterAgentAsync(string token, AgentRegistration reg, CancellationToken ct = default)
    {
        return await PostAsync<Agent>("/api/v1/agents/register", reg, token, ct);
    }

    /// <summary>
    /// List all agents for the current tenant.
    /// </summary>
    public async Task<List<Agent>> ListAgentsAsync(string token, CancellationToken ct = default)
    {
        var data = await GetAsync<JsonElement>("/api/v1/agents", token, ct);
        if (data.ValueKind == JsonValueKind.Array)
            return data.Deserialize<List<Agent>>() ?? new();
        if (data.TryGetProperty("agents", out var agentsEl))
            return agentsEl.Deserialize<List<Agent>>() ?? new();
        return new();
    }

    /// <summary>
    /// Exchange a user token for an agent-scoped token.
    /// </summary>
    public async Task<AgentTokenResponse> ExchangeAgentTokenAsync(string agentId, string subjectToken, List<string>? scopes = null, CancellationToken ct = default)
    {
        var body = new { agent_id = agentId, subject_token = subjectToken, scope = scopes ?? new List<string>() };
        return await PostAsync<AgentTokenResponse>("/api/v1/agents/token", body, token: null, ct);
    }

    /// <summary>
    /// Verify an agent token and return its claims.
    /// </summary>
    public async Task<JsonElement> VerifyAgentTokenAsync(string token, CancellationToken ct = default)
    {
        var body = new { token };
        return await PostAsync<JsonElement>("/api/v1/agents/verify", body, token: null, ct);
    }

    // ── Access Request (IGA) ──

    /// <summary>
    /// Create an access request.
    /// </summary>
    public async Task<AccessRequestResponse> CreateAccessRequestAsync(string token, AccessRequest req, CancellationToken ct = default)
    {
        return await PostAsync<AccessRequestResponse>("/api/v1/access-requests", req, token, ct);
    }

    /// <summary>
    /// List access requests for the current tenant.
    /// </summary>
    public async Task<List<AccessRequestResponse>> ListAccessRequestsAsync(string token, CancellationToken ct = default)
    {
        var data = await GetAsync<JsonElement>("/api/v1/access-requests", token, ct);
        if (data.ValueKind == JsonValueKind.Array)
            return data.Deserialize<List<AccessRequestResponse>>() ?? new();
        if (data.TryGetProperty("requests", out var requestsEl))
            return requestsEl.Deserialize<List<AccessRequestResponse>>() ?? new();
        return new();
    }

    /// <summary>
    /// Approve an access request.
    /// </summary>
    public async Task<AccessRequestResponse> ApproveAccessRequestAsync(string token, string requestId, string comment = "", CancellationToken ct = default)
    {
        var body = new { comment };
        return await PostAsync<AccessRequestResponse>($"/api/v1/access-requests/{requestId}/approve", body, token, ct);
    }

    /// <summary>
    /// Reject an access request.
    /// </summary>
    public async Task<AccessRequestResponse> RejectAccessRequestAsync(string token, string requestId, string comment = "", CancellationToken ct = default)
    {
        var body = new { comment };
        return await PostAsync<AccessRequestResponse>($"/api/v1/access-requests/{requestId}/reject", body, token, ct);
    }

    // ── User Management ──

    /// <summary>
    /// List all users in the tenant.
    /// </summary>
    public async Task<List<User>> ListUsersAsync(string token, CancellationToken ct = default)
    {
        var data = await GetAsync<JsonElement>("/api/v1/users", token, ct);
        if (data.ValueKind == JsonValueKind.Array)
            return data.Deserialize<List<User>>() ?? new();
        if (data.TryGetProperty("users", out var usersEl))
            return usersEl.Deserialize<List<User>>() ?? new();
        return new();
    }

    /// <summary>
    /// Get a single user by ID.
    /// </summary>
    public async Task<User> GetUserAsync(string token, string userId, CancellationToken ct = default)
    {
        return await GetAsync<User>($"/api/v1/users/{userId}", token, ct);
    }

    /// <summary>
    /// Delete a user by ID.
    /// </summary>
    public async Task DeleteUserAsync(string token, string userId, CancellationToken ct = default)
    {
        await DeleteAsync($"/api/v1/users/{userId}", token, ct);
    }

    // ── Internal HTTP helpers ──

    internal async Task<T> GetAsync<T>(string path, string? token, CancellationToken ct)
    {
        var req = new HttpRequestMessage(HttpMethod.Get, _baseUrl + path);
        ApplyHeaders(req, token);
        var resp = await _http.SendAsync(req, ct);
        return await HandleResponse<T>(resp);
    }

    internal async Task<T> PostAsync<T>(string path, object? body, string? token, CancellationToken ct)
    {
        var req = new HttpRequestMessage(HttpMethod.Post, _baseUrl + path);
        if (body != null)
            req.Content = new StringContent(JsonSerializer.Serialize(body), Encoding.UTF8, "application/json");
        ApplyHeaders(req, token);
        var resp = await _http.SendAsync(req, ct);
        return await HandleResponse<T>(resp);
    }

    internal async Task<T> PutAsync<T>(string path, object? body, string? token, CancellationToken ct)
    {
        var req = new HttpRequestMessage(HttpMethod.Put, _baseUrl + path);
        if (body != null)
            req.Content = new StringContent(JsonSerializer.Serialize(body), Encoding.UTF8, "application/json");
        ApplyHeaders(req, token);
        var resp = await _http.SendAsync(req, ct);
        return await HandleResponse<T>(resp);
    }

    internal async Task DeleteAsync(string path, string? token, CancellationToken ct)
    {
        var req = new HttpRequestMessage(HttpMethod.Delete, _baseUrl + path);
        ApplyHeaders(req, token);
        var resp = await _http.SendAsync(req, ct);
        if (!resp.IsSuccessStatusCode)
            throw await CreateException(resp);
    }

    internal async Task DeleteWithBodyAsync(string path, object body, string? token, CancellationToken ct)
    {
        var req = new HttpRequestMessage(HttpMethod.Delete, _baseUrl + path);
        req.Content = new StringContent(JsonSerializer.Serialize(body), Encoding.UTF8, "application/json");
        ApplyHeaders(req, token);
        var resp = await _http.SendAsync(req, ct);
        if (!resp.IsSuccessStatusCode)
            throw await CreateException(resp);
    }

    internal async Task<T> PostFormAsync<T>(string path, Dictionary<string, string> form, CancellationToken ct)
    {
        var req = new HttpRequestMessage(HttpMethod.Post, _baseUrl + path);
        req.Content = new FormUrlEncodedContent(form);
        req.Headers.Add("X-Tenant-ID", _tenantId);
        var resp = await _http.SendAsync(req, ct);
        return await HandleResponse<T>(resp);
    }

    private void ApplyHeaders(HttpRequestMessage req, string? token)
    {
        req.Headers.Add("X-Tenant-ID", _tenantId);
        if (!string.IsNullOrEmpty(token))
            req.Headers.Authorization = new System.Net.Http.Headers.AuthenticationHeaderValue("Bearer", token);
    }

    private static async Task<T> HandleResponse<T>(HttpResponseMessage resp)
    {
        if (!resp.IsSuccessStatusCode)
            throw await CreateException(resp);
        var json = await resp.Content.ReadAsStringAsync();
        if (string.IsNullOrWhiteSpace(json))
            return default!;
        return JsonSerializer.Deserialize<T>(json) ?? default!;
    }

    private static async Task<GGIDException> CreateException(HttpResponseMessage resp)
    {
        var body = await resp.Content.ReadAsStringAsync();
        var msg = body;
        try
        {
            using var doc = JsonDocument.Parse(body);
            if (doc.RootElement.TryGetProperty("detail", out var d)) msg = d.GetString() ?? body;
            else if (doc.RootElement.TryGetProperty("message", out var m)) msg = m.GetString() ?? body;
            else if (doc.RootElement.TryGetProperty("error", out var e)) msg = e.GetString() ?? body;
        }
        catch { /* use raw body */ }
        return new GGIDException((int)resp.StatusCode, msg);
    }
}

    public async Task<JsonDocument> ClientCredentialsAsync(string clientId, string clientSecret, string scope = "") {
        var form = new FormUrlEncodedContent(new Dictionary<string, string> {
            ["grant_type"] = "client_credentials", ["client_id"] = clientId, ["client_secret"] = clientSecret, ["scope"] = scope
        });
        var resp = await _http.PostAsync($"{_baseUrl}/api/v1/oauth/token", form);
        return JsonDocument.Parse(await resp.Content.ReadAsStringAsync());
    }
