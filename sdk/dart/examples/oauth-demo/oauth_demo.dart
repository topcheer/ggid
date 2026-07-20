// GGID OAuth 2.0 Demo (Dart)
// Run: GGID_URL=... dart run oauth_demo.dart
import 'dart:io';
import 'dart:convert';

void main() async {
  final ggidUrl = Platform.environment['GGID_URL'] ?? 'http://localhost:8080';
  final clientId = Platform.environment['CLIENT_ID'] ?? '';
  final clientSecret = Platform.environment['CLIENT_SECRET'] ?? '';
  final redirectUri = Platform.environment['REDIRECT_URI'] ?? 'http://localhost:3000/callback';

  final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 3000);
  print('OAuth demo on http://localhost:3000');

  await for (final request in server) {
    final path = request.uri.path;
    final response = request.response;

    if (path == '/') {
      final authUrl = '$ggidUrl/api/v1/oauth/authorize?response_type=code'
          '&client_id=$clientId&redirect_uri=${Uri.encodeComponent(redirectUri)}'
          '&scope=openid+profile+email&state=demo';
      response.headers.contentType = ContentType.html;
      response.write('<h1>GGID OAuth Demo</h1><a href="$authUrl">Login with GGID</a>');
    } else if (path == '/callback') {
      final code = request.uri.queryParameters['code'] ?? '';
      final httpClient = HttpClient();
      final tokenReq = await httpClient.postUrl(Uri.parse('$ggidUrl/api/v1/oauth/token'));
      tokenReq.headers.contentType = ContentType.parse('application/x-www-form-urlencoded');
      tokenReq.write('grant_type=authorization_code&code=$code'
          '&redirect_uri=${Uri.encodeComponent(redirectUri)}'
          '&client_id=$clientId&client_secret=$clientSecret');
      final tokenRes = await tokenReq.close();
      final tokens = await tokenRes.transform(utf8.decoder).join();
      response.headers.contentType = ContentType.json;
      response.write(tokens);
      httpClient.close();
    }
    await response.close();
  }
}
