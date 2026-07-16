package crypto

import (
	"crypto/elliptic"

	"github.com/emmansun/gmsm/sm2/sm2ec"
)

// isSM2Curve reports whether the given elliptic curve is the SM2 P-256 curve
// (GB/T 32918.1-2016). SM2 and NIST P-256 have different field primes and
// coefficients, so comparing the B coefficient is sufficient.
func isSM2Curve(curve elliptic.Curve) bool {
	if curve == nil {
		return false
	}
	return curve.Params().B.Cmp(sm2ec.P256().Params().B) == 0
}
