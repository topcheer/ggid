package crypto

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// vaultTransitKeyProvider implements KeyProvider using HashiCorp Vault Transit engine.
// It stores the private key in Vault and delegates signing via HTTP API.
type vaultTransitKeyProvider struct {
	cfg      VaultTransitConfig
	token    string
	http     *http.Client
	metadata KeyMetadata
	pubKey   crypto.PublicKey
}

func newVaultTransitKeyProvider(ctx context.Context, cfg VaultTransitConfig) (*vaultTransitKeyProvider, error) {
	if cfg.Address == "" || cfg.KeyName == "" {
		return nil, fmt.Errorf("%w: address and key_name are required", ErrKeyProviderConfig)
	}

	token := cfg.TokenPath
	if token != "" {
		data, err := readFile(token)
		if err != nil {
			return nil, fmt.Errorf("read vault token: %w", err)
		}
		token = strings.TrimSpace(string(data))
	}

	p := &vaultTransitKeyProvider{
		cfg:   cfg,
		token: token,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Fetch public key from Vault
	pub, alg, err := p.fetchPublicKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("vault transit: fetch public key: %w", err)
	}

	p.pubKey = pub
	keyID := cfg.KeyIDHint
	if keyID == "" {
		keyID = cfg.KeyName
	}
	p.metadata = KeyMetadata{
		KeyID:     keyID,
		Algorithm: alg,
		Use:       "sig",
	}

	return p, nil
}

func (p *vaultTransitKeyProvider) Metadata() KeyMetadata { return p.metadata }
func (p *vaultTransitKeyProvider) Public() crypto.PublicKey { return p.pubKey }
func (p *vaultTransitKeyProvider) Close() error { return nil }

// Signer returns a crypto.Signer that delegates to Vault Transit /sign endpoint.
func (p *vaultTransitKeyProvider) Signer() crypto.Signer {
	return &vaultSigner{provider: p}
}

type vaultSigner struct {
	provider *vaultTransitKeyProvider
}

func (s *vaultSigner) Public() crypto.PublicKey { return s.provider.pubKey }

func (s *vaultSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return s.provider.signViaVault(digest, opts)
}

func (p *vaultTransitKeyProvider) signViaVault(digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	body := map[string]any{
		"input": base64.StdEncoding.EncodeToString(digest),
	}
	hashAlg := p.cfg.Algorithm
	if hashAlg == "" {
		hashAlg = "sha2-256"
	}
	body["hash_algorithm"] = hashAlg

	sig, err := p.vaultPost(context.Background(), "/transit/sign/"+p.cfg.KeyName, body)
	if err != nil {
		return nil, err
	}

	sigStr, ok := sig["signature"].(string)
	if !ok {
		return nil, fmt.Errorf("vault: invalid signature response")
	}
	// Vault returns "vault:v1:base64sig" — strip prefix
	parts := strings.SplitN(sigStr, ":", 3)
	if len(parts) == 3 {
		return base64.StdEncoding.DecodeString(parts[2])
	}
	return base64.StdEncoding.DecodeString(sigStr)
}

