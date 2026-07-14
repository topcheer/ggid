/// Unit tests for the GGID Dart SDK.
///
/// Tests cover client initialization, authentication, RBAC, ABAC,
/// OAuth/OIDC, and middleware.
library ggid.test;

import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:test/test.dart';

import 'package:ggid/ggid_client.dart';
import 'package:ggid/src/models.dart';

void main() {
  group('GGIDClient', () {
    test('constructor sets baseUrl and tenantId', () {
      final client = GGIDClient(
        baseUrl: 'https://ggid.example.com',
        tenantId: 'tenant-123',
      );
      expect(client.baseUrl, 'https://ggid.example.com');
      expect(client.tenantId, 'tenant-123');
    });

    test('login returns TokenResponse', () async {
      final mock = MockClient((request) async {
        expect(request.url.path, startsWith('/api/v1/auth/login'));
        expect(request.headers['X-Tenant-ID'], 'tenant-1');
        final body = jsonDecode(request.body) as Map<String, dynamic>;
        expect(body['username'], 'admin');

        return http.Response(
          jsonEncode({
            'access_token': 'jwt-abc',
            'refresh_token': 'r-xyz',
            'token_type': 'Bearer',
            'expires_in': 3600,
          }),
          200,
          headers: {'Content-Type': 'application/json'},
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final tokens = await ggid.login('admin', 'pass');
      expect(tokens.accessToken, 'jwt-abc');
      expect(tokens.refreshToken, 'r-xyz');
      expect(tokens.expiresIn, 3600);
    });

    test('getUserInfo returns UserInfo', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode({
            'sub': 'user-1',
            'name': 'Alice',
            'email': 'alice@test.com',
            'email_verified': true,
          }),
          200,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final info = await ggid.getUserInfo('token');
      expect(info.sub, 'user-1');
      expect(info.name, 'Alice');
      expect(info.email, 'alice@test.com');
      expect(info.emailVerified, true);
    });

    test('getDiscovery returns DiscoveryConfig', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode({
            'issuer': 'https://ggid.example.com',
            'authorization_endpoint': 'https://ggid.example.com/oauth/authorize',
            'token_endpoint': 'https://ggid.example.com/api/v1/oauth/token',
            'jwks_uri': 'https://ggid.example.com/oauth/jwks',
          }),
          200,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final discovery = await ggid.getDiscovery();
      expect(discovery.issuer, 'https://ggid.example.com');
      expect(discovery.authorizationEndpoint, contains('authorize'));
    });

    test('getJwks returns keys', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode({
            'keys': [
              {'kty': 'RSA', 'kid': 'key-1', 'use': 'sig', 'alg': 'RS256', 'n': 'abc', 'e': 'AQAB'}
            ]
          }),
          200,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final jwks = await ggid.getJwks();
      expect(jwks['keys'], isA<List>());
      expect((jwks['keys'] as List).length, 1);
    });

    test('checkPermission returns true when allowed', () async {
      final mock = MockClient((request) async {
        final body = jsonDecode(request.body) as Map<String, dynamic>;
        expect(body['resource'], 'products');
        expect(body['action'], 'read');

        return http.Response(
          jsonEncode({'allowed': true, 'reason': 'role permits read'}),
          200,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final allowed = await ggid.checkPermission('token', 'products', 'read');
      expect(allowed, true);
    });

    test('checkPermission returns false when denied', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode({'allowed': false, 'reason': 'insufficient permissions'}),
          200,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final allowed = await ggid.checkPermission('token', 'products', 'delete');
      expect(allowed, false);
    });

    test('listRoles returns roles from array', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode([
            {'id': 'r1', 'name': 'Admin', 'key': 'admin', 'system_role': true},
            {'id': 'r2', 'name': 'User', 'key': 'user', 'system_role': false},
          ]),
          200,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final roles = await ggid.listRoles('token');
      expect(roles.length, 2);
      expect(roles[0].name, 'Admin');
      expect(roles[0].systemRole, true);
      expect(roles[1].key, 'user');
    });

    test('listRoles returns roles from object', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode({
            'roles': [
              {'id': 'r1', 'name': 'Admin', 'key': 'admin', 'system_role': true},
            ]
          }),
          200,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final roles = await ggid.listRoles('token');
      expect(roles.length, 1);
      expect(roles[0].name, 'Admin');
    });

    test('listPermissions returns permissions', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode([
            {'id': 'p1', 'name': 'Read Products', 'resource': 'products', 'action': 'read'},
            {'id': 'p2', 'name': 'Write Products', 'resource': 'products', 'action': 'write'},
          ]),
          200,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final perms = await ggid.listPermissions('token');
      expect(perms.length, 2);
      expect(perms[0].resource, 'products');
      expect(perms[0].action, 'read');
    });

    test('assignRole succeeds', () async {
      final mock = MockClient((request) async {
        expect(request.url.path, contains('/api/v1/roles/assign'));
        return http.Response('{}', 200);
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      await ggid.assignRole('token', 'user-1', 'role-1');
      // No exception means success
    });

    test('getUserRoles returns roles', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode([
            {'id': 'r1', 'name': 'Admin', 'key': 'admin', 'system_role': true},
          ]),
          200,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final roles = await ggid.getUserRoles('token', 'user-1');
      expect(roles.length, 1);
      expect(roles[0].key, 'admin');
    });

    test('listUsers returns users from object', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode({
            'users': [
              {'id': 'u1', 'username': 'admin', 'email': 'admin@test.com', 'status': 'active'},
            ]
          }),
          200,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final users = await ggid.listUsers('token');
      expect(users.length, 1);
      expect(users[0].username, 'admin');
    });

    test('getAuthorizeUrl builds correct URL', () {
      final ggid = GGIDClient(
        baseUrl: 'https://ggid.example.com',
        tenantId: 'tenant-1',
      );

      final url = ggid.getAuthorizeUrl(
        clientId: 'client-1',
        redirectUri: 'https://app.example.com/callback',
        scope: 'openid profile',
        state: 'state123',
      );

      expect(url, contains('client_id=client-1'));
      expect(url, contains('response_type=code'));
      expect(url, contains('state=state123'));
      expect(url, startsWith('https://ggid.example.com/oauth/authorize?'));
    });

    test('exchangeCode returns tokens', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode({
            'access_token': 'at-xyz',
            'refresh_token': 'rt-abc',
            'token_type': 'Bearer',
            'expires_in': 3600,
          }),
          200,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final tokens = await ggid.exchangeCode(
        code: 'code123',
        redirectUri: 'https://app.example.com/callback',
        clientId: 'client-1',
        clientSecret: 'secret',
      );

      expect(tokens.accessToken, 'at-xyz');
      expect(tokens.tokenType, 'Bearer');
    });

    test('checkPolicy returns PolicyResult', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode({
            'allowed': true,
            'reason': 'ABAC policy matched',
            'matched_rules': ['rule-1'],
          }),
          200,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      final result = await ggid.checkPolicy('token', PolicyCheckRequest(
        subject: 'user-1',
        resource: 'documents',
        action: 'read',
        context: {'department': 'finance'},
      ));

      expect(result.allowed, true);
      expect(result.matchedRules, contains('rule-1'));
    });

    test('API error throws GGIDException', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode({'error': 'forbidden', 'detail': 'insufficient role'}),
          403,
        );
      });

      final ggid = GGIDClient(
        baseUrl: 'http://localhost:9999',
        tenantId: 'tenant-1',
        httpClient: mock,
      );

      try {
        await ggid.listRoles('token');
        fail('should have thrown');
      } on GGIDException catch (e) {
        expect(e.statusCode, 403);
        expect(e.message, contains('insufficient role'));
      }
    });

    test('verifyToken throws on empty token', () async {
      final ggid = GGIDClient(
        baseUrl: 'https://ggid.example.com',
        tenantId: 'tenant-1',
      );

      expect(
        () => ggid.verifyToken(''),
        throwsA(isA<InvalidTokenException>()),
      );
    });

    test('verifyToken throws on invalid format', () async {
      final ggid = GGIDClient(
        baseUrl: 'https://ggid.example.com',
        tenantId: 'tenant-1',
      );

      expect(
        () => ggid.verifyToken('not-a-jwt'),
        throwsA(isA<InvalidTokenException>()),
      );
    });
  });

  group('Models', () {
    test('Claims.fromJson extracts roles', () {
      final claims = Claims.fromJson({
        'sub': 'user-1',
        'tenant_id': 'tenant-1',
        'roles': ['admin', 'editor'],
        'scope': 'openid profile',
        'exp': 1700000000,
        'iat': 1699900000,
        'iss': 'https://ggid.example.com',
      });

      expect(claims.userId, 'user-1');
      expect(claims.tenantId, 'tenant-1');
      expect(claims.roles, ['admin', 'editor']);
      expect(claims.scope, 'openid profile');
    });

    test('Claims.fromJson handles single role string', () {
      final claims = Claims.fromJson({
        'sub': 'user-1',
        'role': 'admin',
      });

      expect(claims.roles, ['admin']);
    });

    test('TokenResponse.fromJson handles minimal response', () {
      final tokens = TokenResponse.fromJson({
        'access_token': 'abc',
      });

      expect(tokens.accessToken, 'abc');
      expect(tokens.tokenType, 'Bearer');
      expect(tokens.expiresIn, 0);
    });

    test('Role.fromJson', () {
      final role = Role.fromJson({
        'id': 'r1',
        'name': 'Admin',
        'key': 'admin',
        'system_role': true,
      });

      expect(role.id, 'r1');
      expect(role.name, 'Admin');
      expect(role.key, 'admin');
      expect(role.systemRole, true);
    });

    test('PolicyResult.fromJson', () {
      final result = PolicyResult.fromJson({
        'allowed': false,
        'reason': 'denied',
      });

      expect(result.allowed, false);
      expect(result.reason, 'denied');
    });

    test('PolicyCheckRequest.toJson', () {
      const req = PolicyCheckRequest(
        subject: 'user-1',
        resource: 'docs',
        action: 'read',
        context: {'dept': 'finance'},
      );

      final json = req.toJson();
      expect(json['subject'], 'user-1');
      expect(json['resource'], 'docs');
      expect(json['action'], 'read');
      expect(json['context']['dept'], 'finance');
    });
  });

  group('Webhooks', () {
    test('listWebhooks returns list from array', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode([
            {'id': 'wh-1', 'url': 'https://example.com/hook', 'events': ['user.created']},
            {'id': 'wh-2', 'url': 'https://example.com/hook2', 'events': ['role.assigned']},
          ]),
          200,
        );
      });
      final ggid = GGIDClient(baseUrl: 'http://localhost:9999', tenantId: 't1', httpClient: mock);
      final hooks = await ggid.listWebhooks('token');
      expect(hooks.length, 2);
      expect(hooks[0]['id'], 'wh-1');
    });

    test('createWebhook returns created webhook', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode({'id': 'wh-3', 'url': 'https://example.com/hook3', 'events': ['user.created']}),
          201,
        );
      });
      final ggid = GGIDClient(baseUrl: 'http://localhost:9999', tenantId: 't1', httpClient: mock);
      final result = await ggid.createWebhook('token', 'https://example.com/hook3', ['user.created']);
      expect(result['id'], 'wh-3');
      expect(result['url'], 'https://example.com/hook3');
    });

    test('deleteWebhook succeeds', () async {
      final mock = MockClient((request) async {
        return http.Response('{}', 200);
      });
      final ggid = GGIDClient(baseUrl: 'http://localhost:9999', tenantId: 't1', httpClient: mock);
      await ggid.deleteWebhook('token', 'wh-1');
    });
  });

  group('Introspect', () {
    test('introspectToken returns active status', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode({'active': true, 'sub': 'user-1', 'exp': 1700000000}),
          200,
        );
      });
      final ggid = GGIDClient(baseUrl: 'http://localhost:9999', tenantId: 't1', httpClient: mock);
      final result = await ggid.introspectToken('token', clientId: 'cid', clientSecret: 'sec');
      expect(result['active'], true);
      expect(result['sub'], 'user-1');
    });

    test('introspectToken returns inactive for revoked token', () async {
      final mock = MockClient((request) async {
        return http.Response(
          jsonEncode({'active': false}),
          200,
        );
      });
      final ggid = GGIDClient(baseUrl: 'http://localhost:9999', tenantId: 't1', httpClient: mock);
      final result = await ggid.introspectToken('revoked-token');
      expect(result['active'], false);
    });
  });
}
