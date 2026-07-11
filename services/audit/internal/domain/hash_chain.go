package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// hashChainSecret is the HMAC key used for audit event hash chaining.
// Loaded from env (AUDIT_HASH_SECRET) at service boot.
var hashChainSecret []byte

// SetHashChainSecret sets the HMAC secret used for audit hash chaining.
func SetHashChainSecret(secret []byte) {
	hashChainSecret = secret
}

// IsHashChainEnabled returns true if a hash chain secret has been configured.
func IsHashChainEnabled() bool {
	return len(hashChainSecret) > 0
}

// ComputeHash computes the HMAC-SHA256 hash chain link for this event.
// hash = HMAC(secret, prev_hash || canonical_data)
func (e *AuditEvent) ComputeHash(prevHash string) string {
	canonical := fmt.Sprintf("%s|%s|%s|%v|%s|%s|%v|%s|%s|%d",
		e.ID, e.TenantID, e.ActorType, e.ActorID,
		e.Action, e.ResourceType, e.ResourceID,
		e.Result, e.IPAddress, e.CreatedAt.UnixNano(),
	)

	mac := hmac.New(sha256.New, hashChainSecret)
	mac.Write([]byte(prevHash))
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHash checks that this event's hash is valid given the previous hash.
func (e *AuditEvent) VerifyHash(prevHash string) bool {
	if e.Hash == "" {
		return false
	}
	return hmac.Equal([]byte(e.Hash), []byte(e.ComputeHash(prevHash)))
}

// VerifyChain validates a sequence of audit events. Returns the index of
// the first broken link, or -1 if the entire chain is valid.
func VerifyChain(events []*AuditEvent) int {
	if len(events) == 0 {
		return -1
	}
	prevHash := ""
	for i, e := range events {
		if !e.VerifyHash(prevHash) {
			return i
		}
		prevHash = e.Hash
	}
	return -1
}

// CanonicalJSON serializes the event to deterministic JSON.
func (e *AuditEvent) CanonicalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID           string         `json:"id"`
		TenantID     string         `json:"tenant_id"`
		ActorType    string         `json:"actor_type"`
		ActorID      string         `json:"actor_id"`
		Action       string         `json:"action"`
		ResourceType string         `json:"resource_type"`
		ResourceID   string         `json:"resource_id"`
		Result       string         `json:"result"`
		IPAddress    string         `json:"ip_address"`
		RequestID    string         `json:"request_id"`
		Metadata     map[string]any `json:"metadata,omitempty"`
		CreatedAt    int64          `json:"created_at_ns"`
	}{
		ID:           e.ID.String(),
		TenantID:     e.TenantID.String(),
		ActorType:    string(e.ActorType),
		ActorID:      uuidToString(e.ActorID),
		Action:       e.Action,
		ResourceType: e.ResourceType,
		ResourceID:   uuidToString(e.ResourceID),
		Result:       string(e.Result),
		IPAddress:    e.IPAddress,
		RequestID:    e.RequestID,
		Metadata:     e.Metadata,
		CreatedAt:    e.CreatedAt.UnixNano(),
	})
}

func uuidToString(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}
