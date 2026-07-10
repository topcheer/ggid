# GGID Java SDK

Java SDK for GGID IAM Platform — Spring Boot integration, JWT verification, RBAC.

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

### Spring Boot Starter

```java
@SpringBootApplication
public class MyApp {
    public static void main(String[] args) {
        SpringApplication.run(MyApp.class, args);
    }
}

@RestController
public class ProfileController {
    
    @GetMapping("/profile")
    public Map<String, Object> profile(@AuthenticationPrincipal GGIDUser user) {
        return Map.of(
            "sub", user.getSubject(),
            "email", user.getEmail(),
            "roles", user.getRoles()
        );
    }
    
    @GetMapping("/admin")
    @RequiresPermission(resource = "admin", action = "access")
    public String admin() {
        return "Welcome, admin!";
    }
}
```

### application.yml
```yaml
ggid:
  gateway-url: https://iam.example.com
  jwks-url: https://iam.example.com/.well-known/jwks.json
  tenant-id: 00000000-0000-0000-0000-000000000001
```

### API Client

```java
GGIDClient client = GGIDClient.builder()
    .gatewayUrl("https://iam.example.com")
    .tenantId("00000000-0000-0000-0000-000000000001")
    .build();

// Login
TokenSet tokens = client.login("admin", "Admin@123456");

// List users
List<User> users = client.listUsers(tokens.getAccessToken());

// Check permission
PolicyResult result = client.checkPermission(tokens.getAccessToken(), "documents", "read");
if (result.isAllowed()) {
    // Access granted
}
```

## License

Apache 2.0
