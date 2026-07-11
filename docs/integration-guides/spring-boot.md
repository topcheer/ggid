# Spring Boot Integration Guide

> Add GGID authentication to a Spring Boot Java app using the Java SDK.

---

## Install

### Maven

```xml
<dependency>
    <groupId>dev.ggid</groupId>
    <artifactId>ggid-sdk-java</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

```groovy
implementation 'dev.ggid:ggid-sdk-java:1.0.0'
```

## Minimal Setup

### JWT Verification Filter

```java
import dev.ggid.sdk.GGIDVerifier;
import dev.ggid.sdk.GGIDClaims;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.security.web.authentication.UsernamePasswordAuthenticationFilter;

@SpringBootApplication
public class App {
    public static void main(String[] args) {
        SpringApplication.run(App.class, args);
    }

    @Bean
    public GGIDVerifier ggidVerifier() {
        return new GGIDVerifier(
            System.getenv().getOrDefault("GGID_URL", "http://localhost:8080"),
            System.getenv("JWT_SECRET")
        );
    }
}
```

### Security Configuration

```java
@Configuration
@EnableWebSecurity
public class SecurityConfig {

    @Autowired
    private GGIDVerifier verifier;

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http
            .csrf(csrf -> csrf.disable())
            .sessionManagement(sm -> sm.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/health", "/api/auth/**").permitAll()
                .anyRequest().authenticated()
            )
            .addFilterBefore(
                new GGIDJwtFilter(verifier),
                UsernamePasswordAuthenticationFilter.class
            );

        return http.build();
    }
}
```

### JWT Filter

```java
import dev.ggid.sdk.GGIDClaims;
import jakarta.servlet.FilterChain;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.web.filter.OncePerRequestFilter;

import java.util.List;
import java.util.stream.Collectors;

public class GGIDJwtFilter extends OncePerRequestFilter {

    private final GGIDVerifier verifier;

    public GGIDJwtFilter(GGIDVerifier verifier) {
        this.verifier = verifier;
    }

    @Override
    protected void doFilterInternal(
            HttpServletRequest request,
            HttpServletResponse response,
            FilterChain chain) throws ServletException, IOException {

        String header = request.getHeader("Authorization");
        if (header == null || !header.startsWith("Bearer ")) {
            chain.doFilter(request, response);
            return;
        }

        String token = header.substring(7);
        try {
            GGIDClaims claims = verifier.verify(token);

            // Set authentication with scopes as authorities
            var authorities = claims.getScopes().stream()
                .map(SimpleGrantedAuthority::new)
                .collect(Collectors.toList());

            var auth = new UsernamePasswordAuthenticationToken(
                claims.getUserId(), null, authorities);

            // Store tenant ID for downstream use
            request.setAttribute("tenantId", claims.getTenantId());

            SecurityContextHolder.getContext().setAuthentication(auth);
        } catch (Exception e) {
            response.sendError(401, "Invalid token");
            return;
        }

        chain.doFilter(request, response);
    }
}
```

### Protected Controller

```java
@RestController
@RequestMapping("/api")
public class UserController {

    @GetMapping("/me")
    public Map<String, Object> me(HttpServletRequest request) {
        var auth = SecurityContextHolder.getContext().getAuthentication();
        String tenantId = (String) request.getAttribute("tenantId");

        return Map.of(
            "userId", auth.getName(),
            "tenantId", tenantId
        );
    }

    @DeleteMapping("/users/{id}")
    @PreAuthorize("hasAuthority('delete:users')")
    public void deleteUser(@PathVariable String id) {
        // ...
    }

    @GetMapping("/users")
    @PreAuthorize("hasAuthority('read:users')")
    public List<User> listUsers() {
        // ...
    }
}
```

### application.yml

```yaml
ggid:
  url: ${GGID_URL:http://localhost:8080}
  jwt-secret: ${JWT_SECRET}

server:
  port: 8081
```

---

*See: [SDK Reference](../sdk-reference.md) | [Java SDK](../sdk-reference.md#java-sdk)*