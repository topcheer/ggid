package scim

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	gerr "github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/ggid/ggid/services/identity/internal/repository"
	"github.com/ggid/ggid/services/identity/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- In-memory mock repo for SCIM bulk tests ---

type bulkMockRepo struct {
	users         map[uuid.UUID]*domain.User
	emails        map[uuid.UUID]*domain.UserEmail
	createUserErr error
}

func newBulkMockRepo() *bulkMockRepo {
	return &bulkMockRepo{
		users:  make(map[uuid.UUID]*domain.User),
		emails: make(map[uuid.UUID]*domain.UserEmail),
	}
}

func (m *bulkMockRepo) Pool() *pgxpool.Pool { return nil }

func (m *bulkMockRepo) CreateUser(_ context.Context, user *domain.User) error {
	if m.createUserErr != nil {
		return m.createUserErr
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}
func (m *bulkMockRepo) GetUserByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, gerr.NotFound("user", id.String())
	}
	return u, nil
}
func (m *bulkMockRepo) GetUserByUsername(_ context.Context, _ uuid.UUID, username string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, gerr.NotFound("user", username)
}
func (m *bulkMockRepo) GetUserByEmail(_ context.Context, _ uuid.UUID, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, gerr.NotFound("email", email)
}
func (m *bulkMockRepo) UpdateUser(_ context.Context, _ uuid.UUID, id uuid.UUID, input *domain.UpdateUserInput) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, gerr.NotFound("user", id.String())
	}
	if input.DisplayName != nil {
		u.DisplayName = *input.DisplayName
	}
	u.UpdatedAt = time.Now()
	return u, nil
}
func (m *bulkMockRepo) DeleteUser(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	u, ok := m.users[id]
	if !ok {
		return gerr.NotFound("user", id.String())
	}
	u.Status = domain.UserStatusDeleted
	return nil
}
func (m *bulkMockRepo) ListUsers(_ context.Context, filter *domain.ListUsersFilter) (*domain.ListUsersResult, error) {
	var result []*domain.User
	for _, u := range m.users {
		result = append(result, u)
	}
	return &domain.ListUsersResult{Users: result, Total: len(result)}, nil
}
func (m *bulkMockRepo) SetUserStatus(_ context.Context, _ uuid.UUID, id uuid.UUID, status domain.UserStatus) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, gerr.NotFound("user", id.String())
	}
	u.Status = status
	return u, nil
}
func (m *bulkMockRepo) UpdateLastLogin(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) error {
	return nil
}
func (m *bulkMockRepo) UpdatePassword(_ context.Context, _ uuid.UUID, id uuid.UUID, hash string) error {
	u, ok := m.users[id]
	if !ok {
		return gerr.NotFound("user", id.String())
	}
	u.PasswordHash = hash
	return nil
}
func (m *bulkMockRepo) GetCredentialByUsername(_ context.Context, _ uuid.UUID, username string) (*authprovider.LocalCredential, error) {
	return nil, gerr.NotFound("user", username)
}
func (m *bulkMockRepo) ListUserEmails(_ context.Context, _ uuid.UUID, userID uuid.UUID) ([]*domain.UserEmail, error) {
	var result []*domain.UserEmail
	for _, e := range m.emails {
		if e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}
func (m *bulkMockRepo) AddUserEmail(_ context.Context, _ uuid.UUID, userID uuid.UUID, email string) (*domain.UserEmail, error) {
	e := &domain.UserEmail{ID: uuid.New(), UserID: userID, Email: email, CreatedAt: time.Now()}
	m.emails[e.ID] = e
	return e, nil
}
func (m *bulkMockRepo) RemoveUserEmail(_ context.Context, _ uuid.UUID, userID uuid.UUID, email string) error {
	for id, e := range m.emails {
		if e.UserID == userID && e.Email == email {
			delete(m.emails, id)
			return nil
		}
	}
	return gerr.NotFound("email", email)
}
func (m *bulkMockRepo) SetPrimaryEmail(_ context.Context, _ uuid.UUID, userID uuid.UUID, emailID uuid.UUID) (*domain.UserEmail, error) {
	for _, e := range m.emails {
		if e.UserID == userID {
			e.IsPrimary = (e.ID == emailID)
		}
	}
	e, ok := m.emails[emailID]
	if !ok {
		return nil, gerr.NotFound("email", emailID.String())
	}
	return e, nil
}
func (m *bulkMockRepo) GetUserByEmailID(_ context.Context, _ uuid.UUID, emailID uuid.UUID) (*domain.UserEmail, error) {
	e, ok := m.emails[emailID]
	if !ok {
		return nil, gerr.NotFound("email", emailID.String())
	}
	return e, nil
}
func (m *bulkMockRepo) CreateEmailVerificationToken(_ context.Context, token *domain.EmailVerificationToken) error {
	return nil
}
func (m *bulkMockRepo) ConsumeEmailVerificationToken(_ context.Context, _ string) (*domain.EmailVerificationToken, error) {
	return nil, gerr.InvalidArgument("not found")
}
func (m *bulkMockRepo) ListExternalIdentities(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]*domain.ExternalIdentity, error) {
	return nil, nil
}
func (m *bulkMockRepo) LinkExternalIdentity(_ context.Context, ei *domain.ExternalIdentity) (*domain.ExternalIdentity, error) {
	return ei, nil
}
func (m *bulkMockRepo) UnlinkExternalIdentity(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID) error {
	return gerr.NotFound("external identity", "")
}
func (m *bulkMockRepo) FindExternalIdentity(_ context.Context, _ uuid.UUID, _, _ string) (*domain.ExternalIdentity, error) {
	return nil, gerr.NotFound("external identity", "")
}

