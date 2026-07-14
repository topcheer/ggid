//go:build pkcs11 && cgo

package crypto

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/miekg/pkcs11"
)

// pkcs11KeyProvider implements KeyProvider using a PKCS#11 token.
type pkcs11KeyProvider struct {
	metadata   KeyMetadata
	ctx        *pkcs11.Ctx
	session    pkcs11.SessionHandle
	privateKey pkcs11.ObjectHandle
	pubKey     crypto.PublicKey
	signer     *pkcs11Signer
}

type pkcs11Signer struct {
	provider *pkcs11KeyProvider
	pubKey   crypto.PublicKey
}

// newPKCS11KeyProvider creates a PKCS#11-backed KeyProvider.
// It can be configured via KeyProviderConfig or environment variables.
func newPKCS11KeyProvider(_ context.Context, cfg PKCS11KeyProviderConfig) (KeyProvider, error) {
	libPath := cfg.LibPath
	if v := os.Getenv("GGID_PKCS11_LIB"); v != "" {
		libPath = v
	}
	if libPath == "" {
		return nil, fmt.Errorf("%w: pkcs11 library path required", ErrKeyProviderConfig)
	}

	pin := cfg.PIN
	if v := os.Getenv("GGID_PKCS11_PIN"); v != "" {
		pin = v
	}

	keyLabel := cfg.KeyLabel
	if v := os.Getenv("GGID_PKCS11_KEY_LABEL"); v != "" {
		keyLabel = v
	}
	if keyLabel == "" {
		return nil, fmt.Errorf("%w: pkcs11 key label required", ErrKeyProviderConfig)
	}

	ctx := pkcs11.New(libPath)
	if ctx == nil {
		return nil, fmt.Errorf("failed to load PKCS#11 library %s", libPath)
	}

	if err := ctx.Initialize(); err != nil {
		return nil, fmt.Errorf("pkcs11 initialize: %w", err)
	}

	slots, err := ctx.GetSlotList(true)
	if err != nil {
		_ = ctx.Finalize()
		return nil, fmt.Errorf("pkcs11 get slot list: %w", err)
	}
	if len(slots) == 0 {
		_ = ctx.Finalize()
		return nil, fmt.Errorf("%w: no PKCS#11 slots with token present", ErrKeyProviderConfig)
	}

	var slot uint
	if v := os.Getenv("GGID_PKCS11_SLOT"); v != "" {
		var slotID uint64
		if _, err := fmt.Sscanf(v, "%d", &slotID); err != nil {
			_ = ctx.Finalize()
			return nil, fmt.Errorf("%w: invalid GGID_PKCS11_SLOT", ErrKeyProviderConfig)
		}
		found := false
		for _, s := range slots {
			if uint64(s) == slotID {
				slot = s
				found = true
				break
			}
		}
		if !found {
			_ = ctx.Finalize()
			return nil, fmt.Errorf("%w: GGID_PKCS11_SLOT not found", ErrKeyProviderConfig)
		}
	} else if cfg.SlotID != 0 {
		found := false
		for _, s := range slots {
			if uint(s) == cfg.SlotID {
				slot = s
				found = true
				break
			}
		}
		if !found {
			_ = ctx.Finalize()
			return nil, fmt.Errorf("%w: configured slot_id not found", ErrKeyProviderConfig)
		}
	} else if cfg.SlotLabel != "" {
		found := false
		for _, s := range slots {
			info, err := ctx.GetSlotInfo(s)
			if err != nil {
				continue
			}
			if info.SlotDescription == cfg.SlotLabel {
				slot = s
				found = true
				break
			}
		}
		if !found {
			_ = ctx.Finalize()
			return nil, fmt.Errorf("%w: configured slot_label not found", ErrKeyProviderConfig)
		}
	} else {
		slot = slots[0]
	}

	session, err := ctx.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		_ = ctx.Finalize()
		return nil, fmt.Errorf("pkcs11 open session: %w", err)
	}

	if err := ctx.Login(session, pkcs11.CKU_USER, pin); err != nil {
		_ = ctx.CloseSession(session)
		_ = ctx.Finalize()
		return nil, fmt.Errorf("pkcs11 login: %w", err)
	}

	privateObj, publicObj, err := findKeyPair(ctx, session, keyLabel)
	if err != nil {
		_ = ctx.Logout(session)
		_ = ctx.CloseSession(session)
		_ = ctx.Finalize()
		return nil, fmt.Errorf("pkcs11 find key: %w", err)
	}

	pubKey, algorithm, err := extractPublicKey(ctx, session, publicObj)
	if err != nil {
		_ = ctx.Logout(session)
		_ = ctx.CloseSession(session)
		_ = ctx.Finalize()
		return nil, fmt.Errorf("pkcs11 extract public key: %w", err)
	}

	keyID := cfg.KeyID
	if keyID == "" {
		keyID = keyLabel
	}

	p := &pkcs11KeyProvider{
		metadata: KeyMetadata{
			KeyID:     keyID,
			Algorithm: algorithm,
			Use:       "sig",
		},
		ctx:        ctx,
		session:    session,
		privateKey: privateObj,
		pubKey:     pubKey,
	}
	p.signer = &pkcs11Signer{provider: p, pubKey: pubKey}
	return p, nil
}

