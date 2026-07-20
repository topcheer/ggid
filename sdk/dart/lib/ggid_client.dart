/// Main GGID client for the Dart SDK.
///
/// Provides JWT verification, RBAC/ABAC authorization, OAuth/OIDC flows,
/// and user management.
///
/// Quick start:
/// ```dart
/// final ggid = GGIDClient(baseUrl: 'https://ggid.iot2.win', tenantId: '00000000-...');
/// final claims = await ggid.verifyToken(jwt);
/// final allowed = await ggid.checkPermission(token, 'products', 'read');
/// ```
library ggid;

import 'dart:convert';
import 'dart:async';
import 'package:http/http.dart' as http;

import 'src/models.dart';
import 'src/auth.dart';

export 'src/models.dart';
export 'src/auth.dart';
export 'src/rbac.dart';
export 'src/abac.dart';
export 'src/middleware.dart';
export 'src/saml.dart';
export 'src/passkey.dart';

/// SDK version.
const String ggidVersion = '1.0.0';

/// Main GGID API client.
///
/// All API methods are async and return typed models.
/// Use [GGIDClient] for authentication, RBAC, ABAC, and OAuth/OIDC.
class GGIDClient {
  /// GGID gateway base URL (e.g. https://ggid.iot2.win).
  final String baseUrl;

  /// Tenant UUID for all requests.
  final String tenantId;

  /// HTTP client (injectable for testing).
  final http.Client _http;

  /// JWT verifier (lazy-initialized).
  JwtVerifier? _verifier;

  /// Default constructor.
  ///
  /// ```dart
  /// final ggid = GGIDClient(
  ///   baseUrl: 'https://ggid.iot2.win',
  ///   tenantId: '00000000-0000-0000-0000-000000000001',
  /// );
  /// ```
  GGIDClient({
    required this.baseUrl,
    required this.tenantId,
    http.Client? httpClient,
  }) : _http = httpClient ?? http.Client();

  /// Enable JWT verification via JWKS.
  GGIDClient withJwks({String? jwksUrl}) {
    _verifier = JwtVerifier(
      jwksUrl ?? '$baseUrl/oauth/jwks',
      _http,
      tenantId,
    );
    return this;
  }

  // ── Authentication ──

  /// Login with username/password and receive tokens.
  Future<TokenResponse> login(String username, String password) async {
    final json = await _post('/api/v1/auth/login', {
      'username': username,
      'password': password,
    });
    return TokenResponse.fromJson(json);
  }

  /// Register a new user account.
  Future<String> register(String username, String email, String password, String name) async {
    final json = await _post('/api/v1/auth/register', {
      'username': username,
      'email': email,
      'password': password,
      'name': name,
    });
    return json['user_id']?.toString() ?? '';
  }

  /// Refresh tokens using a refresh token.
  Future<TokenResponse> refreshToken(String refreshToken) async {
    final json = await _post('/api/v1/auth/refresh', {
      'refresh_token': refreshToken,
    });
    return TokenResponse.fromJson(json);
  }

  // ── JWT Verification ──

  /// Verify a JWT and return claims.
  ///
  /// ```dart
  /// final claims = await ggid.verifyToken(token);
  /// print('User: ${claims.userId}, Roles: ${claims.roles}');
  /// ```
  Future<Claims> verifyToken(String token) async {
    _verifier ??= JwtVerifier('$baseUrl/oauth/jwks', _http, tenantId);
    return _verifier!.verify(token);
  }

  // ── UserInfo ──

  /// Get user info for the given access token.
  Future<UserInfo> getUserInfo(String accessToken) async {
    final json = await _get('/oauth/userinfo', accessToken);
    return UserInfo.fromJson(json as Map<String, dynamic>);
  }

  // ── OAuth/OIDC ──

  /// Get OIDC discovery document.
  Future<DiscoveryConfig> getDiscovery() async {
    final json = await _get('/.well-known/openid-configuration', null);
    return DiscoveryConfig.fromJson(json as Map<String, dynamic>);
  }

  /// Get JWKS for JWT verification.
  Future<Map<String, dynamic>> getJwks() async {
    final json = await _get('/oauth/jwks', null);
    return json as Map<String, dynamic>;
  }