// Compile-time interface check.
var _ repository.UserRepository = (*bulkMockRepo)(nil)

func newBulkTestHandler() *Handler {
	repo := newBulkMockRepo()
	svc := service.NewIdentityService(repo)
	return &Handler{svc: svc}
}

func bulkTestCtx() context.Context {
	return tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.MustParse(testTenantID),
		IsolationLevel: tenant.IsolationShared,
	})
}

// =====================
// Tests
// =====================

func TestHandleBulk_CreateUser(t *testing.T) {
	h := newBulkTestHandler()
	body := `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
		"Operations": [
			{
				"method": "POST",
				"path": "/Users",
				"bulkId": "bulk-1",
				"data": {"userName": "bulkuser1", "emails": [{"value": "bulk1@test.com"}], "displayName": "Bulk User 1"}
			}
		]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp BulkResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(resp.Operations))
	}
	op := resp.Operations[0]
	if op.Status != "201" {
		t.Errorf("expected status 201, got %s", op.Status)
	}
	if op.Method != "POST" {
		t.Errorf("expected method POST, got %s", op.Method)
	}
	if op.BulkID != "bulk-1" {
		t.Errorf("expected bulkId bulk-1, got %s", op.BulkID)
	}
	if !strings.Contains(op.Location, "/scim/v2/Users/") {
		t.Errorf("expected location to contain /scim/v2/Users/, got %s", op.Location)
	}
	if op.Version == "" {
		t.Error("expected non-empty version")
	}
}

func TestHandleBulk_CreateUser_InvalidData(t *testing.T) {
	h := newBulkTestHandler()
	body := `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
		"Operations": [
			{
				"method": "POST",
				"path": "/Users",
				"bulkId": "bad-1",
				"data": "not-valid-json-object"
			}
		]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Operations) != 1 {
		t.Fatalf("expected 1 op, got %d", len(resp.Operations))
	}
	if resp.Operations[0].Status != "400" {
		t.Errorf("expected status 400, got %s", resp.Operations[0].Status)
	}
}

func TestHandleBulk_CreateUser_Duplicate(t *testing.T) {
	h := newBulkTestHandler()
	// First create succeeds
	body := `{
		"Operations": [{
			"method": "POST", "path": "/Users", "bulkId": "a",
			"data": {"userName": "dupuser", "emails": [{"value": "dup@test.com"}]}
		}]
	}`
	req1 := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req1.Header.Set("X-Tenant-ID", testTenantID)
	w1 := httptest.NewRecorder()
	h.handleBulk(w1, req1)

	var resp1 BulkResponse
	json.Unmarshal(w1.Body.Bytes(), &resp1)
	if resp1.Operations[0].Status != "201" {
		t.Fatalf("first create should succeed, got %s", resp1.Operations[0].Status)
	}

	// Second create for same username should fail
	req2 := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req2.Header.Set("X-Tenant-ID", testTenantID)
	w2 := httptest.NewRecorder()
	h.handleBulk(w2, req2)

	var resp2 BulkResponse
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.Operations[0].Status != "409" {
		t.Errorf("expected 409 for duplicate, got %s", resp2.Operations[0].Status)
	}
}

