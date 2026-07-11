/**
 * GGID Java SDK Quickstart — 5-minute JWT authentication integration.
 *
 * Shows how to:
 * 1. Login and get a JWT token
 * 2. Protect servlet routes with GGID filter
 * 3. Access user info from the JWT in your handlers
 *
 * Prerequisites:
 *   - GGID running (cd deploy && docker compose up -d)
 *   - Java 17+
 *   - A servlet container (Tomcat/Jetty) or Spring Boot
 *
 * For Spring Boot, add the GGID filter to your SecurityConfig:
 *   @Bean
 *   public FilterRegistrationBean<GGIDSecurityFilter> ggidFilter() {
 *       FilterRegistrationBean<GGIDSecurityFilter> bean = new FilterRegistrationBean<>();
 *       bean.setFilter(new GGIDSecurityFilter("http://localhost:8080"));
 *       bean.addUrlPatterns("/api/*");
 *       bean.setOrder(1);
 *       return bean;
 *   }
 */
package com.example;

import dev.ggid.sdk.GGIDClient;
import dev.ggid.sdk.GGIDUser;
import dev.ggid.sdk.GGIDSecurityFilter;
import dev.ggid.sdk.TokenSet;

public class QuickstartApp {
    public static void main(String[] args) throws Exception {
        String gatewayUrl = "http://localhost:8080";
        String tenantId = "00000000-0000-0000-0000-000000000001";

        // Step 1: Create client and login
        GGIDClient client = new GGIDClient(gatewayUrl, tenantId);
        TokenSet tokens = client.login("admin", "Admin@123456");
        System.out.println("Login OK — token length: " + tokens.getAccessToken().length());

        // Step 2: Register the GGID filter (in web.xml or programmatically)
        // For Spring Boot, use FilterRegistrationBean as shown in the javadoc above.
        GGIDSecurityFilter filter = new GGIDSecurityFilter(gatewayUrl);
        filter.addSkipPath("/public");
        filter.addSkipPath("/api/health");

        // Step 3: Access user info in your controller
        // @GetMapping("/api/me")
        // public Map<String, Object> me(HttpServletRequest request) {
        //     GGIDUser user = (GGIDUser) request.getAttribute("ggid.user");
        //     return Map.of(
        //         "message", "authenticated!",
        //         "user", user.getUsername(),
        //         "email", user.getEmail(),
        //         "roles", user.getRoles()
        //     );
        // }

        System.out.println("\nSetup complete!");
        System.out.println("Add the GGIDSecurityFilter to your servlet container.");
        System.out.println("GGIDUser is available via request.getAttribute(\"ggid.user\")");
    }
}
