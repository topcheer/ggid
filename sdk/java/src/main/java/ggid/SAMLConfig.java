package ggid;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.time.format.DateTimeFormatter;
import java.time.temporal.ChronoUnit;

/**
 * SAMLConfig holds configuration for a SAML Service Provider.
 * Used to generate SP metadata for IdP integration.
 */
public class SAMLConfig {
    private String entityId;
    private String acsUrl;
    private String sloUrl;
    private boolean signRequests;

    public SAMLConfig() {}

    public SAMLConfig(String entityId, String acsUrl, String sloUrl) {
        this.entityId = entityId;
        this.acsUrl = acsUrl;
        this.sloUrl = sloUrl;
    }

    // Getters and setters
    public String getEntityId() { return entityId; }
    public void setEntityId(String entityId) { this.entityId = entityId; }
    public String getAcsUrl() { return acsUrl; }
    public void setAcsUrl(String acsUrl) { this.acsUrl = acsUrl; }
    public String getSloUrl() { return sloUrl; }
    public void setSloUrl(String sloUrl) { this.sloUrl = sloUrl; }
    public boolean isSignRequests() { return signRequests; }
    public void setSignRequests(boolean signRequests) { this.signRequests = signRequests; }

    /**
     * Generates SAML SP metadata XML for this configuration.
     * @return SP EntityDescriptor XML string
     */
    public String generateSPMetadata() {
        String validUntil = DateTimeFormatter.ISO_INSTANT.format(
            Instant.now().plus(365, ChronoUnit.DAYS));

        StringBuilder sb = new StringBuilder();
        sb.append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n");
        sb.append("<EntityDescriptor xmlns=\"urn:oasis:names:tc:SAML:2.0:metadata\"")
          .append(" entityID=\"").append(escapeXml(entityId)).append("\"")
          .append(" validUntil=\"").append(validUntil).append("\">\n");
        sb.append("  <SPSSODescriptor protocolSupportEnumeration=\"urn:oasis:names:tc:SAML:2.0:protocol\">\n");
        sb.append("    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>\n");
        sb.append("    <AssertionConsumerService index=\"0\" isDefault=\"true\"")
          .append(" Binding=\"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST\"")
          .append(" Location=\"").append(escapeXml(acsUrl)).append("\"/>\n");
        if (sloUrl != null && !sloUrl.isEmpty()) {
            sb.append("    <SingleLogoutService")
              .append(" Binding=\"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect\"")
              .append(" Location=\"").append(escapeXml(sloUrl)).append("\"/>\n");
        }
        sb.append("  </SPSSODescriptor>\n");
        sb.append("</EntityDescriptor>");
        return sb.toString();
    }

    private static String escapeXml(String s) {
        if (s == null) return "";
        return s.replace("&", "&amp;").replace("<", "&lt;")
                .replace(">", "&gt;").replace("\"", "&quot;").replace("'", "&apos;");
    }

    /**
     * Fetches IdP metadata XML from a GGID instance.
     * @param ggidBaseURL The base URL of the GGID server (e.g. "https://ggid.example.com")
     * @return IdP metadata XML bytes
     */
    public static byte[] fetchIdPMetadata(String ggidBaseURL) throws IOException {
        URL url = new URL(ggidBaseURL + "/saml/metadata");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setConnectTimeout(5000);
        conn.setReadTimeout(10000);

        if (conn.getResponseCode() != 200) {
            throw new IOException("IdP metadata request failed: " + conn.getResponseCode());
        }

        try (InputStream is = conn.getInputStream()) {
            return is.readAllBytes();
        }
    }
}
