# GGID Java SDK Guide

This guide covers installing, configuring, and using the GGID Java SDK for user management, authentication, authorization, and organization management.

## Installation

### Maven

```xml
<dependency>
    <groupId>dev.ggid</groupId>
    <artifactId>ggid-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

```groovy
implementation 'dev.ggid:ggid-sdk:1.0.0'
```

### Manual

Build from source:

```bash
cd sdk/java
mvn clean install
# JAR installed to local Maven repository
```

## Configuration

### GGIDClient.Config

```java
GGIDClient.Config config = new GGIDClient.Config(
    "https://api.ggid.example.com",  // Gateway URL
    "00000000-0000-0000-0000-000000000001",  // Tenant ID
    "your-api-key"                    // API Key (for service-to-service)
);

GGIDClient client = new GGIDClient(config);
```

### Configuration Parameters

| Parameter    | Required | Description                          |
|--------------|----------|--------------------------------------|
| gatewayUrl   | Yes      | GGID API Gateway base URL            |
| tenantId     | Yes      | UUID of your tenant                  |
| apiKey       | No       | API key for admin operations         |

## Authentication

### Login

```java
TokenSet tokens = client.login("user@example.com", "password");
// tokens.getAccessToken()  → JWT for API calls
// tokens.getRefreshToken() → Use to refresh after expiry
// tokens.getExpiresIn()    → Seconds until expiry
```

### Refresh Token

```java
TokenSet newTokens = client.refreshToken(tokens.getRefreshToken());
```

### Logout

```java
client.logout(tokens.getAccessToken());
```

### JWT Verification (Server-Side)

```java
JwtVerifier verifier = new JwtVerifier("https://api.ggid.example.com/.well-known/jwks.json");

// Verify and extract claims
GGIDUser user = verifier.verify(accessToken);
// user.getUserId()    → UUID
// user.getTenantId()  → UUID
// user.getScopes()    → List<String>
// user.getExpiresAt() → Instant
```

## User Management

### Create User

```java
User user = client.createUser(
    "newuser@example.com",    // username
    "newuser@example.com",    // email
    "SecurePassword123!"      // password
);
// user.getId()        → UUID
// user.getUsername()  → "newuser@example.com"
// user.getEmail()     → "newuser@example.com"
```

### Get User

```java
User user = client.getUser("user-uuid-here");
```

### Update User

```java
User updated = client.updateUser(
    "user-uuid-here",
    "newemail@example.com",   // email
    "+1234567890"             // phone
);
```

### Delete User

```java
client.deleteUser("user-uuid-here");
```

### List Users (Paginated)

```java
PageResult<User> page = client.listUsers(1, 20);  // page 1, 20 per page
List<User> users = page.getItems();
int totalPages = page.getTotalPages();
long total = page.getTotal();
```

## Role Management

### Create Role

```java
Role role = client.createRole(
    "admin",              // key (unique within tenant)
    "Administrator"       // display name
);
```

> The `key` field must be unique within the tenant. Empty keys cause a UNIQUE constraint violation.

### List Roles

```java
PageResult<Role> roles = client.listRoles();
for (Role r : roles.getItems()) {
    System.out.println(r.getKey() + ": " + r.getName());
}
```

### Assign Role to User

```java
client.assignRole("user-uuid", "role-uuid");
```

## Organization Management

### Create Organization

```java
Organization org = client.createOrg("Engineering Team");
```

### List Organizations

```java
PageResult<Organization> orgs = client.listOrgs();
```

## Authorization / Policy

### Check Permission

```java
PermissionResult result = client.checkPermission(
    "user-uuid",
    "document:report.pdf",   // resource
    "read"                   // action
);

if (result.isAllowed()) {
    // Grant access
} else {
    // Deny
}
```

## Servlet Integration

### Authentication Filter

```java
// web.xml or annotation-based
@WebFilter("/*")
public class AuthFilter extends GGIDAuthFilter {
    @Override
    protected GGIDClient getClient() {
        return client;  // Your configured GGIDClient
    }

    @Override
    protected String getLoginPage() {
        return "/login";
    }
}
```

### Security Filter (RBAC Enforcement)

```java
@WebFilter("/admin/*")
public class AdminFilter extends GGIDSecurityFilter {
    @Override
    protected String[] getRequiredScopes() {
        return new String[]{"admin"};
    }

    @Override
    protected GGIDClient getClient() {
        return client;
    }
}
```

## Error Handling

```java
try {
    User user = client.getUser("invalid-uuid");
} catch (GGIDException e) {
    switch (e.getStatusCode()) {
        case 401:  // Unauthorized — invalid/expired token
            break;
        case 403:  // Forbidden — insufficient scope
            break;
        case 404:  // Not Found
            break;
        case 409:  // Conflict — duplicate resource
            break;
        case 429:  // Rate limited
            break;
        default:
            break;
    }
}
```

## Complete Example

```java
import dev.ggid.sdk.*;

public class GGIDExample {
    public static void main(String[] args) throws Exception {
        // Initialize
        GGIDClient client = new GGIDClient(new GGIDClient.Config(
            "https://api.ggid.example.com",
            "00000000-0000-0000-0000-000000000001",
            System.getenv("GGID_API_KEY")
        ));

        // Login
        TokenSet tokens = client.login("admin@example.com", "password");

        // Create a user
        User newUser = client.createUser("alice", "alice@example.com", "SecurePass1!");

        // Create and assign role
        Role role = client.createRole("developer", "Developer");
        client.assignRole(newUser.getId(), role.getId());

        // Check permission
        PermissionResult perm = client.checkPermission(newUser.getId(), "api:read", "execute");
        System.out.println("Permission granted: " + perm.isAllowed());

        // Cleanup
        client.deleteUser(newUser.getId());
        client.logout(tokens.getAccessToken());
    }
}
```

## See Also

- [Node.js SDK Guide](node-sdk-guide.md)
- [Quick Start](quick-start.md)
- [API Reference](api-reference.md)
- REST API
