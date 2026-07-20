// SAML Service Provider configuration for GGID Rust SDK.
// Provides SP metadata generation and IdP metadata fetching.

use std::time::{SystemTime, UNIX_EPOCH};

/// Configuration for a SAML Service Provider.
#[derive(Debug, Clone)]
pub struct SAMLConfig {
    /// SP Entity ID (e.g. "https://myapp.example.com/saml")
    pub entity_id: String,
    /// Assertion Consumer Service URL
    pub acs_url: String,
    /// Single Logout URL (optional)
    pub slo_url: Option<String>,
    /// Whether to sign SAML authn requests
    pub sign_requests: bool,
}

impl SAMLConfig {
    /// Creates a new SAMLConfig.
    pub fn new(entity_id: &str, acs_url: &str) -> Self {
        Self {
            entity_id: entity_id.to_string(),
            acs_url: acs_url.to_string(),
            slo_url: None,
            sign_requests: false,
        }
    }

    /// Sets the SLO URL.
    pub fn with_slo(mut self, slo_url: &str) -> Self {
        self.slo_url = Some(slo_url.to_string());
        self
    }

    /// Generates SAML SP metadata XML (EntityDescriptor).
    pub fn generate_sp_metadata(&self) -> String {
        let mut xml = String::with_capacity(1024);
        xml.push_str("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n");
        xml.push_str(&format!(
            "<EntityDescriptor xmlns=\"urn:oasis:names:tc:SAML:2.0:metadata\" entityID=\"{}\">\n",
            escape_xml(&self.entity_id)
        ));
        xml.push_str("  <SPSSODescriptor protocolSupportEnumeration=\"urn:oasis:names:tc:SAML:2.0:protocol\">\n");
        xml.push_str("    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>\n");
        xml.push_str(&format!(
            "    <AssertionConsumerService index=\"0\" isDefault=\"true\" Binding=\"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST\" Location=\"{}\"/>\n",
            escape_xml(&self.acs_url)
        ));
        if let Some(ref slo) = self.slo_url {
            xml.push_str(&format!(
                "    <SingleLogoutService Binding=\"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect\" Location=\"{}\"/>\n",
                escape_xml(slo)
            ));
        }
        xml.push_str("  </SPSSODescriptor>\n");
        xml.push_str("</EntityDescriptor>");
        xml
    }
}

fn escape_xml(s: &str) -> String {
    s.replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
        .replace('\'', "&apos;")
}
