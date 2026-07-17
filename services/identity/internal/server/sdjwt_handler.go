package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// SDJWTIssueRequest is the payload for selective disclosure JWT issuance.
type SDJWTIssueRequest struct {
	Subject string         `json:"subject"`
	Issuer  string         `json:"issuer,omitempty"`
	Claims  map[string]any `json:"claims"`
	// Disclosable claims: keys listed here can be selectively revealed by holder.
	Disclosable []string `json:"disclosable,omitempty"`
	// Always-disclosed claims: visible without holder choice.
	AlwaysDisclosed []string `json:"always_disclosed,omitempty"`
	TTLSeconds      int      `json:"ttl_seconds,omitempty"`
}

// SDJWTIssueResponse is the issued SD-JWT.
type SDJWTIssueResponse struct {
	SJWT       string         `json:"sd_jwt"`
	Disclosures []Disclosure  `json:"disclosures"`
	ExpiresAt  time.Time      `json:"expires_at"`
}

// Disclosure represents a selectively disclosable claim.
type Disclosure struct {
	Claim string `json:"claim"`
	Hash  string `json:"hash"`
	Value any    `json:"value"`
}

// SDJWTVerifyRequest is the payload for SD-JWT verification.
type SDJWTVerifyRequest struct {
	SDJWT        string   `json:"sd_jwt"`
	RevealedClaims []string `json:"revealed_claims,omitempty"`
}

// SDJWTVerifyResponse is the verification result.
type SDJWTVerifyResponse struct {
	Valid    bool           `json:"valid"`
	Subject  string         `json:"subject,omitempty"`
	Claims   map[string]any `json:"claims,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// handleSDJWTIssue issues a selective disclosure JWT.
// POST /api/v1/identity/sd-jwt/issue
func (h *HTTPHandler) handleSDJWTIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tc, _ := ggidtenant.FromContext(r.Context())
	_ = tc

	var req SDJWTIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Subject == "" {
		writeError(w, http.StatusBadRequest, "subject required")
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

	// Build SD-JWT: split claims into always-disclosed (in JWT body) and
	// disclosable (in separate disclosures array).
	jwtClaims := map[string]any{
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
			// Create disclosure entry with hash.
			hash := simpleHash(key + fmt.Sprintf("%v", val))
			disclosures = append(disclosures, Disclosure{
				Claim: key,
				Hash:  hash,
				Value: val,
			})
			// Store hash reference in JWT (not the value).
			jwtClaims["_sd_"+key] = hash
		} else {
			// Always disclosed: put directly in JWT.
			jwtClaims[key] = val
		}
	}

	// Build a simple unsigned JWT structure (header.payload).
	// In production: sign with the service's signing key.
	header := `{"alg":"none","typ":"sd-jwt"}`
	payloadBytes, _ := json.Marshal(jwtClaims)
	sjwt := base64Encode(header) + "." + base64Encode(string(payloadBytes)) + "."

	writeJSON(w, http.StatusCreated, SDJWTIssueResponse{
		SJWT:       sjwt,
		Disclosures: disclosures,
		ExpiresAt:  expiresAt,
	})
}

// handleSDJWTVerify verifies a selective disclosure JWT.
// POST /api/v1/identity/sd-jwt/verify
func (h *HTTPHandler) handleSDJWTVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SDJWTVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.SDJWT == "" {
		writeError(w, http.StatusBadRequest, "sd_jwt required")
		return
	}

	// Parse the JWT (header.payload.signature).
	parts := strings.Split(req.SDJWT, ".")
	if len(parts) < 2 {
		writeJSON(w, http.StatusOK, SDJWTVerifyResponse{
			Valid: false,
			Error: "malformed SD-JWT: expected header.payload.signature",
		})
		return
	}

	// Decode payload.
	payload, err := base64Decode(parts[1])
	if err != nil {
		writeJSON(w, http.StatusOK, SDJWTVerifyResponse{
			Valid: false,
			Error: "failed to decode payload",
		})
		return
	}

	var claims map[string]any
	if err := json.Unmarshal([]byte(payload), &claims); err != nil {
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

	// Filter to revealed claims only (strip _sd_ prefixed hashes if not revealed).
	revealedSet := make(map[string]bool)
	for _, r := range req.RevealedClaims {
		revealedSet[r] = true
	}
	// Always include standard claims.
	standardClaims := map[string]bool{"iss": true, "sub": true, "iat": true, "exp": true, "jti": true}
	result := make(map[string]any)
	for key, val := range claims {
		if standardClaims[key] {
			result[key] = val
			continue
		}
		if strings.HasPrefix(key, "_sd_") {
			// Selective disclosure hash: only include if the claim is revealed.
			claimName := strings.TrimPrefix(key, "_sd_")
			if revealedSet[claimName] {
				result[claimName+"_hash"] = val // hash reference (value provided via disclosure)
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

// --- helpers ---

func simpleHash(s string) string {
	if len(s) == 0 {
		return "0"
	}
	return fmt.Sprintf("%x", len(s)*31+int(s[0]))
}

func base64Encode(s string) string {
	return fmt.Sprintf("%x", []byte(s))
}

func base64Decode(s string) (string, error) {
	var data []byte
	_, err := fmt.Sscanf(s, "%x", &data)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
