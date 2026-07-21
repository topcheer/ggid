using System;
using System.Collections.Generic;
using System.IO;
using System.Net;
using System.Text;
using System.Text.Json;
using System.Threading.Tasks;
using GGID.SDK;
using GGID.SDK.Models;
using System.Linq;

class Program
{
    static string ggidUrl = Environment.GetEnvironmentVariable("GGID_URL") ?? "http://localhost:8080";
    static string tenantId = Environment.GetEnvironmentVariable("TENANT_ID") ?? "00000005-0000-0000-0000-000000000001";
    static int port = int.Parse(Environment.GetEnvironmentVariable("PORT") ?? "9200");

    // GGID SDK client with JWKS signature verification
    static GGIDClient ggidClient = new GGIDClient(ggidUrl, tenantId).WithJwks();

    static List<Dictionary<string, object>> inventory = new()
    {
        new() { ["id"] = "p001", ["name"] = "Widget A", ["stock"] = 150, ["price"] = 29.99 },
        new() { ["id"] = "p002", ["name"] = "Widget B", ["stock"] = 80, ["price"] = 49.99 },
    };
    static List<Dictionary<string, object>> orders = new()
    {
        new() { ["id"] = "o001", ["customer"] = "Acme", ["status"] = "pending", ["total"] = 299.90 },
        new() { ["id"] = "o002", ["customer"] = "Beta", ["status"] = "approved", ["total"] = 249.95 },
    };

    static void Main(string[] args)
    {
        var listener = new HttpListener();
        listener.Prefixes.Add($"http://+:{port}/");
        listener.Start();
        Console.WriteLine($"ERP C# Demo on :{port} | GGID: {ggidUrl} | Tenant: {tenantId}");
        while (true)
        {
            var ctx = listener.GetContext();
            _ = Task.Run(() => { try { Handle(ctx); } catch (Exception e) { try { Json(ctx.Response, 500, new { error = e.Message }); } catch {} } ctx.Response.Close(); });
        }
    }

    static void Handle(HttpListenerContext ctx)
    {
        var path = ctx.Request.Url.AbsolutePath;
        var method = ctx.Request.HttpMethod;

        if (path == "/" || path == "/health") { Json(ctx.Response, 200, new { app = "ERP C# Demo", auth = "Password Grant", tenant_id = tenantId }); return; }

        // Login via SDK: GGIDClient.LoginAsync (Password Grant)
        if (path == "/api/auth/login" && method == "POST")
        {
            var body = ReadBody(ctx.Request);
            try
            {
                var tokenResp = ggidClient.LoginAsync(
                    body.GetValueOrDefault("username")?.ToString() ?? "",
                    body.GetValueOrDefault("password")?.ToString() ?? ""
                ).Result;
                Json(ctx.Response, 200, tokenResp);
            }
            catch (Exception e)
            {
                Json(ctx.Response, 401, new { error = "login failed: " + e.InnerException?.Message ?? e.Message });
            }
            return;
        }

        // Token verification via SDK: GGIDClient.VerifyTokenAsync (JWKS + RS256)
        var token = GetToken(ctx.Request);
        if (token == null) { Json(ctx.Response, 401, new { error = "Bearer token required" }); return; }

        Claims claims;
        try
        {
            claims = ggidClient.VerifyTokenAsync(token).Result;
        }
        catch
        {
            Json(ctx.Response, 401, new { error = "invalid token" });
            return;
        }
        var perms = claims.Permissions;

        if (path == "/api/inventory" && method == "GET") { if (!HasPerm(perms, "inventory:read")) { Forbid(ctx, "inventory:read"); return; } Json(ctx.Response, 200, new { items = inventory, count = inventory.Count }); return; }
        if (path == "/api/inventory" && method == "POST") { if (!HasPerm(perms, "inventory:write")) { Forbid(ctx, "inventory:write"); return; } var b = ToObj(ReadBody(ctx.Request)); b["id"] = $"p{inventory.Count + 1:D3}"; inventory.Add(b); Json(ctx.Response, 201, b); return; }
        if (path == "/api/orders" && method == "GET") { if (!HasPerm(perms, "orders:read")) { Forbid(ctx, "orders:read"); return; } Json(ctx.Response, 200, new { orders, count = orders.Count }); return; }
        if (path == "/api/orders" && method == "POST") { if (!HasPerm(perms, "orders:write")) { Forbid(ctx, "orders:write"); return; } var b = ToObj(ReadBody(ctx.Request)); b["id"] = $"o{orders.Count + 1:D3}"; b["status"] = "pending"; orders.Add(b); Json(ctx.Response, 201, b); return; }
        if (path.StartsWith("/api/orders/") && path.EndsWith("/approve") && method == "POST") { if (!HasPerm(perms, "orders:approve")) { Forbid(ctx, "orders:approve"); return; } var id = path.Split("/")[3]; var o = orders.FirstOrDefault(x => x["id"].ToString() == id); if (o == null) { Json(ctx.Response, 404, new { error = "not found" }); return; } o["status"] = "approved"; Json(ctx.Response, 200, o); return; }
        if (path == "/api/my-permissions") { Json(ctx.Response, 200, new { permissions = perms, can_write_orders = HasPerm(perms, "orders:write"), can_approve = HasPerm(perms, "orders:approve") }); return; }

        Json(ctx.Response, 404, new { error = "not found", path });
    }

    static void Forbid(HttpListenerContext ctx, string perm) => Json(ctx.Response, 403, new { error = $"missing {perm}" });
    static string GetToken(HttpListenerRequest req) { var a = req.Headers["Authorization"]; return a != null && a.StartsWith("Bearer ") ? a.Substring(7) : null; }

    static bool HasPerm(List<string> p, string perm) => p.Contains("admin") || p.Contains(perm);

    static Dictionary<string, object> ReadBody(HttpListenerRequest req) { using var r = new StreamReader(req.InputStream); var b = r.ReadToEnd(); if (string.IsNullOrEmpty(b)) return new(); using var doc = JsonDocument.Parse(b); var result = new Dictionary<string, object>(); foreach (var prop in doc.RootElement.EnumerateObject()) { result[prop.Name] = prop.Value.ValueKind switch { JsonValueKind.Number => prop.Value.GetDouble(), JsonValueKind.True => true, JsonValueKind.False => false, _ => prop.Value.GetString() }; } return result; }
    static Dictionary<string, object> ToObj(Dictionary<string, object> d) => d;
    static void Json(HttpListenerResponse resp, int code, object data) { resp.StatusCode = code; resp.ContentType = "application/json"; WriteJson(resp, JsonSerializer.Serialize(data)); }
    static void WriteJson(HttpListenerResponse resp, string json) { var b = Encoding.UTF8.GetBytes(json); resp.ContentLength64 = b.Length; resp.OutputStream.Write(b, 0, b.Length); }
}
