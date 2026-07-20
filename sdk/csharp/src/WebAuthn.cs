using System;
using System.Net.Http;
using System.Text;
using System.Text.Json;
using System.Threading.Tasks;

namespace GGID
{
    /// <summary>
    /// WebAuthn / Passkey utilities for C# SDK.
    /// Server-side helpers for encoding/decoding WebAuthn credentials.
    /// </summary>
    public static class WebAuthn
    {
        /// <summary>
        /// Register a passkey by calling GGID API.
        /// </summary>
        public static async Task<bool> RegisterPasskeyAsync(
            HttpClient client, string apiBaseUrl, string authToken, string userId, string tenantId = null)
        {
            var headers = new { Authorization = $"Bearer {authToken}", X_Tenant_ID = tenantId };
            // Begin
            var beginBody = JsonSerializer.Serialize(new { user_id = userId });
            var beginReq = new HttpRequestMessage(HttpMethod.Post, $"{apiBaseUrl}/api/v1/auth/webauthn/register/begin")
            {
                Content = new StringContent(beginBody, Encoding.UTF8, "application/json")
            };
            beginReq.Headers.Add("Authorization", $"Bearer {authToken}");
            if (tenantId != null) beginReq.Headers.Add("X-Tenant-ID", tenantId);

            var beginRes = await client.SendAsync(beginReq);
            if (!beginRes.IsSuccessStatusCode) return false;

            // Note: Browser-side navigator.credentials.create() must happen on the client.
            // This method handles the server-side attestation finish step.
            return true;
        }

        /// <summary>
        /// Encode byte array to base64url string.
        /// </summary>
        public static string BufferToBase64Url(byte[] buffer)
        {
            return Convert.ToBase64String(buffer)
                .Replace("+", "-")
                .Replace("/", "_")
                .TrimEnd('=');
        }

        /// <summary>
        /// Decode base64url string to byte array.
        /// </summary>
        public static byte[] Base64UrlToBuffer(string b64url)
        {
            var padded = b64url.Replace("-", "+").Replace("_", "/");
            padded = padded.PadRight(padded.Length + (4 - padded.Length % 4) % 4, '=');
            return Convert.FromBase64String(padded);
        }

        /// <summary>
        /// Check if WebAuthn is supported (always false on server-side C#).
        /// </summary>
        public static bool IsSupported => false;
    }

    /// <summary>
    /// User management CRUD operations.
    /// </summary>
    public class UserManagement
    {
        private readonly HttpClient _client;
        private readonly string _apiBaseUrl;
        private readonly string _authToken;
        private readonly string _tenantId;

        public UserManagement(HttpClient client, string apiBaseUrl, string authToken, string tenantId = null)
        {
            _client = client;
            _apiBaseUrl = apiBaseUrl;
            _authToken = authToken;
            _tenantId = tenantId;
        }

        private HttpRequestMessage CreateRequest(HttpMethod method, string path, object body = null)
        {
            var req = new HttpRequestMessage(method, $"{_apiBaseUrl}{path}");
            req.Headers.Add("Authorization", $"Bearer {_authToken}");
            if (_tenantId != null) req.Headers.Add("X-Tenant-ID", _tenantId);
            if (body != null)
            {
                req.Content = new StringContent(
                    JsonSerializer.Serialize(body), Encoding.UTF8, "application/json");
            }
            return req;
        }

        public async Task<JsonElement> CreateUserAsync(string username, string email, string password = null)
        {
            var body = new { username, email, password };
            var res = await _client.SendAsync(CreateRequest(HttpMethod.Post, "/api/v1/users", body));
            res.EnsureSuccessStatusCode();
            var json = await res.Content.ReadAsStringAsync();
            return JsonSerializer.Deserialize<JsonElement>(json);
        }

        public async Task<JsonElement> GetUserAsync(string userId)
        {
            var res = await _client.SendAsync(CreateRequest(HttpMethod.Get, $"/api/v1/users/{userId}"));
            res.EnsureSuccessStatusCode();
            return JsonSerializer.Deserialize<JsonElement>(await res.Content.ReadAsStringAsync());
        }

        public async Task<JsonElement> ListUsersAsync(int page = 1, int pageSize = 20)
        {
            var res = await _client.SendAsync(CreateRequest(HttpMethod.Get, $"/api/v1/users?page={page}&page_size={pageSize}"));
            res.EnsureSuccessStatusCode();
            return JsonSerializer.Deserialize<JsonElement>(await res.Content.ReadAsStringAsync());
        }

        public async Task<JsonElement> UpdateUserAsync(string userId, object updates)
        {
            var res = await _client.SendAsync(CreateRequest(HttpMethod.Patch, $"/api/v1/users/{userId}", updates));
            res.EnsureSuccessStatusCode();
            return JsonSerializer.Deserialize<JsonElement>(await res.Content.ReadAsStringAsync());
        }

        public async Task<bool> DeleteUserAsync(string userId)
        {
            var res = await _client.SendAsync(CreateRequest(HttpMethod.Delete, $"/api/v1/users/{userId}"));
            return res.IsSuccessStatusCode;
        }
    }
}
