package ggid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Policy SDK Tests ---

func TestSDKPolicy_CreateRole(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/roles" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "viewer" {
			t.Errorf("expected key=viewer, got %s", body["key"])
		}
		json.NewEncoder(w).Encode(Role{ID: "r1", Name: body["name"], Key: body["key"]})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	role, err := c.CreateRole(context.Background(), "tok", "Viewer", "viewer", "Read-only access")
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}
	if role.Key != "viewer" {
		t.Errorf("expected key=viewer, got %s", role.Key)
	}
}

func TestSDKPolicy_GetRole(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/roles/r1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Role{ID: "r1", Name: "Admin"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	role, err := c.GetRole(context.Background(), "tok", "r1")
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}
	if role.Name != "Admin" {
		t.Errorf("expected Admin, got %s", role.Name)
	}
}

func TestSDKPolicy_DeleteRole(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/roles/r1" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	if err := c.DeleteRole(context.Background(), "tok", "r1"); err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}
}

func TestSDKPolicy_AssignRevokeRole(t *testing.T) {
	var lastMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	if err := c.AssignRole(context.Background(), "tok", "u1", "r1"); err != nil {
		t.Fatalf("AssignRole failed: %v", err)
	}
	if lastMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", lastMethod)
	}

	if err := c.RevokeRole(context.Background(), "tok", "u1", "r1"); err != nil {
		t.Fatalf("RevokeRole failed: %v", err)
	}
	if lastMethod != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", lastMethod)
	}
}

func TestSDKPolicy_GetUserRoles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/u1/roles" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]Role{
			{ID: "r1", Name: "Admin"},
			{ID: "r2", Name: "Viewer"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	roles, err := c.GetUserRoles(context.Background(), "tok", "u1")
	if err != nil {
		t.Fatalf("GetUserRoles failed: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(roles))
	}
	if roles[0].Name != "Admin" {
		t.Errorf("expected Admin, got %s", roles[0].Name)
	}
}

func TestSDKPolicy_ListPermissions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Permission{
			{ID: "p1", Name: "read:users", Resource: "users", Action: "read"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	perms, err := c.ListPermissions(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListPermissions failed: %v", err)
	}
	if len(perms) != 1 {
		t.Fatalf("expected 1 perm, got %d", len(perms))
	}
	if perms[0].Resource != "users" {
		t.Errorf("expected users, got %s", perms[0].Resource)
	}
}

func TestSDKPolicy_CheckPolicy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policies/check" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(PolicyResult{Allowed: true, Reason: "permit"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	result, err := c.CheckPolicy(context.Background(), "tok", &PolicyCheckRequest{
		Subject:  "user:u1",
		Resource: "documents:d1",
		Action:   "read",
		Context:  map[string]string{"ip": "10.0.0.1"},
	})
	if err != nil {
		t.Fatalf("CheckPolicy failed: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed=true")
	}
}

// --- Identity SDK Tests ---

func TestSDKIdentity_CreateUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(User{ID: "u1", Username: "johndoe", Email: "john@example.com"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	user, err := c.CreateUser(context.Background(), "tok", &CreateUserRequest{
		Username: "johndoe",
		Email:    "john@example.com",
		Password: "S3cure!",
	})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.ID != "u1" {
		t.Errorf("expected u1, got %s", user.ID)
	}
}

func TestSDKIdentity_UpdateUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/u1" || r.Method != http.MethodPut {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(User{ID: "u1", Email: "new@example.com"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	newEmail := "new@example.com"
	user, err := c.UpdateUser(context.Background(), "tok", "u1", &UpdateUserRequest{Email: &newEmail})
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}
	if user.Email != "new@example.com" {
		t.Errorf("expected new email, got %s", user.Email)
	}
}

func TestSDKIdentity_LockUnlockUser(t *testing.T) {
	var lastPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	if err := c.LockUser(context.Background(), "tok", "u1"); err != nil {
		t.Fatalf("LockUser failed: %v", err)
	}
	if lastPath != "/api/v1/users/u1/lock" {
		t.Errorf("expected lock path, got %s", lastPath)
	}

	if err := c.UnlockUser(context.Background(), "tok", "u1"); err != nil {
		t.Fatalf("UnlockUser failed: %v", err)
	}
	if lastPath != "/api/v1/users/u1/unlock" {
		t.Errorf("expected unlock path, got %s", lastPath)
	}
}

func TestSDKIdentity_SearchUsers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "q=john" {
			t.Errorf("expected q=john, got %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode([]User{
			{ID: "u1", Username: "johndoe"},
			{ID: "u2", Username: "johnsmith"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	users, err := c.SearchUsers(context.Background(), "tok", "john")
	if err != nil {
		t.Fatalf("SearchUsers failed: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
}

// --- Org SDK Tests ---

func TestSDKOrg_ListOrganizations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Organization{
			{ID: "o1", Name: "Acme Corp"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	orgs, err := c.ListOrganizations(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListOrganizations failed: %v", err)
	}
	if len(orgs) != 1 || orgs[0].Name != "Acme Corp" {
		t.Errorf("unexpected: %+v", orgs)
	}
}

func TestSDKOrg_CreateOrganization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Organization{ID: "o1", Name: "NewCo"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	org, err := c.CreateOrganization(context.Background(), "tok", "NewCo", "A new company")
	if err != nil {
		t.Fatalf("CreateOrganization failed: %v", err)
	}
	if org.Name != "NewCo" {
		t.Errorf("expected NewCo, got %s", org.Name)
	}
}

func TestSDKOrg_DeleteOrganization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/orgs/o1" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	if err := c.DeleteOrganization(context.Background(), "tok", "o1"); err != nil {
		t.Fatalf("DeleteOrganization failed: %v", err)
	}
}

func TestSDKOrg_Departments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/orgs/o1/departments" && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]Department{
				{ID: "d1", Name: "Engineering"},
			})
			return
		}
		if r.URL.Path == "/api/v1/orgs/o1/departments" && r.Method == http.MethodPost {
			json.NewEncoder(w).Encode(Department{ID: "d2", Name: "Marketing"})
			return
		}
		t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	depts, err := c.ListDepartments(context.Background(), "tok", "o1")
	if err != nil {
		t.Fatalf("ListDepartments failed: %v", err)
	}
	if len(depts) != 1 || depts[0].Name != "Engineering" {
		t.Errorf("unexpected: %+v", depts)
	}

	dept, err := c.CreateDepartment(context.Background(), "tok", "o1", "Marketing", "")
	if err != nil {
		t.Fatalf("CreateDepartment failed: %v", err)
	}
	if dept.Name != "Marketing" {
		t.Errorf("expected Marketing, got %s", dept.Name)
	}
}

func TestSDKOrg_Membership(t *testing.T) {
	var lastMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	if err := c.AddMember(context.Background(), "tok", "o1", "u1", "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}
	if lastMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", lastMethod)
	}

	if err := c.RemoveMember(context.Background(), "tok", "o1", "u1"); err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}
	if lastMethod != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", lastMethod)
	}
}