  /// Build an authorization URL for the OAuth code flow.
  String getAuthorizeUrl({
    required String clientId,
    required String redirectUri,
    String? scope,
    String? state,
  }) {
    final params = <String, String>{
      'client_id': clientId,
      'redirect_uri': redirectUri,
      'response_type': 'code',
    };
    if (scope != null) params['scope'] = scope;
    if (state != null) params['state'] = state;

    final qs = params.entries.map((e) =>
      '${Uri.encodeQueryComponent(e.key)}=${Uri.encodeQueryComponent(e.value)}').join('&');
    return '$baseUrl/oauth/authorize?$qs';
  }

  /// Exchange an authorization code for tokens.
  Future<TokenResponse> exchangeCode({
    required String code,
    required String redirectUri,
    required String clientId,
    required String clientSecret,
  }) async {
    final json = await _postForm('/api/v1/oauth/token', {
      'grant_type': 'authorization_code',
      'code': code,
      'redirect_uri': redirectUri,
      'client_id': clientId,
      'client_secret': clientSecret,
    });
    return TokenResponse.fromJson(json);
  }

  /// Revoke a token (RFC 7009).
  Future<void> revokeToken(String token) async {
    await _post('/api/v1/oauth/revoke', {'token': token});
  }

  /// Introspect a token (RFC 7662). Returns {active, sub, exp, ...}.
  Future<Map<String, dynamic>> introspectToken(String token, {String? clientId, String? clientSecret}) async {
    final body = <String, dynamic>{'token': token};
    if (clientId != null) body['client_id'] = clientId;
    if (clientSecret != null) body['client_secret'] = clientSecret;
    final json = await _post('/api/v1/oauth/introspect', body);
    return json as Map<String, dynamic>;
  }

  // ── Webhooks ──

  /// List all webhooks in the tenant.
  Future<List<Map<String, dynamic>>> listWebhooks(String token) async {
    final json = await _get('/api/v1/webhooks', token);
    if (json is List) {
      return json.cast<Map<String, dynamic>>();
    }
    if (json is Map && json['webhooks'] is List) {
      return (json['webhooks'] as List).cast<Map<String, dynamic>>();
    }
    return [];
  }

  /// Create a new webhook.
  Future<Map<String, dynamic>> createWebhook(String token, String url, List<String> events) async {
    final json = await _post('/api/v1/webhooks', {
      'url': url,
      'events': events,
    }, token);
    return json as Map<String, dynamic>;
  }

  /// Delete a webhook by ID.
  Future<void> deleteWebhook(String token, String webhookId) async {
    await _delete('/api/v1/webhooks/$webhookId', token);
  }

  // ── User Management ──

  /// List all users in the tenant.
  Future<List<User>> listUsers(String token) async {
    final json = await _get('/api/v1/users', token);
    List? list;
    if (json is List) {
      list = json;
    } else if (json is Map && json['users'] is List) {
      list = json['users'] as List;
    }
    if (list != null) {
      return list.map((u) => User.fromJson(u as Map<String, dynamic>)).toList();
    }
    return [];
  }

  /// Get a single user by ID.
  Future<User> getUser(String token, String userId) async {
    final json = await _get('/api/v1/users/$userId', token);
    return User.fromJson(json as Map<String, dynamic>);
  }

  /// Delete a user by ID.
  Future<void> deleteUser(String token, String userId) async {
    await _delete('/api/v1/users/$userId', token);
  }

  // ── RBAC ──

  /// Check if the token's user can perform an action on a resource.
  ///
  /// ```dart
  /// final allowed = await ggid.checkPermission(token, 'products', 'read');
  /// ```
  Future<bool> checkPermission(String token, String resource, String action) async {
    final result = await checkPermissionResult(token, resource, action);
    return result.allowed;
  }

  /// Check permission and return full policy result.
  Future<PolicyResult> checkPermissionResult(String token, String resource, String action) async {
    final json = await _post('/api/v1/policies/check', {
      'resource': resource,
      'action': action,
    }, token);
    return PolicyResult.fromJson(json as Map<String, dynamic>);
  }

  /// Assign a role to a user.
  Future<void> assignRole(String token, String userId, String roleId) async {
    await _post('/api/v1/roles/assign', {
      'user_id': userId,
      'role_id': roleId,
    }, token);
  }

