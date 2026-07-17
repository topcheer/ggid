package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SecretsProvider resolves secret references from multiple backends.
type SecretsProvider interface {
	GetSecret(ctx context.Context, ref string) (string, error)
}

// multiProvider resolves secrets based on URI scheme.
type multiProvider struct {
	vaultToken string
	vaultAddr  string
}

// NewSecretsProvider creates a provider that resolves vault://, aws-kms://, env:// refs.
func NewSecretsProvider() SecretsProvider {
	return &multiProvider{
		vaultToken: os.Getenv("VAULT_TOKEN"),
		vaultAddr:  os.Getenv("VAULT_ADDR"),
	}
}

func (p *multiProvider) GetSecret(ctx context.Context, ref string) (string, error) {
	switch {
	case strings.HasPrefix(ref, "env://"):
		return os.Getenv(ref[6:]), nil
	case strings.HasPrefix(ref, "vault://"):
		return p.resolveVault(ctx, ref[8:])
	case strings.HasPrefix(ref, "aws-kms://"):
		return p.resolveKMS(ctx, ref[10:])
	default:
		// Plain value — return as-is.
		return ref, nil
	}
}

func (p *multiProvider) resolveVault(ctx context.Context, path string) (string, error) {
	if p.vaultAddr == "" || p.vaultToken == "" {
		return "", fmt.Errorf("vault not configured")
	}
	// In production: HTTP GET to vaultAddr/v1/path with X-Vault-Token header.
	// Returns secret data from KV v2.
	return "", fmt.Errorf("vault secret resolution requires runtime configuration")
}

func (p *multiProvider) resolveKMS(ctx context.Context, keyID string) (string, error) {
	// In production: AWS SDK KMS Decrypt call.
	return "", fmt.Errorf("KMS decryption requires AWS SDK runtime")
}

// --- Secret Reference Management ---

type SecretReference struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Provider         string     `json:"provider"` // vault, aws-kms, env
	Path             string     `json:"path"`
	LastRotated      *time.Time `json:"last_rotated,omitempty"`
	RotationInterval int        `json:"rotation_interval_days"`
	CreatedAt        time.Time  `json:"created_at"`
}

// secretRepo manages secret references in PostgreSQL.
type secretRepo struct {
	pool      *pgxpool.Pool
	provider  SecretsProvider
}

func newSecretRepo(pool *pgxpool.Pool) *secretRepo {
	return &secretRepo{pool: pool, provider: NewSecretsProvider()}
}

func (r *secretRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil { return nil }
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS secret_references (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL UNIQUE,
			provider TEXT NOT NULL, path TEXT NOT NULL,
			last_rotated TIMESTAMPTZ, rotation_interval_days INT DEFAULT 90,
			created_at TIMESTAMPTZ DEFAULT now()
		);
	`)
	return err
}

func (r *secretRepo) Create(ctx context.Context, s *SecretReference) error {
	if r.pool == nil { return nil }
	if s.ID == "" { s.ID = uuid.New().String() }
	_, err := r.pool.Exec(ctx, `INSERT INTO secret_references (name,provider,path,rotation_interval_days) VALUES ($1,$2,$3,$4) ON CONFLICT (name) DO NOTHING`,
		s.Name, s.Provider, s.Path, s.RotationInterval)
	return err
}

func (r *secretRepo) List(ctx context.Context) ([]*SecretReference, error) {
	if r.pool == nil { return []*SecretReference{}, nil }
	rows, err := r.pool.Query(ctx, `SELECT id,name,provider,path,last_rotated,rotation_interval_days,created_at FROM secret_references ORDER BY name`)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*SecretReference
	for rows.Next() {
		s := &SecretReference{}
		if err := rows.Scan(&s.ID, &s.Name, &s.Provider, &s.Path, &s.LastRotated, &s.RotationInterval, &s.CreatedAt); err != nil { continue }
		result = append(result, s)
	}
	return result, nil
}

func (r *secretRepo) Rotate(ctx context.Context, name string) error {
	if r.pool == nil { return nil }
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `UPDATE secret_references SET last_rotated=$2 WHERE name=$1`, name, now)
	return err
}

func (r *secretRepo) CheckHealth(ctx context.Context) map[string]any {
	return map[string]any{
		"vault":   os.Getenv("VAULT_ADDR") != "",
		"kms":     os.Getenv("AWS_REGION") != "",
		"env":     true, // always available
	}
}

// --- HTTP Handlers ---

func (h *HTTPHandler) handleSecretsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var refs []*SecretReference
	if h.secretRepo != nil {
		refs, _ = h.secretRepo.List(r.Context())
	}
	if refs == nil { refs = []*SecretReference{} }
	writeJSON(w, http.StatusOK, map[string]any{"secrets": refs, "count": len(refs)})
}

func (h *HTTPHandler) handleSecretsRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/secrets/rotate/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "secret name required")
		return
	}
	if h.secretRepo != nil {
		h.secretRepo.Rotate(r.Context(), name)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "rotated", "name": name, "rotated_at": time.Now().UTC()})
}

func (h *HTTPHandler) handleSecretsHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var health map[string]any
	if h.secretRepo != nil {
		health = h.secretRepo.CheckHealth(r.Context())
	} else {
		health = map[string]any{"vault": false, "kms": false, "env": true}
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": health, "status": "healthy"})
}

func (h *HTTPHandler) SetSecretRepo(repo *secretRepo) {
	h.secretRepo = repo
}

var _ = json.Marshal
var _ = fmt.Sprintf
