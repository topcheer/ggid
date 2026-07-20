/// WebAuthn / Passkey utilities for GGID SDK (Dart)
///
/// Provides browser-side WebAuthn API helpers for passkey registration
/// and authentication via JS interop.

/// Check if the current browser supports WebAuthn.
bool get isWebAuthnSupported {
  // In Flutter web, check via JS interop
  // In non-web contexts, WebAuthn is not available
  return false; // Override in web implementation
}

/// Configuration for passkey operations.
class PasskeyConfig {
  final String apiBaseUrl;
  final String? authToken;
  final String? tenantId;

  const PasskeyConfig({
    required this.apiBaseUrl,
    this.authToken,
    this.tenantId,
  });
}

/// Result of a passkey registration attempt.
class PasskeyResult {
  final bool success;
  final String? error;

  const PasskeyResult({required this.success, this.error});
}

/// Register a new passkey (browser-side, requires Flutter Web + JS interop).
///
/// This is a stub for non-web platforms. On Flutter Web, override with
/// actual `navigator.credentials.create()` via dart:js_interop.
Future<PasskeyResult> registerPasskey(PasskeyConfig config, String userId) async {
  return const PasskeyResult(
    success: false,
    error: 'WebAuthn is only available in browser environments. '
        'Use the Flutter Web implementation with dart:js_interop.',
  );
}

/// Authenticate with a passkey (browser-side).
///
/// Returns the encoded assertion for server verification, or null if cancelled.
Future<Map<String, dynamic>?> authenticateWithPasskey(PasskeyConfig config) async {
  return null; // Override in web implementation
}

/// Encode bytes to base64url string.
String bufferToBase64url(List<int> bytes) {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_';
  final result = StringBuffer();
  for (var i = 0; i < bytes.length; i += 3) {
    final b1 = bytes[i];
    final b2 = i + 1 < bytes.length ? bytes[i + 1] : 0;
    final b3 = i + 2 < bytes.length ? bytes[i + 2] : 0;
    result.write(chars[(b1 >> 2) & 0x3F]);
    if (i + 1 < bytes.length) {
      result.write(chars[((b1 << 4) | (b2 >> 4)) & 0x3F]);
      result.write(i + 2 < bytes.length ? chars[((b2 << 2) | (b3 >> 6)) & 0x3F] : '=');
      result.write(i + 2 < bytes.length ? chars[b3 & 0x3F] : '=');
    }
  }
  return result.toString();
}

/// Decode base64url string to bytes.
List<int> base64urlToBuffer(String b64url) {
  // Use dart:convert for actual implementation
  // This is a simplified decoder
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_';
  final result = <int>[];
  final sanitized = b64url.replaceAll('=', '');
  for (var i = 0; i < sanitized.length; i += 4) {
    final c1 = chars.indexOf(sanitized[i]);
    final c2 = i + 1 < sanitized.length ? chars.indexOf(sanitized[i + 1]) : 0;
    final c3 = i + 2 < sanitized.length ? chars.indexOf(sanitized[i + 2]) : -1;
    final c4 = i + 3 < sanitized.length ? chars.indexOf(sanitized[i + 3]) : -1;
    result.add(((c1 << 2) | (c2 >> 4)) & 0xFF);
    if (c3 >= 0) result.add((((c2 << 4) | (c3 >> 2)) & 0xFF));
    if (c4 >= 0) result.add((((c3 << 6) | c4) & 0xFF));
  }
  return result;
}
