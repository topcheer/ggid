// Cross-Board ERP Demo — C# implementation
// Tests all GGID core features via C# SDK.
//
// Run: GGID_URL=https://ggid.iot2.win dotnet run

using System;
using System.Collections.Generic;
using System.IO;
using System.Net;
using System.Text;
using System.Text.Json;
using System.Web;
using Ggid.Sdk; // GGID C# SDK

var ggidUrl = Environment.GetEnvironmentVariable("GGID_URL") ?? "http://localhost:8080";
var tenantId = Environment.GetEnvironmentVariable("TENANT_ID") ?? "00000005-0000-0000-0000-000000000001";
var port = int.Parse(Environment.GetEnvironmentVariable("PORT") ?? "9200");

// --- In-memory data ---
var inventory = new List<Dictionary<string, object>> {
    new() { ["id"] = "p001", ["name"] = "Widget A", ["stock"] = 150, ["price"] = 29.99 },
    new() { ["id"] = "p002", ["name"] = "Widget B", ["stock"] = 80, ["price"] = 49.99 },
    new() { ["id"] = "p003", ["name"] = "Gadget C", ["stock"] = 200, ["price"] = 19.99 },
};
var orders = new List<Dictionary<string, object>> {
    new() { ["id"] = "o001", ["customer"] = "Acme Corp", ["product_id"] = "p001", ["qty"] = 10, ["status"] = "pending", ["total"] = 299.90 },
    new() { ["id"] = "o002", ["customer"] = "Beta Inc", ["product_id"] = "p002", ["qty"] = 5, ["status"] = "approved", ["total"] = 249.95 },
};

var client = new GGIDClient(ggidUrl, tenantId);
var listener = new HttpListener();
listener.Prefixes.Add($"http://+:{port}/");
listener.Start();
Console.WriteLine($"ERP C# Demo on :{port} | GGID: {ggidUrl}");

while (true)
{
    var ctx = listener.GetContext();
    _ = Task.Run(() => HandleRequest(ctx, client, inventory, orders));
}

