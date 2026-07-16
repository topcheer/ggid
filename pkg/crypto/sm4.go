package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"github.com/emmansun/gmsm/sm4"
)

// SM4Encrypt encrypts plaintext using SM4-GCM with the given key.
// The key is normalized to 16 bytes via SM3 hashing (any input length accepted).
// Output format: nonce || ciphertext || tag (same layout as AESEncrypt).
func SM4Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := sm4.NewCipher(sm4Key(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// SM4Decrypt decrypts ciphertext produced by SM4Encrypt (SM4-GCM).
func SM4Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := sm4.NewCipher(sm4Key(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// sm4Key normalizes an arbitrary-length key to the 16-byte SM4 key size
// using SHA-256 (first 16 bytes), matching the hashKey convention used for AES.
func sm4Key(key []byte) []byte {
	if len(key) == sm4.BlockSize {
		return key
	}
	return hashKey(key)[:sm4.BlockSize]
}