func (p *pkcs11KeyProvider) Metadata() KeyMetadata      { return p.metadata }
func (p *pkcs11KeyProvider) Public() crypto.PublicKey { return p.pubKey }
func (p *pkcs11KeyProvider) Signer() crypto.Signer    { return p.signer }

func (p *pkcs11KeyProvider) Close() error {
	_ = p.ctx.Logout(p.session)
	_ = p.ctx.CloseSession(p.session)
	_ = p.ctx.Finalize()
	p.ctx.Destroy()
	return nil
}

func (s *pkcs11Signer) Public() crypto.PublicKey { return s.pubKey }

func (s *pkcs11Signer) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	p := s.provider

	var mech *pkcs11.Mechanism
	switch s.pubKey.(type) {
	case *rsa.PublicKey:
		mech = pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS, nil)
	case *ecdsa.PublicKey:
		mech = pkcs11.NewMechanism(pkcs11.CKM_ECDSA, nil)
	default:
		return nil, errors.New("pkcs11: unsupported key type")
	}

	if err := p.ctx.SignInit(p.session, []*pkcs11.Mechanism{mech}, p.privateKey); err != nil {
		return nil, fmt.Errorf("pkcs11 sign init: %w", err)
	}

	data := digest
	if _, ok := s.pubKey.(*rsa.PublicKey); ok {
		var err error
		data, err = encodeDigestInfo(digest, opts.HashFunc())
		if err != nil {
			return nil, fmt.Errorf("pkcs11 digest info: %w", err)
		}
	}

	sig, err := p.ctx.Sign(p.session, data)
	if err != nil {
		return nil, fmt.Errorf("pkcs11 sign: %w", err)
	}

	if _, ok := s.pubKey.(*ecdsa.PublicKey); ok {
		return encodeECDSASignatureDER(sig, p.pubKey.(*ecdsa.PublicKey).Curve)
	}
	return sig, nil
}

func findKeyPair(ctx *pkcs11.Ctx, session pkcs11.SessionHandle, label string) (privateObj, publicObj pkcs11.ObjectHandle, err error) {
	privateTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
	}
	if err := ctx.FindObjectsInit(session, privateTemplate); err != nil {
		return 0, 0, err
	}
	privateHandles, _, err := ctx.FindObjects(session, 2)
	if err != nil {
		_ = ctx.FindObjectsFinal(session)
		return 0, 0, err
	}
	_ = ctx.FindObjectsFinal(session)
	if len(privateHandles) == 0 {
		return 0, 0, fmt.Errorf("private key with label %q not found", label)
	}
	if len(privateHandles) > 1 {
		return 0, 0, fmt.Errorf("multiple private keys with label %q", label)
	}

	publicTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
	}
	if err := ctx.FindObjectsInit(session, publicTemplate); err != nil {
		return 0, 0, err
	}
	publicHandles, _, err := ctx.FindObjects(session, 2)
	if err != nil {
		_ = ctx.FindObjectsFinal(session)
		return 0, 0, err
	}
	_ = ctx.FindObjectsFinal(session)
	if len(publicHandles) == 0 {
		return 0, 0, fmt.Errorf("public key with label %q not found", label)
	}
	if len(publicHandles) > 1 {
		return 0, 0, fmt.Errorf("multiple public keys with label %q", label)
	}

	return privateHandles[0], publicHandles[0], nil
}

