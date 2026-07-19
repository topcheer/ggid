package com.example.erp.config;

import dev.ggid.sdk.GGIDClient;
import dev.ggid.sdk.JwtVerifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * GGID configuration.
 *
 * The ERP app authenticates users via GGID login API and verifies JWTs
 * via the GGID JWKS endpoint. For permission checks, it uses a service
 * account (admin) token because the gateway blocks non-admin scopes
 * from the policy API.
 */
@Configuration
public class GgidConfig {

    @Value("${ggid.gateway-url:https://ggid.iot2.win}")
    private String gatewayUrl;

    @Value("${ggid.tenant-id:00000000-0000-0000-0000-000000000001}")
    private String tenantId;

    @Value("${ggid.jwks-url:https://ggid.iot2.win/oauth/jwks}")
    private String jwksUrl;

    @Value("${ggid.service-account-user:admin}")
    private String svcUser;

    @Value("${ggid.service-account-pass:}")
    private String svcPass;

    @Bean
    public GGIDClient ggidClient() {
        GGIDClient.Config config = new GGIDClient.Config(gatewayUrl);
        config.tenantId = tenantId;
        return new GGIDClient(config);
    }

    @Bean
    public JwtVerifier jwtVerifier() {
        // Issuer is "ggid-auth" (the auth service), not the gateway URL
        return new JwtVerifier(jwksUrl, "ggid-auth", 60);
    }

    public String getTenantId() { return tenantId; }
    public String getSvcUser() { return svcUser; }
    public String getSvcPass() { return svcPass; }
}
