package dev.ggid.sdk;

import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.List;

/**
 * SAML Service Provider configuration and metadata generation.
 * <p>
 * Usage:
 * <pre>{@code
 * SAMLConfig config = new SAMLConfig(
 *     "https://myapp.example.com/saml",
 *     "https://myapp.example.com/saml/acs",
 *     "https://myapp.example.com/saml/slo"
 * );
 * String metadata = SAMLConfig.generateSPMetadata(config);
 * }</pre>
 */
public class SAMLConfig {
    private final String entityId;
    private final String acsUrl;
    private final String sloUrl;
    private final boolean signRequests;

    public SAMLConfig(String entityId, String acsUrl, String sloUrl) {
        this.entityId = entityId;
        this.acsUrl = acsUrl;
        this.sloUrl = sloUrl;
        this.signRequests = false;
    }

    public SAMLConfig(String entityId, String acsUrl, String sloUrl, boolean signRequests) {
        this.entityId = entityId;
        this.acsUrl = acsUrl;
        this.sloUrl = sloUrl;
        this.signRequests = signRequests;
    }

    public String getEntityId() { return entityId; }
    public String getAcsUrl() { return acsUrl; }
    public String getSloUrl() { return sloUrl; }
    public boolean isSignRequests() { return signRequests; }

    /**
     * Generates SAML SP metadata XML for IdP configuration.
     *
     * @param config the SP configuration
     * @return SP EntityDescriptor XML string
     * @throws IllegalArgumentException if entityId or acsUrl is empty
     */
    public static String generateSPMetadata(SAMLConfig config) {
        if (config == null) throw new IllegalArgumentException("SAML config is null");
        if (config.entityId == null || config.entityId.isEmpty())
            throw new IllegalArgumentException("entity ID is required");
        if (config.acsUrl == null || config.acsUrl.isEmpty())
            throw new IllegalArgumentException("ACS URL is required");

        String validUntil = Instant.now().plus(365, ChronoUnit.DAYS).toString();
        StringBuilder sb = new StringBuilder();
        sb.append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n");
        sb.append("<EntityDescriptor xmlns=\"urn:oasis:names:tc:SAML:2.0:metadata\"");
        sb.append(" entityID=\"").append(escapeXml(config.entityId)).append("\"");
        sb.append(" validUntil=\"").append(validUntil).append("\">");
        sb.append("<SPSSODescriptor protocolSupportEnumeration=\"urn:oasis:names:tc:SAML:2.0:protocol\">");
        sb.append("<NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>");
        sb.append("<AssertionConsumerService Binding=\"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST\"");
        sb.append(" Location=\"").append(escapeXml(config.acsUrl)).append("\" index=\"0\"/>");
        if (config.sloUrl != null && !config.sloUrl.isEmpty()) {
            sb.append("<SingleLogoutService Binding=\"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect\"");
            sb.append(" Location=\"").append(escapeXml(config.sloUrl)).append("\"/>");
        }
        sb.append("</SPSSODescriptor></EntityDescriptor>");
        return sb.toString();
    }

    private static String escapeXml(String s) {
        return s.replace("&", "&amp;").replace("<", "&lt;")
                .replace(">", "&gt;").replace("\"", "&quot;").replace("'", "&apos;");
    }
}
