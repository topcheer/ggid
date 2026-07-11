# Java Spring Boot Integration Example

> Complete Spring Boot application using the GGID Java SDK for JWT verification and route protection.

---

## Prerequisites

- Java 17+
- Spring Boot 3.x
- GGID Gateway running at `http://localhost:8080`

---

## Project Setup

### Maven (`pom.xml`)

```xml
<dependencies>
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-web</artifactId>
    </dependency>
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-security</artifactId>
    </dependency>
    <dependency>
        <groupId>dev.ggid</groupId>
        <artifactId>ggid-sdk</artifactId>
        <version>1.0.0</version>
    </dependency>
</dependencies>
```

### Gradle (`build.gradle`)

```groovy
dependencies {
    implementation 'org.springframework.boot:spring-boot-starter-web'
    implementation 'org.springframework.boot:spring-boot-starter-security'
    implementation 'dev.ggid:ggid-sdk:1.0.0'
}
```

---

## Configuration

`src/main/resources/application.yml`:

```yaml
ggid:
  gateway-url: ${GGID_URL:http://localhost:8080}
  jwks-url: ${GGID_URL:http://localhost:8080}/.well-known/jwks.json
  tenant-id: ${TENANT_ID:00000000-0000-0000-0000-000000000001}
  api-key: ${GGID_API_KEY:your-api-key}

server:
  port: 8081
```

---

## Security Configuration

`src/main/java/com/example/config/SecurityConfig.java`:

```java
package com.example.config;

import dev.ggid.sdk.GGIDClient;
import dev.ggid.sdk.GGIDSecurityFilter;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.security.web.authentication.UsernamePasswordAuthenticationFilter;

@Configuration
@EnableWebSecurity
public class SecurityConfig {

    @Value("${ggid.gateway-url}")
    private String gatewayUrl;

    @Value("${ggid.jwks-url}")
    private String jwksUrl;

    @Bean
    public GGIDClient ggidClient() {
        return new GGIDClient(new GGIDClient.Config(gatewayUrl));
    }

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http, GGIDClient client) throws Exception {
        http
            .csrf(csrf -> csrf.disable())
            .sessionManagement(session -> session.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/health").permitAll()
                .anyRequest().authenticated()
            )
            .addFilterBefore(
                new GGIDSecurityFilter(client, jwksUrl),
                UsernamePasswordAuthenticationFilter.class
            );

        return http.build();
    }
}
```

---

## REST Controller

`src/main/java/com/example/controller/UserController.java`:

```java
package com.example.controller;

import dev.ggid.sdk.GGIDUser;
import jakarta.servlet.http.HttpServletRequest;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api")
public class UserController {

    @GetMapping("/me")
    public ResponseEntity<?> me(HttpServletRequest request) {
        GGIDUser user = (GGIDUser) request.getAttribute("ggid.user");
        if (user == null) {
            return ResponseEntity.status(401).body(Map.of("error", "not_authenticated"));
        }
        return ResponseEntity.ok(Map.of(
            "user_id", user.getId(),
            "email", user.getEmail(),
            "roles", user.getRoles(),
            "tenant_id", user.getTenantId()
        ));
    }

    @GetMapping("/users")
    public ResponseEntity<?> listUsers(HttpServletRequest request) {
        GGIDUser user = (GGIDUser) request.getAttribute("ggid.user");
        if (user == null || !user.getRoles().contains("admin")) {
            return ResponseEntity.status(403).body(Map.of("error", "admin_role_required"));
        }

        // In production, query your database scoped to user.getTenantId()
        List<Map<String, String>> users = List.of(
            Map.of("id", "usr_001", "username", "alice", "tenant_id", user.getTenantId()),
            Map.of("id", "usr_002", "username", "bob", "tenant_id", user.getTenantId())
        );
        return ResponseEntity.ok(Map.of("users", users, "count", users.size()));
    }

    @PostMapping("/users")
    public ResponseEntity<?> createUser(@RequestBody Map<String, String> body, HttpServletRequest request) {
        GGIDUser user = (GGIDUser) request.getAttribute("ggid.user");
        if (user == null || !user.getRoles().contains("admin")) {
            return ResponseEntity.status(403).body(Map.of("error", "admin_role_required"));
        }

        String username = body.get("username");
        String email = body.get("email");

        if (username == null || email == null) {
            return ResponseEntity.badRequest().body(Map.of("error", "username_and_email_required"));
        }

        return ResponseEntity.status(HttpStatus.CREATED).body(Map.of(
            "id", "usr_" + username,
            "username", username,
            "email", email,
            "tenant_id", user.getTenantId(),
            "created_by", user.getId()
        ));
    }

    @GetMapping("/health")
    public ResponseEntity<?> health() {
        return ResponseEntity.ok(Map.of("status", "ok", "service", "spring-demo"));
    }
}
```

---

## Application Entry Point

`src/main/java/com/example/Application.java`:

```java
package com.example;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication
public class Application {
    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }
}
```

---

## Run

```bash
export GGID_URL=http://localhost:8080
export GGID_API_KEY=your-api-key
export TENANT_ID=00000000-0000-0000-0000-000000000001

mvn spring-boot:run
# → Tomcat started on port 8081
```

---

## Test the Endpoints

### Health Check (public)

```bash
curl http://localhost:8081/health
# → {"status":"ok","service":"spring-demo"}
```

### Protected Route Without Token (401)

```bash
curl http://localhost:8081/api/me
# → {"error":"not_authenticated"}
```

### Get User Info

```bash
JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"Admin123!"}' | jq -r .access_token)

curl -s http://localhost:8081/api/me \
  -H "Authorization: Bearer $JWT" | jq .
```

### List Users (requires admin role)

```bash
curl -s http://localhost:8081/api/users \
  -H "Authorization: Bearer $JWT" | jq .
```

---

## Key Takeaways

1. **`GGIDSecurityFilter`** verifies JWT and sets `ggid.user` request attribute.
2. **Spring Security** filter chain integrates with GGID for stateless auth.
3. **`GGIDUser`** provides `getId()`, `getEmail()`, `getRoles()`, `getTenantId()`.
4. **Role checks** done via `user.getRoles().contains("admin")`.
5. **Tenant isolation** — use `user.getTenantId()` for all database queries.

---

*See also: [SDK Quickstart](../quickstart/sdk-quickstart.md) | [3-Line Integration](../quickstart/3-line-integration.md) | [RBAC Guide](../guides/role-based-access.md)*

*Last updated: 2025-07-11*
