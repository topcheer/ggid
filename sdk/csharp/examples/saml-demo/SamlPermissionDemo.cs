// GGID SAML SSO Demo with Permissions (C#)
// Run: GGID_URL=... dotnet run
using System;
using System.Net;
using System.Text;
using GGID;

class User {
  public string Username = "demo_user";
  public string[] Roles = { "viewer" };
  public string[] Permissions = { "dashboard:read", "orders:read", "inventory:read" };
  public bool HasPermission(string perm) =>
    Array.IndexOf(Permissions, "admin") >= 0 || Array.IndexOf(Permissions, perm) >= 0;
}

class Demo {
  static User user = new();
  static string ggidUrl = Environment.GetEnvironmentVariable("GGID_URL") ?? "http://localhost:8080";
  static string entityId = Environment.GetEnvironmentVariable("SP_ENTITY_ID") ?? "http://localhost:3102/saml/metadata";
  static string acsUrl = Environment.GetEnvironmentVariable("ACS_URL") ?? "http://localhost:3102/saml/acs";

  static async Task Main() {
    using var listener = new HttpListener();
    listener.Prefixes.Add("http://localhost:3102/");
    listener.Start();
    Console.WriteLine("SAML Demo on http://localhost:3102");
    while (true) {
      var ctx = await listener.GetContextAsync();
      var path = ctx.Request.Url.AbsolutePath;
      var res = ctx.Response;

      string html = path switch {
        "/" => Dashboard(),
        "/saml/metadata" => SAML.GenerateSPMetadata(entityId, acsUrl),
        "/login" => $"<a href='{ggidUrl}/saml/sso'>Login</a>",
        "/inventory" when !user.HasPermission("inventory:read") => Forbidden("inventory:read"),
        "/inventory" => Page("Inventory", user.HasPermission("inventory:write")),
        "/orders" when !user.HasPermission("orders:read") => Forbidden("orders:read"),
        "/orders" => Page("Orders", user.HasPermission("orders:write")),
        _ => "Not found",
      };

      res.ContentType = path == "/saml/metadata" ? "application/xml" : "text/html";
      if (html.StartsWith("403")) res.StatusCode = 403;
      var buf = Encoding.UTF8.GetBytes(html);
      res.OutputStream.Write(buf, 0, buf.Length);
      res.Close();
    }
  }

  static string Menu() {
    var items = "<li><a href='/'>Dashboard</a></li>";
    if (user.HasPermission("orders:read")) items += "<li><a href='/orders'>Orders</a></li>";
    if (user.HasPermission("inventory:read")) items += "<li><a href='/inventory'>Inventory</a></li>";
    return $"<aside><ul>{items}</ul><p>Roles: {string.Join(", ", user.Roles)}</p></aside>";
  }

  static string Dashboard() => $"<html><body>{Menu()}<main><h1>Dashboard</h1><p>{user.Username}</p></main></body></html>";

  static string Page(string title, bool canWrite) =>
    $"<html><body>{Menu()}<main><h1>{title}</h1>{(canWrite ? "<button>New</button>" : "<p>Read-only</p>")}</main></body></html>";

  static string Forbidden(string perm) => $"403 - Need: {perm}";
}
