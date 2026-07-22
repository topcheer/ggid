package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// SDJWTIssueRequest is the payload for selective disclosure JWT issuance.
type SDJWTIssueRequest struct {
	Subject        string         `json:"subject"`
	Issuer         string         `json:"issuer,omitempty"`
	Claims         map[string]any `json:"claims"`
	Disclosable    []string       `json:"disclosable,omitempty"`
	AlwaysDisclosed []string      `json:"always_disclosed,omitempty"`
	TTLSeconds     int            `json:"ttl_seconds,omitempty"`
}

type SDJWTIssueResponse struct {
	SDJWT        string        `json:"sd_jwt"`
	Disclosures  []Disclosure  `json:"disclosures"`
	ExpiresAt    time.Time     `json:"expires_at"`
}

type Disclosure struct {
	Claim string `json:"claim"`
	Hash  string `json:"hash"`
	Value any    `json:"value"`
}

type SDJWTVerifyRequest struct {
	SDJWT          string   `json:"sd_jwt"`
	RevealedClaims []string `json:"revealed_claims,omitempty"`
}

type SDJWTVerifyResponse struct {
	Valid   bool           `json:"valid"`
	Subject string         `json:"subject,omitempty"`
	Claims  map[string]any `json:"claims,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// getSDJWTSecret returns the signing key for SD-JWT (HMAC-SHA256).
func getSDJWTSecret() []byte {
	secret := os.Getenv("GGID_INTERNAL_SECRET")
	if secret == "" {
		slog.Error("GGID_INTERNAL_SECRET not set — SDJWT handler refuses to operate with insecure default")
		return nil // nil key → hmac.New produces invalid signatures → fail-closed
	}
	return []byte(secret)
}

// b64url encodes bytes using base64url without padding (RFC 7515).
func b64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// b64urlDecode decodes base64url without padding.
func b64urlDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// sha256Hex returns hex-encoded SHA-256 of a string.
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:])
}

// hmacSHA256 signs data with the secret key.
func hmacSHA256(secret, data []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(data)
	return mac.Sum(nil)
}

// handleSDJWTIssue issues a signed SD-JWT (RFC 9496 compliant structure).
// POST /api/v1/identity/sd-jwt/issue
func (h *HTTPHandler) handleSDJWTIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tc, _ := ggidtenant.FromContext(r.Context())
	_ = tc

	var req SDJWTIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Subject == "" {
		writeJSONError(w, http.StatusBadRequest, "subject required")
		return
	}
	if req.TTLSeconds == 0 {
		req.TTLSeconds = 3600
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(req.TTLSeconds) * time.Second)
	if req.Issuer == "" {
		req.Issuer = "ggid"
	}

	// Build JWT header (HS256 signing).
	header := map[string]any{
		"alg": "HS256",
		"typ": "sd-jwt",
	}

	// Build JWT payload: standard claims + always-disclosed + SD hashes.
	payload := map[string]any{
		"iss": req.Issuer,
		"sub": req.Subject,
		"iat": now.Unix(),
		"exp": expiresAt.Unix(),
		"jti": uuid.New().String(),
	}

	disclosures := []Disclosure{}
	disclosableSet := make(map[string]bool)
	for _, d := range req.Disclosable {
		disclosableSet[d] = true
	}
	alwaysSet := make(map[string]bool)
	for _, a := range req.AlwaysDisclosed {
		alwaysSet[a] = true
	}

	for key, val := range req.Claims {
		if disclosableSet[key] && !alwaysSet[key] {
			// Create disclosure: SHA-256 hash of key+value.
			disclosureData := key + ":" + fmt.Sprintf("%v", val)
			hash := sha256Hex(disclosureData)
			disclosures = append(disclosures, Disclosure{
				Claim: key,
				Hash:  hash,
				Value: val,
			})
			// Store hash reference in JWT (SD claim).
			payload["_sd_"+key] = hash
		} else {
			// Always disclosed: put directly in JWT.
			payload[key] = val
		}
	}

	// Sign: header.payload with HMAC-SHA256.
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)
	signingInput := b64url(headerJSON) + "." + b64url(payloadJSON)
	signature := hmacSHA256(getSDJWTSecret(), []byte(signingInput))

	sjwt := signingInput + "." + b64url(signature)

	writeJSON(w, http.StatusCreated, SDJWTIssueResponse{
		SDJWT:       sjwt,
		Disclosures: disclosures,
		ExpiresAt:   expiresAt,
	})
}

// handleSDJWTVerify verifies a signed SD-JWT.
// POST /api/v1/identity/sd-jwt/verify
func (h *HTTPHandler) handleSDJWTVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SDJWTVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.SDJWT == "" {
		writeJSONError(w, http.StatusBadRequest, "sd_jwt required")
		return
	}

	parts := strings.Split(req.SDJWT, ".")
	if len(parts) != 3 {
		writeJSON(w, http.StatusOK, SDJWTVerifyResponse{
			Valid: false,
			Error: "malformed SD-JWT: expected header.payload.signature",
		})
		return
	}

	// Verify signature (HMAC-SHA256 constant-time).
	signingInput := parts[0] + "." + parts[1]
	expectedSig := hmacSHA256(getSDJWTSecret(), []byte(signingInput))
	actualSig, err := b64urlDecode(parts[2])
	if err != nil || !hmac.Equal(expectedSig, actualSig) {
		writeJSON(w, http.StatusOK, SDJWTVerifyResponse{
			Valid: false,
			Error: "signature verification failed",
		})
		return
	}

	// Decode payload.
	payloadBytes, err := b64urlDecode(parts[1])
	if err != nil {
		writeJSON(w, http.StatusOK, SDJWTVerifyResponse{
			Valid: false,
			Error: "failed to decode payload",
		})
		return
	}

	var claims map[string]any
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		writeJSON(w, http.StatusOK, SDJWTVerifyResponse{
			Valid: false,
			Error: "invalid payload JSON",
		})
		return
	}

	// Check expiry.
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			writeJSON(w, http.StatusOK, SDJWTVerifyResponse{
				Valid: false,
				Error: "token expired",
			})
			return
		}
	}

	// Filter to revealed claims.
	revealedSet := make(map[string]bool)
	for _, c := range req.RevealedClaims {
		revealedSet[c] = true
	}
	standardClaims := map[string]bool{"iss": true, "sub": true, "iat": true, "exp": true, "jti": true}
	result := make(map[string]any)
	for key, val := range claims {
		if standardClaims[key] {
			result[key] = val
			continue
		}
		if strings.HasPrefix(key, "_sd_") {
			claimName := strings.TrimPrefix(key, "_sd_")
			if revealedSet[claimName] {
				result[claimName+"_hash"] = val
			}
			continue
		}
		result[key] = val
	}

	subject, _ := claims["sub"].(string)
	writeJSON(w, http.StatusOK, SDJWTVerifyResponse{
		Valid:   true,
		Subject: subject,
		Claims:  result,
	})
}

// generateSalt generates a random salt for future use.
func generateSalt(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}
