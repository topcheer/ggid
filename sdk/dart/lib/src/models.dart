/// Data models for the GGID Dart SDK.
library ggid.models;

/// JWT claims extracted from a verified access token.
class Claims {
  final String? userId;
  final String? tenantId;
  final List<String> roles;
  final String? scope;
  final int exp;
  final int iat;
  final String? iss;
  final String? email;
  final String? name;

  const Claims({
    this.userId,
    this.tenantId,
    this.roles = const [],
    this.scope,
    this.exp = 0,
    this.iat = 0,
    this.iss,
    this.email,
    this.name,
  });

  factory Claims.fromJson(Map<String, dynamic> json) {
    final sub = json['sub'] ?? json['user_id'];
    final roles = <String>[];
    if (json['roles'] is List) {
      roles.addAll((json['roles'] as List).map((e) => e.toString()));
    } else if (json['role'] is String) {
      roles.add(json['role'] as String);
    }
    return Claims(
      userId: sub?.toString(),
      tenantId: json['tenant_id']?.toString(),
      roles: roles,
      scope: json['scope']?.toString(),
      exp: (json['exp'] as num?)?.toInt() ?? 0,
      iat: (json['iat'] as num?)?.toInt() ?? 0,
      iss: json['iss']?.toString(),
      email: json['email']?.toString(),
      name: json['name']?.toString(),
    );
  }

  Map<String, dynamic> toJson() => {
        'sub': userId,
        'tenant_id': tenantId,
        'roles': roles,
        'scope': scope,
        'exp': exp,
        'iat': iat,
        'iss': iss,
        'email': email,
        'name': name,
      };
}

/// OpenID Connect UserInfo response.
class UserInfo {
  final String? sub;
  final String? name;
  final String? email;
  final bool emailVerified;
  final String? preferredUsername;
  final String? picture;
  final String? locale;
  final int updatedAt;

  const UserInfo({
    this.sub,
    this.name,
    this.email,
    this.emailVerified = false,
    this.preferredUsername,
    this.picture,
    this.locale,
    this.updatedAt = 0,
  });

  factory UserInfo.fromJson(Map<String, dynamic> json) => UserInfo(
        sub: json['sub']?.toString(),
        name: json['name']?.toString(),
        email: json['email']?.toString(),
        emailVerified: json['email_verified'] == true,
        preferredUsername: json['preferred_username']?.toString(),
        picture: json['picture']?.toString(),
        locale: json['locale']?.toString(),
        updatedAt: (json['updated_at'] as num?)?.toInt() ?? 0,
      );
}

/// OAuth 2.0 token response from login or token exchange.
class TokenResponse {
  final String accessToken;
  final String? refreshToken;
  final String? idToken;
  final String tokenType;
  final int expiresIn;

  const TokenResponse({
    required this.accessToken,
    this.refreshToken,
    this.idToken,
    this.tokenType = 'Bearer',
    this.expiresIn = 0,
  });

  factory TokenResponse.fromJson(Map<String, dynamic> json) => TokenResponse(
        accessToken: json['access_token']?.toString() ?? '',
        refreshToken: json['refresh_token']?.toString(),
        idToken: json['id_token']?.toString(),
        tokenType: json['token_type']?.toString() ?? 'Bearer',
        expiresIn: (json['expires_in'] as num?)?.toInt() ?? 0,
      );
}

/// A GGID role.
class Role {
  final String id;
  final String name;
  final String key;
  final String? description;
  final bool systemRole;

  const Role({
    required this.id,
    required this.name,
    required this.key,
    this.description,
    this.systemRole = false,
  });

  factory Role.fromJson(Map<String, dynamic> json) => Role(
        id: json['id']?.toString() ?? '',
        name: json['name']?.toString() ?? '',
        key: json['key']?.toString() ?? '',
        description: json['description']?.toString(),
        systemRole: json['system_role'] == true,
      );
}

/// A GGID permission entry.
class Permission {
  final String id;
  final String name;
  final String resource;
  final String action;
  final String? description;

  const Permission({
    required this.id,
    required this.name,
    required this.resource,
    required this.action,
    this.description,
  });

  factory Permission.fromJson(Map<String, dynamic> json) => Permission(
        id: json['id']?.toString() ?? '',
        name: json['name']?.toString() ?? '',
        resource: json['resource']?.toString() ?? '',
        action: json['action']?.toString() ?? '',
        description: json['description']?.toString(),
      );
}

