<?php
/**
 * GGID SDK - SAML Service Provider utilities (PHP)
 */

namespace GGID;

class SAML
{
    /**
     * Generate SAML SP metadata XML.
     *
     * @param string $entityId SP Entity ID
     * @param string $acsUrl Assertion Consumer Service URL
     * @param string|null $sloUrl Single Logout URL (optional)
     * @return string XML metadata
     */
    public static function generateSPMetadata(
        string $entityId,
        string $acsUrl,
        ?string $sloUrl = null
    ): string {
        $slo = $sloUrl
            ? '  <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="' . self::escapeXml($sloUrl) . "\" />\n"
            : '';

        return '<?xml version="1.0" encoding="UTF-8"?>' . "\n"
            . '<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="' . self::escapeXml($entityId) . "\">\n"
            . '  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">' . "\n"
            . '    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>' . "\n"
            . '    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"' . "\n"
            . '      Location="' . self::escapeXml($acsUrl) . '" index="0" />' . "\n"
            . $slo
            . '  </SPSSODescriptor>' . "\n"
            . '</EntityDescriptor>';
    }

    /**
     * Fetch IdP metadata XML from GGID.
     */
    public static function fetchIdPMetadata(string $ggidBaseUrl): string
    {
        $url = rtrim($ggidBaseUrl, '/') . '/saml/metadata';
        $xml = @file_get_contents($url);
        if ($xml === false) {
            throw new \RuntimeException("Failed to fetch IdP metadata from: $url");
        }
        return $xml;
    }

    /**
     * Extract entity ID from IdP metadata XML.
     */
    public static function parseEntityId(string $metadataXml): ?string
    {
        if (preg_match('/entityID="([^"]+)"/', $metadataXml, $m)) {
            return $m[1];
        }
        return null;
    }

    /**
     * Extract SSO URL from IdP metadata XML.
     */
    public static function parseSsoUrl(string $metadataXml): ?string
    {
        if (preg_match('/SingleSignOnService[^>]*Location="([^"]+)"/', $metadataXml, $m)) {
            return $m[1];
        }
        return null;
    }

    /**
     * Build a SAML AuthnRequest redirect URL.
     */
    public static function buildAuthnRequestUrl(
        string $ssoUrl,
        string $entityId,
        string $acsUrl,
        ?string $relayState = null
    ): string {
        $id = '_' . bin2hex(random_bytes(8));
        $instant = gmdate('Y-m-d\TH:i:s\Z');
        $request = '<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"'
            . ' ID="' . $id . '" Version="2.0" IssueInstant="' . $instant . '"'
            . ' Destination="' . self::escapeXml($ssoUrl) . '"'
            . ' AssertionConsumerServiceURL="' . self::escapeXml($acsUrl) . '">'
            . '<saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">'
            . self::escapeXml($entityId)
            . '</saml:Issuer></samlp:AuthnRequest>';

        $encoded = base64_encode($request);
        $sep = str_contains($ssoUrl, '?') ? '&' : '?';
        $url = $ssoUrl . $sep . 'SAMLRequest=' . urlencode($encoded);
        if ($relayState) $url .= '&RelayState=' . urlencode($relayState);
        return $url;
    }

    private static function escapeXml(string $str): string
    {
        return htmlspecialchars($str, ENT_XML1 | ENT_QUOTES, 'UTF-8');
    }
}
