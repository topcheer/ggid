package tap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// AAGUIDEntry represents an approved authenticator model.
type AAGUIDEntry struct {
	AAGUID      string    `json:"aaguid"`       // Authenticator Attestation GUID
	Name        string    `json:"name"`         // Human-readable name
	Description string    `json:"description"`
	Status      string    `json:"status"`       // approved | denied | deprecated
	AddedBy     string    `json:"added_by"`
	AddedAt     time.Time `json:"added_at"`
}

// AAGUIDStatus constants.
const (
	AAGUIDApproved   = "approved"
	AAGUIDDenied     = "denied"
	AAGUIDDeprecated = "deprecated"
)

// Allowlist manages the AAGUID approval list.
type Allowlist struct {
	mu      sync.RWMutex
	entries map[string]*AAGUIDEntry // aaguid → entry
}

// NewAllowlist creates an empty allowlist.
func NewAllowlist() *Allowlist {
	return &Allowlist{entries: make(map[string]*AAGUIDEntry)}
}

// Add registers an AAGUID as approved.
func (a *Allowlist) Add(entry AAGUIDEntry) {
	if entry.Status == "" {
		entry.Status = AAGUIDApproved
	}
	entry.AddedAt = time.Now()
	a.mu.Lock()
	a.entries[entry.AAGUID] = &entry
	a.mu.Unlock()
}

// Remove deletes an AAGUID from the allowlist.
func (a *Allowlist) Remove(aaguid string) {
	a.mu.Lock()
	delete(a.entries, aaguid)
	a.mu.Unlock()
}

// IsApproved checks if an AAGUID is in the approved allowlist.
func (a *Allowlist) IsApproved(aaguid string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	e, ok := a.entries[aaguid]
	return ok && e.Status == AAGUIDApproved
}

// List returns all allowlist entries.
func (a *Allowlist) List() []AAGUIDEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var result []AAGUIDEntry
	for _, e := range a.entries {
		result = append(result, *e)
	}
	return result
}

// PopulateFromMDS loads known-good authenticators from FIDO MDS data.
// mdsData is a map of AAGUID → {name, description}.
func (a *Allowlist) PopulateFromMDS(mdsData map[string]struct {
	Name        string
	Description string
}, addedBy string) int {
	count := 0
	for aaguid, info := range mdsData {
		a.Add(AAGUIDEntry{
			AAGUID:      aaguid,
			Name:        info.Name,
			Description: info.Description,
			Status:      AAGUIDApproved,
			AddedBy:     addedBy,
		})
		count++
	}
	return count
}

// DefaultKnownAuthenticators returns common FIDO-certified AAGUIDs.
func DefaultKnownAuthenticators() map[string]struct {
	Name        string
	Description string
} {
	return map[string]struct {
		Name        string
		Description string
	}{
		// YubiKey 5 Series
		"cb69481e-8ff7-4039-93ec-0a2729a154a8": {"YubiKey 5 NFC", "Yubico YubiKey 5 NFC (USB+NFC)"},
		"08987058-cadc-49b2-ab1f-77a3b49c6f9b": {"YubiKey 5C NFC", "Yubico YubiKey 5C NFC"},
		"34f5766d-1536-4a24-9035-52a172e6330d": {"YubiKey 5 Nano", "Yubico YubiKey 5 Nano"},
		"fa2b99dc-9e39-4257-8f92-4a30d23c4df8": {"YubiKey 5C Nano", "Yubico YubiKey 5C Nano"},
		// Windows Hello
		"6028b017-b1d4-4c02-bf5d-a2737c972a47": {"Windows Hello", "Microsoft Windows Hello"},
		// Google Titan
		"7e66b7f5-8d63-4bec-b3f1-38a06480d6a7": {"Google Titan", "Google Titan Security Key"},
		// Apple
		"adce0002-35bc-c60a-648b-0b25f1f05503": {"Apple Face ID/Touch ID", "Apple platform authenticator"},
	}
}

// EnsureAAGUIDSchema creates webauthn_aaguid_allowlist table.
func EnsureAAGUIDSchema(ctx context.Context, pool interface{ Exec(context.Context, string, ...any) (any, error) }) error {
	// Schema for PG — uses interface for testability.
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS webauthn_aaguid_allowlist (
			aaguid TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'approved',
			added_by TEXT,
			added_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	return err
}

// CheckAttestation verifies that a registration's AAGUID is approved.
// Returns nil if approved, error with message if not.
func (a *Allowlist) CheckAttestation(aaguid string) error {
	if aaguid == "" {
		return fmt.Errorf("no AAGUID in attestation — cannot verify authenticator")
	}
	if !a.IsApproved(aaguid) {
		return fmt.Errorf("authenticator (AAGUID: %s) is not in approved allowlist", aaguid)
	}
	return nil
}

// HashAAGUID returns a hex hash for logging (never log raw AAGUID in production).
func HashAAGUID(aaguid string) string {
	h := sha256.Sum256([]byte(aaguid))
	return hex.EncodeToString(h[:])[:16]
}
