/**
 * SAML Service Provider utilities for GGID SDK (Node.js)
 *
 * Generate SP metadata, fetch IdP metadata, and build SAML auth request URLs.
 */

export interface SAMLConfig {
  /** SP Entity ID (e.g. "https://myapp.example.com/saml") */
  entityId: string;
  /** Assertion Consumer Service URL (e.g. "https://myapp.example.com/saml/acs") */
  acsUrl: string;
  /** Single Logout URL (optional) */
  sloUrl?: string;
  /** Whether to sign SAML requests (default: false) */
  signRequests?: boolean;
}

/**
 * Generate SAML SP metadata XML from configuration.
 *
 * @example
 * ```ts
 * const xml = generateSPMetadata({
 *   entityId: "https://myapp.example.com/saml",
 *   acsUrl: "https://myapp.example.com/saml/acs",
 * });
 * fs.writeFileSync("sp-metadata.xml", xml);
 * ```
 */
export function generateSPMetadata(config: SAMLConfig): string {
  const slo = config.sloUrl
    ? `  <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="${escapeXml(config.sloUrl)}" />\n`
    : "";

  return `<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="${escapeXml(config.entityId)}">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      Location="${escapeXml(config.acsUrl)}" index="0" />
${slo}  </SPSSODescriptor>
</EntityDescriptor>`;
}

/**
 * Fetch IdP metadata XML from a GGID instance.
 *
 * @param ggidBaseUrl - Base URL of GGID (e.g. "https://ggid.example.com")
 */
export async function fetchIdPMetadata(ggidBaseUrl: string): Promise<string> {
  const res = await fetch(`${ggidBaseUrl.replace(/\/$/, "")}/saml/metadata`, {
    headers: { Accept: "application/xml" },
  });
  if (!res.ok) {
    throw new Error(`Failed to fetch IdP metadata: ${res.status} ${res.statusText}`);
  }
  return res.text();
}

/**
 * Extract entity ID from IdP metadata XML.
 */
export function parseEntityId(metadataXml: string): string | null {
  const match = metadataXml.match(/entityID="([^"]+)"/);
  return match ? match[1] : null;
}

/**
 * Extract SSO URL (SingleSignOnService) from IdP metadata XML.
 */
export function parseSsoUrl(metadataXml: string): string | null {
  const match = metadataXml.match(/SingleSignOnService[^>]*Location="([^"]+)"/);
  return match ? match[1] : null;
}

/**
 * Build a SAML AuthnRequest redirect URL (SP-initiated SSO).
 *
 * @param ssoUrl - IdP SSO URL from metadata
 * @param entityId - SP Entity ID
 * @param acsUrl - SP ACS URL
 * @param relayState - Optional relay state (return URL after auth)
 */
export function buildAuthnRequestUrl(
  ssoUrl: string,
  entityId: string,
  acsUrl: string,
  relayState?: string,
): string {
  const request = `<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="${generateId()}" Version="2.0" IssueInstant="${new Date().toISOString()}" Destination="${escapeXml(ssoUrl)}" AssertionConsumerServiceURL="${escapeXml(acsUrl)}"><saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">${escapeXml(entityId)}</saml:Issuer></samlp:AuthnRequest>`;

  const encoded = Buffer.from(request).toString("base64");
  const params = new URLSearchParams({ SAMLRequest: encoded });
  if (relayState) params.set("RelayState", relayState);

  const separator = ssoUrl.includes("?") ? "&" : "?";
  return `${ssoUrl}${separator}${params.toString()}`;
}

function escapeXml(str: string): string {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;");
}

function generateId(): string {
  return `_${Math.random().toString(36).slice(2, 12)}${Date.now().toString(36)}`;
}
