/**
 * SAML SP utilities for GGID React SDK
 *
 * Generate SP metadata, fetch IdP metadata, and build SAML auth request URLs.
 */

export interface SAMLConfig {
  entityId: string;
  acsUrl: string;
  sloUrl?: string;
  signRequests?: boolean;
}

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

export async function fetchIdPMetadata(ggidBaseUrl: string): Promise<string> {
  const res = await fetch(`${ggidBaseUrl.replace(/\/$/, "")}/saml/metadata`);
  if (!res.ok) throw new Error(`Failed to fetch IdP metadata: ${res.status}`);
  return res.text();
}

export function parseEntityId(metadataXml: string): string | null {
  const match = metadataXml.match(/entityID="([^"]+)"/);
  return match ? match[1] : null;
}

export function parseSsoUrl(metadataXml: string): string | null {
  const match = metadataXml.match(/SingleSignOnService[^>]*Location="([^"]+)"/);
  return match ? match[1] : null;
}

export function buildAuthnRequestUrl(
  ssoUrl: string, entityId: string, acsUrl: string, relayState?: string,
): string {
  const id = `_${Math.random().toString(36).slice(2, 12)}${Date.now().toString(36)}`;
  const request = `<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="${id}" Version="2.0" IssueInstant="${new Date().toISOString()}" Destination="${escapeXml(ssoUrl)}" AssertionConsumerServiceURL="${escapeXml(acsUrl)}"><saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">${escapeXml(entityId)}</saml:Issuer></samlp:AuthnRequest>`;
  const encoded = typeof Buffer !== "undefined"
    ? Buffer.from(request).toString("base64")
    : btoa(request);
  const params = new URLSearchParams({ SAMLRequest: encoded });
  if (relayState) params.set("RelayState", relayState);
  const sep = ssoUrl.includes("?") ? "&" : "?";
  return `${ssoUrl}${sep}${params.toString()}`;
}

function escapeXml(str: string): string {
  return str.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&apos;");
}
