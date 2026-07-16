package middleware

// supportedAlgs is the whitelist of JWS algorithms that GGID may issue.
// Kept in sync with github.com/ggid/ggid/sdk/go (IsSupportedAlg).
var supportedAlgs = []string{
	"RS256", "RS384", "RS512",
	"PS256", "PS384", "PS512",
	"ES256", "ES384", "ES512",
	"EdDSA",
	"SM2SM3",
}

// isSupportedAlg reports whether the given JWS alg identifier is acceptable
// for tokens issued by GGID.
func isSupportedAlg(alg string) bool {
	for _, a := range supportedAlgs {
		if a == alg {
			return true
		}
	}
	return false
}