/// Result of a permission/policy check.
class PolicyResult {
  final bool allowed;
  final String? reason;
  final List<String> matchedRules;

  const PolicyResult({
    required this.allowed,
    this.reason,
    this.matchedRules = const [],
  });

  factory PolicyResult.fromJson(Map<String, dynamic> json) => PolicyResult(
        allowed: json['allowed'] == true,
        reason: json['reason']?.toString(),
        matchedRules: (json['matched_rules'] as List?)
                ?.map((e) => e.toString())
                .toList() ??
            const [],
      );
}

/// ABAC policy check request.
class PolicyCheckRequest {
  final String subject;
  final String resource;
  final String action;
  final Map<String, String> context;

  const PolicyCheckRequest({
    required this.subject,
    required this.resource,
    required this.action,
    this.context = const {},
  });

  Map<String, dynamic> toJson() => {
        'subject': subject,
        'resource': resource,
        'action': action,
        'context': context,
      };
}

/// ABAC condition for attribute-based evaluation.
class AbacCondition {
  final String field;
  final String operator;
  final String value;

  const AbacCondition({
    required this.field,
    required this.operator,
    required this.value,
  });

  Map<String, dynamic> toJson() => {
        'field': field,
        'operator': operator,
        'value': value,
      };
}

/// ABAC evaluation request.
class AbacEvalRequest {
  final String action;
  final String resource;
  final List<AbacCondition> conditions;

  const AbacEvalRequest({
    required this.action,
    required this.resource,
    this.conditions = const [],
  });

  Map<String, dynamic> toJson() => {
        'action': action,
        'resource': resource,
        'conditions': conditions.map((c) => c.toJson()).toList(),
      };
}

/// ABAC evaluation result.
class AbacEvalResult {
  final bool matched;
  final List<String> matchedRules;

  const AbacEvalResult({
    required this.matched,
    this.matchedRules = const [],
  });

  factory AbacEvalResult.fromJson(Map<String, dynamic> json) => AbacEvalResult(
        matched: json['matched'] == true,
        matchedRules: (json['matched_rules'] as List?)
                ?.map((e) => e.toString())
                .toList() ??
            const [],
      );
}

/// OpenID Connect discovery document.
class DiscoveryConfig {
  final String? issuer;
  final String? authorizationEndpoint;
  final String? tokenEndpoint;
  final String? userInfoEndpoint;
  final String? jwksUri;
  final String? revocationEndpoint;

  const DiscoveryConfig({
    this.issuer,
    this.authorizationEndpoint,
    this.tokenEndpoint,
    this.userInfoEndpoint,
    this.jwksUri,
    this.revocationEndpoint,
  });

  factory DiscoveryConfig.fromJson(Map<String, dynamic> json) => DiscoveryConfig(
        issuer: json['issuer']?.toString(),
        authorizationEndpoint: json['authorization_endpoint']?.toString(),
        tokenEndpoint: json['token_endpoint']?.toString(),
        userInfoEndpoint: json['userinfo_endpoint']?.toString(),
        jwksUri: json['jwks_uri']?.toString(),
        revocationEndpoint: json['revocation_endpoint']?.toString(),
      );
}

/// Represents a GGID user.
class User {
  final String id;
  final String username;
  final String email;
  final String status;
  final String? displayName;
  final String? createdAt;

  const User({
    required this.id,
    required this.username,
    required this.email,
    this.status = '',
    this.displayName,
    this.createdAt,
  });

  factory User.fromJson(Map<String, dynamic> json) => User(
        id: json['id']?.toString() ?? '',
        username: json['username']?.toString() ?? '',
        email: json['email']?.toString() ?? '',
        status: json['status']?.toString() ?? '',
        displayName: json['display_name']?.toString(),
        createdAt: json['created_at']?.toString(),
      );
}

/// Custom exception for GGID API errors.
class GGIDException implements Exception {
  final int statusCode;
  final String message;

  const GGIDException(this.statusCode, this.message);

  @override
  String toString() => 'GGIDException($statusCode): $message';
}

/// Exception thrown when JWT verification fails.
class InvalidTokenException implements Exception {
  final String message;
  const InvalidTokenException(this.message);

  @override
  String toString() => 'InvalidTokenException: $message';
}

/// Exception thrown when a token has expired.
class TokenExpiredException implements Exception {
  const TokenExpiredException();

  @override
  String toString() => 'TokenExpiredException: token expired';
}
