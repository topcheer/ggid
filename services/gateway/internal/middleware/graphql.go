// Package middleware implements HTTP middleware for the API Gateway.
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GraphQLRequest represents a GraphQL query request.
type GraphQLRequest struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operationName,omitempty"`
}

// GraphQLResponse represents a GraphQL response.
type GraphQLResponse struct {
	Data   map[string]any `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error.
type GraphQLError struct {
	Message string `json:"message"`
	Path    []string `json:"path,omitempty"`
}

// GraphQLResolver resolves a GraphQL field by calling backend REST APIs.
type GraphQLResolver struct {
	BackendURLs map[string]string // type → URL (e.g. "users" → "http://localhost:8081")
	HTTPClient  *http.Client
}

// NewGraphQLResolver creates a resolver from backend URLs.
func NewGraphQLResolver(urls map[string]string) *GraphQLResolver {
	return &GraphQLResolver{
		BackendURLs: urls,
		HTTPClient:  &http.Client{},
	}
}

// GraphQLHandler handles POST /graphql requests.
// It supports a simple subset of GraphQL: top-level field resolution by type name.
// Example queries:
//   { users { id email name } }
//   { user(id: "123") { id email } }
//   { roles { id name key } }
//   { orgs { id name slug } }
func (r *GraphQLResolver) GraphQLHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			writeGraphQLError(w, http.StatusMethodNotAllowed, "only POST is supported")
			return
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeGraphQLError(w, http.StatusBadRequest, "failed to read body")
			return
		}
		defer req.Body.Close()

		var gqlReq GraphQLRequest
		if err := json.Unmarshal(body, &gqlReq); err != nil {
			writeGraphQLError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		// Resolve the query
		data, errs := r.resolveQuery(req, gqlReq.Query)
		resp := GraphQLResponse{Data: data}
		if len(errs) > 0 {
			resp.Errors = errs
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// resolveQuery parses the GraphQL query and resolves top-level fields.
func (r *GraphQLResolver) resolveQuery(req *http.Request, query string) (map[string]any, []GraphQLError) {
	fields := parseGraphQLFields(query)
	if len(fields) == 0 {
		return nil, []GraphQLError{{Message: "no fields found in query"}}
	}

	data := make(map[string]any)
	var errors []GraphQLError

	tenantID, _ := TenantIDFromRequest(req)
	authHeader := req.Header.Get("Authorization")

	for _, field := range fields {
		result, err := r.resolveField(req.Context(), field, tenantID, authHeader)
		if err != nil {
			errors = append(errors, GraphQLError{
				Message: err.Error(),
				Path:    []string{field.Name},
			})
			continue
		}
		data[field.Name] = result
	}

	return data, errors
}

// resolveField resolves a single GraphQL field by proxying to the appropriate backend.
func (r *GraphQLResolver) resolveField(ctx context.Context, field graphqlField, tenantID, authHeader string) (any, error) {
	backendURL, ok := r.BackendURLs[field.Type]
	if !ok {
		return nil, fmt.Errorf("no backend configured for type '%s'", field.Type)
	}

	// Build the REST URL
	url := backendURL + field.Path
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if authHeader != "" {
		httpReq.Header.Set("Authorization", authHeader)
	}
	if tenantID != "" {
		httpReq.Header.Set("X-Tenant-ID", tenantID)
	}

	resp, err := r.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("backend request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read backend response: %w", err)
	}

	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		// Return raw body as string if not JSON
		return string(body), nil
	}
	return result, nil
}

// graphqlField represents a parsed top-level GraphQL field.
type graphqlField struct {
	Name string // field name (e.g. "users", "user")
	Type string // backend type (e.g. "users", "roles", "orgs")
	Path string // REST path (e.g. "/api/v1/users", "/api/v1/users/123")
}

// parseGraphQLFields extracts top-level field names from a GraphQL query.
// This is a simple parser — not a full GraphQL engine.
func parseGraphQLFields(query string) []graphqlField {
	// Remove query wrapper
	query = strings.TrimSpace(query)
	query = strings.TrimPrefix(query, "query")
	query = strings.TrimPrefix(query, "mutation")
	// Remove operation name if present
	if idx := strings.Index(query, "{"); idx >= 0 {
		query = query[idx:]
	}

	var fields []graphqlField
	// Simple regex-like extraction: { fieldName(args) { ... } }
	lines := strings.Split(query, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "{") || strings.HasPrefix(line, "}") {
			continue
		}
		// Extract field name
		fieldName := extractFieldName(line)
		if fieldName == "" {
			continue
		}

		args := extractArgs(line)
		gf := graphqlField{
			Name: fieldName,
			Type: fieldName,
			Path: typeToPath(fieldName, args),
		}
		fields = append(fields, gf)
	}
	return fields
}

func extractFieldName(line string) string {
	// Remove leading/trailing braces
	line = strings.Trim(line, " {}")
	// Get the first word
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func extractArgs(line string) map[string]string {
	args := make(map[string]string)
	// Find content within parentheses
	start := strings.Index(line, "(")
	if start < 0 {
		return args
	}
	end := strings.Index(line[start:], ")")
	if end < 0 {
		return args
	}
	argStr := line[start+1 : start+end]
	// Parse key: value pairs
	for _, pair := range strings.Split(argStr, ",") {
		pair = strings.TrimSpace(pair)
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.Trim(strings.TrimSpace(kv[1]), "\"")
			args[key] = val
		}
	}
	return args
}

func typeToPath(typeName string, args map[string]string) string {
	// Map GraphQL type names to REST paths
	switch typeName {
	case "users", "user":
		if id, ok := args["id"]; ok {
			return "/api/v1/users/" + id
		}
		return "/api/v1/users"
	case "roles", "role":
		if id, ok := args["id"]; ok {
			return "/api/v1/roles/" + id
		}
		return "/api/v1/roles"
	case "orgs", "org":
		if id, ok := args["id"]; ok {
			return "/api/v1/orgs/" + id
		}
		return "/api/v1/orgs"
	case "audit":
		return "/api/v1/audit"
	default:
		return "/api/v1/" + typeName
	}
}

func writeGraphQLError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(GraphQLResponse{
		Errors: []GraphQLError{{Message: msg}},
	})
}
