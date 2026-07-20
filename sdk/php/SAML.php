<?php
/**
 * GGID SAML SDK for PHP.
 * Generates SP metadata for IdP integration.
 */

class GGIDSAML {
    private string $entityId;
    private string $acsUrl;
    private string $sloUrl;

    /**
     * @param array $config ['entity_id' => '...', 'acs_url' => '...', 'slo_url' => '...']
     */
    public function __construct(array $config) {
        if (empty($config['entity_id'])) {
            throw new InvalidArgumentException('entity_id is required');
        }
        if (empty($config['acs_url'])) {
            throw new InvalidArgumentException('acs_url is required');
        }
        $this->entityId = $config['entity_id'];
        $this->acsUrl = $config['acs_url'];
        $this->sloUrl = $config['slo_url'] ?? '';
    }

    /**
     * Generate SP metadata XML.
     * @return string XML string
     */
    public function generateSPMetadata(): string {
        $validUntil = date('c', strtotime('+1 year'));
        $slo = '';
        if ($this->sloUrl) {
            $slo = "<SingleLogoutService Binding=\"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect\" Location=\"{$this->sloUrl}\"/>";
        }
        return <<<XML
<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="{$this->entityId}" validUntil="{$validUntil}">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="{$this->acsUrl}" index="0"/>
    {$slo}
  </SPSSODescriptor>
</EntityDescriptor>
XML;
    }

    /**
     * Fetch IdP metadata from GGID instance.
     * @param string $ggidBaseUrl e.g. https://ggid.example.com
     * @return string XML string
     */
    public static function fetchIdPMetadata(string $ggidBaseUrl): string {
        $url = rtrim($ggidBaseUrl, '/') . '/saml/idp/metadata';
        $xml = file_get_contents($url);
        if ($xml === false) {
            throw new RuntimeException("Failed to fetch IdP metadata from $url");
        }
        return $xml;
    }
}
