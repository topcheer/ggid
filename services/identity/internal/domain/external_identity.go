package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExternalIdentity links a local user to an external identity provider
// such as LDAP, Google, GitHub, SAML, or OIDC.
type ExternalIdentity struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TenantID   uuid.UUID
	Provider   string // ldap, google, github, saml:xxx, oidc:yyy
	ExternalID string // unique ID in the external system (e.g. LDAP DN, OAuth sub)
	Metadata   map[string]any
	LinkedAt   time.Time
}

// MetadataJSON returns the metadata as a json.RawMessage suitable for pgx.
func (e *ExternalIdentity) MetadataJSON() json.RawMessage {
	if e.Metadata == nil {
		return json.RawMessage("{}")
	}
	b, _ := json.Marshal(e.Metadata)
	return b
}