func extractPublicKey(ctx *pkcs11.Ctx, session pkcs11.SessionHandle, obj pkcs11.ObjectHandle) (crypto.PublicKey, KeyAlgorithm, error) {
	attrs := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, 0),
	}
	attrs, err := ctx.GetAttributeValue(session, obj, attrs)
	if err != nil {
		return nil, "", fmt.Errorf("get key type: %w", err)
	}
	keyType := binary.LittleEndian.Uint32(attrs[0].Value)

	switch keyType {
	case pkcs11.CKK_RSA:
		attrs = []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_MODULUS, nil),
			pkcs11.NewAttribute(pkcs11.CKA_PUBLIC_EXPONENT, nil),
		}
		attrs, err = ctx.GetAttributeValue(session, obj, attrs)
		if err != nil {
			return nil, "", fmt.Errorf("get RSA attributes: %w", err)
		}
		modulus := new(big.Int).SetBytes(attrs[0].Value)
		exponent := new(big.Int).SetBytes(attrs[1].Value)
		pub := &rsa.PublicKey{N: modulus, E: int(exponent.Int64())}
		return pub, RS256, nil

	case pkcs11.CKK_EC:
		attrs = []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, nil),
			pkcs11.NewAttribute(pkcs11.CKA_EC_POINT, nil),
		}
		attrs, err = ctx.GetAttributeValue(session, obj, attrs)
		if err != nil {
			return nil, "", fmt.Errorf("get EC attributes: %w", err)
		}
		curve, err := ecParamsToCurve(attrs[0].Value)
		if err != nil {
			return nil, "", err
		}
		pub, err := ecPointToPublicKey(attrs[1].Value, curve)
		if err != nil {
			return nil, "", fmt.Errorf("parse EC point: %w", err)
		}
		var alg KeyAlgorithm
		switch curve {
		case elliptic.P256():
			alg = ES256
		case elliptic.P384():
			alg = ES384
		case elliptic.P521():
			alg = ES512
		default:
			alg = ES256
		}
		return pub, alg, nil

	default:
		return nil, "", fmt.Errorf("unsupported PKCS#11 key type: %d", keyType)
	}
}

func ecParamsToCurve(params []byte) (elliptic.Curve, error) {
	oid := asn1.ObjectIdentifier{}
	if _, err := asn1.Unmarshal(params, &oid); err != nil {
		return nil, fmt.Errorf("parse EC params: %w", err)
	}
	switch {
	case oid.Equal(asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7}):
		return elliptic.P256(), nil
	case oid.Equal(asn1.ObjectIdentifier{1, 3, 132, 0, 34}):
		return elliptic.P384(), nil
	case oid.Equal(asn1.ObjectIdentifier{1, 3, 132, 0, 35}):
		return elliptic.P521(), nil
	default:
		return nil, fmt.Errorf("unsupported EC curve OID: %v", oid)
	}
}

func ecPointToPublicKey(point []byte, curve elliptic.Curve) (*ecdsa.PublicKey, error) {
	// PKCS#11 encodes EC points as OCTET STRING containing the uncompressed point 0x04 || X || Y.
	if len(point) == 0 {
		return nil, errors.New("empty EC point")
	}
	if point[0] == 0x04 {
		point = point[1:]
	}
	byteLen := (curve.Params().BitSize + 7) / 8
	if len(point) != 2*byteLen {
		return nil, fmt.Errorf("invalid EC point length: got %d, want %d", len(point), 2*byteLen)
	}
	x := new(big.Int).SetBytes(point[:byteLen])
	y := new(big.Int).SetBytes(point[byteLen:])
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

func encodeECDSASignatureDER(raw []byte, curve elliptic.Curve) ([]byte, error) {
	byteLen := (curve.Params().BitSize + 7) / 8
	if len(raw) != 2*byteLen {
		return nil, fmt.Errorf("invalid ECDSA signature length: got %d, want %d", len(raw), 2*byteLen)
	}
	r := new(big.Int).SetBytes(raw[:byteLen])
	s := new(big.Int).SetBytes(raw[byteLen:])
	type ecdsaSignature struct {
		R, S *big.Int
	}
	return asn1.Marshal(ecdsaSignature{R: r, S: s})
}

func encodeDigestInfo(digest []byte, hash crypto.Hash) ([]byte, error) {
	var oid asn1.ObjectIdentifier
	switch hash {
	case crypto.SHA256:
		oid = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	case crypto.SHA384:
		oid = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}
	case crypto.SHA512:
		oid = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}
	case crypto.SHA1:
		oid = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
	default:
		return nil, fmt.Errorf("unsupported hash for PKCS#1 v1.5: %v", hash)
	}
	type pkixAlgorithmIdentifier struct {
		Algorithm  asn1.ObjectIdentifier
		Parameters asn1.RawValue `asn1:"optional"`
	}
	type digestInfo struct {
		Algorithm pkixAlgorithmIdentifier
		Digest    []byte
	}
	di := digestInfo{
		Algorithm: pkixAlgorithmIdentifier{Algorithm: oid},
		Digest:    digest,
	}
	return asn1.Marshal(di)
}

// hashDigest is not used directly but documents expected hash inputs for Sign.
func hashDigest(h crypto.Hash) []byte { return nil }

// compile-time interface assertions.
var _ KeyProvider = (*pkcs11KeyProvider)(nil)
var _ crypto.Signer = (*pkcs11Signer)(nil)
var _ crypto.PublicKey = (*rsa.PublicKey)(nil)
