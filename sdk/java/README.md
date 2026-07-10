# GGID Java SDK

A production-ready Java client SDK for the [GGID](https://github.com/ggid/ggid) IAM platform.

## Requirements

- Java 17+
- Maven or Gradle

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

## Quick Start

```java
import dev.ggid.sdk.GGIDClient;

GGIDClient client = new GGIDClient(new GGIDClient.Config("https://iam.example.com"));

// Login
GGIDClient.TokenSet tokens = client.login("alice", "SecurePass@123");
System.out.println("Access token: " + tokens.accessToken);

// Create user
GGIDClient.User user = client.createUser("bob", "bob@example.com", "SecurePass@123");

// Check permission
GGIDClient.PermissionResult result = client.checkPermission(user.id, "documents", "read");
System.out.println("Allowed: " + result.allowed);
```

## API Reference

| Method | Description |
|--------|-------------|
| `login(username, password)` | Authenticate with username/password |
| `refreshToken(refreshToken)` | Refresh an access token |
| `logout(accessToken)` | Invalidate an access token |
| `createUser(username, email, password)` | Create a new user |
| `getUser(userId)` | Get user by ID |
| `deleteUser(userId)` | Delete a user |
| `listUsers(page, pageSize)` | List users with pagination |
| `assignRole(userId, roleId)` | Assign role to user |
| `createRole(key, name)` | Create a role |
| `listRoles()` | List roles |
| `createOrg(name)` | Create an organization |
| `listOrgs()` | List organizations |
| `checkPermission(userId, resource, action)` | Check authorization |

## Error Handling

```java
try {
    client.getUser("nonexistent");
} catch (GGIDException e) {
    if (e.isNotFound()) {
        // 404
    } else if (e.isRateLimited()) {
        // 429
    } else if (e.isConflict()) {
        // 409
    }
    System.out.println(e.getStatusCode() + ": " + e.getMessage());
}
```

## Configuration

```java
GGIDClient.Config config = new GGIDClient.Config("https://iam.example.com");
config.tenantId = "your-tenant-uuid";
config.apiKey = "your-api-key";

GGIDClient client = new GGIDClient(config);
```

## License

Apache 2.0
