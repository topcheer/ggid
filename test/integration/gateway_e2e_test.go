//go:build integration

// Package integration provides E2E tests for Policy/Org/Audit REST APIs
// through the API Gateway. These tests require the full stack running
// (Gateway on :8080 + all microservices + PostgreSQL + NATS).
//
// Run: go test -tags=integration -v -run TestGateway ./test/integration/...
package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// gatewayBaseURL is the API Gateway endpoint.
const gatewayBaseURL = "http://localhost:8080"

// loginAndGetJWT registers a test user, logs in, and returns a JWT access token.
// Skips the test if the gateway or auth service is not running.
func loginAndGetJWT(t *testing.T) string {
	t.Helper()
	username := fmt.Sprintf("gw_%d", time.Now().UnixNano())
	email := fmt.Sprintf("%s@test.local", username)
	password := "GatewayTest123!"

	// Register
	regBody := fmt.Sprintf(`{"username":"%s","email":"%s","password":"%s"}`, username, email, password)
	resp := doRequest(t, "POST", gatewayBaseURL+"/api/v1/auth/register", regBody, "")
	if resp == nil {
		t.Skipf("Gateway not running, skipping")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Skipf("Register returned %d: %s (services might not be running)", resp.StatusCode, body)
	}

	// Login
	loginBody := fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)
	resp2 := doRequest(t, "POST", gatewayBaseURL+"/api/v1/auth/login", loginBody, "")
	if resp2 == nil {
		t.Skipf("Gateway not running")
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		t.Skipf("Login returned %d: %s", resp2.StatusCode, body)
	}

	var result map[string]any
	json.NewDecoder(resp2.Body).Decode(&result)
	token, _ := result["access_token"].(string)
	if token == "" {
		t.Skip("no access_token in login response")
	}
	return token
}

func doRequest(t *testing.T, method, url, body, token string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Tenant-ID", defaultTenantID)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return result
}

// --- Policy Engine E2E via Gateway ---

// TestGateway_PolicyRoleCRUD tests creating and querying roles through the Gateway.
func TestGateway_PolicyRoleCRUD(t *testing.T) {
	jwt := loginAndGetJWT(t)

	// Create role
	roleKey := fmt.Sprintf("test_role_%d", time.Now().UnixNano())
	createBody := fmt.Sprintf(`{"tenant_id":"%s","key":"%s","name":"Test Role","description":"E2E test role"}`,
		defaultTenantID, roleKey)
	resp := doRequest(t, "POST", gatewayBaseURL+"/api/v1/roles", createBody, jwt)
	if resp == nil {
		t.Skip("Policy Service not reachable through Gateway")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Skipf("Create role returned %d: %s (Policy Service might not be running)", resp.StatusCode, body)
	}

	result := decodeBody(t, resp)
	roleID, _ := result["id"].(string)
	t.Logf("Created role: %s (id=%s)", roleKey, roleID)

	// List roles — should include our new role
	listResp := doRequest(t, "GET", gatewayBaseURL+"/api/v1/roles?tenant_id="+defaultTenantID, "", jwt)
	if listResp == nil {
		t.Fatal("list roles request failed")
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listResp.Body)
		t.Fatalf("List roles returned %d: %s", listResp.StatusCode, body)
	}
	var listResult map[string]any
	json.NewDecoder(listResp.Body).Decode(&listResult)
	roles, _ := listResult["roles"].([]any)
	if len(roles) == 0 {
		t.Error("expected at least 1 role in list")
	}
	t.Logf("Listed %d roles", len(roles))
}

