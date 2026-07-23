// Package webhook provides utilities for verifying webhook payload signatures.
// SDK consumers use this to verify that webhook events came from GGID.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
)

// VerifySignature verifies the HMAC-SHA256 signature of a webhook payload.
//
// GGID signs webhook payloads with HMAC-SHA256 using the webhook's secret.
// The signature is sent in the X-GGID-Signature header as hex-encoded.
//
// Usage (Go SDK):
//
//	body, _ := io.ReadAll(r.Body)
//	sig := r.Header.Get("X-GGID-Signature")
//	if !webhook.VerifySignature(body, sig, "your-webhook-secret") {
//	    http.Error(w, "invalid signature", http.StatusUnauthorized)
//	    return
//	}
//
// Usage (Node SDK):
//
//	import crypto from 'crypto';
//	function verifySignature(body, signature, secret) {
//	    const expected = crypto.createHmac('sha256', secret).update(body).digest('hex');
//	    return crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(expected));
//	}
//
// Usage (Python SDK):
//
//	import hmac, hashlib
//	def verify_signature(body: bytes, signature: str, secret: str) -> bool:
//	    expected = hmac.new(secret.encode(), body, hashlib.sha256).hexdigest()
//	    return hmac.compare_digest(signature, expected)
func VerifySignature(payload []byte, signature string, secret string) bool {
	if signature == "" || secret == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// VerifyRequest is a convenience wrapper that extracts the signature header
// and body from an HTTP request, then verifies it.
func VerifyRequest(r *http.Request, secret string) error {
	sig := r.Header.Get("X-GGID-Signature")
	if sig == "" {
		sig = r.Header.Get("X-GGID-Signature-256") // Alternate header
	}
	if sig == "" {
		return fmt.Errorf("missing X-GGID-Signature header")
	}

	// Read body (caller must handle body reading if already consumed)
	body := make([]byte, r.ContentLength)
	if r.Body != nil {
		n, err := r.Body.Read(body)
		if err != nil && n == 0 {
			return fmt.Errorf("failed to read body: %w", err)
		}
		body = body[:n]
	}

	if !VerifySignature(body, sig, secret) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

// SignPayload signs a payload with HMAC-SHA256 for outbound webhooks.
// This is used by the GGID webhook dispatcher to sign events.
func SignPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
