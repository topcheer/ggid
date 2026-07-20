# GGID SAML SDK for Ruby.
# Generates SP metadata for IdP integration.

require 'net/http'
require 'uri'

module GGID
  class SAML
    attr_reader :entity_id, :acs_url, :slo_url

    # @param config [Hash] :entity_id, :acs_url, :slo_url (optional)
    def initialize(config)
      raise ArgumentError, 'entity_id is required' if config[:entity_id].nil? || config[:entity_id].empty?
      raise ArgumentError, 'acs_url is required' if config[:acs_url].nil? || config[:acs_url].empty?

      @entity_id = config[:entity_id]
      @acs_url = config[:acs_url]
      @slo_url = config[:slo_url]
    end

    # Generate SP metadata XML string.
    # @return [String] XML
    def generate_sp_metadata
      valid_until = (Time.now + 365 * 24 * 3600).utc.iso8601
      slo_element = ''
      if @slo_url
        slo_element = %(<SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="#{@slo_url}"/>)
      end

      %(<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="#{@entity_id}" validUntil="#{valid_until}">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="#{@acs_url}" index="0"/>
    #{slo_element}
  </SPSSODescriptor>
</EntityDescriptor>)
    end

    # Fetch IdP metadata from GGID instance.
    # @param ggid_base_url [String] e.g. https://ggid.example.com
    # @return [String] XML
    def self.fetch_idp_metadata(ggid_base_url)
      url = URI("#{ggid_base_url.chomp('/')}/saml/idp/metadata")
      res = Net::HTTP.get(url)
      res
    rescue StandardError => e
      raise "Failed to fetch IdP metadata: #{e.message}"
    end
  end
end
