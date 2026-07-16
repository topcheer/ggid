package crypto

// supportedAlgs is the whitelist of JWS algorithms that GGID services may
// issue or accept. It covers RSA, RSA-PSS, ECDSA, EdDSA, and Chinese GM SM2.
var supportedAlgs = []string{
	"RS256", "RS384", "RS512",
	"PS256", "PS384", "PS512",
	"ES256", "ES384", "ES512",
	"EdDSA",
	SM2SM3Alg,
}

// SupportedAlgs returns the whitelist of acceptable JWS algorithm identifiers,
// suitable for jwt.WithValidMethods.
func SupportedAlgs() []string {
	out := make([]string, len(supportedAlgs))
	copy(out, supportedAlgs)
	return out
}

// IsSupportedAlg reports whether the given JWS alg identifier is in the GGID
// whitelist. Use this in JWT keyfunc callbacks to prevent alg-confusion attacks
// while allowing all algorithms the platform can legitimately issue.
func IsSupportedAlg(alg string) bool {
	for _, a := range supportedAlgs {
		if a == alg {
			return true
		}
	}
	return false
}
