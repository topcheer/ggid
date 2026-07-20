/// SAML Service Provider utilities for GGID SDK (Dart)
///
/// Generate SP metadata, fetch IdP metadata, and build SAML auth request URLs.

/// Configuration for a SAML Service Provider.
class SAMLConfig {
  /// SP Entity ID (e.g. "https://myapp.example.com/saml")
  final String entityId;

  /// Assertion Consumer Service URL
  final String acsUrl;

  /// Single Logout URL (optional)
  final String? sloUrl;

  /// Whether to sign SAML requests (default: false)
  final bool signRequests;

  const SAMLConfig({
    required this.entityId,
    required this.acsUrl,
    this.sloUrl,
    this.signRequests = false,
  });
}

/// Generate SAML SP metadata XML from configuration.
///
/// Example:
/// ```dart
/// final xml = generateSPMetadata(SAMLConfig(
///   entityId: 'https://myapp.example.com/saml',
///   acsUrl: 'https://myapp.example.com/saml/acs',
/// ));
/// ```
String generateSPMetadata(SAMLConfig config) {
  final slo = config.sloUrl != null
      ? '  <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="${_escapeXml(config.sloUrl!)}" />\n'
      : '';

  return '''<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="${_escapeXml(config.entityId)}">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      Location="${_escapeXml(config.acsUrl)}" index="0" />
$slo  </SPSSODescriptor>
</EntityDescriptor>''';
}

/// Extract entity ID from IdP metadata XML.
String? parseEntityId(String metadataXml) {
  final match = RegExp(r'entityID="([^"]+)"').firstMatch(metadataXml);
  return match?.group(1);
}

/// Extract SSO URL from IdP metadata XML.
String? parseSsoUrl(String metadataXml) {
  final match = RegExp(r'SingleSignOnService[^>]*Location="([^"]+)"').firstMatch(metadataXml);
  return match?.group(1);
}

/// Build a SAML AuthnRequest redirect URL (SP-initiated SSO).
String buildAuthnRequestUrl({
  required String ssoUrl,
  required String entityId,
  required String acsUrl,
  String? relayState,
}) {
  final id = '_${DateTime.now().millisecondsSinceEpoch.toRadixString(36)}${DateTime.now().microsecond}';
  final request = '''<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="$id" Version="2.0" IssueInstant="${DateTime.now().toUtc().toIso8601String()}" Destination="${_escapeXml(ssoUrl)}" AssertionConsumerServiceURL="${_escapeXml(acsUrl)}"><saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">${_escapeXml(entityId)}</saml:Issuer></samlp:AuthnRequest>''';

  final encoded = base64Encode(request);
  final params = 'SAMLRequest=${Uri.encodeComponent(encoded)}';
  final relay = relayState != null ? '&RelayState=${Uri.encodeComponent(relayState)}' : '';
  final sep = ssoUrl.contains('?') ? '&' : '?';

  return '$ssoUrl$sep$params$relay';
}

String _escapeXml(String str) {
  return str
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&apos;');
}

// Simple base64 encode (no dart:convert dependency needed for standalone SDK)
String base64Encode(String input) {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
  final bytes = input.codeUnits;
  final result = StringBuffer();
  for (var i = 0; i < bytes.length; i += 3) {
    final b1 = bytes[i];
    final b2 = i + 1 < bytes.length ? bytes[i + 1] : 0;
    final b3 = i + 2 < bytes.length ? bytes[i + 2] : 0;
    result.write(chars[(b1 >> 2) & 0x3F]);
    result.write(chars[((b1 << 4) | (b2 >> 4)) & 0x3F]);
    result.write(i + 1 < bytes.length ? chars[((b2 << 2) | (b3 >> 6)) & 0x3F] : '=');
    result.write(i + 2 < bytes.length ? chars[b3 & 0x3F] : '=');
  }
  return result.toString();
}
