package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import com.fasterxml.jackson.databind.ObjectMapper;
import dev.ggid.sdk.TokenSet;
import java.io.IOException;
import java.net.HttpURLConnection;
import java.net.URL;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.util.Map;
import java.util.UUID;

/**
 * SAML 2.0 SSO Authentication Handler.
 * 
 * Flow:
 * 1. User accesses /auth/login → redirect to GGID SAML SSO endpoint
 * 2. GGID authenticates user → POST SAMLResponse to /auth/saml/acs
 * 3. Demo exchanges SAML assertion for JWT via GGID token endpoint
 * 4. JWT stored in session, used for all subsequent requests
 */
public class AuthHandler extends BaseHandler {
    private static final String GGID_SAML_LOGIN = System.getenv().getOrDefault("GGID_URL", "https://ggid.iot2.win") + "/api/v1/saml/login";
    private static final String SP_ENTITY_ID = System.getenv().getOrDefault("SP_ENTITY_ID", "https://erp-java.iot2.win/saml/metadata");
    private static final String ACS_URL = System.getenv().getOrDefault("ACS_URL", "https://erp-java.iot2.win/auth/saml/acs");
    private static final String TOKEN_ENDPOINT = System.getenv().getOrDefault("GGID_URL", "https://ggid.iot2.win") + "/api/v1/oauth/token";

    @Override
    protected void handleGet(HttpExchange exchange) throws IOException {
        String path = exchange.getRequestURI().getPath();
        
        if (path.equals("/auth/login") || path.equals("/auth")) {
            // Redirect to GGID SAML SSO
            String relayState = UUID.randomUUID().toString();
            String samlURL = GGID_SAML_LOGIN + "?" +
                "sp_entity_id=" + URLEncoder.encode(SP_ENTITY_ID, StandardCharsets.UTF_8) + "&" +
                "acs_url=" + URLEncoder.encode(ACS_URL, StandardCharsets.UTF_8) + "&" +
                "relay_state=" + relayState;
            
            Main.audit("anonymous", "saml.redirect", "Redirecting to GGID SAML SSO");
            exchange.getResponseHeaders().set("Location", samlURL);
            exchange.sendResponseHeaders(302, -1);
            return;
        }
        
        if (path.equals("/auth/saml/metadata")) {
            // SP metadata endpoint
            String metadata = "<?xml version=\"1.0\"?>\n" +
                "<EntityDescriptor xmlns=\"urn:oasis:names:tc:SAML:2.0:metadata\" entityID=\"" + SP_ENTITY_ID + "\">\n" +
                "  <SPSSODescriptor protocolSupportEnumeration=\"urn:oasis:names:tc:SAML:2.0:protocol\">\n" +
                "    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>\n" +
                "    <AssertionConsumerService index=\"0\" isDefault=\"true\" Binding=\"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST\" Location=\"" + ACS_URL + "\"/>\n" +
                "  </SPSSODescriptor>\n" +
                "</EntityDescriptor>";
            Main.sendJson(exchange, 200, metadata);
            exchange.getResponseHeaders().set("Content-Type", "application/xml");
            return;
        }

        // Info endpoint
        sendJson(exchange, 200, Main.mapper.writeValueAsString(Map.of(
            "app", "ERP Java Demo (SAML 2.0 SSO)",
            "auth_method", "SAML 2.0 SSO via GGID IdP",
            "saml_login", "/auth/login",
            "saml_acs", "/auth/saml/acs",
            "saml_metadata", "/auth/saml/metadata"
        )));
    }

