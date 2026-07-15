using GGID.SDK;
using GGID.SDK.Middleware;

// ── ASP.NET Core Minimal API Quickstart ──

var builder = WebApplication.CreateBuilder(args);

// Register GGID client as singleton with JWKS verification
builder.Services.AddGGID(
    baseUrl: "https://ggid.iot2.win",
    tenantId: "00000000-0000-0000-0000-000000000001"
);

var app = builder.Build();

// Public endpoint (no auth required)
app.MapGet("/healthz", () => Results.Ok(new { status = "ok" }));

// Add JWT authentication middleware
app.UseGGIDAuth();

// Protected: requires valid JWT
app.MapGet("/api/me", [Authorize] (HttpContext ctx) =>
{
    var claims = ctx.GetGGIDClaims()!;
    return Results.Ok(new
    {
        user_id = claims.UserId,
        email = claims.Email,
        roles = claims.Roles,
    });
})
.WithName("GetMe");

// Protected: requires permission products:read
app.MapGet("/api/products", [Authorize]
    [RequirePermission("products", "read")]
    (HttpContext ctx, GGIDClient ggid) =>
{
    var claims = ctx.GetGGIDClaims()!;
    var token = ctx.GetBearerToken()!;

    // Optionally check permission via API for server-side enforcement
    // var allowed = await ggid.CheckPermissionAsync(token, "products", "read");

    return Results.Ok(new
    {
        user = claims.UserId,
        products = new[]
        {
            new { id = 1, name = "Widget", price = 9.99m },
            new { id = 2, name = "Gadget", price = 19.99m },
            new { id = 3, name = "Doohickey", price = 4.99m },
        },
    });
})
.WithName("ListProducts");

// Protected: requires admin role
app.MapPost("/api/products", [Authorize]
    [RequireRole("admin")]
    (HttpContext ctx, ProductInput input) =>
{
    return Results.Created($"/api/products/{42}", new { id = 42, input.name, input.price });
})
.WithName("CreateProduct");

// OAuth login flow example
app.MapGet("/auth/login", (GGIDClient ggid, string redirectUri) =>
{
    var authorizeUrl = ggid.GetAuthorizeUrl(
        clientId: "gcid_your_client_id",
        redirectUri: redirectUri,
        scope: "openid profile email",
        state: Guid.NewGuid().ToString("N")
    );
    return Results.Redirect(authorizeUrl);
})
.WithName("OAuthLogin");

// OAuth callback
app.MapGet("/auth/callback", async (GGIDClient ggid, string code, string redirectUri) =>
{
    var tokens = await ggid.ExchangeCodeAsync(
        code: code,
        redirectUri: redirectUri,
        clientId: "gcid_your_client_id",
        clientSecret: "your_client_secret"
    );
    return Results.Ok(new { access_token = tokens.AccessToken, expires_in = tokens.ExpiresIn });
})
.WithName("OAuthCallback");

app.Run();

// ── Models ──

public record ProductInput(string name, decimal price);
