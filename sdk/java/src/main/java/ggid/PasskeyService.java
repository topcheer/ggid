package ggid;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

/**
 * Passkey/WebAuthn API calls for GGID Java SDK.
 * Provides server-side methods to initiate passkey registration and authentication.
 */
public class PasskeyService {

    private final String baseUrl;
    private final String tenantId;
    private final HttpClient httpClient;

    public PasskeyService(String baseUrl, String tenantId) {
        this.baseUrl = baseUrl;
        this.tenantId = tenantId;
        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(5))
                .build();
    }

    /**
     * Initiates passkey registration for a user.
     * @param accessToken Admin or user JWT token
     * @param userId The user ID to register a passkey for
     * @return JSON response containing session_id, challenge, and rp_id
     */
    public String registerBegin(String accessToken, String userId) throws Exception {
        String body = String.format("{\"user_id\":\"%s\"}", userId);
        return post("/api/v1/auth/webauthn/register/begin", accessToken, body);
    }

    /**
     * Completes passkey registration by submitting the credential.
     * @param accessToken JWT token
     * @param sessionId The session ID from registerBegin
     * @param credentialJson The WebAuthn credential JSON from the browser
     * @return JSON response confirming registration
     */
    public String registerFinish(String accessToken, String sessionId, String credentialJson) throws Exception {
        String body = String.format("{\"session_id\":\"%s\",\"credential\":%s}", sessionId, credentialJson);
        return post("/api/v1/auth/webauthn/register/finish", accessToken, body);
    }

    /**
     * Initiates passkey authentication.
     * @param accessToken JWT token
     * @return JSON response containing session_id and challenge
     */
    public String authBegin(String accessToken) throws Exception {
        return post("/api/v1/auth/webauthn/login/begin", accessToken, "{}");
    }

    /**
     * Completes passkey authentication.
     * @param accessToken JWT token
     * @param sessionId The session ID from authBegin
     * @param assertionJson The WebAuthn assertion JSON from the browser
     * @return JSON response with user info and tokens
     */
    public String authFinish(String accessToken, String sessionId, String assertionJson) throws Exception {
        String body = String.format("{\"session_id\":\"%s\",\"assertion\":%s}", sessionId, assertionJson);
        return post("/api/v1/auth/webauthn/login/finish", accessToken, body);
    }

    /**
     * Lists registered passkeys for a user.
     * @param accessToken JWT token
     * @param userId Optional user ID filter
     * @return JSON response with passkey list
     */
    public String listPasskeys(String accessToken, String userId) throws Exception {
        String url = "/api/v1/auth/webauthn/passkeys";
        if (userId != null && !userId.isEmpty()) {
            url += "?user_id=" + userId;
        }
        return get(url, accessToken);
    }

    /**
     * Revokes a passkey credential.
     * @param accessToken JWT token
     * @param credentialId The credential ID to revoke
     */
    public void revokePasskey(String accessToken, String credentialId) throws Exception {
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + "/api/v1/auth/webauthn/passkeys/" + credentialId))
                .header("Authorization", "Bearer " + accessToken)
                .header("X-Tenant-ID", tenantId)
                .DELETE()
                .timeout(Duration.ofSeconds(10))
                .build();
        httpClient.send(request, HttpResponse.BodyHandlers.discarding());
    }

    private String post(String path, String accessToken, String body) throws Exception {
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + path))
                .header("Authorization", "Bearer " + accessToken)
                .header("X-Tenant-ID", tenantId)
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(body))
                .timeout(Duration.ofSeconds(10))
                .build();
        HttpResponse<String> resp = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
        return resp.body();
    }

    private String get(String path, String accessToken) throws Exception {
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + path))
                .header("Authorization", "Bearer " + accessToken)
                .header("X-Tenant-ID", tenantId)
                .timeout(Duration.ofSeconds(10))
                .GET()
                .build();
        HttpResponse<String> resp = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
        return resp.body();
    }
}
