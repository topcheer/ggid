// schemas.go implements the SCIM 2.0 /Schemas discovery endpoint (RFC 7643 §4).
//
// SCIM clients (Okta, Entra ID, Google Workspace) query /Schemas during
// initial configuration to discover attribute definitions. Without it,
// standards-based provisioning setup fails.
package scim

import (
	"net/http"
	"strings"
)

// Schema URNs (RFC 7643 §3.3, §4.5).
const (
	userSchemaURN       = "urn:ietf:params:scim:schemas:core:2.0:User"
	groupSchemaURN      = "urn:ietf:params:scim:schemas:core:2.0:Group"
	entUserSchemaURN    = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	schemaResourceURN   = "urn:ietf:params:scim:schemas:core:2.0:Schema"
)

type schemaAttribute struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	MultiValued   bool              `json:"multiValued"`
	Description   string            `json:"description,omitempty"`
	Required      bool              `json:"required"`
	CaseExact     bool              `json:"caseExact"`
	Mutability    string            `json:"mutability"`
	Returned      string            `json:"returned"`
	Uniqueness    string            `json:"uniqueness"`
	SubAttributes []schemaAttribute `json:"subAttributes,omitempty"`
}

type schemaResource struct {
	Schemas     []string          `json:"schemas"`
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Attributes  []schemaAttribute `json:"attributes"`
	Meta        map[string]string `json:"meta"`
}

func attr(name, typ string, multiValued, required, caseExact bool, mutability, uniqueness string) schemaAttribute {
	return schemaAttribute{
		Name: name, Type: typ, MultiValued: multiValued, Required: required,
		CaseExact: caseExact, Mutability: mutability, Returned: "default", Uniqueness: uniqueness,
	}
}

func userSchema() schemaResource {
	return schemaResource{
		Schemas:     []string{schemaResourceURN},
		ID:          userSchemaURN,
		Name:        "User",
		Description: "User Account",
		Attributes: []schemaAttribute{
			attr("userName", "string", false, true, false, "readWrite", "server"),
			attr("externalId", "string", false, false, true, "readWrite", "none"),
			{
				Name: "name", Type: "complex", MultiValued: false, Required: false,
				CaseExact: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none",
				SubAttributes: []schemaAttribute{
					attr("givenName", "string", false, false, false, "readWrite", "none"),
					attr("familyName", "string", false, false, false, "readWrite", "none"),
				},
			},
			attr("displayName", "string", false, false, false, "readWrite", "none"),
			attr("nickName", "string", false, false, false, "readWrite", "none"),
			attr("profileUrl", "reference", false, false, false, "readWrite", "none"),
			attr("title", "string", false, false, false, "readWrite", "none"),
			attr("userType", "string", false, false, false, "readWrite", "none"),
			attr("preferredLanguage", "string", false, false, false, "readWrite", "none"),
			attr("locale", "string", false, false, false, "readWrite", "none"),
			attr("timezone", "string", false, false, false, "readWrite", "none"),
			{
				Name: "emails", Type: "complex", MultiValued: true, Required: false,
				CaseExact: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none",
				SubAttributes: []schemaAttribute{
					attr("value", "string", false, false, false, "readWrite", "none"),
					attr("type", "string", false, false, false, "readWrite", "none"),
					attr("primary", "boolean", false, false, false, "readWrite", "none"),
				},
			},
			{
				Name: "phoneNumbers", Type: "complex", MultiValued: true, Required: false,
				CaseExact: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none",
				SubAttributes: []schemaAttribute{
					attr("value", "string", false, false, false, "readWrite", "none"),
					attr("type", "string", false, false, false, "readWrite", "none"),
				},
			},
			attr("active", "boolean", false, false, false, "readWrite", "none"),
		},
		Meta: map[string]string{"resourceType": "Schema", "location": "/scim/v2/Schemas/" + userSchemaURN},
	}
}

func groupSchema() schemaResource {
	return schemaResource{
		Schemas:     []string{schemaResourceURN},
		ID:          groupSchemaURN,
		Name:        "Group",
		Description: "Group",
		Attributes: []schemaAttribute{
			attr("displayName", "string", false, true, false, "readWrite", "none"),
			{
				Name: "members", Type: "complex", MultiValued: true, Required: false,
				CaseExact: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none",
				SubAttributes: []schemaAttribute{
					attr("value", "string", false, false, true, "readWrite", "none"),
					attr("$ref", "reference", false, false, true, "readWrite", "none"),
					attr("display", "string", false, false, false, "readOnly", "none"),
				},
			},
		},
		Meta: map[string]string{"resourceType": "Schema", "location": "/scim/v2/Schemas/" + groupSchemaURN},
	}
}

func enterpriseUserSchema() schemaResource {
	return schemaResource{
		Schemas:     []string{schemaResourceURN},
		ID:          entUserSchemaURN,
		Name:        "EnterpriseUser",
		Description: "Enterprise User",
		Attributes: []schemaAttribute{
			attr("employeeNumber", "string", false, false, false, "readWrite", "none"),
			attr("department", "string", false, false, false, "readWrite", "none"),
			attr("division", "string", false, false, false, "readWrite", "none"),
			{
				Name: "manager", Type: "complex", MultiValued: false, Required: false,
				CaseExact: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none",
				SubAttributes: []schemaAttribute{
					attr("value", "string", false, false, true, "readWrite", "none"),
					attr("$ref", "reference", false, false, true, "readWrite", "none"),
					attr("displayName", "string", false, false, false, "readOnly", "none"),
				},
			},
		},
		Meta: map[string]string{"resourceType": "Schema", "location": "/scim/v2/Schemas/" + entUserSchemaURN},
	}
}

// handleSchemasCollection serves GET /scim/v2/Schemas (RFC 7643 §4).
func (h *Handler) handleSchemasCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeSCIMError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeSCIMJSON(w, http.StatusOK, []schemaResource{userSchema(), groupSchema(), enterpriseUserSchema()})
}

// handleSchemaResource serves GET /scim/v2/Schemas/{schema-urn}.
func (h *Handler) handleSchemaResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeSCIMError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/scim/v2/Schemas/")
	switch id {
	case userSchemaURN:
		writeSCIMJSON(w, http.StatusOK, userSchema())
	case groupSchemaURN:
		writeSCIMJSON(w, http.StatusOK, groupSchema())
	case entUserSchemaURN:
		writeSCIMJSON(w, http.StatusOK, enterpriseUserSchema())
	default:
		writeSCIMError(w, http.StatusNotFound, "schema not found: "+id)
	}
}