  /// Revoke a role from a user.
  Future<void> revokeRole(String token, String userId, String roleId) async {
    await _deleteWithBody('/api/v1/roles/revoke', {
      'user_id': userId,
      'role_id': roleId,
    }, token);
  }

  /// Get all roles assigned to a user.
  Future<List<Role>> getUserRoles(String token, String userId) async {
    final json = await _get('/api/v1/users/$userId/roles', token);
    List? list;
    if (json is List) {
      list = json;
    } else if (json is Map && json['roles'] is List) {
      list = json['roles'] as List;
    }
    if (list != null) {
      return list.map((r) => Role.fromJson(r as Map<String, dynamic>)).toList();
    }
    return [];
  }

  /// List all roles in the tenant.
  Future<List<Role>> listRoles(String token) async {
    final json = await _get('/api/v1/roles', token);
    List? list;
    if (json is List) {
      list = json;
    } else if (json is Map && json['roles'] is List) {
      list = json['roles'] as List;
    }
    if (list != null) {
      return list.map((r) => Role.fromJson(r as Map<String, dynamic>)).toList();
    }
    return [];
  }

  /// List all available permissions.
  Future<List<Permission>> listPermissions(String token) async {
    final json = await _get('/api/v1/permissions', token);
    List? list;
    if (json is List) {
      list = json;
    } else if (json is Map && json['permissions'] is List) {
      list = json['permissions'] as List;
    }
    if (list != null) {
      return list.map((p) => Permission.fromJson(p as Map<String, dynamic>)).toList();
    }
    return [];
  }

  /// Create a new role.
  Future<Role> createRole(
    String token, {
    required String name,
    required String key,
    String? description,
  }) async {
    final json = await _post('/api/v1/roles', {
      'name': name,
      'key': key,
      'description': description ?? '',
    }, token);
    return Role.fromJson(json as Map<String, dynamic>);
  }

  /// Delete a role by ID.
  Future<void> deleteRole(String token, String roleId) async {
    await _delete('/api/v1/roles/$roleId', token);
  }

  // ── ABAC ──

  /// Evaluate an ABAC policy with conditions.
  Future<AbacEvalResult> evaluateAbac(String token, AbacEvalRequest request) async {
    final json = await _post('/api/v1/policies/abac/evaluate', request.toJson(), token);
    return AbacEvalResult.fromJson(json as Map<String, dynamic>);
  }

  /// Check a policy with subject, resource, action, and context.
  Future<PolicyResult> checkPolicy(String token, PolicyCheckRequest request) async {
    final json = await _post('/api/v1/policies/check', request.toJson(), token);
    return PolicyResult.fromJson(json as Map<String, dynamic>);
  }

  // ── Agent Identity ──

  /// Register a new AI agent.
  Future<Map<String, dynamic>> registerAgent(
    String token, {
    required String name,
    required String type,
    String ownerUserId = '',
    String description = '',
    List<String> allowedScopes = const [],
    int maxDelegationDepth = 3,
    int rateLimitPerMin = 60,
  }) async {
    final json = await _post('/api/v1/agents/register', {
      'name': name,
      'type': type,
      'owner_user_id': ownerUserId,
      'description': description,
      'allowed_scopes': allowedScopes,
      'max_delegation_depth': maxDelegationDepth,
      'rate_limit_per_min': rateLimitPerMin,
    }, token);
    return json as Map<String, dynamic>;
  }

  /// List all agents for the current tenant.
  Future<List<Map<String, dynamic>>> listAgents(String token) async {
    final json = await _get('/api/v1/agents', token);
    List? list;
    if (json is List) {
      list = json;
    } else if (json is Map && json['agents'] is List) {
      list = json['agents'] as List;
    }
    if (list != null) {
      return list.cast<Map<String, dynamic>>();
    }
    return [];
  }

  /// Exchange a user token for an agent-scoped token.
  Future<Map<String, dynamic>> exchangeAgentToken({
    required String agentId,
    required String subjectToken,
    List<String> scopes = const [],
  }) async {
    final json = await _post('/api/v1/agents/token', {
      'agent_id': agentId,
      'subject_token': subjectToken,
      'scope': scopes,
    });
    return json as Map<String, dynamic>;
  }

