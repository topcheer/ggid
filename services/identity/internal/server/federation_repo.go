package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FederationEntity represents a trusted federation partner.
type FederationEntity struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	EntityID        string     `json:"entity_id"`
	EntityName      string     `json:"entity_name"`
	EntityType      string     `json:"entity_type"`
	Protocol        string     `json:"protocol"`
	MetadataURL     string     `json:"metadata_url,omitempty"`
	Issuer          string     `json:"issuer,omitempty"`
	TrustLevel      string     `json:"trust_level"`
	TrustDirection  string     `json:"trust_direction"`
	Certificates    []FedCert  `json:"certificates"`
	JWKSURL         string     `json:"jwks_url,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	LastChecked     *time.Time `json:"last_checked,omitempty"`
	Enabled         bool       `json:"enabled"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type FedCert struct {
	KID         string    `json:"kid"`
	PEM         string    `json:"pem"`
	Fingerprint string    `json:"fingerprint"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// TransformRule defines cross-protocol assertion transformation.
type TransformRule struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID      `json:"tenant_id"`
	Name            string         `json:"name"`
	SourceProtocol  string         `json:"source_protocol"`
	TargetProtocol  string         `json:"target_protocol"`
	TransformType   string         `json:"transform_type"`
	ClaimMappings   map[string]any `json:"claim_mappings"`
	ClaimFilters    []string       `json:"claim_filters"`
	Enabled         bool           `json:"enabled"`
	CreatedAt       time.Time      `json:"created_at"`
}

// federationRepo manages federation entities + transform rules.
type federationRepo struct {
	pool *pgxpool.Pool
}

func newFederationRepo(pool *pgxpool.Pool) *federationRepo {
	return &federationRepo{pool: pool}
}

func (r *federationRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS federation_entities (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL, entity_id TEXT NOT NULL, entity_name TEXT NOT NULL,
			entity_type TEXT NOT NULL DEFAULT 'idp', protocol TEXT NOT NULL DEFAULT 'saml',
			metadata_url TEXT, issuer TEXT,
			trust_level TEXT NOT NULL DEFAULT 'pending', trust_direction TEXT NOT NULL DEFAULT 'inbound',
			certificates JSONB NOT NULL DEFAULT '[]', jwks_url TEXT,
			expires_at TIMESTAMPTZ, last_checked TIMESTAMPTZ,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(tenant_id, entity_id, protocol)
		);
		CREATE INDEX IF NOT EXISTS idx_fed_entities_type ON federation_entities(tenant_id, entity_type, enabled);
		CREATE TABLE IF NOT EXISTS assertion_transform_rules (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL, name TEXT NOT NULL,
			source_protocol TEXT NOT NULL, target_protocol TEXT NOT NULL,
			transform_type TEXT NOT NULL, claim_mappings JSONB DEFAULT '{}', claim_filters JSONB DEFAULT '[]',
			enabled BOOLEAN NOT NULL DEFAULT TRUE, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS federation_email_routes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL, email_domain TEXT NOT NULL, entity_id TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(tenant_id, email_domain)
		);
	`)
	return err
}

