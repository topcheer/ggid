/// JWT verification and OAuth helpers for the GGID Dart SDK.
library ggid.auth;

import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:dart_jsonwebtoken/dart_jsonwebtoken.dart';

import 'models.dart';

/// Verifies RS256 JWTs against GGID's JWKS endpoint.
///
/// Caches signing keys with a 5-minute TTL.
class JwtVerifier {
  final String _jwksUrl;
  final http.Client _http;
  final String _tenantId;

  Map<String, dynamic>? _cachedKeys;
  DateTime? _cachedAt;
  static const _ttl = Duration(minutes: 5);

  JwtVerifier(this._jwksUrl, this._http, this._tenantId);

  /// Verify a JWT and return its claims.
  Future<Claims> verify(String token) async {
    if (token.isEmpty) {
      throw const InvalidTokenException('token is empty');
    }

    // Decode without verification first to extract claims and kid
    final parts = token.split('.');
    if (parts.length != 3) {
      throw const InvalidTokenException('invalid JWT format');
    }

    String normalizeBase64(String s) {
      var padded = s.replaceAll('-', '+').replaceAll('_', '/');
      while (padded.length % 4 != 0) {
        padded += '=';
      }
      return padded;
    }

    Map<String, dynamic> header;
    Map<String, dynamic> payload;
    try {
      header = jsonDecode(
        utf8.decode(base64.decode(normalizeBase64(parts[0]))),
      ) as Map<String, dynamic>;
      payload = jsonDecode(
        utf8.decode(base64.decode(normalizeBase64(parts[1]))),
      ) as Map<String, dynamic>;
    } catch (e) {
      throw InvalidTokenException('failed to decode JWT: $e');
    }

    // Check expiration with 60s clock skew
    final exp = payload['exp'];
    if (exp != null) {
      final expTime = (exp as num).toInt();
      final now = DateTime.now().millisecondsSinceEpoch ~/ 1000;
      if (now > expTime + 60) {
        throw const TokenExpiredException();
      }
    }

    // SECURITY: Verify JWT signature against JWKS public key.
    final kid = header['kid'] as String?;
    if (kid == null) {
      throw const InvalidTokenException('JWT header missing kid');
    }

    final jwks = await _getKeys();
    final keys = jwks['keys'] as List<dynamic>?;
    if (keys == null) {
      throw const InvalidTokenException('JWKS response missing keys');
    }

    Map<String, dynamic>? jwk;
    for (final k in keys) {
      if ((k as Map<String, dynamic>)['kid'] == kid) {
        jwk = k;
        break;
      }
    }
    if (jwk == null) {
      throw InvalidTokenException('no JWKS key found for kid=$kid');
    }

    // Verify signature: reconstruct signing input and compare with RSA verification.
    // dart_jsonwebtoken's JWT.verify requires the correct key type.
    // For RS256, we need to build an RSAPublicKey from the JWK n and e fields.
    try {
      final alg = header['alg'] as String? ?? 'RS256';
      if (alg != 'RS256') {
        throw InvalidTokenException('unsupported algorithm: $alg');
      }

      // Use dart_jsonwebtoken with proper RSA key from JWK
      final n = jwk['n'] as String?;
      final e = jwk['e'] as String?;
      if (n == null || e == null) {
        throw const InvalidTokenException('JWK missing n or e');
      }

      // dart_jsonwebtoken can verify with RSAPublicKey constructed from JWK
      final rsaKey = RSAPublicKey(
        BigInt.parse(
          base64Url.decode(n).map((b) => b.toRadixString(16).padLeft(2, '0')).join(),
          radix: 16,
        ),
        BigInt.parse(
          base64Url.decode(e).map((b) => b.toRadixString(16).padLeft(2, '0')).join(),
          radix: 16,
        ),
      );

      // This will throw if signature is invalid.
      JWT.verify(token, rsaKey, checkExpiresIn: false);
    } on JWTInvalidException {
      throw const InvalidTokenException('JWT signature verification failed');
    } on InvalidTokenException {
      rethrow;
    } catch (e) {
      throw InvalidTokenException('JWT verification error: $e');
    }

    // Build claims from decoded payload
    return Claims.fromJson(payload);
  }

  /// Fetch and cache JWKS keys.
  Future<Map<String, dynamic>> _getKeys({bool forceRefresh = false}) async {
    if (!forceRefresh &&
        _cachedKeys != null &&
        _cachedAt != null &&
        DateTime.now().difference(_cachedAt!) < _ttl) {
      return _cachedKeys!;
    }

    final resp = await _http.get(
      Uri.parse(_jwksUrl),
      headers: {'X-Tenant-ID': _tenantId},
    );

    if (resp.statusCode != 200) {
      throw GGIDException(resp.statusCode, 'failed to fetch JWKS: ${resp.body}');
    }

    _cachedKeys = jsonDecode(resp.body) as Map<String, dynamic>;
    _cachedAt = DateTime.now();
    return _cachedKeys!;
  }
}
