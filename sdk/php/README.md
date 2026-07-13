# GGID PHP SDK

PHP 8.1+ SDK for the GGID IAM Platform — JWT verification, OAuth/OIDC, RBAC, ABAC, and framework middleware.

## Quick Start (5 Minutes)

### 1. Install

```bash
composer require ggid/sdk
```

### 2. Initialize

```php
use Ggid\Sdk\GGIDClient;

$ggid = new GGIDClient('https://ggid.iot2.win', '00000000-0000-0000-0000-000000000001');
```

### 3. Authenticate

```php
// Login
$tokens = $ggid->login('admin', 'Admin@123456');

// Verify JWT
$claims = $ggid->verifyToken($tokens['access_token']);
echo $claims->userId;    // "user-001"
echo $claims->email;     // "admin@example.com"

// Check permission
$result = $ggid->checkPermission($tokens['access_token'], 'products', 'read');
if ($result->allowed) {
    // User can read products
}
```

## API Reference

### Client

```php
$ggid = new GGIDClient(
    baseUrl: 'https://ggid.iot2.win',
    tenantId: '00000000-0000-0000-0000-000000000001',
    timeout: 30,
);
```

### Authentication

| Method | Description |
|--------|-------------|
| `register($username, $email, $password)` | Register a new user |
| `login($username, $password)` | Login and get tokens |
| `verifyToken($jwt): Claims` | Verify JWT via JWKS |
| `getUserInfo($accessToken): UserInfo` | OIDC userinfo endpoint |

### OAuth/OIDC

| Method | Description |
|--------|-------------|
| `getDiscovery(): array` | OIDC discovery document |
| `getJwks(): array` | JWKS public keys |
| `getAuthorizeUrl($clientId, $redirectUri, $scope, $state): string` | Build authorize URL |
| `exchangeCode($code, $redirectUri, $clientId, $clientSecret): TokenResponse` | Exchange auth code for tokens |
| `refreshToken($refreshToken, $clientId, $clientSecret): TokenResponse` | Refresh access token |
| `revokeToken($token, $clientId, $clientSecret)` | Revoke token (RFC 7009) |
| `introspectToken($token, $clientId, $clientSecret): array` | Introspect token (RFC 7662) |

### RBAC

| Method | Description |
|--------|-------------|
| `checkPermission($token, $resource, $action): PermissionCheckResult` | Check user permission |
| `assignRole($token, $userId, $roleId)` | Assign role to user |
| `revokeRole($token, $userId, $roleId)` | Revoke role from user |
| `getUserRoles($token, $userId): Role[]` | Get user's roles |
| `listRoles($token): Role[]` | List all roles |
| `listPermissions($token): Permission[]` | List permission tree |

### ABAC

| Method | Description |
|--------|-------------|
| `evaluateABAC($token, $action, $resource, $subject, $conditions, $tenantId): ABACResult` | Evaluate ABAC policy |
| `checkPolicy($token, $subject, $resource, $action, $context): ABACResult` | Full policy check |

### User Management

| Method | Description |
|--------|-------------|
| `getUser($token, $userId)` | Get user by ID |
| `listUsers($token, $params)` | List users |
| `createUser($token, $data)` | Create user |
| `updateUser($token, $userId, $data)` | Update user |
| `deleteUser($token, $userId)` | Delete user |

## Framework Integration

### PHP 8 Attributes (any framework)

```php
use Ggid\Sdk\Attribute\RequiresAuth;
use Ggid\Sdk\Attribute\RequirePermission;
use Ggid\Sdk\Attribute\RequireRole;
use Ggid\Sdk\MiddlewareResolver;

class ProductController
{
    #[RequiresAuth]
    #[RequirePermission('products', 'read')]
    public function list(): void
    {
        // Only called if user has products:read permission
    }

    #[RequiresAuth]
    #[RequireRole('admin')]
    public function adminPanel(): void
    {
        // Only called if user has admin role
    }
}

// In your router/dispatcher:
$resolver = new MiddlewareResolver($ggid);
$claims = $resolver->resolve(ProductController::class, 'list', $requestToken);
```

### Slim / Callable Middleware

```php
use Ggid\Sdk\AuthMiddleware;

// Auth required for all routes
$app->add(AuthMiddleware::create($ggid));

// Permission check on specific route
$app->get('/api/products', function ($req, $res) {
    // ...
})->add(AuthMiddleware::requirePermission($ggid, 'products', 'read'));

// Role check
$app->get('/admin', function ($req, $res) {
    // ...
})->add(AuthMiddleware::requireRole($ggid, 'admin'));
```

## Claims Object

```php
$claims = $ggid->verifyToken($jwt);

$claims->userId;      // string
$claims->tenantId;    // string
$claims->roles;       // string[]
$claims->scope;       // string (space-separated)
$claims->exp;         // int (Unix timestamp)
$claims->email;       // ?string

$claims->hasRole('admin');     // bool
$claims->hasScope('read');     // bool
$claims->isExpired();          // bool
```

## Dependencies

- PHP 8.1+
- guzzlehttp/guzzle ^7.0
- firebase/php-jwt ^6.10

## License

Apache-2.0
