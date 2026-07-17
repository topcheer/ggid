package crypto

import (
	"context"
	"testing"
)

func TestDataKeyProvider_GenerateAndDecrypt(t *testing.T) {
	provider := NewEnvelopeEncryptionProvider([]byte("test-kek-material-32-bytes!!"))
	ctx := context.Background()

	plaintextDEK, encryptedDEK, err := provider.GenerateDataKey(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("GenerateDataKey failed: %v", err)
	}
	if len(plaintextDEK) != 32 {
		t.Fatalf("expected 32-byte DEK, got %d", len(plaintextDEK))
	}
	if len(encryptedDEK) == 0 {
		t.Fatal("expected non-empty encrypted DEK")
	}

	// Decrypt should return the same DEK.
	decryptedDEK, err := provider.DecryptDataKey(ctx, encryptedDEK)
	if err != nil {
		t.Fatalf("DecryptDataKey failed: %v", err)
	}
	if string(decryptedDEK) != string(plaintextDEK) {
		t.Fatal("decrypted DEK does not match original")
	}
}

func TestEnvelopeEncryption_EncryptDecryptField(t *testing.T) {
	provider := NewEnvelopeEncryptionProvider([]byte("my-kek-secret-key-material"))
	ctx := context.Background()
	plaintext := []byte("sensitive customer data")

	ciphertext, err := provider.EncryptField(ctx, "tenant-1", plaintext)
	if err != nil {
		t.Fatalf("EncryptField failed: %v", err)
	}
	if ciphertext == "" {
		t.Fatal("expected non-empty ciphertext")
	}
	if ciphertext == string(plaintext) {
		t.Fatal("ciphertext should not equal plaintext")
	}

	decrypted, err := provider.DecryptField(ctx, ciphertext)
	if err != nil {
		t.Fatalf("DecryptField failed: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted text mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEnvelopeEncryption_SM4Variant(t *testing.T) {
	provider := NewEnvelopeEncryptionProvider([]byte("china-compliance-kek")).WithSM4()
	ctx := context.Background()
	plaintext := []byte("国密合规数据加密")

	ciphertext, err := provider.EncryptField(ctx, "tenant-cn", plaintext)
	if err != nil {
		t.Fatalf("SM4 EncryptField failed: %v", err)
	}

	decrypted, err := provider.DecryptField(ctx, ciphertext)
	if err != nil {
		t.Fatalf("SM4 DecryptField failed: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("SM4 decrypted mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEnvelopeEncryption_DifferentCiphertexts(t *testing.T) {
	provider := NewEnvelopeEncryptionProvider([]byte("deterministic-test-key"))
	ctx := context.Background()

	ct1, _ := provider.EncryptField(ctx, "tenant-1", []byte("same data"))
	ct2, _ := provider.EncryptField(ctx, "tenant-1", []byte("same data"))

	if ct1 == ct2 {
		t.Fatal("same plaintext should produce different ciphertexts (random DEK + nonce)")
	}
}

func TestEnvelopeEncryption_WrongKEK(t *testing.T) {
	provider1 := NewEnvelopeEncryptionProvider([]byte("correct-kek-material"))
	provider2 := NewEnvelopeEncryptionProvider([]byte("wrong-kek-material!!!"))
	ctx := context.Background()

	ct, err := provider1.EncryptField(ctx, "tenant-1", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = provider2.DecryptField(ctx, ct)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong KEK")
	}
}
