package crypto

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"

	"github.com/emmansun/gmsm/sm2"
	"github.com/golang-jwt/jwt/v5"
)

// SM2SM3Alg is the JWS algorithm identifier for SM2 signature with SM3 hash.
// The signature is computed over the full signing input following GB/T 32918.2-2016
// (SM3 hash of ZA || M with the default user ID), encoded as ASN.1 DER.
const SM2SM3Alg = "SM2SM3"

// signingMethodSM2 implements jwt.SigningMethod for the SM2SM3 algorithm.
type signingMethodSM2 struct{}

// SigningMethodSM2 is the singleton JWT signing method for alg "SM2SM3".
var SigningMethodSM2 jwt.SigningMethod = &signingMethodSM2{}

func init() {
	jwt.RegisterSigningMethod(SM2SM3Alg, func() jwt.SigningMethod {
		return SigningMethodSM2
	})
}

// Sign signs the signing string with an SM2 private key.
// The key may be a *sm2.PrivateKey or any crypto.Signer holding an SM2 key
// (e.g. the Signer returned by an SM2 KeyProvider).
func (m *signingMethodSM2) Sign(signingString string, key interface{}) ([]byte, error) {
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, jwt.ErrInvalidKeyType
	}
	// ForceGMSign=true: treat signingString as the raw message and run the full
	// GB/T 32918.2-2016 process (SM3 over ZA || M) with the default user ID.
	sig, err := signer.Sign(rand.Reader, []byte(signingString), sm2.NewSM2SignerOption(true, nil))
	if err != nil {
		return nil, err
	}
	return sig, nil
}

// Verify verifies an SM2SM3 signature against an SM2 public key.
// In gmsm the SM2 public key is a plain *ecdsa.PublicKey on the SM2 curve.
func (m *signingMethodSM2) Verify(signingString string, sig []byte, key interface{}) error {
	pub, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return jwt.ErrInvalidKeyType
	}
	if !sm2.VerifyASN1WithSM2(pub, nil, []byte(signingString), sig) {
		return jwt.ErrSignatureInvalid
	}
	return nil
}

// Alg returns the JWS algorithm identifier.
func (m *signingMethodSM2) Alg() string {
	return SM2SM3Alg
}
