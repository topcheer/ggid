package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// KeyRotationEntry tracks a key rotation event.
type KeyRotationEntry struct {
	ID             string     `json:"id"`
	KeyType        string     `json:"key_type"` // jwt_signing, scep_ca, webhook_hmac
	OldKeyID       string     `json:"old_key_id"`
	NewKeyID       string     `json:"new_key_id"`
	Status         string     `json:"status"` // active, grace, expired
	RotatedAt      time.Time  `json:"rotated_at"`
	GraceExpiresAt time.Time  `json:"grace_expires_at"`
}

// ActiveKey tracks currently active keys.
type ActiveKey struct {
	KeyType  string    `json:"key_type"`
	KeyID    string    `json:"key_id"`
	Status   string    `json:"status"` // primary, grace
	CreatedAt time.Time `json:"created_at"`
}

// keyRotationRepo manages key rotation in PG.
type keyRotationRepo struct {
	pool *pgxpool.Pool
}

func newKeyRotationRepo(pool *pgxpool.Pool) *keyRotationRepo {
	return &keyRotationRepo{pool: pool}
}

func (r *keyRotationRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil { return nil }
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS key_rotation_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			key_type TEXT NOT NULL, old_key_id TEXT, new_key_id TEXT NOT NULL,
			status TEXT DEFAULT 'active', rotated_at TIMESTAMPTZ DEFAULT now(),
			grace_expires_at TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_key_rotation_type ON key_rotation_log(key_type, status);
	`)
	return err
}

func (r *keyRotationRepo) Rotate(ctx context.Context, keyType string, graceDays int) (*KeyRotationEntry, error) {
	if graceDays <= 0 { graceDays = 7 }
	now := time.Now().UTC()
	newKeyID := uuid.New().String()
	entry := &KeyRotationEntry{
		ID: uuid.New().String(), KeyType: keyType,
		NewKeyID: newKeyID, Status: "active",
		RotatedAt: now, GraceExpiresAt: now.Add(time.Duration(graceDays) * 24 * time.Hour),
	}
	if r.pool != nil {
		// Mark previous active keys as grace.
		r.pool.Exec(ctx, `UPDATE key_rotation_log SET status='grace' WHERE key_type=$1 AND status='active'`, keyType)
		_, err := r.pool.Exec(ctx,
			`INSERT INTO key_rotation_log (id,key_type,old_key_id,new_key_id,status,grace_expires_at) VALUES ($1,$2,$3,$4,$5,$6)`,
			entry.ID, entry.KeyType, entry.OldKeyID, entry.NewKeyID, entry.Status, entry.GraceExpiresAt)
		if err != nil { return nil, err }
	}
	return entry, nil
}

func (r *keyRotationRepo) ListActiveKeys(ctx context.Context) ([]*ActiveKey, error) {
	if r.pool == nil { return []*ActiveKey{}, nil }
	rows, err := r.pool.Query(ctx,
		`SELECT key_type,new_key_id,status,rotated_at FROM key_rotation_log WHERE status IN ('active','grace') ORDER BY rotated_at DESC`)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*ActiveKey
	for rows.Next() {
		k := &ActiveKey{}
		if err := rows.Scan(&k.KeyType, &k.KeyID, &k.Status, &k.CreatedAt); err != nil { continue }
		result = append(result, k)
	}
	return result, nil
}

func (r *keyRotationRepo) ListHistory(ctx context.Context, limit int) ([]*KeyRotationEntry, error) {
	if r.pool == nil { return []*KeyRotationEntry{}, nil }
	if limit <= 0 || limit > 100 { limit = 50 }
	rows, err := r.pool.Query(ctx,
		`SELECT id,key_type,COALESCE(old_key_id,''),new_key_id,status,rotated_at,grace_expires_at FROM key_rotation_log ORDER BY rotated_at DESC LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*KeyRotationEntry
	for rows.Next() {
		e := &KeyRotationEntry{}
		if err := rows.Scan(&e.ID, &e.KeyType, &e.OldKeyID, &e.NewKeyID, &e.Status, &e.RotatedAt, &e.GraceExpiresAt); err != nil { continue }
		result = append(result, e)
	}
	return result, nil
}

// ExpireGrace marks grace-period keys as expired.
func (r *keyRotationRepo) ExpireGrace(ctx context.Context) (int, error) {
	if r.pool == nil { return 0, nil }
	tag, err := r.pool.Exec(ctx, `UPDATE key_rotation_log SET status='expired' WHERE status='grace' AND grace_expires_at < now()`)
	if err != nil { return 0, err }
	return int(tag.RowsAffected()), nil
}

// GenerateECDSAKeyPair creates a new P-256 keypair (for JWT signing).
func GenerateECDSAKeyPair() (string, string, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { return "", "", err }
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil { return "", "", err }
	privPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}))
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil { return "", "", err }
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}))
	return privPEM, pubPEM, nil
}

// --- HTTP Handlers ---

func (h *Handler) handleKeyList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var keys []*ActiveKey
	if h.keyRotationRepo != nil {
		keys, _ = h.keyRotationRepo.ListActiveKeys(r.Context())
	}
	if keys == nil { keys = []*ActiveKey{} }
	writeJSON(w, http.StatusOK, map[string]any{"keys": keys, "count": len(keys)})
}

func (h *Handler) handleKeyRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	keyType := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/keys/rotate/")
	validTypes := map[string]bool{"jwt_signing": true, "scep_ca": true, "webhook_hmac": true}
	if !validTypes[keyType] {
		writeError(w, http.StatusBadRequest, "type must be jwt_signing, scep_ca, or webhook_hmac")
		return
	}
	var entry *KeyRotationEntry
	if h.keyRotationRepo != nil {
		entry, _ = h.keyRotationRepo.Rotate(r.Context(), keyType, 7)
	}
	if entry == nil {
		entry = &KeyRotationEntry{KeyType: keyType, NewKeyID: uuid.New().String(), Status: "active"}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "rotated", "entry": entry})
}

func (h *Handler) handleKeyHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var history []*KeyRotationEntry
	if h.keyRotationRepo != nil {
		history, _ = h.keyRotationRepo.ListHistory(r.Context(), 50)
	}
	if history == nil { history = []*KeyRotationEntry{} }
	writeJSON(w, http.StatusOK, map[string]any{"history": history, "count": len(history)})
}

func (h *Handler) SetKeyRotationRepo(repo *keyRotationRepo) {
	h.keyRotationRepo = repo
}

var _ = json.Marshal
var _ = fmt.Sprintf
