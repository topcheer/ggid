package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// AuthorizationDetail is a single RAR element per RFC 9396.
type AuthorizationDetail struct {
	Type        string         `json:"type"`
	Locations   []string       `json:"locations,omitempty"`
	Actions     []string       `json:"actions,omitempty"`
	Datatypes   []string       `json:"datatypes,omitempty"`
	Identifier  map[string]any `json:"identifier,omitempty"`
	Privileges  []string       `json:"privileges,omitempty"`
	Fields      []string       `json:"fields,omitempty"`
	Constraints map[string]any `json:"constraints,omitempty"`
}

// ConsentLine is a human-readable RAR description for the consent page.
type ConsentLine struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Details string `json:"details"`
}

// RARRegistry maps RAR types to handlers for validation + consent rendering.
type RARRegistry struct {
	handlers map[string]RARTypeHandler
}

// RARTypeHandler validates and renders a specific RAR type.
type RARTypeHandler interface {
	Type() string
	Validate(detail *AuthorizationDetail, clientID string) error
	RenderForConsent(detail *AuthorizationDetail) ConsentLine
}

func NewRARRegistry() *RARRegistry {
	r := &RARRegistry{handlers: make(map[string]RARTypeHandler)}
	// Register 6 built-in types.
	r.Register(&userProfileType{})
	r.Register(&userRolesType{})
	r.Register(&auditEventsType{})
	r.Register(&appAccessType{})
	r.Register(&paymentInitiationType{})
	r.Register(&vcIssueType{})
	return r
}

func (r *RARRegistry) Register(h RARTypeHandler) {
	r.handlers[h.Type()] = h
}

func (r *RARRegistry) Get(typeName string) (RARTypeHandler, bool) {
	h, ok := r.handlers[typeName]
	return h, ok
}

// ValidateDetails validates all authorization_details elements.
func (r *RARRegistry) ValidateDetails(details []AuthorizationDetail, clientID string) error {
	for i, d := range details {
		h, ok := r.handlers[d.Type]
		if !ok {
			return fmt.Errorf("unsupported authorization_details[%d].type: %s", i, d.Type)
		}
		if err := h.Validate(&d, clientID); err != nil {
			return fmt.Errorf("invalid authorization_details[%d]: %w", i, err)
		}
	}
	return nil
}

// RenderConsentLines generates human-readable descriptions for consent UI.
func (r *RARRegistry) RenderConsentLines(details []AuthorizationDetail) []ConsentLine {
	lines := make([]ConsentLine, 0, len(details))
	for _, d := range details {
		if h, ok := r.handlers[d.Type]; ok {
			lines = append(lines, h.RenderForConsent(&d))
		} else {
			lines = append(lines, ConsentLine{
				Type:    d.Type,
				Title:   d.Type,
				Details: fmt.Sprintf("Request access: %v", d.Actions),
			})
		}
	}
	return lines
}

// --- Built-in RAR type handlers ---

type userProfileType struct{}

func (h *userProfileType) Type() string { return "user_profile" }
func (h *userProfileType) Validate(d *AuthorizationDetail, clientID string) error {
	if len(d.Actions) == 0 {
		return fmt.Errorf("actions required")
	}
	for _, a := range d.Actions {
		if a != "read" {
			return fmt.Errorf("user_profile only supports 'read' action, got '%s'", a)
		}
	}
	return nil
}
func (h *userProfileType) RenderForConsent(d *AuthorizationDetail) ConsentLine {
	fields := "profile"
	if len(d.Fields) > 0 {
		fields = strings.Join(d.Fields, ", ")
	}
	return ConsentLine{Type: d.Type, Title: "Read Profile", Details: fmt.Sprintf("Read your profile fields: %s", fields)}
}

type userRolesType struct{}

func (h *userRolesType) Type() string { return "user_roles" }
func (h *userRolesType) Validate(d *AuthorizationDetail, clientID string) error {
	for _, a := range d.Actions {
		if a != "read" {
			return fmt.Errorf("user_roles only supports 'read' action")
		}
	}
	return nil
}
func (h *userRolesType) RenderForConsent(d *AuthorizationDetail) ConsentLine {
	return ConsentLine{Type: d.Type, Title: "View Roles", Details: "View your assigned roles and permissions"}
}

type auditEventsType struct{}

func (h *auditEventsType) Type() string { return "audit_events" }
func (h *auditEventsType) Validate(d *AuthorizationDetail, clientID string) error {
	for _, a := range d.Actions {
		if a != "read" && a != "export" {
			return fmt.Errorf("audit_events supports 'read' and 'export', got '%s'", a)
		}
	}
	return nil
}
func (h *auditEventsType) RenderForConsent(d *AuthorizationDetail) ConsentLine {
	actions := strings.Join(d.Actions, "/")
	return ConsentLine{Type: d.Type, Title: "Audit Access", Details: fmt.Sprintf("%s audit event logs", actions)}
}

type appAccessType struct{}

func (h *appAccessType) Type() string { return "app_access" }
func (h *appAccessType) Validate(d *AuthorizationDetail, clientID string) error {
	for _, a := range d.Actions {
		if a != "read" && a != "manage" {
			return fmt.Errorf("app_access supports 'read' and 'manage'")
		}
	}
	return nil
}
func (h *appAccessType) RenderForConsent(d *AuthorizationDetail) ConsentLine {
	app := "application"
	if id, ok := d.Identifier["slug"].(string); ok {
		app = id
	}
	return ConsentLine{Type: d.Type, Title: "App Access", Details: fmt.Sprintf("Access %s application", app)}
}

type paymentInitiationType struct{}

func (h *paymentInitiationType) Type() string { return "payment_initiation" }
func (h *paymentInitiationType) Validate(d *AuthorizationDetail, clientID string) error {
	if len(d.Actions) == 0 {
		return fmt.Errorf("actions required for payment_initiation")
	}
	return nil
}
func (h *paymentInitiationType) RenderForConsent(d *AuthorizationDetail) ConsentLine {
	amount := ""
	if at, ok := d.Constraints["instructedAmount"].(map[string]any); ok {
		amount = fmt.Sprintf("%s %s", at["amount"], at["currency"])
	}
	if amount == "" {
		amount = "unspecified amount"
	}
	return ConsentLine{Type: d.Type, Title: "Payment", Details: fmt.Sprintf("Initiate payment of %s", amount)}
}

type vcIssueType struct{}

func (h *vcIssueType) Type() string { return "vc_issue" }
func (h *vcIssueType) Validate(d *AuthorizationDetail, clientID string) error {
	return nil
}
func (h *vcIssueType) RenderForConsent(d *AuthorizationDetail) ConsentLine {
	return ConsentLine{Type: d.Type, Title: "Issue Credential", Details: "Issue a verifiable credential for your identity"}
}

// --- API handlers ---

// RARConsentPreviewHandler renders human-readable consent lines from authorization_details.
// POST /api/v1/oauth/rar/consent-preview
func RARConsentPreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		AuthorizationDetails []AuthorizationDetail `json:"authorization_details"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	registry := NewRARRegistry()
	if err := registry.ValidateDetails(req.AuthorizationDetails, ""); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	lines := registry.RenderConsentLines(req.AuthorizationDetails)
	writeJSON(w, http.StatusOK, map[string]any{
		"consent_lines": lines,
		"total":         len(lines),
	})
}

// suppress unused
var _ = uuid.Nil
