package crypto

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/emmansun/gmsm/sm2"
	"github.com/emmansun/gmsm/smx509"
)

// sm2KeyProvider implements KeyProvider using a Chinese GM SM2 key pair.
// Keys are loaded from PEM files (PKCS#8 private / PKIX public, SM2 OID)
// or generated ephemeral for development when Generate is true.
type sm2KeyProvider struct {
	metadata KeyMetadata
	priv     *sm2.PrivateKey
}

func newSM2KeyProvider(cfg SM2KeyProviderConfig) (*sm2KeyProvider, error) {
	var priv *sm2.PrivateKey

	if cfg.PrivateKeyPath != "" {
		privPEM, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read SM2 private key: %w", err)
		}
		block, _ := pem.Decode(privPEM)
		if block == nil {
			return nil, fmt.Errorf("%w: failed to decode PEM SM2 private key", ErrKeyProviderConfig)
		}
		keyAny, err := smx509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: parse SM2 private key: %v", ErrKeyProviderConfig, err)
		}
		var ok bool
		priv, ok = keyAny.(*sm2.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("%w: key is not SM2", ErrKeyProviderConfig)
		}
	} else if cfg.Generate {
		var err error
		priv, err = sm2.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate SM2 key: %w", err)
		}
	} else {
		return nil, fmt.Errorf("%w: sm2 provider requires private_key_path or generate=true", ErrKeyProviderConfig)
	}

	keyID := cfg.KeyID
	if keyID == "" {
		keyID = "sm2-signing-key"
	}

	return &sm2KeyProvider{
		metadata: KeyMetadata{KeyID: keyID, Algorithm: SM2SM3, Use: "sig"},
		priv:     priv,
	}, nil
}

func (p *sm2KeyProvider) Metadata() KeyMetadata   { return p.metadata }
func (p *sm2KeyProvider) Public() crypto.PublicKey { return p.priv.Public() }
func (p *sm2KeyProvider) Signer() crypto.Signer    { return p.priv }
func (p *sm2KeyProvider) Close() error             { return nil }

// MarshalSM2PrivateKeyPEM encodes an SM2 private key as PKCS#8 PEM (SM2 OID).
// Useful for generating and persisting keys via CLI tooling.
func MarshalSM2PrivateKeyPEM(priv *sm2.PrivateKey) ([]byte, error) {
	der, err := smx509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal SM2 private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

// MarshalSM2PublicKeyPEM encodes an SM2 public key as PKIX PEM.
func MarshalSM2PublicKeyPEM(pub *ecdsa.PublicKey) ([]byte, error) {
	der, err := smx509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("marshal SM2 public key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

// GenerateSM2KeyPair creates a fresh SM2 key pair (used by tooling and tests).
func GenerateSM2KeyPair() (*sm2.PrivateKey, error) {
	return sm2.GenerateKey(rand.Reader)
}
