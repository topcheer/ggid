/// PKCE (Proof Key for Code Exchange) utilities — RFC 7636.
///
/// Since the GGID OAuth server enforces PKCE S256, SDK consumers must generate
/// a code_verifier and code_challenge pair before redirecting to the authorize
/// endpoint, then send the code_verifier with the token exchange request.

library ggid.pkce;

import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';
import 'package:crypto/crypto.dart' as crypto;

/// Generates a cryptographically random code_verifier (43-128 chars, RFC 7636 §4.1).
String generateCodeVerifier() {
  final random = Random.secure();
  final bytes = Uint8List(64);
  for (var i = 0; i < 64; i++) {
    bytes[i] = random.nextInt(256);
  }
  return base64Url.encode(bytes).replaceAll('=', '');
}

/// Derives the S256 code_challenge from a code_verifier (RFC 7636 §4.2).
/// code_challenge = BASE64URL(SHA256(code_verifier))
String generateCodeChallenge(String verifier) {
  final bytes = utf8.encode(verifier);
  final digest = crypto.sha256.convert(bytes);
  return base64Url.encode(digest.bytes).replaceAll('=', '');
}

/// PKCE pair returned by [generatePKCEPair].
class PKCEPair {
  final String codeVerifier;
  final String codeChallenge;
  final String codeChallengeMethod;

  const PKCEPair({
    required this.codeVerifier,
    required this.codeChallenge,
    this.codeChallengeMethod = 'S256',
  });
}

/// Generates a complete PKCE pair for use with the authorize URL.
PKCEPair generatePKCEPair() {
  final verifier = generateCodeVerifier();
  final challenge = generateCodeChallenge(verifier);
  return PKCEPair(
    codeVerifier: verifier,
    codeChallenge: challenge,
    codeChallengeMethod: 'S256',
  );
}