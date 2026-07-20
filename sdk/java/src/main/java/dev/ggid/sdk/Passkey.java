package dev.ggid.sdk;

import com.google.gson.Gson;
import java.io.IOException;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.net.URI;
import java.util.Map;

/** Passkey/WebAuthn registration and authentication API calls. */
public class Passkey {
    private final Config config;
    private final HttpClient httpClient;
    private final Gson gson = new Gson();

    public Passkey(Config config, HttpClient httpClient) {
        this.config = config;
        this.httpClient = httpClient;
    }

    /** Begin WebAuthn registration. Returns server challenge options. */
    @SuppressWarnings("unchecked")
    public Map<String, Object> beginRegistration(String accessToken, String deviceName)
            throws GGIDException, IOException {
        String body = gson.toJson(Map.of("type", "webauthn", "name", deviceName));
        HttpRequest req = HttpRequest.newBuilder()
                .uri(URI.create(config.getBaseUrl() + "/api/v1/auth/mfa/enroll"))
                .header("Authorization", "Bearer " + accessToken)
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(body))
                .build();
        try {
            HttpResponse<String> resp = httpClient.send(req, HttpResponse.BodyHandlers.ofString());
            if (resp.statusCode() >= 400) {
                throw new GGIDException("Passkey registration failed: " + resp.body(), resp.statusCode());
            }
            return gson.fromJson(resp.body(), Map.class);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new GGIDException("Interrupted", 0);
        }
    }

    /** Finish WebAuthn registration by verifying the attestation. */
    public void finishRegistration(String accessToken, String deviceID, String attestationResponse)
            throws GGIDException, IOException {
        String body = gson.toJson(Map.of("device_id", deviceID, "code", attestationResponse));
        HttpRequest req = HttpRequest.newBuilder()
                .uri(URI.create(config.getBaseUrl() + "/api/v1/auth/mfa/verify"))
                .header("Authorization", "Bearer " + accessToken)
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(body))
                .build();
        try {
            HttpResponse<String> resp = httpClient.send(req, HttpResponse.BodyHandlers.ofString());
            if (resp.statusCode() >= 400) {
                throw new GGIDException("Passkey verify failed: " + resp.body(), resp.statusCode());
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new GGIDException("Interrupted", 0);
        }
    }
}
