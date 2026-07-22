package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// hashChainSecrets maps secret version → key bytes.
// Version 0 is the default (backward compat with pre-versioning events).
var hashChainSecrets map[int][]byte
var hashChainCurrentVersion int

func init() {
	hashChainSecrets = make(map[int][]byte)
}

// SetHashChainSecret sets the current-version HMAC secret.
func SetHashChainSecret(secret []byte) {
	hashChainSecrets[hashChainCurrentVersion] = secret
}

// SetHashChainSecretVersioned sets a secret at a specific version.
// Use this when rotating keys: keep old versions for backward verification.
func SetHashChainSecretVersioned(version int, secret []byte) {
	hashChainSecrets[version] = secret
	if version > hashChainCurrentVersion {
		hashChainCurrentVersion = version
	}
}

// SetHashChainCurrentVersion sets which version to use for new events.
func SetHashChainCurrentVersion(v int) { hashChainCurrentVersion = v }

// IsHashChainEnabled returns true if a hash chain secret has been configured.
func IsHashChainEnabled() bool {
	return len(hashChainSecrets[hashChainCurrentVersion]) > 0
}

// canonicalEventData produces a deterministic byte representation of an event
// for hash chain computation. P2-7: uses length-prefixed fields to prevent
// delimiter collision (old code used "|" which could collide if field values
// contain "|"). Each field is prefixed with its byte length as a hex uint16.
func canonicalEventData(e *AuditEvent) []byte {
	fields := []string{
		e.ID.String(),
		e.TenantID.String(),
		string(e.ActorType),
		uuidToString(e.ActorID),
		e.Action,
		e.ResourceType,
		uuidToString(e.ResourceID),
		string(e.Result),
		e.IPAddress,
		fmt.Sprintf("%d", e.CreatedAt.UnixNano()),
	}
	var buf []byte
	for _, f := range fields {
		n := len(f)
		buf = append(buf, []byte(fmt.Sprintf("%04x", n))...)
		buf = append(buf, []byte(f)...)
	}
	return buf
}

// ComputeHash computes the HMAC-SHA256 hash chain link for this event.
// P2-6: secret version is prepended to the HMAC input so events hashed
// with different secret versions can be identified and verified with
// the correct key. Format: HMAC(vN_secret, "vN" || prev_hash || canonical)
func (e *AuditEvent) ComputeHash(prevHash string) string {
	secret := hashChainSecrets[hashChainCurrentVersion]
	if secret == nil {
		secret = hashChainSecrets[0] // backward compat
	}

	mac := hmac.New(sha256.New, secret)
	// Write version tag + prev_hash + canonical data
	mac.Write([]byte(fmt.Sprintf("v%d:", hashChainCurrentVersion)))
	mac.Write([]byte(prevHash))
	mac.Write(canonicalEventData(e))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHashWithVersion checks the hash using the secret at the event's
// recorded version. Falls back to version 0 for legacy events.
func (e *AuditEvent) VerifyHashWithVersion(prevHash string, secretVersion int) bool {
	if e.Hash == "" {
		return false
	}
	secret := hashChainSecrets[secretVersion]
	if secret == nil {
		secret = hashChainSecrets[0]
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(fmt.Sprintf("v%d:", secretVersion)))
	mac.Write([]byte(prevHash))
	mac.Write(canonicalEventData(e))
	return hmac.Equal([]byte(e.Hash), []byte(hex.EncodeToString(mac.Sum(nil))))
}

// VerifyHash checks that this event's hash is valid given the previous hash.
// Uses version 0 (backward compat — works for events hashed before versioning).
func (e *AuditEvent) VerifyHash(prevHash string) bool {
	return e.VerifyHashWithVersion(prevHash, 0)
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
