package server

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"

	"github.com/ggid/ggid/services/oauth/internal/service"
)

// POST /api/v1/oauth/dpop/verify — verify a DPoP proof JWT.
// Body: {"proof": "...", "htm": "POST", "htu": "https://example.com/token"}
// Returns: {"is_valid": true, "thumbprint": "...", "key_jwk": {...}, "jti": "..."}
func handleDPoPVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	var req struct {
		Proof string `json:"proof"`
		HTM   string `json:"htm"`
		HTU   string `json:"htu"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
		return
	}
	if req.Proof == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "proof is required"})
		return
	}
	if req.HTM == "" {
		req.HTM = "POST"
	}
	if req.HTU == "" {
		req.HTU = "https://example.com/token"
	}

	proof, err := service.ParseDPoPHeader(req.Proof, req.HTM, req.HTU)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"is_valid": false,
			"error":    err.Error(),
		})
		return
	}

	keyJWK := publicKeyToJWKMap(proof.PublicKey)
	thumbprint := computeKeyThumbprint(proof.PublicKey)

	writeJSON(w, http.StatusOK, map[string]any{
		"is_valid":   true,
		"jti":        proof.JTI,
		"iat":        proof.IssuedAt,
		"htm":        proof.HTTPMethod,
		"htu":        proof.HTTPURI,
		"thumbprint": thumbprint,
		"key_jwk":    keyJWK,
	})
}

// publicKeyToJWKMap converts a supported public key to a JWK map.
func publicKeyToJWKMap(pub interface{}) map[string]any {
	switch k := pub.(type) {
	case *ecdsa.PublicKey:
		return map[string]any{
			"kty": "EC",
			"crv": "P-256",
			"x":   base64.RawURLEncoding.EncodeToString(k.X.Bytes()),
			"y":   base64.RawURLEncoding.EncodeToString(k.Y.Bytes()),
		}
	case *rsa.PublicKey:
		return map[string]any{
			"kty": "RSA",
			"n":   base64.RawURLEncoding.EncodeToString(k.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.E)).Bytes()),
		}
	case ed25519.PublicKey:
		return map[string]any{
			"kty": "OKP",
			"crv": "Ed25519",
			"x":   base64.RawURLEncoding.EncodeToString(k),
		}
	default:
		return map[string]any{"kty": "unknown"}
	}
}

// computeKeyThumbprint computes a SHA-256 JWK thumbprint (RFC 7638).
func computeKeyThumbprint(pub interface{}) string {
	jwk := publicKeyToJWKMap(pub)
	// Build canonical JSON for thumbprint
	data, _ := json.Marshal(jwk)
	hash := sha256.Sum256(data)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