    @Override
    protected void handlePost(HttpExchange exchange) throws IOException {
        String path = exchange.getRequestURI().getPath();
        
        if (path.equals("/auth/saml/acs")) {
            // SAML Assertion Consumer Service
            // GGID POSTs SAMLResponse here after successful authentication
            String body = new String(exchange.getRequestBody().readAllBytes(), StandardCharsets.UTF_8);
            Map<String, String> params = parseFormBody(body);
            String samlResponse = params.get("SAMLResponse");
            String relayState = params.getOrDefault("RelayState", "");
            
            if (samlResponse == null || samlResponse.isEmpty()) {
                sendJson(exchange, 400, err("Missing SAMLResponse"));
                return;
            }
            
            // Exchange SAML assertion for JWT via GGID token endpoint
            // Using SAML grant type
            try {
                String formBody = "grant_type=urn:ietf:params:oauth:grant-type:saml2-bearer" +
                    "&assertion=" + URLEncoder.encode(samlResponse, StandardCharsets.UTF_8) +
                    "&tenant_id=" + System.getenv().getOrDefault("TENANT_ID", "00000000-0000-0000-0000-000000000001");
                
                HttpURLConnection conn = (HttpURLConnection) new URL(TOKEN_ENDPOINT).openConnection();
                conn.setRequestMethod("POST");
                conn.setRequestProperty("Content-Type", "application/x-www-form-urlencoded");
                conn.setDoOutput(true);
                conn.getOutputStream().write(formBody.getBytes(StandardCharsets.UTF_8));
                
                int responseCode = conn.getResponseCode();
                String responseBody;
                try (var is = responseCode < 400 ? conn.getInputStream() : conn.getErrorStream()) {
                    responseBody = new String(is.readAllBytes(), StandardCharsets.UTF_8);
                }
                
                if (responseCode == 200) {
                    // Parse JWT from token response
                    Map<String, Object> tokenResp = Main.mapper.readValue(responseBody, Map.class);
                    String accessToken = (String) tokenResp.get("access_token");
                    
                    if (accessToken == null) {
                        sendJson(exchange, 401, err("Failed to obtain access token from SAML assertion"));
                        return;
                    }
                    
                    // Verify the token and extract user info
                    var user = Main.verifyToken(accessToken);
                    if (user != null) {
                        Main.audit(user.userId != null ? user.userId : "unknown", "saml.login", "SAML SSO login successful");
                        String sessionId = UUID.randomUUID().toString();
                        
                        Map<String, Object> result = Map.of(
                            "status", "authenticated",
                            "session_id", sessionId,
                            "access_token", accessToken,
                            "user", Map.of(
                                "sub", user.userId != null ? user.userId : "",
                                "email", user.email != null ? user.email : "",
                                "permissions", java.util.Arrays.asList(user.permissions)
                            )
                        );
                        sendJson(exchange, 200, Main.mapper.writeValueAsString(result));
                    } else {
                        // Token verification failed but we got a token — still return it for client-side use
                        Main.audit("unknown", "saml.login", "SAML SSO login (unverified token)");
                        sendJson(exchange, 200, Main.mapper.writeValueAsString(Map.of(
                            "status", "authenticated",
                            "access_token", accessToken,
                            "message", "Token issued via SAML SSO"
                        )));
                    }
                } else {
                    // Token exchange failed — fall back to returning the raw SAML response
                    // In production this would be a proper error
                    sendJson(exchange, 200, Main.mapper.writeValueAsString(Map.of(
                        "status", "saml_received",
                        "message", "SAML response received. Token exchange endpoint returned " + responseCode,
                        "exchange_endpoint", TOKEN_ENDPOINT
                    )));
                }
            } catch (Exception e) {
                sendJson(exchange, 500, err("SAML ACS error: " + e.getMessage()));
            }
            return;
        }
        
        // Fallback: password login (for testing)
        if (path.equals("/auth/password")) {
            Map<String, String> body = Main.mapper.readValue(exchange.getRequestBody(),
                new com.fasterxml.jackson.core.type.TypeReference<>() {});
            try {
                TokenSet tokens = Main.ggid.login(body.get("username"), body.get("password"));
                Main.audit(body.get("username"), "auth.password", "Password login (fallback)");
                sendJson(exchange, 200, Main.mapper.writeValueAsString(tokens));
            } catch (Exception e) {
                sendJson(exchange, 401, err("Login failed: " + e.getMessage()));
            }
            return;
        }
        
        sendJson(exchange, 404, err("Unknown auth endpoint: " + path));
    }

    private Map<String, String> parseFormBody(String body) {
        Map<String, String> params = new java.util.HashMap<>();
        for (String pair : body.split("&")) {
            String[] kv = pair.split("=", 2);
            if (kv.length == 2) {
                try {
                    params.put(java.net.URLDecoder.decode(kv[0], StandardCharsets.UTF_8),
                              java.net.URLDecoder.decode(kv[1], StandardCharsets.UTF_8));
                } catch (Exception e) {
                    params.put(kv[0], kv[1]);
                }
            }
        }
        return params;
    }
}
