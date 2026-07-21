package server

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TrustLevel constants for federation trust validation.
const (
	TrustLevelFull      = "full"
	TrustLevelConditional = "conditional"
	TrustLevelNone      = "none"
)

// TrustChainError describes why trust chain validation failed.
type TrustChainError struct {
	Reason string
}

func (e *TrustChainError) Error() string { return e.Reason }

// TrustChainValidator validates that an incoming federation assertion or
// client auth request originates from a trusted, enabled, non-expired entity.
type TrustChainValidator struct {
	pool *pgxpool.Pool
}

// NewTrustChainValidator creates a validator backed by the federation_entities table.
func NewTrustChainValidator(pool *pgxpool.Pool) *TrustChainValidator {
	return &TrustChainValidator{pool: pool}
}

type fedEntityRow struct {
	EntityID     string
	Protocol     string
	TrustLevel   string
	Enabled      bool
	ExpiresAt    *time.Time
}

// ValidateSAMLIssuer checks that the SAML assertion issuer is a trusted federation entity.
// Must be called before saml.ParseAssertion to reject untrusted IdPs early.
func (v *TrustChainValidator) ValidateSAMLIssuer(ctx context.Context, tenantID, issuer string) error {
	if v == nil || v.pool == nil {
		return nil // nil validator = no enforcement (backward compat)
	}
	entity, err := v.lookupEntity(ctx, tenantID, issuer, "saml")
	if err != nil {
		return err
	}
	if entity.TrustLevel == TrustLevelNone {
		return &TrustChainError{Reason: fmt.Sprintf("entity %s has trust_level=none", issuer)}
	}
	return nil
}

// ValidateOIDCClient checks that an OIDC client_id corresponds to a trusted
// federation entity (for cross-org OIDC federation flows).
func (v *TrustChainValidator) ValidateOIDCClient(ctx context.Context, tenantID, clientID string) error {
	if v == nil || v.pool == nil {
		return nil
	}
	// First-party clients (registered in oauth_clients) are always trusted.
	// Federation trust check only applies to external OIDC federation entities.
	var isInOAuthClients bool
	_ = v.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM oauth_clients WHERE client_id = $1)`, clientID).Scan(&isInOAuthClients)
	if isInOAuthClients {
		return nil
	}
	entity, err := v.lookupEntity(ctx, tenantID, clientID, "oidc")
	if err != nil {
		return err
	}
	if entity.TrustLevel == TrustLevelNone {
		return &TrustChainError{Reason: fmt.Sprintf("oidc entity %s has trust_level=none", clientID)}
	}
	return nil
}

func (v *TrustChainValidator) lookupEntity(ctx context.Context, tenantID, entityID, protocol string) (*fedEntityRow, error) {
	row := v.pool.QueryRow(ctx, `
		SELECT entity_id, protocol, trust_level, enabled, expires_at
		FROM federation_entities
		WHERE entity_id = $1 AND protocol = $2 AND enabled = TRUE
		ORDER BY updated_at DESC LIMIT 1`, entityID, protocol)

	var e fedEntityRow
	if err := row.Scan(&e.EntityID, &e.Protocol, &e.TrustLevel, &e.Enabled, &e.ExpiresAt); err != nil {
		return nil, &TrustChainError{Reason: fmt.Sprintf("untrusted entity %s: not found or disabled", entityID)}
	}
	if !e.Enabled {
		return nil, &TrustChainError{Reason: fmt.Sprintf("entity %s is disabled", entityID)}
	}
	if e.ExpiresAt != nil && e.ExpiresAt.Before(time.Now()) {
		return nil, &TrustChainError{Reason: fmt.Sprintf("entity %s trust expired at %s", entityID, e.ExpiresAt.Format(time.RFC3339))}
	}
	return &e, nil
}

// extractSAMLIssuer pulls the <Issuer> value from raw SAML XML without full parsing.
// This is used to validate the trust chain before ParseAssertion.
func extractSAMLIssuer(rawXML []byte) string {	type issuerWrapper struct {
		XMLName xml.Name `xml:"Response"`
		Issuer  string   `xml:"Issuer"`
	}
	// Try Response-level Issuer first.
	var resp issuerWrapper
	if err := xml.Unmarshal(rawXML, &resp); err == nil && resp.Issuer != "" {
		return resp.Issuer
	}
	// Fallback: Assertion-level Issuer.
	type assertionWrapper struct {
		XMLName xml.Name `xml:"Assertion"`
		Issuer  string   `xml:"Issuer"`
	}
	var assertion assertionWrapper
	if err := xml.Unmarshal(rawXML, &assertion); err == nil && assertion.Issuer != "" {
		return assertion.Issuer
	}
	return ""
}

// samlACSTrustConfig holds the trust anchors for inbound SAML assertions.
type samlTrustConfig struct {
	Cert          *x509.Certificate // trusted IdP signing certificate
	IdPEntityID   string            // expected IdP entityID (optional)
	SPEntityID    string            // our SP entityID for audience validation (optional)
}

// samlACSTrustConfig loads the trust configuration for the SAML ACS endpoint
// from sys_config (key "saml_config"). Fields:
//   - idp_cert      (required) PEM-encoded IdP signing certificate
//   - idp_entity_id (optional) expected assertion Issuer
//   - sp_entity_id  (optional) expected Audience; falls back to BaseURL+"/saml"
//
// Consistent with the auth service SAML handler's sys_config format.
func samlACSTrustConfig(r *http.Request, pool *pgxpool.Pool) (*samlTrustConfig, error) {
	if pool == nil {
		return nil, fmt.Errorf("database not available")
	}

	var configJSON string
	err := pool.QueryRow(r.Context(),
		`SELECT value::text FROM sys_config WHERE key = 'saml_config'`).Scan(&configJSON)
	if err != nil {
		return nil, fmt.Errorf("saml_config not found in sys_config: %w", err)
	}

	var cfg struct {
		IDPCert     string `json:"idp_cert"`
		IdPEntityID string `json:"idp_entity_id"`
		SPEntityID  string `json:"sp_entity_id"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("invalid saml_config JSON: %w", err)
	}
	if cfg.IDPCert == "" {
		return nil, fmt.Errorf("idp_cert not configured")
	}

	block, _ := pem.Decode([]byte(cfg.IDPCert))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM certificate in idp_cert")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse idp_cert: %w", err)
	}
	return &samlTrustConfig{
		Cert:        cert,
		IdPEntityID: cfg.IdPEntityID,
		SPEntityID:  cfg.SPEntityID,
	}, nil
}
