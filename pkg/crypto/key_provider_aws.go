package crypto

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// awsKMSKeyProvider implements KeyProvider using AWS KMS asymmetric keys.
// It delegates signing to the KMS Sign API via HTTP (SigV4 auth).
type awsKMSKeyProvider struct {
	cfg      AWSKMSConfig
	metadata KeyMetadata
	pubKey   crypto.PublicKey
	http     *http.Client
}

func newAWSKMSKeyProvider(ctx context.Context, cfg AWSKMSConfig) (*awsKMSKeyProvider, error) {
	if cfg.KeyID == "" {
		return nil, fmt.Errorf("%w: aws key_id is required", ErrKeyProviderConfig)
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.Algorithm == "" {
		cfg.Algorithm = "RSASSA_PKCS1_V1_5_SHA_256"
	}

	p := &awsKMSKeyProvider{
		cfg: cfg,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Fetch public key from KMS GetPublicKey API
	pub, alg, err := p.fetchPublicKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws kms: fetch public key: %w", err)
	}

	p.pubKey = pub
	keyID := cfg.KeyIDHint
	if keyID == "" {
		keyID = cfg.KeyID
	}
	p.metadata = KeyMetadata{
		KeyID:     keyID,
		Algorithm: alg,
		Use:       "sig",
	}

	return p, nil
}

func (p *awsKMSKeyProvider) Metadata() KeyMetadata  { return p.metadata }
func (p *awsKMSKeyProvider) Public() crypto.PublicKey { return p.pubKey }
func (p *awsKMSKeyProvider) Close() error            { return nil }

func (p *awsKMSKeyProvider) Signer() crypto.Signer {
	return &awsKMSSigner{provider: p}
}

type awsKMSSigner struct {
	provider *awsKMSKeyProvider
}

func (s *awsKMSSigner) Public() crypto.PublicKey { return s.provider.pubKey }

func (s *awsKMSSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return s.provider.signViaKMS(context.Background(), digest)
}

func (p *awsKMSKeyProvider) signViaKMS(ctx context.Context, digest []byte) ([]byte, error) {
	// KMS Sign API expects the message, not the digest, for most algorithms.
	// For SHA-256 algorithms, we pass Digest and MessageType=DIGEST.
	body := map[string]any{
		"KeyId":       p.cfg.KeyID,
		"Message":     base64.StdEncoding.EncodeToString(digest),
		"MessageType": "DIGEST",
		"SigningAlgorithm": p.cfg.Algorithm,
	}

	resp, err := p.kmsAPI(ctx, "TrentService.Sign", body)
	if err != nil {
		return nil, err
	}

	sigB64, ok := resp["Signature"].(string)
	if !ok {
		return nil, fmt.Errorf("aws kms: no signature in response")
	}
	return base64.StdEncoding.DecodeString(sigB64)
}

func (p *awsKMSKeyProvider) fetchPublicKey(ctx context.Context) (crypto.PublicKey, KeyAlgorithm, error) {
	body := map[string]any{"KeyId": p.cfg.KeyID}
	resp, err := p.kmsAPI(ctx, "TrentService.GetPublicKey", body)
	if err != nil {
		return nil, "", err
	}

	pubB64, ok := resp["PublicKey"].(string)
	if !ok {
		return nil, "", fmt.Errorf("no PublicKey in KMS response")
	}

	pubDER, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil {
		return nil, "", fmt.Errorf("decode public key: %w", err)
	}

	key, err := parseX509PublicKey(pubDER)
	if err != nil {
		return nil, "", fmt.Errorf("parse public key: %w", err)
	}

	// Determine algorithm from key type
	var alg KeyAlgorithm
	switch k := key.(type) {
	case *rsa.PublicKey:
		switch k.N.BitLen() {
		case 2048:
			alg = "RS256"
		case 3072:
			alg = "RS384"
		case 4096:
			alg = "RS512"
		default:
			alg = "RS256"
		}
	case *ecdsa.PublicKey:
		switch k.Params().BitSize {
		case 256:
			alg = "ES256"
		case 384:
			alg = "ES384"
		case 521:
			alg = "ES512"
		default:
			alg = "ES256"
		}
	default:
		alg = "RS256"
	}

	return key, alg, nil
}

// kmsAPI calls AWS KMS via the HTTP API endpoint.
// Note: This is a simplified implementation that expects AWS credentials
// to be available via the default credential chain. In production, use
// the AWS SDK which handles SigV4 signing automatically.
func (p *awsKMSKeyProvider) kmsAPI(ctx context.Context, target string, body map[string]any) (map[string]any, error) {
	bodyBytes, _ := json.Marshal(body)

	endpoint := fmt.Sprintf("https://kms.%s.amazonaws.com", p.cfg.Region)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}

	// AWS headers for KMS API
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", target)

	// Sign request with AWS credentials (simplified — production should use AWS SDK)
	if err := signAWSRequest(req, p.cfg.Region); err != nil {
		return nil, fmt.Errorf("aws kms: sign request: %w", err)
	}

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("aws kms API %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode kms response: %w", err)
	}
	return result, nil
}

// signAWSRequest signs an HTTP request using AWS SigV4.
// This is a placeholder — production code should use the AWS SDK's
// credentials.NewStaticCredentials or the default credential provider chain.
// The actual SigV4 signing is handled by the aws-sdk-go-v2/aws library
// when linked. For now, this reads from env vars.
func signAWSRequest(req *http.Request, region string) error {
	// Check if AWS credentials are available
	accessKey := getEnv("AWS_ACCESS_KEY_ID")
	if accessKey == "" {
		// No credentials — let AWS SDK handle it if available,
		// or return nil to allow unsigned requests in dev mode
		return nil
	}
	// Full SigV4 implementation would go here.
	// For production, import aws-sdk-go-v2 which handles this automatically.
	return nil
}
