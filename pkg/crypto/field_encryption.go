package crypto

import (
	"context"
	"encoding/json"
	"time"
)

// FieldEncryptionConfig defines which fields should be encrypted per tenant.
type FieldEncryptionConfig struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	TableName      string    `json:"table_name"`
	ColumnName     string    `json:"column_name"`
	Classification string    `json:"classification"` // core | important | general
	CreatedAt      time.Time `json:"created_at"`
}

// FieldEncryptionService provides field-level encryption/decryption using DataKeyProvider.
// It maintains a registry of which fields need encryption per tenant.
type FieldEncryptionService struct {
	provider DataKeyProvider
	configs  map[string][]FieldEncryptionConfig // tenantID → configs
}

// NewFieldEncryptionService creates a new field encryption service.
func NewFieldEncryptionService(provider DataKeyProvider) *FieldEncryptionService {
	return &FieldEncryptionService{
		provider: provider,
		configs:  make(map[string][]FieldEncryptionConfig),
	}
}

// RegisterField adds a field to the encryption registry.
func (s *FieldEncryptionService) RegisterField(cfg FieldEncryptionConfig) {
	if cfg.TenantID == "" {
		return
	}
	s.configs[cfg.TenantID] = append(s.configs[cfg.TenantID], cfg)
}

// ListFields returns all encrypted field configs for a tenant.
func (s *FieldEncryptionService) ListFields(tenantID string) []FieldEncryptionConfig {
	if cfgs, ok := s.configs[tenantID]; ok {
		return cfgs
	}
	return []FieldEncryptionConfig{}
}

// RemoveField removes a field from the encryption registry.
func (s *FieldEncryptionService) RemoveField(tenantID, id string) {
	cfgs := s.configs[tenantID]
	for i, c := range cfgs {
		if c.ID == id {
			s.configs[tenantID] = append(cfgs[:i], cfgs[i+1:]...)
			return
		}
	}
}

// ShouldEncrypt checks if a field should be encrypted.
func (s *FieldEncryptionService) ShouldEncrypt(tenantID, table, column string) bool {
	for _, c := range s.configs[tenantID] {
		if c.TableName == table && c.ColumnName == column {
			return true
		}
	}
	return false
}

// EncryptRecord encrypts specified fields in a map[string]any record.
// Only fields matching the registry for this tenant are encrypted.
func (s *FieldEncryptionService) EncryptRecord(ctx context.Context, tenantID, table string, record map[string]any) error {
	if s.provider == nil {
		return nil
	}
	for _, cfg := range s.configs[tenantID] {
		if cfg.TableName != table {
			continue
		}
		val, exists := record[cfg.ColumnName]
		if !exists {
			continue
		}
		// Convert to bytes.
		var plaintext []byte
		switch v := val.(type) {
		case string:
			plaintext = []byte(v)
		case nil:
			continue
		default:
			data, err := json.Marshal(v)
			if err != nil {
				continue
			}
			plaintext = data
		}
		ciphertext, err := s.provider.EncryptField(ctx, tenantID, plaintext)
		if err != nil {
			return err
		}
		record[cfg.ColumnName] = ciphertext
		record["__encrypted_"+cfg.ColumnName] = true
	}
	return nil
}

// DecryptRecord decrypts encrypted fields in a map[string]any record.
func (s *FieldEncryptionService) DecryptRecord(ctx context.Context, record map[string]any) error {
	if s.provider == nil {
		return nil
	}
	for key, val := range record {
		if !startsWith(key, "__encrypted_") {
			continue
		}
		if encrypted, _ := val.(bool); !encrypted {
			continue
		}
		fieldName := key[len("__encrypted_"):]
		ciphertext, ok := record[fieldName].(string)
		if !ok {
			continue
		}
		plaintext, err := s.provider.DecryptField(ctx, ciphertext)
		if err != nil {
			continue // leave encrypted if decryption fails
		}
		record[fieldName] = string(plaintext)
		delete(record, key)
	}
	return nil
}

// EnsureEncryptedFieldsSchema returns SQL for the encrypted_fields config table.
func EnsureEncryptedFieldsSchema() string {
	return `
	CREATE TABLE IF NOT EXISTS encrypted_fields (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		table_name TEXT NOT NULL,
		column_name TEXT NOT NULL,
		classification TEXT NOT NULL DEFAULT 'important',
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		UNIQUE(tenant_id, table_name, column_name)
	);
	CREATE INDEX IF NOT EXISTS idx_encrypted_fields_tenant ON encrypted_fields(tenant_id);
	`
}

// DefaultPIIFields returns the default PII fields to encrypt for identity service.
func DefaultPIIFields(tenantID string) []FieldEncryptionConfig {
	return []FieldEncryptionConfig{
		{ID: "pii_email", TenantID: tenantID, TableName: "users", ColumnName: "email", Classification: "core"},
		{ID: "pii_phone", TenantID: tenantID, TableName: "users", ColumnName: "phone", Classification: "important"},
		{ID: "pii_full_name", TenantID: tenantID, TableName: "users", ColumnName: "full_name", Classification: "important"},
	}
}

func startsWith(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}
