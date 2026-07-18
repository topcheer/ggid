package server

import (
	"strings"
	"testing"
)

func TestKeyRotationRepo_NilPool(t *testing.T) {
	repo := newKeyRotationRepo(nil)
	keys, err := repo.ListActiveKeys(nil)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if len(keys) != 0 { t.Error("nil pool should return empty") }
	history, err := repo.ListHistory(nil, 10)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if len(history) != 0 { t.Error("nil pool should return empty") }
}

func TestKeyRotationRepo_RotateNilPool(t *testing.T) {
	repo := newKeyRotationRepo(nil)
	entry, err := repo.Rotate(nil, "jwt_signing", 7)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if entry.KeyType != "jwt_signing" { t.Error("type mismatch") }
	if entry.Status != "active" { t.Error("should be active") }
	if entry.NewKeyID == "" { t.Error("should have new key ID") }
}

func TestKeyRotationRepo_ExpireGraceNilPool(t *testing.T) {
	repo := newKeyRotationRepo(nil)
	count, err := repo.ExpireGrace(nil)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if count != 0 { t.Error("nil pool should expire 0") }
}

func TestGenerateECDSAKeyPair(t *testing.T) {
	priv, pub, err := GenerateECDSAKeyPair()
	if err != nil { t.Fatalf("key generation should not fail: %v", err) }
	if !strings.Contains(priv, "EC PRIVATE KEY") { t.Error("private key PEM should contain EC PRIVATE KEY") }
	if !strings.Contains(pub, "PUBLIC KEY") { t.Error("public key PEM should contain PUBLIC KEY") }
}

func TestKeyRotationEntry_Struct(t *testing.T) {
	e := KeyRotationEntry{KeyType: "webhook_hmac", Status: "active"}
	if e.KeyType != "webhook_hmac" { t.Error("type mismatch") }
}
