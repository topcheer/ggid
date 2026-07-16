package middleware

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"

	"github.com/emmansun/gmsm/sm2"
	"github.com/golang-jwt/jwt/v5"
)

// sm2sm3Alg is the JWS algorithm identifier for Chinese GM SM2 signature
// with SM3 hash (GB/T 32918.2-2016, default user ID, ASN.1 DER encoding).
const sm2sm3Alg = "SM2SM3"

// signingMethodSM2 implements jwt.SigningMethod for SM2SM3 so that the
// middleware can verify GGID tokens issued with a Chinese GM SM2 key.
type signingMethodSM2 struct{}

func init() {
	jwt.RegisterSigningMethod(sm2sm3Alg, func() jwt.SigningMethod {
		return &signingMethodSM2{}
	})
}

func (m *signingMethodSM2) Sign(signingString string, key interface{}) ([]byte, error) {
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, jwt.ErrInvalidKeyType
	}
	sig, err := signer.Sign(rand.Reader, []byte(signingString), sm2.NewSM2SignerOption(true, nil))
	if err != nil {
		return nil, err
	}
	return sig, nil
}

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

func (m *signingMethodSM2) Alg() string {
	return sm2sm3Alg
}