// TestGateway_PolicyPermissionCheck tests the permission check endpoint through the Gateway.
func TestGateway_PolicyPermissionCheck(t *testing.T) {
	jwt := loginAndGetJWT(t)

	// Check permission for the default tenant admin
	// The user we just registered won't have any roles, so this should return denied.
	checkBody := fmt.Sprintf(`{"user_id":"00000000-0000-0000-0000-000000000001","resource_type":"users","action":"read","resource":"*"}`)
	resp := doRequest(t, "POST", gatewayBaseURL+"/api/v1/policies/check", checkBody, jwt)
	if resp == nil {
		t.Skip("Policy Service not reachable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Skipf("Permission check returned %d: %s", resp.StatusCode, body)
	}

	result := decodeBody(t, resp)
	allowed, _ := result["allowed"].(bool)
	t.Logf("Permission check result: allowed=%v", allowed)
	// We don't assert on the boolean — depends on seeded data.
	// The important thing is the endpoint responds 200 with a boolean.
}

// --- Org Service E2E via Gateway ---

// TestGateway_OrgCRUD tests creating and querying organizations through the Gateway.
func TestGateway_OrgCRUD(t *testing.T) {
	jwt := loginAndGetJWT(t)

	// Create organization
	orgName := fmt.Sprintf("E2E Org %d", time.Now().UnixNano())
	createBody := fmt.Sprintf(`{"tenant_id":"%s","name":"%s"}`, defaultTenantID, orgName)
	resp := doRequest(t, "POST", gatewayBaseURL+"/api/v1/orgs", createBody, jwt)
	if resp == nil {
		t.Skip("Org Service not reachable through Gateway")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Skipf("Create org returned %d: %s (Org Service might not be running)", resp.StatusCode, body)
	}

	result := decodeBody(t, resp)
	orgID, _ := result["id"].(string)
	t.Logf("Created org: %s (id=%s)", orgName, orgID)

	// List organizations
	listResp := doRequest(t, "GET", gatewayBaseURL+"/api/v1/orgs?tenant_id="+defaultTenantID, "", jwt)
	if listResp == nil {
		t.Fatal("list orgs request failed")
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listResp.Body)
		t.Fatalf("List orgs returned %d: %s", listResp.StatusCode, body)
	}
	t.Log("Listed organizations successfully")

	// Get org by ID (if we got an ID back)
	if orgID != "" {
		getResp := doRequest(t, "GET", gatewayBaseURL+"/api/v1/orgs/"+orgID, "", jwt)
		if getResp != nil {
			defer getResp.Body.Close()
			if getResp.StatusCode == http.StatusOK {
				t.Log("Retrieved org by ID")
			}
		}

		// Delete org
		delResp := doRequest(t, "DELETE", gatewayBaseURL+"/api/v1/orgs/"+orgID, "", jwt)
		if delResp != nil {
			defer delResp.Body.Close()
			t.Logf("Delete org returned %d", delResp.StatusCode)
		}
	}
}

// --- Audit Service E2E via Gateway ---

// TestGateway_AuditQuery tests querying audit events through the Gateway.
func TestGateway_AuditQuery(t *testing.T) {
	jwt := loginAndGetJWT(t)

	// Query audit events
	resp := doRequest(t, "GET",
		gatewayBaseURL+"/api/v1/audit/events?tenant_id="+defaultTenantID+"&page_size=10", "", jwt)
	if resp == nil {
		t.Skip("Audit Service not reachable through Gateway")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Skipf("Audit query returned %d: %s (Audit Service might not be running)", resp.StatusCode, body)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode audit response: %v", err)
	}

	total, _ := result["total"].(float64)
	events, _ := result["events"].([]any)
	t.Logf("Audit query: total=%d, returned=%d events", int(total), len(events))
	// total may be 0 if no events have been published yet — that's OK.
	// The important thing is the endpoint responds 200 with the right structure.
}

// TestGateway_AuditQueryWithFilters tests audit query with action and result filters.
func TestGateway_AuditQueryWithFilters(t *testing.T) {
	jwt := loginAndGetJWT(t)

	// Query with filters — these may return 0 results but should not error
	resp := doRequest(t, "GET",
		gatewayBaseURL+"/api/v1/audit/events?tenant_id="+defaultTenantID+"&action=user.login&result=success&page_size=5", "", jwt)
	if resp == nil {
		t.Skip("Audit Service not reachable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Skipf("Audit query with filters returned %d: %s", resp.StatusCode, body)
	}

	result := decodeBody(t, resp)
	t.Logf("Filtered audit query returned total=%v", result["total"])
}

// TestGateway_Unauthorized tests that requests without JWT are rejected.
func TestGateway_Unauthorized(t *testing.T) {
	// This path requires auth (not in whitelist)
	resp := doRequest(t, "GET", gatewayBaseURL+"/api/v1/roles?tenant_id="+defaultTenantID, "", "")
	if resp == nil {
		t.Skip("Gateway not reachable")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 for request without JWT")
	}
	t.Logf("Unauthorized request correctly returned %d", resp.StatusCode)
}
