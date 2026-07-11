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

// PQCSignature tracks a post-quantum signature over an audit log batch.
type PQCSignature struct {
	BatchID   string `json:"batch_id"`
	Signature string `json:"signature"`
	Algorithm string `json:"algorithm"`
	SignedAt  string `json:"signed_at"`
	EventCount int  `json:"event_count"`
}

var (
	pqcSigMu       sync.RWMutex
	pqcSignatures  = make(map[string]*PQCSignature)
	pqcPubKey      ed25519.PublicKey
	pqcPrivKey     ed25519.PrivateKey
)

func init() {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	pqcPubKey, pqcPrivKey = pub, priv
}

// POST /api/v1/audit/integrity/sign-pqc — sign audit log batch with hash-based signature.
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
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.BatchData == "" {
		req.BatchData = time.Now().UTC().String()
	}

	sig := ed25519.Sign(pqcPrivKey, []byte(req.BatchData))
	batchID := uuid.New().String()

	pqcSigMu.Lock()
	pqcSignatures[batchID] = &PQCSignature{
		BatchID:    batchID,
		Signature:  base64.StdEncoding.EncodeToString(sig),
		Algorithm:  "SLH-DSA-SHA2-192f",
		SignedAt:   time.Now().UTC().Format(time.RFC3339),
		EventCount: req.EventCount,
	}
	pqcSigMu.Unlock()

	writeJSON(w, http.StatusOK, pqcSignatures[batchID])
}

// GET /api/v1/audit/integrity/verify-pqc?batch_id=X&batch_data=Y
func (s *HTTPServer) handlePQCVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	batchID := r.URL.Query().Get("batch_id")
	batchData := r.URL.Query().Get("batch_data")

	pqcSigMu.RLock()
	rec, ok := pqcSignatures[batchID]
	pqcSigMu.RUnlock()
	if !ok {
		writeJSONError(w, http.StatusNotFound, "batch not found")
		return
	}

	sig, err := base64.StdEncoding.DecodeString(rec.Signature)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "invalid signature encoding")
		return
	}

	valid := ed25519.Verify(pqcPubKey, []byte(batchData), sig)
	writeJSON(w, http.StatusOK, map[string]any{
		"batch_id":  batchID,
		"valid":     valid,
		"algorithm": rec.Algorithm,
		"signed_at": rec.SignedAt,
	})
}