static async Task HandleRequest(HttpListenerContext ctx, GGIDClient client,
    List<Dictionary<string, object>> inventory, List<Dictionary<string, object>> orders)
{
    var req = ctx.Request;
    var resp = ctx.Response;
    var path = req.Url!.AbsolutePath;
    var method = req.HttpMethod;

    try
    {
        // Public routes
        if (path == "/" || path == "/health")
        {
            await Json(resp, 200, new { app = "ERP C# Demo", status = "ok" });
            return;
        }

        if (path == "/api/auth/login" && method == "POST")
        {
            var body = await ReadBody(req);
            var result = await client.Login(body["username"]!, body["password"]!);
            await Json(resp, 200, result);
            return;
        }

        // Authenticated routes
        var token = GetToken(req);
        if (string.IsNullOrEmpty(token))
        {
            await Json(resp, 401, new { error = "Bearer token required" });
            return;
        }

        var perms = ExtractPermissions(token);

        // --- Inventory ---
        if (path == "/api/inventory" && method == "GET")
        {
            if (!HasPerm(perms, "inventory:read")) { await Json(resp, 403, new { error = "missing inventory:read" }); return; }
            await Json(resp, 200, new { items = inventory, count = inventory.Count });
            return;
        }
        if (path == "/api/inventory" && method == "POST")
        {
            if (!HasPerm(perms, "inventory:write")) { await Json(resp, 403, new { error = "missing inventory:write" }); return; }
            var body = await ReadBody(req);
            body["id"] = $"p{inventory.Count + 1:D3}";
            inventory.Add(body);
            await Json(resp, 201, body);
            return;
        }

        // --- Orders ---
        if (path == "/api/orders" && method == "GET")
        {
            if (!HasPerm(perms, "orders:read")) { await Json(resp, 403, new { error = "missing orders:read" }); return; }
            await Json(resp, 200, new { orders, count = orders.Count });
            return;
        }
        if (path == "/api/orders" && method == "POST")
        {
            if (!HasPerm(perms, "orders:write")) { await Json(resp, 403, new { error = "missing orders:write" }); return; }
            var body = await ReadBody(req);
            body["id"] = $"o{orders.Count + 1:D3}";
            body["status"] = "pending";
            orders.Add(body);
            await Json(resp, 201, body);
            return;
        }
        if (path.StartsWith("/api/orders/") && path.EndsWith("/approve") && method == "POST")
        {
            if (!HasPerm(perms, "orders:approve")) { await Json(resp, 403, new { error = "missing orders:approve" }); return; }
            var orderId = path.Split("/")[3];
            var order = orders.Find(o => o["id"]!.ToString() == orderId);
            if (order == null) { await Json(resp, 404, new { error = "order not found" }); return; }
            order["status"] = "approved";
            await Json(resp, 200, order);
            return;
        }

        // --- Users ---
        if (path == "/api/users" && method == "GET")
        {
            if (!HasPerm(perms, "users:read")) { await Json(resp, 403, new { error = "missing users:read" }); return; }
            var users = await client.ListUsers(token);
            await Json(resp, 200, users);
            return;
        }

        // --- Roles ---
        if (path == "/api/roles" && method == "GET")
        {
            if (!HasPerm(perms, "roles:read")) { await Json(resp, 403, new { error = "missing roles:read" }); return; }
            var roles = await client.ListRoles(token);
            await Json(resp, 200, roles);
            return;
        }

        // --- Audit ---
        if (path == "/api/audit" && method == "GET")
        {
            if (!HasPerm(perms, "audit:read")) { await Json(resp, 403, new { error = "missing audit:read" }); return; }
            var events = await client.ListAuditEvents(token, tenantId: "00000000-0000-0000-0000-000000000001");
            await Json(resp, 200, events);
            return;
        }

        // --- My Permissions ---
        if (path == "/api/my-permissions" && method == "GET")
        {
            await Json(resp, 200, new
            {
                permissions = perms,
                can_read_inventory = HasPerm(perms, "inventory:read"),
                can_write_orders = HasPerm(perms, "orders:write"),
                can_approve_orders = HasPerm(perms, "orders:approve"),
            });
            return;
        }

        await Json(resp, 404, new { error = "not found", path });
    }
    catch (Exception ex)
    {
        await Json(resp, 500, new { error = ex.Message });
    }
    finally
    {
        resp.Close();
    }
}

static string? GetToken(HttpListenerRequest req)
{
    var auth = req.Headers["Authorization"];
    return auth?.StartsWith("Bearer ") == true ? auth[7..] : null;
}

static List<string> ExtractPermissions(string token)
{
    // Extract permissions claim from JWT (no verification in demo)
    var parts = token.Split('.');
    if (parts.Length < 2) return new();
    var payload = parts[1];
    payload += new string('=', (4 - payload.Length % 4) % 4);
    try
    {
        var bytes = Convert.FromBase64String(payload);
        var json = Encoding.UTF8.GetString(bytes);
        using var doc = JsonDocument.Parse(json);
        if (doc.RootElement.TryGetProperty("permissions", out var perms))
        {
            var result = new List<string>();
            foreach (var p in perms.EnumerateArray())
                result.Add(p.GetString());
            return result;
        }
    }
    catch { }
    return new();
}

static bool HasPerm(List<string> perms, string perm) =>
    perms.Contains("admin") || perms.Contains(perm);

static async Task<Dictionary<string, string>> ReadBody(HttpListenerRequest req)
{
    using var reader = new StreamReader(req.InputStream);
    var body = await reader.ReadToEndAsync();
    return JsonSerializer.Deserialize<Dictionary<string, string>>(body) ?? new();
}

static async Task Json(HttpListenerResponse resp, int status, object data)
{
    resp.StatusCode = status;
    resp.ContentType = "application/json";
    var json = JsonSerializer.Serialize(data);
    var bytes = Encoding.UTF8.GetBytes(json);
    resp.ContentLength64 = bytes.Length;
    await resp.OutputStream.WriteAsync(bytes);
}
