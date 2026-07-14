# GGID C# .NET SDK

Official C# SDK for the **GGID IAM Platform** — JWT verification, RBAC/ABAC authorization, and ASP.NET Core middleware.

## Quick Start

### 1. Install

```bash
dotnet add package GGID.SDK
```

Or from source:
```bash
cd sdk/csharp
dotnet build
dotnet pack
dotnet add package GGID.SDK --source ./bin/Release
```

### 2. Initialize

```csharp
using GGID.SDK;

var ggid = new GGIDClient("https://ggid.iot2.win", "00000000-0000-0000-0000-000000000001")
    .WithJwks();
```

### 3. Verify Token

```csharp
var claims = await ggid.VerifyTokenAsync(jwt);
Console.WriteLine($"User: {claims.UserId}, Roles: {string.Join(", ", claims.Roles)}");
```

### 4. Check Permission (RBAC)

```csharp
var allowed = await ggid.CheckPermissionAsync(token, "products", "read");
if (!allowed) return Forbid();
```

### 5. Check Policy (ABAC)

```csharp
var allowed = await ggid.CheckPolicyAsync(token, "user-123", "documents", "read",
    new() { ["department"] = "finance" });
```

## ASP.NET Core Integration

### Program.cs

```csharp
using GGID.SDK;
using GGID.SDK.Middleware;

var builder = WebApplication.CreateBuilder(args);

// Register GGID client
builder.Services.AddGGID("https://ggid.iot2.win", "00000000-0000-0000-0000-000000000001");

var app = builder.Build();

// Add JWT auth middleware
app.UseGGIDAuth();

// Protected endpoint
app.MapGet("/api/products", [Authorize]
    [RequirePermission("products", "read")]
    (HttpContext ctx) =>
    {
        var claims = ctx.GetGGIDClaims();
        return Results.Ok(new { user = claims?.UserId, products = new[] { "widget", "gadget" } });
    });

// Admin-only endpoint
app.MapDelete("/api/products/{id}", [Authorize]
    [RequireRole("admin")]
    (string id) => Results.Ok(new { deleted = id }));

app.Run();
```

## API Reference

### Authentication

| Method | Description |
|--------|-------------|
| `LoginAsync(username, password)` | Login with credentials, returns `TokenResponse` |
| `RegisterAsync(username, email, password, name)` | Register new user |
| `RefreshTokenAsync(refreshToken)` | Refresh tokens |
| `VerifyTokenAsync(token)` | Verify JWT, returns `Claims` |
| `GetUserInfoAsync(token)` | Get OIDC UserInfo |

### OAuth/OIDC

| Method | Description |
|--------|-------------|
| `GetDiscoveryAsync()` | Get OIDC discovery document |
| `GetJwksAsync()` | Get JWKS keys |
| `GetAuthorizeUrl(clientId, redirectUri, scope?, state?)` | Build authorize URL |
| `ExchangeCodeAsync(code, redirectUri, clientId, clientSecret)` | Exchange auth code for tokens |
| `RevokeTokenAsync(token)` | Revoke a token (RFC 7009) |

### RBAC

| Method | Description |
|--------|-------------|
| `CheckPermissionAsync(token, resource, action)` | Check if user can perform action |
| `AssignRoleAsync(token, userId, roleId)` | Assign role to user |
| `RevokeRoleAsync(token, userId, roleId)` | Revoke role from user |
| `GetUserRolesAsync(token, userId)` | Get user's roles |
| `ListRolesAsync(token)` | List all roles |
| `ListPermissionsAsync(token)` | List all permissions |

### ABAC

| Method | Description |
|--------|-------------|
| `EvaluateAbacAsync(token, AbacEvalRequest)` | Evaluate ABAC policy |
| `CheckPolicyAsync(token, PolicyCheckRequest)` | Check policy with context |

### Attributes (ASP.NET Core)

| Attribute | Description |
|-----------|-------------|
| `[Authorize]` | Require authenticated user |
| `[RequirePermission("res", "act")]` | Require specific permission |
| `[RequireRole("admin")]` | Require specific role |

## License

Apache-2.0
