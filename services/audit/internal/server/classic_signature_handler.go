package httpserver

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// NOTE: This handler uses Ed25519 (classic cryptography), NOT post-quantum cryptography.
// The endpoint paths retain "pqc" for backward API compatibility, but the algorithm
// field correctly reports "Ed25519". When ML-DSA (FIPS 205) becomes available in
// Go's standard library or via cloudflare/circl, this should be migrated to true PQC.
// See: docs/guides/post-quantum-crypto-migration.md

// IntegritySignature tracks a cryptographic signature over an audit log batch.
type IntegritySignature struct {
	BatchID    string `json:"batch_id"`
	Signature  string `json:"signature"`
	Algorithm  string `json:"algorithm"`
	SignedAt   string `json:"signed_at"`
	EventCount int    `json:"event_count"`
}

var (
	sigMu      sync.RWMutex
	sigRecords  = make(map[string]*IntegritySignature)
	classicPub  ed25519.PublicKey
	classicPriv ed25519.PrivateKey
)

func init() {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	classicPub, classicPriv = pub, priv
}

// POST /api/v1/audit/integrity/sign-pqc — sign audit log batch with Ed25519.
// NOTE: Uses Ed25519 (classic), not post-quantum. Algorithm field reports "Ed25519".
// Body: {"batch_data": "...", "event_count": N}
func (s *HTTPServer) handlePQCSign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		BatchData  string `json:"batch_data"`
		EventCount int    `json:"event_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.BatchData == "" {
		req.BatchData = time.Now().UTC().String()
	}

	sig := ed25519.Sign(classicPriv, []byte(req.BatchData))
	batchID := uuid.New().String()

	record := &IntegritySignature{
		BatchID:    batchID,
		Signature:  base64.StdEncoding.EncodeToString(sig),
		Algorithm:  "Ed25519", // Honest: NOT post-quantum. Migrate to ML-DSA when available.
		SignedAt:   time.Now().UTC().Format(time.RFC3339),
		EventCount: req.EventCount,
	}

	sigMu.Lock()
	sigRecords[batchID] = record
	sigMu.Unlock()

	writeJSON(w, http.StatusOK, record)
}

// GET /api/v1/audit/integrity/verify-pqc?batch_id=X&batch_data=Y
// NOTE: Verifies Ed25519 signatures, NOT post-quantum.
func (s *HTTPServer) handlePQCVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	batchID := r.URL.Query().Get("batch_id")
	batchData := r.URL.Query().Get("batch_data")

	sigMu.RLock()
	rec, ok := sigRecords[batchID]
	sigMu.RUnlock()
	if !ok {
		writeJSONError(w, http.StatusNotFound, "batch not found")
		return
	}

	sig, err := base64.StdEncoding.DecodeString(rec.Signature)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "invalid signature encoding")
		return
	}

	valid := ed25519.Verify(classicPub, []byte(batchData), sig)
	writeJSON(w, http.StatusOK, map[string]any{
		"batch_id":  batchID,
		"valid":     valid,
		"algorithm": rec.Algorithm,
		"signed_at": rec.SignedAt,
	})
}
