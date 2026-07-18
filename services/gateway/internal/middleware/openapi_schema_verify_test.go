package middleware

import (
	"encoding/json"
	"testing"
)

func TestGenerateOpenAPISpec_SchemasPresent(t *testing.T) {
	spec := GenerateOpenAPISpec()

	// Verify core schemas are present
	if len(spec.Components.Schemas) == 0 {
		t.Fatal("expected non-empty schemas in components")
	}

	requiredSchemas := []string{
		"LoginRequest", "TokenResponse", "RegisterRequest",
		"UserResponse", "CreateUserRequest", "ListUsersResponse",
		"OAuthTokenRequest", "OAuthTokenResponse", "OAuthClientResponse",
		"CreatePolicyRequest", "PolicyResponse",
		"ErrorResponse", "OKResponse",
	}
	for _, name := range requiredSchemas {
		if _, ok := spec.Components.Schemas[name]; !ok {
			t.Errorf("missing schema: %s", name)
		}
	}
}

func TestGenerateOpenAPISpec_EnhancedPathsWithBodies(t *testing.T) {
	spec := GenerateOpenAPISpec()

	// Count paths with requestBody (enhanced operations)
	pathsWithBody := 0
	for _, path := range spec.Paths {
		for _, method := range []any{path.Get, path.Post, path.Put, path.Delete} {
			if method == nil {
				continue
			}
			raw, _ := json.Marshal(method)
			var m map[string]any
			if json.Unmarshal(raw, &m) == nil {
				if _, hasBody := m["requestBody"]; hasBody {
					pathsWithBody++
				}
			}
		}
	}

	// We should have at least 20 enhanced paths with request bodies
	if pathsWithBody < 20 {
		t.Errorf("expected >= 20 paths with requestBody, got %d", pathsWithBody)
	}
	t.Logf("Total paths with requestBody: %d", pathsWithBody)
}

func TestGenerateOpenAPISpec_AuthLoginSchema(t *testing.T) {
	spec := GenerateOpenAPISpec()

	loginPath, ok := spec.Paths["/api/v1/auth/login"]
	if !ok {
		t.Fatal("missing /api/v1/auth/login path")
	}
	if loginPath.Post == nil {
		t.Fatal("missing POST method on /api/v1/auth/login")
	}

	// Verify it has requestBody
	raw, _ := json.Marshal(loginPath.Post)
	var m map[string]any
	json.Unmarshal(raw, &m)

	body, hasBody := m["requestBody"]
	if !hasBody {
		t.Fatal("login POST missing requestBody")
	}

	bodyMap := body.(map[string]any)
	jsonContent := bodyMap["content"].(map[string]any)
	appJSON := jsonContent["application/json"].(map[string]any)
	schema := appJSON["schema"].(map[string]any)
	ref := schema["$ref"].(string)

	if ref != "#/components/schemas/LoginRequest" {
		t.Errorf("expected $ref to LoginRequest, got %s", ref)
	}
}

func TestGenerateOpenAPISpec_ResponseSchemas(t *testing.T) {
	spec := GenerateOpenAPISpec()

	// Check that login has TokenResponse in 200 response
	loginPath := spec.Paths["/api/v1/auth/login"]
	raw, _ := json.Marshal(loginPath.Post)
	var m map[string]any
	json.Unmarshal(raw, &m)

	responses := m["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	content := resp200["content"].(map[string]any)
	appJSON := content["application/json"].(map[string]any)
	schema := appJSON["schema"].(map[string]any)
	ref := schema["$ref"].(string)

	if ref != "#/components/schemas/TokenResponse" {
		t.Errorf("expected TokenResponse in 200, got %s", ref)
	}
}
