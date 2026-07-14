# GGID Dart SDK

Official Dart SDK for the **GGID IAM Platform** — JWT verification, RBAC/ABAC authorization, and Shelf middleware.

## Quick Start

### 1. Install

Add to your `pubspec.yaml`:

```yaml
dependencies:
  ggid: ^1.0.0
```

Or from source:
```bash
cd sdk/dart
dart pub get
dart test
```

### 2. Initialize

```dart
import 'package:ggid/ggid_client.dart';

final ggid = GGIDClient(
  baseUrl: 'https://ggid.iot2.win',
  tenantId: '00000000-0000-0000-0000-000000000001',
).withJwks();
```

### 3. Verify Token

```dart
final claims = await ggid.verifyToken(jwt);
print('User: ${claims.userId}, Roles: ${claims.roles}');
```

### 4. Check Permission (RBAC)

```dart
final allowed = await ggid.checkPermission(token, 'products', 'read');
if (!allowed) return Response.forbidden('denied');
```

### 5. Check Policy (ABAC)

```dart
final allowed = await ggid.checkPolicySimple(
  token, 'user-123', 'documents', 'read',
  context: {'department': 'finance'},
);
```

## Shelf Integration

```dart
import 'package:shelf/shelf.dart';
import 'package:shelf/shelf_io.dart' as io;
import 'package:ggid/ggid_client.dart';
import 'package:ggid/src/middleware.dart';

final ggid = GGIDClient(
  baseUrl: 'https://ggid.iot2.win',
  tenantId: '00000000-0000-0000-0000-000000000001',
).withJwks();

final handler = const Pipeline()
  .addMiddleware(logRequests())
  .addMiddleware(ggid.authMiddleware())
  .addMiddleware(ggid.requirePermission('products', 'read'))
  .addHandler((Request request) async {
    final claims = getClaims(request)!;
    return Response.ok('Hello ${claims.userId}');
  });

await io.serve(handler, 'localhost', 8080);
```

## API Reference

### Authentication

| Method | Description |
|--------|-------------|
| `login(username, password)` | Login with credentials |
| `register(username, email, password, name)` | Register new user |
| `refreshToken(refreshToken)` | Refresh tokens |
| `verifyToken(token)` | Verify JWT, returns `Claims` |
| `getUserInfo(token)` | Get OIDC UserInfo |

### OAuth/OIDC

| Method | Description |
|--------|-------------|
| `getDiscovery()` | Get OIDC discovery document |
| `getJwks()` | Get JWKS keys |
| `getAuthorizeUrl(clientId, redirectUri, scope?, state?)` | Build authorize URL |
| `exchangeCode(code, redirectUri, clientId, clientSecret)` | Exchange auth code for tokens |
| `revokeToken(token)` | Revoke a token (RFC 7009) |

### RBAC

| Method | Description |
|--------|-------------|
| `checkPermission(token, resource, action)` | Check if user can perform action |
| `assignRole(token, userId, roleId)` | Assign role to user |
| `revokeRole(token, userId, roleId)` | Revoke role from user |
| `getUserRoles(token, userId)` | Get user's roles |
| `listRoles(token)` | List all roles |
| `listPermissions(token)` | List all permissions |
| `createRole(token, name, key, description?)` | Create a new role |
| `deleteRole(token, roleId)` | Delete a role |
| `hasRole(token, userId, roleKey)` | Check if user has role (extension) |
| `hasAnyRole(token, userId, roleKeys)` | Check if user has any of roles (extension) |

### ABAC

| Method | Description |
|--------|-------------|
| `evaluateAbac(token, AbacEvalRequest)` | Evaluate ABAC policy |
| `checkPolicy(token, PolicyCheckRequest)` | Check policy with context |
| `checkPolicySimple(token, subject, resource, action, context?)` | Convenience check (extension) |

### Middleware (Shelf)

| Middleware | Description |
|-----------|-------------|
| `ggid.authMiddleware()` | JWT authentication |
| `ggid.requirePermission('res', 'act')` | Require specific permission |
| `ggid.requireRole('admin')` | Require specific role |

## License

Apache-2.0
