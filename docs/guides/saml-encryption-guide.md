# SAML Encryption Guide

This guide covers XML Encryption in SAML, EncryptedAssertion/EncryptedKey handling, key transport algorithms, data encryption, decryption flow, certificate rotation impact, and GGID's implementation.

## Overview

SAML encryption protects assertion content from interception. While SAML responses are typically signed (for integrity), encryption provides confidentiality — only the intended recipient (SP) can read the assertion content.

## XML Encryption Syntax

### EncryptedAssertion

```xml
<saml:EncryptedAssertion>
  <xenc:EncryptedData Type="http://www.w3.org/2001/04/xmlenc#Element"
                      Id="_encrypted_assertion">
    <xenc:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes256-gcm"/>
    <ds:KeyInfo>
      <xenc:EncryptedKey Id="_encrypted_key">
        <xenc:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p">
          <ds:DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"/>
        </xenc:EncryptionMethod>
        <ds:KeyInfo>
          <ds:X509Data>
            <ds:X509Certificate>...</ds:X509Certificate>
          </ds:X509Data>
        </ds:KeyInfo>
        <xenc:CipherData>
          <xenc:CipherValue>base64-encoded-encrypted-key</xenc:CipherValue>
        </xenc:CipherData>
      </xenc:EncryptedKey>
    </ds:KeyInfo>
    <xenc:CipherData>
      <xenc:CipherValue>base64-encoded-encrypted-assertion</xenc:CipherValue>
    </xenc:CipherData>
  </xenc:EncryptedData>
</saml:EncryptedAssertion>
```

### Structure Breakdown

| Element | Purpose |
|---|---|
| `EncryptedAssertion` | Container for encrypted SAML assertion |
| `EncryptedData` | The encrypted assertion content |
| `EncryptionMethod` (data) | Data encryption algorithm (AES-GCM) |
| `EncryptedKey` | The encrypted symmetric key used for data encryption |
| `EncryptionMethod` (key) | Key transport algorithm (RSA-OAEP) |
| `X509Certificate` | SP's public key certificate for key encryption |
| `CipherValue` (key) | RSA-encrypted symmetric key |
| `CipherValue` (data) | AES-encrypted assertion |

## Key Transport Algorithms

### RSA-OAEP (Recommended)

```
Algorithm: http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p
```

RSA-OAEP (Optimal Asymmetric Encryption Padding) encrypts the symmetric key using the SP's RSA public key.

| Property | Value |
|---|---|
| Key size | 2048 or 3072 bits |
| Padding | OAEP with MGF1 |
| Hash | SHA-1 (standard) or SHA-256 (xmlenc11) |
| Use | Encrypt symmetric key for recipient |

```go
func decryptKey(encryptedKey []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
    // RSA-OAEP decryption
    hash := sha1.New()  // or sha256.New() for xmlenc11
    sessionKey, err := rsa.DecryptOAEP(privateKey, hash, encryptedKey, nil)
    if err != nil {
        return nil, fmt.Errorf("RSA-OAEP decrypt: %w", err)
    }
    return sessionKey, nil
}
```

### RSA-v1.5 (Deprecated)

```
Algorithm: http://www.w3.org/2001/04/xmlenc#rsa-1_5
```

**Do not use.** Vulnerable to padding oracle attacks (Bleichenbacher). GGID rejects this algorithm.

### ECDH (Advanced)

```
Algorithm: http://www.w3.org/2009/xmlenc11#ecdh-es
```

Key agreement using Elliptic Curve Diffie-Hellman. Supported but less common in SAML deployments.

## Data Encryption Algorithms

### AES-256-GCM (Recommended)

```
Algorithm: http://www.w3.org/2009/xmlenc11#aes256-gcm
```

| Property | Value |
|---|---|
| Key size | 256 bits |
| Mode | GCM (authenticated encryption) |
| IV | 96 bits |
| Tag | 128 bits |
| Use | Encrypt assertion content |

### AES-128-CBC (Legacy)

```
Algorithm: http://www.w3.org/2001/04/xmlenc#aes128-cbc
```

| Property | Value |
|---|---|
| Key size | 128 bits |
| Mode | CBC (requires separate signature) |
| IV | 128 bits |
| Status | Legacy, not recommended |

### Triple-DES (Deprecated)

```
Algorithm: http://www.w3.org/2001/04/xmlenc#tripledes-cbc
```

**Do not use.** Deprecated due to small block size and security concerns.

