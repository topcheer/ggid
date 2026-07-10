# GGID Java SDK

JWT verification, user management, and RBAC for Java / Spring Boot / Jakarta EE.

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

### API Client

```java
GGIDClient client = new GGIDClient("https://iam.example.com");

// Login
TokenSet tokens = client.login("admin", "Admin@123456");

// List users
JsonNode users = client.listUsers(tokens.getAccessToken());

// Check permission
JsonNode result = client.checkPermission(tokens.getAccessToken(), "documents", "read");
boolean allowed = result.path("allowed").asBoolean();
```

### Spring Boot Security Filter

```java
@Bean
public FilterRegistrationBean<GGIDFilter> ggidFilter() {
    FilterRegistrationBean<GGIDFilter> bean = new FilterRegistrationBean<>();
    bean.setFilter(new GGIDFilter());
    bean.addUrlPatterns("/api/*");
    return bean;
}
```

### Get User from Request

```java
@GetMapping("/profile")
public Map<String, String> profile(HttpServletRequest request) {
    return Map.of(
        "sub", (String) request.getAttribute("ggid.sub"),
        "email", (String) request.getAttribute("ggid.email"),
        "tenant_id", (String) request.getAttribute("ggid.tenant_id")
    );
}
```

## License

Apache 2.0
