package com.example.erp.config;

import dev.ggid.sdk.GGIDClient;
import dev.ggid.sdk.JwtVerifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * GGID configuration — creates the GGIDClient and JwtVerifier beans.
 *
 * The GGID gateway URL and tenant ID come from application.yml.
 * In production, the JWKS URL would be https://your-ggid/oauth/jwks.
 */
@Configuration
public class GgidConfig {

    @Value("${ggid.gateway-url:https://ggid.iot2.win}")
    private String gatewayUrl;

    @Value("${ggid.tenant-id:00000000-0000-0000-0000-000000000001}")
    private String tenantId;

    @Value("${ggid.jwks-url:https://ggid.iot2.win/oauth/jwks}")
    private String jwksUrl;

    @Bean
    public GGIDClient ggidClient() {
        GGIDClient.Config config = new GGIDClient.Config(gatewayUrl);
        config.tenantId = tenantId;
        return new GGIDClient(config);
    }

    @Bean
    public JwtVerifier jwtVerifier() {
        return new JwtVerifier(jwksUrl, gatewayUrl, 30);
    }
}
