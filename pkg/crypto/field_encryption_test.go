package crypto

import (
	"context"
	"testing"
)

func TestFieldEncryption_EncryptDecryptRecord(t *testing.T) {
	provider := NewEnvelopeEncryptionProvider([]byte("field-encryption-kek-key"))
	svc := NewFieldEncryptionService(provider)

	// Register fields to encrypt.
	svc.RegisterField(FieldEncryptionConfig{
		TenantID: "t1", TableName: "users",
		ColumnName: "email", Classification: "core",
	})

	record := map[string]any{
		"id":    "user-123",
		"email": "user@example.com",
		"name":  "John Doe",
		"role":  "viewer",
	}

	ctx := context.Background()
	if err := svc.EncryptRecord(ctx, "t1", "users", record); err != nil {
		t.Fatalf("EncryptRecord failed: %v", err)
	}

	// Email should be encrypted (not the original plaintext).
	encryptedEmail, _ := record["email"].(string)
	if encryptedEmail == "user@example.com" {
		t.Fatal("email should be encrypted")
	}
	if encryptedEmail == "" {
		t.Fatal("email should not be empty")
	}
	// Other fields should be untouched.
	if record["name"] != "John Doe" {
		t.Fatal("non-encrypted field should be unchanged")
	}
	// __encrypted_email marker should be set.
	if marker, _ := record["__encrypted_email"].(bool); !marker {
		t.Fatal("expected __encrypted_email marker")
	}

	// Decrypt.
	if err := svc.DecryptRecord(ctx, record); err != nil {
		t.Fatalf("DecryptRecord failed: %v", err)
	}
	if record["email"] != "user@example.com" {
		t.Fatalf("decrypted email should match: got %v", record["email"])
	}
	if _, exists := record["__encrypted_email"]; exists {
		t.Fatal("marker should be removed after decrypt")
	}
}

func TestFieldEncryption_NonRegisteredFieldNotEncrypted(t *testing.T) {
	provider := NewEnvelopeEncryptionProvider([]byte("test-kek"))
	svc := NewFieldEncryptionService(provider)
	svc.RegisterField(FieldEncryptionConfig{
		TenantID: "t1", TableName: "users",
		ColumnName: "email", Classification: "core",
	})

	record := map[string]any{
		"email":   "test@test.com",
		"phone":   "+1234567890", // not registered for encryption
	}

	ctx := context.Background()
	svc.EncryptRecord(ctx, "t1", "users", record)

	// Phone should NOT be encrypted.
	if record["phone"] != "+1234567890" {
		t.Fatal("non-registered field should not be encrypted")
	}
}

func TestFieldEncryption_DifferentTenants(t *testing.T) {
	provider := NewEnvelopeEncryptionProvider([]byte("multi-tenant-kek"))
	svc := NewFieldEncryptionService(provider)
	svc.RegisterField(FieldEncryptionConfig{
		TenantID: "t1", TableName: "users", ColumnName: "email", Classification: "core",
	})

	// Tenant 2 has no encrypted fields registered.
	record := map[string]any{"email": "user@example.com"}
	ctx := context.Background()
	svc.EncryptRecord(ctx, "t2", "users", record)

	if record["email"] != "user@example.com" {
		t.Fatal("tenant without registration should not encrypt")
	}
}

func TestFieldEncryption_ListAndRemove(t *testing.T) {
	provider := NewEnvelopeEncryptionProvider([]byte("kek"))
	svc := NewFieldEncryptionService(provider)
	svc.RegisterField(FieldEncryptionConfig{ID: "f1", TenantID: "t1", TableName: "users", ColumnName: "email"})
	svc.RegisterField(FieldEncryptionConfig{ID: "f2", TenantID: "t1", TableName: "users", ColumnName: "phone"})

	fields := svc.ListFields("t1")
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}

	svc.RemoveField("t1", "f1")
	fields = svc.ListFields("t1")
	if len(fields) != 1 {
		t.Fatalf("expected 1 field after remove, got %d", len(fields))
	}
}

func TestFieldEncryption_ShouldEncrypt(t *testing.T) {
	provider := NewEnvelopeEncryptionProvider([]byte("kek"))
	svc := NewFieldEncryptionService(provider)
	svc.RegisterField(FieldEncryptionConfig{TenantID: "t1", TableName: "users", ColumnName: "email"})

	if !svc.ShouldEncrypt("t1", "users", "email") {
		t.Fatal("ShouldEncrypt should return true for registered field")
	}
	if svc.ShouldEncrypt("t1", "users", "phone") {
		t.Fatal("ShouldEncrypt should return false for unregistered field")
	}
	if svc.ShouldEncrypt("t2", "users", "email") {
		t.Fatal("ShouldEncrypt should return false for unregistered tenant")
	}
}

func TestDefaultPIIFields(t *testing.T) {
	fields := DefaultPIIFields("t1")
	if len(fields) != 3 {
		t.Fatalf("expected 3 default PII fields, got %d", len(fields))
	}
	found := map[string]bool{}
	for _, f := range fields {
		found[f.ColumnName] = true
	}
	if !found["email"] || !found["phone"] || !found["full_name"] {
		t.Fatal("expected email, phone, full_name in default PII fields")
	}
}
