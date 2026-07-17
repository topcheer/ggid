package server

import (
	"context"
	"testing"
)

func TestSecretsProvider_Env(t *testing.T) {
	provider := NewSecretsProvider()
	val, err := provider.GetSecret(context.Background(), "env://PATH")
	if err != nil { t.Fatalf("env resolution should not error: %v", err) }
	if val == "" { t.Error("PATH env var should be non-empty") }
}

func TestSecretsProvider_PlainValue(t *testing.T) {
	provider := NewSecretsProvider()
	val, err := provider.GetSecret(context.Background(), "plain-secret-value")
	if err != nil { t.Fatalf("plain value should not error: %v", err) }
	if val != "plain-secret-value" { t.Error("plain value should pass through") }
}

func TestSecretsProvider_VaultNotConfigured(t *testing.T) {
	provider := NewSecretsProvider()
	_, err := provider.GetSecret(context.Background(), "vault://secret/data/myapp")
	if err == nil { t.Error("vault without config should error") }
}

func TestSecretRepo_NilPool(t *testing.T) {
	repo := newSecretRepo(nil)
	refs, err := repo.List(nil)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if len(refs) != 0 { t.Error("nil pool should return empty") }
	health := repo.CheckHealth(nil)
	if !health["env"].(bool) { t.Error("env provider should always be available") }
}

func TestSecretRepo_RotateNilPool(t *testing.T) {
	repo := newSecretRepo(nil)
	if err := repo.Rotate(nil, "jwt-signing-key"); err != nil {
		t.Errorf("nil pool Rotate should not error: %v", err)
	}
}