## Decryption Flow

### Step-by-Step

```
1. Parse SAML response → find EncryptedAssertion
2. Extract EncryptedData → get EncryptionMethod (data algorithm)
3. Extract EncryptedKey → get EncryptionMethod (key transport algorithm)
4. Decrypt EncryptedKey using SP private key → obtain symmetric session key
5. Decrypt EncryptedData using session key → obtain plaintext assertion XML
6. Parse decrypted assertion → verify signature if present
7. Process assertion claims
```

### Implementation

```go
func (s *SAMLService) DecryptAssertion(
    encryptedAssertion *xmlenc.EncryptedAssertion,
    spPrivateKey *rsa.PrivateKey,
) (*saml.Assertion, error) {

    // Step 1: Extract encrypted key
    encKey := encryptedAssertion.EncryptedData.KeyInfo.EncryptedKey
    if encKey == nil {
        return nil, ErrMissingEncryptedKey
    }

    // Step 2: Verify key transport algorithm
    keyAlg := encKey.EncryptionMethod.Algorithm
    if keyAlg != "http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p" &&
       keyAlg != "http://www.w3.org/2009/xmlenc11#rsa-oaep" {
        return nil, fmt.Errorf("unsupported key transport: %s", keyAlg)
    }

    // Step 3: Decrypt symmetric key
    cipherKey, err := base64.StdEncoding.DecodeString(encKey.CipherData.CipherValue)
    if err != nil {
        return nil, fmt.Errorf("decode encrypted key: %w", err)
    }

    sessionKey, err := rsa.DecryptOAEP(spPrivateKey, sha1.New(), cipherKey, nil)
    if err != nil {
        return nil, fmt.Errorf("decrypt session key: %w", err)
    }

    // Step 4: Verify data encryption algorithm
    dataAlg := encryptedAssertion.EncryptedData.EncryptionMethod.Algorithm
    if dataAlg != "http://www.w3.org/2009/xmlenc11#aes256-gcm" &&
       dataAlg != "http://www.w3.org/2001/04/xmlenc#aes256-cbc" {
        return nil, fmt.Errorf("unsupported data encryption: %s", dataAlg)
    }

    // Step 5: Decrypt assertion
    cipherData, err := base64.StdEncoding.DecodeString(
        encryptedAssertion.EncryptedData.CipherData.CipherValue)
    if err != nil {
        return nil, fmt.Errorf("decode encrypted data: %w", err)
    }

    var plaintext []byte
    if strings.Contains(dataAlg, "gcm") {
        plaintext, err = decryptAESGCM(sessionKey, cipherData)
    } else {
        plaintext, err = decryptAESCBC(sessionKey, cipherData)
    }
    if err != nil {
        return nil, fmt.Errorf("decrypt assertion: %w", err)
    }

    // Step 6: Parse decrypted assertion
    assertion := &saml.Assertion{}
    if err := xml.Unmarshal(plaintext, assertion); err != nil {
        return nil, fmt.Errorf("parse decrypted assertion: %w", err)
    }

    // Step 7: Verify signature if present
    if assertion.Signature != nil {
        if err := verifyAssertionSignature(assertion, s.idpCertificate); err != nil {
            return nil, fmt.Errorf("signature verification: %w", err)
        }
    }

    return assertion, nil
}

func decryptAESGCM(key, ciphertext []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return nil, ErrCiphertextTooShort
    }
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    return gcm.Open(nil, nonce, ciphertext, nil)
}
```

## Encryption Flow (SP → IdP request)

### SP Metadata Declaration

The SP advertises its encryption capabilities in its metadata:

```xml
<SPSSODescriptor>
  <KeyDescriptor use="encryption">
    <ds:KeyInfo>
      <ds:X509Data>
        <ds:X509Certificate>SP's encryption certificate</ds:X509Certificate>
      </ds:X509Data>
    </ds:KeyInfo>
    <EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes256-gcm"/>
    <EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p"/>
  </KeyDescriptor>
</SPSSODescriptor>
```

The IdP reads this metadata to know:
1. Which certificate to use for encrypting the symmetric key
2. Which encryption algorithms the SP supports

## Certificate Rotation Impact

### Impact of SP Certificate Rotation

When the SP rotates its encryption certificate:

1. **Before rotation**: IdP encrypts with old cert, SP decrypts with old private key
2. **Transition**: IdP may use either old or new cert
3. **After rotation**: IdP encrypts with new cert, SP decrypts with new private key