  /// Verify an agent token and return its claims.
  Future<Map<String, dynamic>> verifyAgentToken(String token) async {
    final json = await _post('/api/v1/agents/verify', {'token': token});
    return json as Map<String, dynamic>;
  }

  // ── Access Request (IGA) ──

  /// Create an access request.
  Future<Map<String, dynamic>> createAccessRequest(
    String token, {
    required String userId,
    required String resource,
    required String action,
    String reason = '',
  }) async {
    final json = await _post('/api/v1/access-requests', {
      'user_id': userId,
      'resource': resource,
      'action': action,
      'reason': reason,
    }, token);
    return json as Map<String, dynamic>;
  }

  /// List access requests for the current tenant.
  Future<List<Map<String, dynamic>>> listAccessRequests(String token) async {
    final json = await _get('/api/v1/access-requests', token);
    List? list;
    if (json is List) {
      list = json;
    } else if (json is Map && json['requests'] is List) {
      list = json['requests'] as List;
    }
    if (list != null) {
      return list.cast<Map<String, dynamic>>();
    }
    return [];
  }

  /// Approve an access request.
  Future<Map<String, dynamic>> approveAccessRequest(
    String token,
    String requestId, {
    String comment = '',
  }) async {
    final json = await _post('/api/v1/access-requests/$requestId/approve', {
      'comment': comment,
    }, token);
    return json as Map<String, dynamic>;
  }

  /// Reject an access request.
  Future<Map<String, dynamic>> rejectAccessRequest(
    String token,
    String requestId, {
    String comment = '',
  }) async {
    final json = await _post('/api/v1/access-requests/$requestId/reject', {
      'comment': comment,
    }, token);
    return json as Map<String, dynamic>;
  }

  // ── Internal HTTP ──

  Future<dynamic> _get(String path, String? token) async {
    final resp = await _http.get(
      Uri.parse('$baseUrl$path'),
      headers: _headers(token),
    );
    return _handleResponse(resp);
  }

  Future<Map<String, dynamic>> _post(String path, Map<String, dynamic> body, [String? token]) async {
    final resp = await _http.post(
      Uri.parse('$baseUrl$path'),
      headers: _headers(token),
      body: jsonEncode(body),
    );
    return _handleResponse(resp);
  }

  Future<Map<String, dynamic>> _postForm(String path, Map<String, String> form) async {
    final resp = await _http.post(
      Uri.parse('$baseUrl$path'),
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
        'X-Tenant-ID': tenantId,
      },
      body: form,
    );
    return _handleResponse(resp);
  }

  Future<void> _delete(String path, String? token) async {
    final resp = await _http.delete(
      Uri.parse('$baseUrl$path'),
      headers: _headers(token),
    );
    if (resp.statusCode >= 400) {
      throw _createException(resp.statusCode, resp.body);
    }
  }

  Future<void> _deleteWithBody(String path, Map<String, dynamic> body, String? token) async {
    final req = http.Request('DELETE', Uri.parse('$baseUrl$path'));
    req.headers.addAll(_headers(token));
    req.body = jsonEncode(body);
    final streamed = await _http.send(req);
    final resp = await http.Response.fromStream(streamed);
    if (resp.statusCode >= 400) {
      throw _createException(resp.statusCode, resp.body);
    }
  }

  Map<String, String> _headers(String? token) {
    final h = <String, String>{
      'X-Tenant-ID': tenantId,
      'Content-Type': 'application/json',
    };
    if (token != null && token.isNotEmpty) {
      h['Authorization'] = 'Bearer $token';
    }
    return h;
  }

  dynamic _handleResponse(http.Response resp) {
    if (resp.statusCode >= 400) {
      throw _createException(resp.statusCode, resp.body);
    }
    final body = resp.body;
    if (body.isEmpty) return <String, dynamic>{};
    return jsonDecode(body);
  }

  GGIDException _createException(int statusCode, String body) {
    var msg = body;
    try {
      final json = jsonDecode(body) as Map<String, dynamic>;
      msg = json['detail']?.toString() ??
          json['message']?.toString() ??
          json['error']?.toString() ??
          body;
    } catch (_) {
      // use raw body
    }
    return GGIDException(statusCode, msg);
  }
}
