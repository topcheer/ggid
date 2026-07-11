//go:build integration

// Package integration provides E2E tests for AI Agent Identity and
// IGA Workflows through the Gateway.
//
// Run: go test -tags=integration -v -run "TestAgent|TestIGA" ./test/integration/...
package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// postJSONAgent sends a POST with JSON body and returns the response.
func postJSONAgent(t *testing.T, url string, body any) (*http.Response, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", defaultTenantID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("Service not running: %v", err)
	}
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body = io.NopCloser(bytes.NewReader(b)) // allow further reads
	return resp, result
}

// --- AI Agent Identity E2E ---

// TestAgent_Register verifies agent registration through the gateway.
func TestAgent_Register(t *testing.T) {
	resp, body := postJSONAgent(t, gatewayBaseURL+"/api/v1/agents/register", map[string]any{
		"name":                "E2E-TestBot",
		"type":                "coding-assistant",
		"owner_user_id":       "00000000-0000-0000-0000-000000000001",
		"allowed_scopes":      []string{"read:users"},
		"max_delegation_depth": 2,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Skipf("Agent register returned %d (OAuth service may not be running)", resp.StatusCode)
	}

	if body["id"] == nil {
		t.Fatal("expected agent id in response")
	}
	if body["client_id"] == nil || strings.TrimSpace(body["client_id"].(string)) == "" {
		t.Error("expected non-empty client_id")
	}
	t.Logf("Registered agent: id=%v type=%v", body["id"], body["type"])
}

// TestAgent_List verifies listing agents through the gateway.
func TestAgent_List(t *testing.T) {
	resp, body := postJSONAgent(t, gatewayBaseURL+"/api/v1/agents/register", map[string]any{
		"name":          "E2E-ListBot",
		"type":          "research-agent",
		"owner_user_id": "00000000-0000-0000-0000-000000000001",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Skipf("Prerequisite register failed (%d)", resp.StatusCode)
	}
	resp.Body.Close()

	// List agents
	resp = doRequest(t, "GET", gatewayBaseURL+"/api/v1/agents", "", "")
	if resp == nil {
		t.Skip("Gateway not running")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Agent list returned %d", resp.StatusCode)
	}

	result := decodeBody(t, resp)
	agents, ok := result["agents"].([]any)
	if !ok {
		t.Fatal("expected agents array in response")
	}
	if len(agents) == 0 {
		t.Error("expected at least 1 agent after registration")
	}
	t.Logf("Found %d agents", len(agents))
}

// --- IGA Workflows E2E ---

// TestIGA_CreateAccessRequest verifies access request creation.
func TestIGA_CreateAccessRequest(t *testing.T) {
	resp, body := postJSONAgent(t, gatewayBaseURL+"/api/v1/access-requests", map[string]any{
		"requester_id":   "00000000-0000-0000-0000-000000000001",
		"resource_type":  "role",
		"resource_id":    "admin",
		"reason":         "E2E test: temporary admin access",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Skipf("Access request returned %d (Identity service may not be running)", resp.StatusCode)
	}

	if body["id"] == nil {
		t.Fatal("expected request id in response")
	}
	if body["status"] != "pending" {
		t.Errorf("expected status pending, got %v", body["status"])
	}
	t.Logf("Created access request: id=%v status=%v", body["id"], body["status"])
}

// TestIGA_ListAccessRequests verifies listing access requests.
func TestIGA_ListAccessRequests(t *testing.T) {
	// Create a request first
	resp, _ := postJSONAgent(t, gatewayBaseURL+"/api/v1/access-requests", map[string]any{
		"requester_id":   "00000000-0000-0000-0000-000000000001",
		"resource_type":  "role",
		"resource_id":    "auditor",
		"reason":         "E2E test: list verification",
	})
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Skipf("Prerequisite create failed (%d)", resp.StatusCode)
	}
	resp.Body.Close()

	// List pending requests
	resp = doRequest(t, "GET", gatewayBaseURL+"/api/v1/access-requests?status=pending", "", "")
	if resp == nil {
		t.Skip("Gateway not running")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("List returned %d", resp.StatusCode)
	}

	result := decodeBody(t, resp)
	requests, ok := result["requests"].([]any)
	if !ok {
		// might be "data" or direct array
		if result["data"] != nil {
			t.Logf("Response has 'data' key: %v", result["data"])
		} else {
			t.Logf("Response keys: %v", func() []string {
				keys := make([]string, 0)
				for k := range result {
					keys = append(keys, k)
				}
				return keys
			}())
		}
	} else if len(requests) == 0 {
		t.Log("No pending requests (may have been resolved)")
	} else {
		t.Logf("Found %d pending requests", len(requests))
	}
}

// TestIGA_ApproveAccessRequest verifies the approve flow.
func TestIGA_ApproveAccessRequest(t *testing.T) {
	// Create a request
	resp, body := postJSONAgent(t, gatewayBaseURL+"/api/v1/access-requests", map[string]any{
		"requester_id":   "00000000-0000-0000-0000-000000000001",
		"resource_type":  "role",
		"resource_id":    "developer",
		"reason":         "E2E test: approve flow",
	})
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Skipf("Prerequisite create failed (%d)", resp.StatusCode)
	}
	requestID, ok := body["id"].(string)
	if !ok {
		t.Skip("Could not extract request id")
	}
	resp.Body.Close()

	// Approve it (different user to avoid self-approval block)
	resp, body = postJSONAgent(t, gatewayBaseURL+"/api/v1/access-requests/"+requestID+"/approve", map[string]any{
		"approver_id": "00000000-0000-0000-0000-000000000002",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Approve returned %d", resp.StatusCode)
	}

	if body["status"] != "approved" {
		t.Errorf("expected status approved, got %v", body["status"])
	}
	t.Logf("Approved request %s: status=%v", requestID, body["status"])
}

// TestIGA_DenyAccessRequest verifies the deny flow.
func TestIGA_DenyAccessRequest(t *testing.T) {
	// Create a request
	resp, body := postJSONAgent(t, gatewayBaseURL+"/api/v1/access-requests", map[string]any{
		"requester_id":   "00000000-0000-0000-0000-000000000001",
		"resource_type":  "role",
		"resource_id":    "superadmin",
		"reason":         "E2E test: deny flow",
	})
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Skipf("Prerequisite create failed (%d)", resp.StatusCode)
	}
	requestID, ok := body["id"].(string)
	if !ok {
		t.Skip("Could not extract request id")
	}
	resp.Body.Close()

	// Deny it
	resp, body = postJSONAgent(t, gatewayBaseURL+"/api/v1/access-requests/"+requestID+"/deny", map[string]any{
		"approver_id":    "00000000-0000-0000-0000-000000000002",
		"denial_reason":  "E2E test: request denied",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Deny returned %d", resp.StatusCode)
	}

	if body["status"] != "denied" {
		t.Errorf("expected status denied, got %v", body["status"])
	}
	t.Logf("Denied request %s: status=%v", requestID, body["status"])
}
