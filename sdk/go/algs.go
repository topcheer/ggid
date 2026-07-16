package ggid

// supportedAlgs is the whitelist of JWS algorithms that GGID may issue.
// Kept in sync with pkg/crypto.SupportedAlgs in the main module.
var supportedAlgs = []string{
	"RS256", "RS384", "RS512",
	"PS256", "PS384", "PS512",
	"ES256", "ES384", "ES512",
	"EdDSA",
	"SM2SM3",
}

// IsSupportedAlg reports whether the given JWS alg identifier is acceptable
// for tokens issued by GGID. Use in JWT keyfunc callbacks to prevent
// alg-confusion attacks while allowing all platform algorithms.
func IsSupportedAlg(alg string) bool {
	for _, a := range supportedAlgs {
		if a == alg {
			return true
		}
	}
	return false
}