func (p *vaultTransitKeyProvider) fetchPublicKey(ctx context.Context) (crypto.PublicKey, KeyAlgorithm, error) {
	resp, err := p.vaultGet(ctx, "/transit/keys/"+p.cfg.KeyName)
	if err != nil {
		return nil, "", err
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("unexpected vault response")
	}

	keyType, _ := data["type"].(string)
	// data.keys has versioned public keys
	keys, ok := data["keys"].(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("no key versions in vault response")
	}

	// Get latest version
	latestVer := data["latest_version"].(json.Number)
	latestKey, ok := keys[string(latestVer)].(map[string]any)
	if !ok {
		// try numeric index
		for _, v := range keys {
			latestKey, ok = v.(map[string]any)
			if ok {
				break
			}
		}
	}
	if latestKey == nil {
		return nil, "", fmt.Errorf("no key version found")
	}

	pubStr, _ := latestKey["public_key"].(string)
	if pubStr == "" {
		return nil, "", fmt.Errorf("no public_key in vault response")
	}

	// Parse based on key type
	alg := KeyAlgorithm("")
	var pub crypto.PublicKey

	switch keyType {
	case "rsa-2048", "rsa-3072", "rsa-4096":
		alg = "RS256"
		pub = parseRSAPublicKeyPEM(pubStr)
	case "ecdsa-p256":
		alg = "ES256"
		pub = parseECDSAPublicKeyPEM(pubStr)
	case "ed25519":
		alg = "EdDSA"
		pub = parseEd25519PublicKeyPEM(pubStr)
	default:
		// Default to RSA
		alg = "RS256"
		pub = parseRSAPublicKeyPEM(pubStr)
	}

	if pub == nil {
		return nil, "", fmt.Errorf("failed to parse public key (type=%s)", keyType)
	}
	return pub, alg, nil
}

func (p *vaultTransitKeyProvider) vaultGet(ctx context.Context, path string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.cfg.Address+"/v1"+path, nil)
	if err != nil {
		return nil, err
	}
	if p.token != "" {
		req.Header.Set("X-Vault-Token", p.token)
	}
	return p.doRequest(req)
}

func (p *vaultTransitKeyProvider) vaultPost(ctx context.Context, path string, body map[string]any) (map[string]any, error) {
	bodyBytes, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.Address+"/v1"+path, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.token != "" {
		req.Header.Set("X-Vault-Token", p.token)
	}
	return p.doRequest(req)
}

func (p *vaultTransitKeyProvider) doRequest(req *http.Request) (map[string]any, error) {
	resp, err := p.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vault API %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode vault response: %w", err)
	}
	return result, nil
}

// Helper to read token file (injectable for testing).
var readFile = func(path string) ([]byte, error) {
	return osReadFile(path)
}

// parseRSAPublicKeyPEM parses a PEM-encoded RSA public key.
func parseRSAPublicKeyPEM(pemStr string) crypto.PublicKey {
	pemStr = strings.TrimSpace(pemStr)
	if !strings.Contains(pemStr, "BEGIN") {
		// Vault returns raw base64 — wrap in PEM
		pemStr = "-----BEGIN PUBLIC KEY-----\n" + pemStr + "\n-----END PUBLIC KEY-----"
	}
	block := decodePEM(pemStr)
	if block == nil {
		return nil
	}
	key, err := parseX509PublicKey(block.Bytes)
	if err != nil {
		return nil
	}
	if rsa, ok := key.(*rsa.PublicKey); ok {
		return rsa
	}
	return nil
}

func parseECDSAPublicKeyPEM(pemStr string) crypto.PublicKey {
	pemStr = strings.TrimSpace(pemStr)
	if !strings.Contains(pemStr, "BEGIN") {
		pemStr = "-----BEGIN PUBLIC KEY-----\n" + pemStr + "\n-----END PUBLIC KEY-----"
	}
	block := decodePEM(pemStr)
	if block == nil {
		return nil
	}
	key, err := parseX509PublicKey(block.Bytes)
	if err != nil {
		return nil
	}
	if ecdsa, ok := key.(*ecdsa.PublicKey); ok {
		return ecdsa
	}
	return nil
}

func parseEd25519PublicKeyPEM(pemStr string) crypto.PublicKey {
	pemStr = strings.TrimSpace(pemStr)
	if !strings.Contains(pemStr, "BEGIN") {
		pemStr = "-----BEGIN PUBLIC KEY-----\n" + pemStr + "\n-----END PUBLIC KEY-----"
	}
	block := decodePEM(pemStr)
	if block == nil {
		return nil
	}
	key, err := parseX509PublicKey(block.Bytes)
	if err != nil {
		return nil
	}
	if ed, ok := key.(ed25519.PublicKey); ok {
		return ed
	}
	return nil
}
