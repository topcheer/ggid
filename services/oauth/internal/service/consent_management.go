package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ConsentMgmtRecord represents a user's consent for a client to access certain scopes.
// This is separate from ConsentRecord in consent.go which is the basic store record.
type ConsentMgmtRecord struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	ClientID    string     `json:"client_id"`
	Scopes      []string   `json:"scopes"`
	GrantedAt   time.Time  `json:"granted_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Withdrawn   bool       `json:"withdrawn"`
	WithdrawnAt *time.Time `json:"withdrawn_at,omitempty"`
}

// ConsentManager manages OAuth consent records with withdrawal and expiry support.
type ConsentManager struct {
	mu      sync.RWMutex
	records map[string]*ConsentMgmtRecord
}

// NewConsentManager creates a new ConsentManager.
func NewConsentManager() *ConsentManager {
	return &ConsentManager{records: make(map[string]*ConsentMgmtRecord)}
}

// GrantConsent creates or updates a consent record.
func (cm *ConsentManager) GrantConsent(userID uuid.UUID, clientID string, scopes []string, expiresAt *time.Time) (*ConsentMgmtRecord, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}
	if clientID == "" {
		return nil, fmt.Errorf("client_id is required")
	}
	key := consentMgmtKey(userID, clientID)
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if existing, ok := cm.records[key]; ok {
		existing.Scopes = mergeConsentScopes(existing.Scopes, scopes)
		existing.GrantedAt = time.Now()
		existing.ExpiresAt = expiresAt
		existing.Withdrawn = false
		existing.WithdrawnAt = nil
		return existing, nil
	}
	record := &ConsentMgmtRecord{ID: uuid.New(), UserID: userID, ClientID: clientID, Scopes: scopes, GrantedAt: time.Now(), ExpiresAt: expiresAt}
	cm.records[key] = record
	return record, nil
}

// GetConsent retrieves the consent record for a user-client pair.
func (cm *ConsentManager) GetConsent(userID uuid.UUID, clientID string) (*ConsentMgmtRecord, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	record, ok := cm.records[consentMgmtKey(userID, clientID)]
	if !ok {
		return nil, nil
	}
	return record, nil
}

// WithdrawConsent marks a consent as withdrawn.
func (cm *ConsentManager) WithdrawConsent(userID uuid.UUID, clientID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	record, ok := cm.records[consentMgmtKey(userID, clientID)]
	if !ok {
		return fmt.Errorf("consent not found")
	}
	now := time.Now()
	record.Withdrawn = true
	record.WithdrawnAt = &now
	return nil
}

// ListConsents returns all consent records for a user.
func (cm *ConsentManager) ListConsents(userID uuid.UUID) ([]ConsentMgmtRecord, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	var result []ConsentMgmtRecord
	for _, r := range cm.records {
		if r.UserID == userID {
			result = append(result, *r)
		}
	}
	return result, nil
}

// IsConsentValid checks if a valid consent exists for the requested scopes.
func (cm *ConsentManager) IsConsentValid(userID uuid.UUID, clientID string, requestedScopes []string) (bool, string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	record, ok := cm.records[consentMgmtKey(userID, clientID)]
	if !ok {
		return false, "consent not found"
	}
	if record.Withdrawn {
		return false, "consent withdrawn"
	}
	if record.ExpiresAt != nil && time.Now().After(*record.ExpiresAt) {
		return false, "consent expired"
	}
	granted := make(map[string]bool)
	for _, s := range record.Scopes {
		granted[s] = true
	}
	for _, req := range requestedScopes {
		if !granted[req] && !granted["*"] {
			return false, fmt.Sprintf("scope '%s' not granted", req)
		}
	}
	return true, ""
}

// Reset clears all consent records.
func (cm *ConsentManager) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.records = make(map[string]*ConsentMgmtRecord)
}

func consentMgmtKey(userID uuid.UUID, clientID string) string {
	return fmt.Sprintf("cm:%s:%s", userID, clientID)
}

func mergeConsentScopes(existing, newScopes []string) []string {
	seen := make(map[string]bool)
	for _, s := range existing {
		seen[s] = true
	}
	for _, s := range newScopes {
		if !seen[s] {
			existing = append(existing, s)
			seen[s] = true
		}
	}
	return existing
}