### Transition Strategy

```yaml
saml:
  encryption:
    key_rotation:
      overlap_period: 7d  # Both certs valid during overlap
      sp_metadata_update: true  # Auto-publish new metadata
      idp_notification: true  # Notify IdP of cert change
```

### During Overlap Period

SP must be able to decrypt with BOTH old and new private keys:

```go
func (s *SAMLService) DecryptAssertion(encAssertion *EncryptedAssertion) (*Assertion, error) {
    // Try new key first
    assertion, err := tryDecrypt(encAssertion, s.newPrivateKey)
    if err == nil {
        return assertion, nil
    }

    // Fall back to old key during rotation
    if s.oldPrivateKey != nil {
        assertion, err = tryDecrypt(encAssertion, s.oldPrivateKey)
        if err == nil {
            log.Warn("decrypted with old key during rotation overlap")
            return assertion, nil
        }
    }

    return nil, fmt.Errorf("decryption failed with both keys: %w", err)
}
```

### Rotation Steps

1. Generate new key pair
2. Publish new SP metadata with new certificate
3. Keep both old and new private keys available
4. Wait for IdP to pick up new metadata (or notify manually)
5. After overlap period, remove old private key

## Algorithm Security Policy

### Allowed Algorithms

| Type | Algorithm | Status |
|---|---|---|
| Key transport | RSA-OAEP-MGF1P | Allowed |
| Key transport | RSA-OAEP-MGF1P-SHA256 | Allowed (preferred) |
| Key transport | RSA-v1.5 | **Rejected** |
| Data encryption | AES-256-GCM | Allowed (preferred) |
| Data encryption | AES-192-GCM | Allowed |
| Data encryption | AES-128-GCM | Allowed |
| Data encryption | AES-256-CBC | Allowed (legacy) |
| Data encryption | Triple-DES-CBC | **Rejected** |

### Configuration

```yaml
saml:
  encryption:
    require_encrypted_assertions: false  # Optional but recommended
    allowed_key_transport:
      - "http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p"
      - "http://www.w3.org/2009/xmlenc11#rsa-oaep"
    allowed_data_encryption:
      - "http://www.w3.org/2009/xmlenc11#aes256-gcm"
      - "http://www.w3.org/2009/xmlenc11#aes128-gcm"
      - "http://www.w3.org/2001/04/xmlenc#aes256-cbc"
    reject_weak_algorithms: true
    min_rsa_key_size: 2048
```

## GGID Implementation

### Configuration

```yaml
saml:
  encryption:
    enabled: true
    sp_decrypt_key: /etc/ggid/saml/sp-key.pem
    sp_decrypt_cert: /etc/ggid/saml/sp-cert.pem
    preferred_algorithm: "aes256-gcm"
    key_transport: "rsa-oaep"
    require_encrypted: false  # Set true for high-security
```

### Decryption Service

```go
type SAMLEncryptionService struct {
    privateKey  *rsa.PrivateKey
    oldKey      *rsa.PrivateKey  // During rotation
    config      EncryptionConfig
}

func (s *SAMLEncryptionService) HandleEncryptedAssertion(
    response *saml.Response,
) (*saml.Assertion, error) {

    if len(response.EncryptedAssertions) == 0 {
        if s.config.RequireEncrypted {
            return nil, ErrEncryptionRequired
        }
        // Use unencrypted assertion
        if len(response.Assertions) == 0 {
            return nil, ErrNoAssertion
        }
        return response.Assertions[0], nil
    }

    // Decrypt the assertion
    encAssertion := response.EncryptedAssertions[0]
    return s.DecryptAssertion(encAssertion)
}
```

## Best Practices

1. **Use AES-256-GCM** — Authenticated encryption, no separate MAC needed
2. **Use RSA-OAEP** — Never RSA-v1.5 (vulnerable to padding oracle)
3. **Require encryption for high-security** — Don't accept unencrypted assertions
4. **Rotate certificates carefully** — Overlap period prevents decryption failures
5. **Publish metadata promptly** — IdP needs current encryption certificate
6. **Reject weak algorithms** — Triple-DES, RSA-v1.5, small key sizes
7. **Verify after decryption** — Check signature on decrypted assertion
8. **Log encryption details** — Algorithm, key size, cert fingerprint for audit
9. **Handle both encrypted and unencrypted** — For interoperability with IdPs
10. **Test with multiple IdPs** — Different IdPs use different encryption configs