func TestHandleBulk_DeleteUser(t *testing.T) {
	h := newBulkTestHandler()
	// First, create a user to get its ID
	createBody := `{
		"Operations": [{
			"method": "POST", "path": "/Users", "bulkId": "c",
			"data": {"userName": "deluser", "emails": [{"value": "del@test.com"}]}
		}]
	}`
	reqC := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(createBody))
	reqC.Header.Set("X-Tenant-ID", testTenantID)
	wC := httptest.NewRecorder()
	h.handleBulk(wC, reqC)
	var respC BulkResponse
	json.Unmarshal(wC.Body.Bytes(), &respC)
	userID := strings.TrimPrefix(respC.Operations[0].Location, "/scim/v2/Users/")

	// Now delete
	delBody := `{
		"Operations": [{
			"method": "DELETE", "path": "/Users/` + userID + `", "bulkId": "d"
		}]
	}`
	reqD := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(delBody))
	reqD.Header.Set("X-Tenant-ID", testTenantID)
	wD := httptest.NewRecorder()
	h.handleBulk(wD, reqD)

	var respD BulkResponse
	json.Unmarshal(wD.Body.Bytes(), &respD)
	if respD.Operations[0].Status != "204" {
		t.Errorf("expected 204 for delete, got %s", respD.Operations[0].Status)
	}
}

func TestHandleBulk_DeleteUser_NotFound(t *testing.T) {
	h := newBulkTestHandler()
	body := `{
		"Operations": [{
			"method": "DELETE", "path": "/Users/` + uuid.New().String() + `", "bulkId": "d"
		}]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Operations[0].Status != "404" {
		t.Errorf("expected 404, got %s", resp.Operations[0].Status)
	}
}

func TestHandleBulk_PutUser(t *testing.T) {
	h := newBulkTestHandler()
	// Create first
	createBody := `{
		"Operations": [{
			"method": "POST", "path": "/Users", "bulkId": "c",
			"data": {"userName": "putuser", "emails": [{"value": "put@test.com"}], "displayName": "Before"}
		}]
	}`
	reqC := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(createBody))
	reqC.Header.Set("X-Tenant-ID", testTenantID)
	wC := httptest.NewRecorder()
	h.handleBulk(wC, reqC)
	var respC BulkResponse
	json.Unmarshal(wC.Body.Bytes(), &respC)
	userID := strings.TrimPrefix(respC.Operations[0].Location, "/scim/v2/Users/")

	// Replace
	putBody := `{
		"Operations": [{
			"method": "PUT", "path": "/Users/` + userID + `", "bulkId": "p",
			"data": {"userName": "putuser", "displayName": "After Update"}
		}]
	}`
	reqP := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(putBody))
	reqP.Header.Set("X-Tenant-ID", testTenantID)
	wP := httptest.NewRecorder()
	h.handleBulk(wP, reqP)

	var respP BulkResponse
	json.Unmarshal(wP.Body.Bytes(), &respP)
	if respP.Operations[0].Status != "200" {
		t.Errorf("expected 200, got %s", respP.Operations[0].Status)
	}
}

func TestHandleBulk_PutUser_NotFound(t *testing.T) {
	h := newBulkTestHandler()
	body := `{
		"Operations": [{
			"method": "PUT", "path": "/Users/` + uuid.New().String() + `",
			"data": {"userName": "x", "displayName": "Y"}
		}]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Operations[0].Status != "404" {
		t.Errorf("expected 404, got %s", resp.Operations[0].Status)
	}
}

func TestHandleBulk_PatchUser(t *testing.T) {
	h := newBulkTestHandler()
	// Create first
	createBody := `{
		"Operations": [{
			"method": "POST", "path": "/Users", "bulkId": "c",
			"data": {"userName": "patchuser", "emails": [{"value": "patch@test.com"}], "displayName": "Old"}
		}]
	}`
	reqC := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(createBody))
	reqC.Header.Set("X-Tenant-ID", testTenantID)
	wC := httptest.NewRecorder()
	h.handleBulk(wC, reqC)
	var respC BulkResponse
	json.Unmarshal(wC.Body.Bytes(), &respC)
	userID := strings.TrimPrefix(respC.Operations[0].Location, "/scim/v2/Users/")

	// Patch
	patchBody := `{
		"Operations": [{
			"method": "PATCH", "path": "/Users/` + userID + `", "bulkId": "p",
			"data": {"Operations": [{"op": "replace", "path": "displayName", "value": "New Name"}]}
		}]
	}`
	reqP := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(patchBody))
	reqP.Header.Set("X-Tenant-ID", testTenantID)
	wP := httptest.NewRecorder()
	h.handleBulk(wP, reqP)

	var respP BulkResponse
	json.Unmarshal(wP.Body.Bytes(), &respP)
	if respP.Operations[0].Status != "200" {
		t.Errorf("expected 200, got %s: %s", respP.Operations[0].Status, wP.Body.String())
	}
}

