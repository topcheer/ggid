/// Shelf middleware for the GGID Dart SDK.
///
/// Provides authentication, permission, and role-based middleware for
/// Shelf-based Dart server applications.
///
/// ```dart
/// import 'package:shelf/shelf.dart';
/// import 'package:shelf/shelf_io.dart' as io;
/// import 'package:ggid/ggid_client.dart';
/// import 'package:ggid/src/middleware.dart';
///
/// final ggid = GGIDClient(baseUrl: '...', tenantId: '...');
/// final handler = const Pipeline()
///   .addMiddleware(logRequests())
///   .addMiddleware(ggid.authMiddleware())
///   .addMiddleware(ggid.requirePermission('products', 'read'))
///   .addHandler(myHandler);
/// ```
library ggid.middleware;

import 'dart:convert';
import 'package:shelf/shelf.dart';

import 'package:ggid/ggid_client.dart';
import 'models.dart';

/// Set of public paths that bypass authentication.
const _publicPaths = <String>{
  '/', '/healthz', '/docs', '/api-docs', '/login', '/register',
};

/// Extension providing Shelf middleware on [GGIDClient].
extension GgidMiddleware on GGIDClient {
  /// Authentication middleware — verifies JWT and injects claims into request context.
  ///
  /// Public paths (login, register, healthz) bypass authentication.
  ///
  /// ```dart
  /// final pipeline = const Pipeline()
  ///   .addMiddleware(ggid.authMiddleware())
  ///   .addHandler(myHandler);
  /// ```
  Middleware authMiddleware() {
    return (Handler innerHandler) {
      return (Request request) async {
        final path = request.url.path;

        // Skip public paths and auth endpoints
        if (_publicPaths.contains('/$path') ||
            path.startsWith('api/v1/auth/') ||
            path.startsWith('oauth/')) {
          return innerHandler(request);
        }

        // Extract Bearer token
        final authHeader = request.headers['authorization'] ?? '';
        if (!authHeader.startsWith('Bearer ')) {
          return _jsonResponse(401, {'error': 'missing bearer token'});
        }

        final token = authHeader.substring(7);

        try {
          final claims = await verifyToken(token);
          // Inject claims into request context
          final newRequest = request.change(context: {
            'ggid.claims': claims,
            'ggid.token': token,
          });
          return innerHandler(newRequest);
        } on TokenExpiredException {
          return _jsonResponse(401, {'error': 'token expired'});
        } on InvalidTokenException catch (e) {
          return _jsonResponse(401, {'error': 'invalid token', 'detail': e.message});
        }
      };
    };
  }

  /// Permission middleware — checks if the user has permission for resource:action.
  ///
  /// Must be used after [authMiddleware].
  ///
  /// ```dart
  /// final pipeline = const Pipeline()
  ///   .addMiddleware(ggid.authMiddleware())
  ///   .addMiddleware(ggid.requirePermission('products', 'read'))
  ///   .addHandler(myHandler);
  /// ```
  Middleware requirePermission(String resource, String action) {
    return (Handler innerHandler) {
      return (Request request) async {
        final claims = request.context['ggid.claims'] as Claims?;
        if (claims == null) {
          return _jsonResponse(401, {'error': 'not authenticated'});
        }

        final token = request.context['ggid.token'] as String? ?? '';
        try {
          final allowed = await checkPermission(token, resource, action);
          if (!allowed) {
            return _jsonResponse(403, {'error': 'forbidden'});
          }
          return innerHandler(request);
        } catch (e) {
          return _jsonResponse(403, {'error': 'permission check failed', 'detail': e.toString()});
        }
      };
    };
  }

  /// Role middleware — checks if the user has the specified role.
  ///
  /// Must be used after [authMiddleware].
  ///
  /// ```dart
  /// final pipeline = const Pipeline()
  ///   .addMiddleware(ggid.authMiddleware())
  ///   .addMiddleware(ggid.requireRole('admin'))
  ///   .addHandler(myHandler);
  /// ```
  Middleware requireRole(String role) {
    return (Handler innerHandler) {
      return (Request request) async {
        final claims = request.context['ggid.claims'] as Claims?;
        if (claims == null) {
          return _jsonResponse(401, {'error': 'not authenticated'});
        }

        if (!claims.roles.contains(role)) {
          return _jsonResponse(403, {
            'error': 'forbidden',
            'detail': 'requires role: $role',
          });
        }

        return innerHandler(request);
      };
    };
  }
}

/// Helper to create a JSON response.
Response _jsonResponse(int status, Map<String, dynamic> body) {
  return Response(
    status,
    body: jsonEncode(body),
    headers: {'Content-Type': 'application/json'},
  );
}

/// Extract GGID claims from a Shelf request context.
///
/// ```dart
/// final claims = getClaims(request);
/// print(claims?.userId);
/// ```
Claims? getClaims(Request request) {
  return request.context['ggid.claims'] as Claims?;
}

/// Extract the Bearer token from a Shelf request context.
String? getToken(Request request) {
  return request.context['ggid.token'] as String?;
}
