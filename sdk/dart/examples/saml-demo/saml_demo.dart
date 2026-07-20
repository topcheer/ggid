// GGID SAML SSO Demo (Dart)
// Run: GGID_URL=... dart run saml_demo.dart
import 'dart:io';
import '../../lib/src/saml.dart';

void main() async {
  final ggidUrl = Platform.environment['GGID_URL'] ?? 'http://localhost:8080';
  final entityId = Platform.environment['SP_ENTITY_ID'] ?? 'http://localhost:3001/saml/metadata';
  final acsUrl = Platform.environment['ACS_URL'] ?? 'http://localhost:3001/saml/acs';

  final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 3001);
  print('SAML demo on http://localhost:3001');

  await for (final request in server) {
    final path = request.uri.path;
    final response = request.response;

    if (path == '/') {
      response.headers.contentType = ContentType.html;
      response.write('<h1>GGID SAML Demo</h1><a href="/login">Login with SAML SSO</a>');
    } else if (path == '/saml/metadata') {
      response.headers.contentType = ContentType.parse('application/xml');
      response.write(generateSPMetadata(SAMLConfig(entityId: entityId, acsUrl: acsUrl)));
    } else if (path == '/login') {
      final ssoUrl = '$ggidUrl/saml/sso';
      final url = buildAuthnRequestUrl(
        ssoUrl: ssoUrl, entityId: entityId, acsUrl: acsUrl, relayState: '/profile',
      );
      response.statusCode = HttpStatus.movedPermanently;
      response.headers.set(HttpHeaders.locationHeader, url);
    } else if (path == '/saml/acs') {
      response.headers.contentType = ContentType.html;
      response.write('<h1>SAML ACS</h1><p>Received SAML response</p><a href="/profile">Continue</a>');
    } else if (path == '/profile') {
      response.headers.contentType = ContentType.html;
      response.write('<h1>Profile</h1><p>Authenticated via SAML SSO</p>');
    }
    await response.close();
  }
}