func TestHandleBulk_PatchUser_NotFound(t *testing.T) {
	h := newBulkTestHandler()
	body := `{
		"Operations": [{
			"method": "PATCH", "path": "/Users/` + uuid.New().String() + `",
			"data": {"Operations": [{"op": "replace", "path": "displayName", "value": "X"}]}
		}]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Operations[0].Status != "404" {
		t.Errorf("expected 404, got %s", resp.Operations[0].Status)
	}
}

func TestHandleBulk_UnsupportedMethod(t *testing.T) {
	h := newBulkTestHandler()
	body := `{
		"Operations": [{
			"method": "GET", "path": "/Users", "bulkId": "bad"
		}]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Operations[0].Status != "400" {
		t.Errorf("expected 400 for GET, got %s", resp.Operations[0].Status)
	}
}

func TestHandleBulk_MalformedJSON(t *testing.T) {
	h := newBulkTestHandler()
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader("{invalid"))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", w.Code)
	}
}

func TestHandleBulk_NoOperations(t *testing.T) {
	h := newBulkTestHandler()
	body := `{"schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"], "Operations": []}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty ops, got %d", w.Code)
	}
}

func TestHandleBulk_TooManyOperations(t *testing.T) {
	h := newBulkTestHandler()
	// Build > maxBulkOperations operations
	var ops []string
	for i := 0; i <= maxBulkOperations; i++ {
		ops = append(ops, `{"method":"POST","path":"/Users","bulkId":"b"}`)
	}
	body := `{"Operations": [` + strings.Join(ops, ",") + `]}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 for too many ops, got %d", w.Code)
	}
}

func TestHandleBulk_FailOnErrors(t *testing.T) {
	h := newBulkTestHandler()
	// 3 ops: all with unsupported method so they all fail
	failOn := 2
	body := `{
		"failOnErrors": 2,
		"Operations": [
			{"method": "GET", "path": "/Users", "bulkId": "1"},
			{"method": "GET", "path": "/Users", "bulkId": "2"},
			{"method": "GET", "path": "/Users", "bulkId": "3"}
		]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	// failOnErrors = 2, so we should stop after 2 errors
	if len(resp.Operations) != failOn {
		t.Errorf("expected %d operations (stopped at failOnErrors), got %d", failOn, len(resp.Operations))
	}
}

func TestHandleBulk_MultipleOperations(t *testing.T) {
	h := newBulkTestHandler()
	body := `{
		"Operations": [
			{"method": "POST", "path": "/Users", "bulkId": "m1",
			 "data": {"userName": "multi1", "emails": [{"value": "m1@test.com"}], "displayName": "M1"}},
			{"method": "POST", "path": "/Users", "bulkId": "m2",
			 "data": {"userName": "multi2", "emails": [{"value": "m2@test.com"}], "displayName": "M2"}},
			{"method": "POST", "path": "/Users", "bulkId": "m3",
			 "data": {"userName": "multi3", "emails": [{"value": "m3@test.com"}], "displayName": "M3"}}
		]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Operations) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(resp.Operations))
	}
	for i, op := range resp.Operations {
		if op.Status != "201" {
			t.Errorf("op[%d]: expected 201, got %s", i, op.Status)
		}
		if op.Location == "" {
			t.Errorf("op[%d]: expected non-empty location", i)
		}
	}
}

func TestHandleBulk_MissingTenant(t *testing.T) {
	h := newBulkTestHandler()
	body := `{"Operations": [{"method": "POST", "path": "/Users", "data": {"userName": "x"}}]}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	// No X-Tenant-ID header
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing tenant, got %d", w.Code)
	}
}

func TestHandleBulk_InvalidTenant(t *testing.T) {
	h := newBulkTestHandler()
	body := `{"Operations": [{"method": "POST", "path": "/Users", "data": {"userName": "x"}}]}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", "not-a-uuid")
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid tenant, got %d", w.Code)
	}
}

