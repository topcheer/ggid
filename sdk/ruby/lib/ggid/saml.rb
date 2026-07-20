# GGID SDK - SAML SP utilities (Ruby)
module GGID
  module SAML
    module_function

    # Generate SAML SP metadata XML
    def generate_sp_metadata(entity_id:, acs_url:, slo_url: nil)
      slo = slo_url ?
        %(  <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="#{escape_xml(slo_url)}" />\n) : ''
      %(<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="#{escape_xml(entity_id)}">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      Location="#{escape_xml(acs_url)}" index="0" />
#{slo}  </SPSSODescriptor>
</EntityDescriptor>)
    end

    # Fetch IdP metadata XML from GGID
    def fetch_idp_metadata(ggid_base_url)
      require 'net/http'
      require 'uri'
      uri = URI("#{ggid_base_url.gsub(/\/$/, '')}/saml/metadata")
      response = Net::HTTP.get_response(uri)
      raise "Failed to fetch IdP metadata: #{response.code}" unless response.is_a?(Net::HTTPSuccess)
      response.body
    end

    # Extract entity ID from IdP metadata XML
    def parse_entity_id(metadata_xml)
      match = metadata_xml.match(/entityID="([^"]+)"/)
      match && match[1]
    end

    # Extract SSO URL from IdP metadata XML
    def parse_sso_url(metadata_xml)
      match = metadata_xml.match(/SingleSignOnService[^>]*Location="([^"]+)"/)
      match && match[1]
    end

    # Build a SAML AuthnRequest redirect URL
    def build_authn_request_url(sso_url:, entity_id:, acs_url:, relay_state: nil)
      require 'base64'
      require 'cgi'
      id = "_#{Time.now.to_i}#{rand(36**8).to_s(36)}"
      request = %(<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="#{id}" Version="2.0" IssueInstant="#{Time.now.utc.iso8601}" Destination="#{escape_xml(sso_url)}" AssertionConsumerServiceURL="#{escape_xml(acs_url)}"><saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">#{escape_xml(entity_id)}</saml:Issuer></samlp:AuthnRequest>)
      encoded = Base64.strict_encode64(request)
      sep = sso_url.include?('?') ? '&' : '?'
      url = "#{sso_url}#{sep}SAMLRequest=#{CGI.escape(encoded)}"
      url += "&RelayState=#{CGI.escape(relay_state)}" if relay_state
      url
    end

    def escape_xml(str)
      str.gsub('&', '&amp;').gsub('<', '&lt;').gsub('>', '&gt;').gsub('"', '&quot;').gsub("'", '&apos;')
    end
  end
end