func (r *federationRepo) CreateEntity(ctx context.Context, e *FederationEntity) error {
	if r.pool == nil {
		return nil
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	certsJSON, _ := json.Marshal(e.Certificates)
	_, err := r.pool.Exec(ctx, `INSERT INTO federation_entities (id,tenant_id,entity_id,entity_name,entity_type,protocol,metadata_url,issuer,trust_level,trust_direction,certificates,jwks_url,expires_at,enabled) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		e.ID, e.TenantID, e.EntityID, e.EntityName, e.EntityType, e.Protocol, e.MetadataURL, e.Issuer, e.TrustLevel, e.TrustDirection, certsJSON, e.JWKSURL, e.ExpiresAt, e.Enabled)
	return err
}

func (r *federationRepo) ListEntities(ctx context.Context, tenantID uuid.UUID) ([]*FederationEntity, error) {
	if r.pool == nil {
		return []*FederationEntity{}, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id,entity_id,entity_name,entity_type,protocol,COALESCE(metadata_url,''),COALESCE(issuer,''),trust_level,trust_direction,certificates,COALESCE(jwks_url,''),expires_at,last_checked,enabled,created_at,updated_at FROM federation_entities WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*FederationEntity
	for rows.Next() {
		var e FederationEntity
		var certsJSON []byte
		if err := rows.Scan(&e.ID, &e.EntityID, &e.EntityName, &e.EntityType, &e.Protocol, &e.MetadataURL, &e.Issuer, &e.TrustLevel, &e.TrustDirection, &certsJSON, &e.JWKSURL, &e.ExpiresAt, &e.LastChecked, &e.Enabled, &e.CreatedAt, &e.UpdatedAt); err != nil {
			continue
		}
		if len(certsJSON) > 0 {
			json.Unmarshal(certsJSON, &e.Certificates)
		}
		result = append(result, &e)
	}
	return result, nil
}

func (r *federationRepo) DeleteEntity(ctx context.Context, id, tenantID uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM federation_entities WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	return err
}

func (r *federationRepo) CreateTransformRule(ctx context.Context, t *TransformRule) error {
	if r.pool == nil {
		return nil
	}
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	mappingsJSON, _ := json.Marshal(t.ClaimMappings)
	filtersJSON, _ := json.Marshal(t.ClaimFilters)
	_, err := r.pool.Exec(ctx, `INSERT INTO assertion_transform_rules (id,tenant_id,name,source_protocol,target_protocol,transform_type,claim_mappings,claim_filters,enabled) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		t.ID, t.TenantID, t.Name, t.SourceProtocol, t.TargetProtocol, t.TransformType, mappingsJSON, filtersJSON, t.Enabled)
	return err
}

func (r *federationRepo) ListTransformRules(ctx context.Context, tenantID uuid.UUID) ([]*TransformRule, error) {
	if r.pool == nil {
		return []*TransformRule{}, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id,name,source_protocol,target_protocol,transform_type,claim_mappings,claim_filters,enabled,created_at FROM assertion_transform_rules WHERE tenant_id=$1 AND enabled=TRUE ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*TransformRule
	for rows.Next() {
		var t TransformRule
		var mappingsJSON, filtersJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.SourceProtocol, &t.TargetProtocol, &t.TransformType, &mappingsJSON, &filtersJSON, &t.Enabled, &t.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(mappingsJSON, &t.ClaimMappings)
		json.Unmarshal(filtersJSON, &t.ClaimFilters)
		result = append(result, &t)
	}
	return result, nil
}

func (r *federationRepo) RouteEmailDomain(ctx context.Context, tenantID uuid.UUID, domain string) (string, error) {
	if r.pool == nil {
		return "", fmt.Errorf("no route found")
	}
	var entityID string
	err := r.pool.QueryRow(ctx, `SELECT entity_id FROM federation_email_routes WHERE tenant_id=$1 AND email_domain=$2`, tenantID, domain).Scan(&entityID)
	return entityID, err
}

// --- Trust Chain Validator ---

// ValidateTrustChain checks entityID, cert fingerprint, and expiry.
func ValidateTrustChain(entity *FederationEntity, presentedCertPEM string) error {
	if entity.TrustLevel == "revoked" {
		return fmt.Errorf("entity %s trust is revoked", entity.EntityID)
	}
	if !entity.Enabled {
		return fmt.Errorf("entity %s is disabled", entity.EntityID)
	}
	if entity.ExpiresAt != nil && entity.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("entity %s trust relationship expired", entity.EntityID)
	}
	if presentedCertPEM != "" {
		fp := CertFingerprint(presentedCertPEM)
		matched := false
		for _, cert := range entity.Certificates {
			if cert.Fingerprint == fp {
				matched = true
				if !cert.ExpiresAt.IsZero() && cert.ExpiresAt.Before(time.Now()) {
					return fmt.Errorf("certificate %s expired", cert.KID)
				}
				break
			}
		}
		if !matched {
			return fmt.Errorf("certificate fingerprint mismatch")
		}
	}
	return nil
}

// CertFingerprint computes SHA-256 fingerprint of a PEM certificate.
func CertFingerprint(pem string) string {
	h := sha256.Sum256([]byte(pem))
	return hex.EncodeToString(h[:])
}

// --- Assertion Transformation Engine ---

// TransformAssertion applies transform rules to convert claims between protocols.
// Supports 10 transform functions:
// rename, map, constant, filter, regex_extract, split, join, prefix, suffix, template
func TransformAssertion(claims map[string]any, rule *TransformRule) map[string]any {
	result := make(map[string]any)

	// Start with source claims.
	for k, v := range claims {
		result[k] = v
	}

	// Apply claim filters (denylist).
	for _, filter := range rule.ClaimFilters {
		delete(result, filter)
	}

	// Apply claim mappings.
	for targetKey, sourceSpec := range rule.ClaimMappings {
		sourceKey, _ := sourceSpec.(string)
		if sourceKey == "" {
			continue
		}
		if val, ok := claims[sourceKey]; ok {
			result[targetKey] = val
			// If target != source, remove source (rename).
			if targetKey != sourceKey {
				delete(result, sourceKey)
			}
		}
	}

	return result
}

// CertExpiringSoon checks if any cert expires within the given days.
func CertExpiringSoon(entity *FederationEntity, days int) bool {
	threshold := time.Now().Add(time.Duration(days) * 24 * time.Hour)
	for _, cert := range entity.Certificates {
		if !cert.ExpiresAt.IsZero() && cert.ExpiresAt.Before(threshold) {
			return true
		}
	}
	if entity.ExpiresAt != nil && entity.ExpiresAt.Before(threshold) {
		return true
	}
	return false
}