func TestHandleBulk_MixedSuccessAndFailure(t *testing.T) {
	h := newBulkTestHandler()
	randomID := uuid.New().String()
	body := `{
		"Operations": [
			{"method": "POST", "path": "/Users", "bulkId": "ok",
			 "data": {"userName": "mixuser", "emails": [{"value": "mix@test.com"}]}},
			{"method": "DELETE", "path": "/Users/` + randomID + `", "bulkId": "fail"},
			{"method": "GET", "path": "/Users", "bulkId": "unsupported"}
		]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Operations) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(resp.Operations))
	}
	if resp.Operations[0].Status != "201" {
		t.Errorf("op[0]: expected 201, got %s", resp.Operations[0].Status)
	}
	if resp.Operations[1].Status != "404" {
		t.Errorf("op[1]: expected 404, got %s", resp.Operations[1].Status)
	}
	if resp.Operations[2].Status != "400" {
		t.Errorf("op[2]: expected 400, got %s", resp.Operations[2].Status)
	}
}

func TestHandleBulk_ResponseSchema(t *testing.T) {
	h := newBulkTestHandler()
	body := `{
		"Operations": [{
			"method": "POST", "path": "/Users", "bulkId": "s",
			"data": {"userName": "schemauser", "emails": [{"value": "sch@test.com"}]}
		}]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var raw map[string]any
	json.Unmarshal(w.Body.Bytes(), &raw)
	schemas, ok := raw["schemas"].([]any)
	if !ok || len(schemas) == 0 {
		t.Fatal("missing schemas in response")
	}
	if schemas[0] != bulkResponseSchema {
		t.Errorf("expected schema %s, got %v", bulkResponseSchema, schemas[0])
	}
	// Content-Type must be application/scim+json
	ct := w.Header().Get("Content-Type")
	if ct != "application/scim+json" {
		t.Errorf("expected Content-Type application/scim+json, got %s", ct)
	}
}

func TestHandleBulk_FailOnErrors_Zero(t *testing.T) {
	h := newBulkTestHandler()
	// failOnErrors=0 means no early stopping — process all ops
	body := `{
		"failOnErrors": 0,
		"Operations": [
			{"method": "GET", "path": "/Users", "bulkId": "1"},
			{"method": "GET", "path": "/Users", "bulkId": "2"},
			{"method": "GET", "path": "/Users", "bulkId": "3"}
		]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Operations) != 3 {
		t.Errorf("expected 3 ops (failOnErrors=0 means no stop), got %d", len(resp.Operations))
	}
}

func TestHandleBulk_DeleteUser_InvalidPath(t *testing.T) {
	h := newBulkTestHandler()
	body := `{"Operations": [{"method": "DELETE", "path": "/Users", "bulkId": "d"}]}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Operations[0].Status != "400" {
		t.Errorf("expected 400 for path without ID, got %s", resp.Operations[0].Status)
	}
}

func TestHandleBulk_PutUser_InvalidPath(t *testing.T) {
	h := newBulkTestHandler()
	body := `{"Operations": [{"method": "PUT", "path": "/Users", "data": {"userName": "x"}}]}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Operations[0].Status != "400" {
		t.Errorf("expected 400 for PUT without ID, got %s", resp.Operations[0].Status)
	}
}

func TestHandleBulk_FailOnErrors_One(t *testing.T) {
	h := newBulkTestHandler()
	body := `{
		"failOnErrors": 1,
		"Operations": [
			{"method": "GET", "path": "/Users", "bulkId": "1"},
			{"method": "POST", "path": "/Users", "bulkId": "2",
			 "data": {"userName": "afterfail", "emails": [{"value": "af@test.com"}]}}
		]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Bulk", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", testTenantID)
	w := httptest.NewRecorder()
	h.handleBulk(w, req)

	var resp BulkResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	// failOnErrors=1, first op errors → should stop after 1
	if len(resp.Operations) != 1 {
		t.Errorf("expected 1 op (stopped after 1 error), got %d", len(resp.Operations))
	}
	if resp.Operations[0].Status != "400" {
		t.Errorf("expected 400, got %s", resp.Operations[0].Status)
	}
}

func TestExecuteBulkOp_POST_WithService(t *testing.T) {
	h := newBulkTestHandler()
	op := BulkOperationRequest{
		Method: "POST",
		Path:   "/Users",
		BulkID: "x",
		Data:   json.RawMessage(`{"userName":"execuser","emails":[{"value":"exec@test.com"}]}`),
	}
	resp, err := h.executeBulkOp(bulkTestCtx(), op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "201" {
		t.Errorf("expected 201, got %s", resp.Status)
	}
	if resp.BulkID != "x" {
		t.Errorf("expected bulkId x, got %s", resp.BulkID)
	}
}

func TestExecuteBulkOp_DELETE_NotFound(t *testing.T) {
	h := newBulkTestHandler()
	op := BulkOperationRequest{
		Method: "DELETE",
		Path:   "/Users/" + uuid.New().String(),
		BulkID: "x",
	}
	resp, err := h.executeBulkOp(bulkTestCtx(), op)
	if err == nil {
		t.Error("expected error for non-existent user")
	}
	if resp.Status != "404" {
		t.Errorf("expected 404, got %s", resp.Status)
	}
}

func TestExtractIDFromPath_GroupsPath(t *testing.T) {
	id, err := extractIDFromPath("/Groups/550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("got %s", id)
	}
}
