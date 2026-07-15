// ── Dart Shelf Quickstart ──
//
// A minimal Shelf server using GGID SDK for authentication and authorization.

import 'dart:convert';
import 'package:shelf/shelf.dart';
import 'package:shelf/shelf_io.dart' as io;
import 'package:ggid/ggid_client.dart';
import 'package:ggid/src/middleware.dart';

void main() async {
  // Initialize GGID client
  final ggid = GGIDClient(
    baseUrl: 'https://ggid.iot2.win',
    tenantId: '00000000-0000-0000-0000-000000000001',
  ).withJwks();

  // ── Public endpoint (no auth) ──
  final publicHandler = (Request request) {
    return Response.ok(jsonEncode({'status': 'ok'}),
        headers: {'Content-Type': 'application/json'});
  };

  // ── Protected: requires valid JWT ──
  final meHandler = (Request request) {
    final claims = getClaims(request)!;
    return Response.ok(jsonEncode({
      'user_id': claims.userId,
      'email': claims.email,
      'roles': claims.roles,
    }), headers: {'Content-Type': 'application/json'});
  };

  // ── Protected: requires permission products:read ──
  final productsHandler = (Request request) {
    final claims = getClaims(request)!;
    return Response.ok(jsonEncode({
      'user': claims.userId,
      'products': [
        {'id': 1, 'name': 'Widget', 'price': 9.99},
        {'id': 2, 'name': 'Gadget', 'price': 19.99},
        {'id': 3, 'name': 'Doohickey', 'price': 4.99},
      ],
    }), headers: {'Content-Type': 'application/json'});
  };

  // ── Protected: requires admin role ──
  final adminHandler = (Request request) {
    final claims = getClaims(request)!;
    return Response.ok(jsonEncode({
      'message': 'Admin access granted',
      'user': claims.userId,
    }), headers: {'Content-Type': 'application/json'});
  };

  // ── OAuth login flow ──
  final oauthLoginHandler = (Request request) {
    final redirectUri = request.url.queryParameters['redirect_uri'] ?? '';
    final authorizeUrl = ggid.getAuthorizeUrl(
      clientId: 'gcid_your_client_id',
      redirectUri: redirectUri,
      scope: 'openid profile email',
      state: DateTime.now().millisecondsSinceEpoch.toString(),
    );
    return Response.found(authorizeUrl);
  };

  // ── OAuth callback ──
  final oauthCallbackHandler = (Request request) async {
    final code = request.url.queryParameters['code'] ?? '';
    final redirectUri = request.url.queryParameters['redirect_uri'] ?? '';
    if (code.isEmpty) {
      return Response.badRequest(body: 'missing code parameter');
    }

    final tokens = await ggid.exchangeCode(
      code: code,
      redirectUri: redirectUri,
      clientId: 'gcid_your_client_id',
      clientSecret: 'your_client_secret',
    );

    return Response.ok(jsonEncode({
      'access_token': tokens.accessToken,
      'expires_in': tokens.expiresIn,
    }), headers: {'Content-Type': 'application/json'});
  };

  // ── Router ──
  Handler router() {
    return (Request request) async {
      final path = request.url.path;

      // Public routes
      if (path == 'healthz') return publicHandler(request);
      if (path == 'auth/login') return oauthLoginHandler(request);
      if (path == 'auth/callback') return oauthCallbackHandler(request);

      // Apply auth middleware to protected routes
      final authPipeline = const Pipeline()
          .addMiddleware(ggid.authMiddleware());

      // Route with auth + permission middleware
      if (path == 'api/me') {
        return authPipeline.addHandler(meHandler)(request);
      }
      if (path == 'api/products') {
        final fullPipeline = authPipeline
            .addMiddleware(ggid.requirePermission('products', 'read'));
        return fullPipeline.addHandler(productsHandler)(request);
      }
      if (path == 'api/admin') {
        final fullPipeline = authPipeline
            .addMiddleware(ggid.requireRole('admin'));
        return fullPipeline.addHandler(adminHandler)(request);
      }

      return Response.notFound('Not Found');
    };
  }

  // ── Start server ──
  final server = await io.serve(router(), 'localhost', 8080);
  print('Server running at http://${server.address.host}:${server.port}');
}
