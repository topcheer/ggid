package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestUserStatus_IsValid(t *testing.T) {
	valid := []UserStatus{UserStatusActive, UserStatusLocked, UserStatusDisabled, UserStatusDeleted}
	for _, s := range valid {
		if !s.IsValid() {
			t.Errorf("expected %s to be valid", s)
		}
	}
	if UserStatus("invalid").IsValid() {
		t.Error("expected invalid status to be false")
	}
}

func TestUserStatus_CanAuthenticate(t *testing.T) {
	if !UserStatusActive.CanAuthenticate() {
		t.Error("active should authenticate")
	}
	if UserStatusLocked.CanAuthenticate() {
		t.Error("locked should not authenticate")
	}
	if UserStatusDisabled.CanAuthenticate() {
		t.Error("disabled should not authenticate")
	}
	if UserStatusDeleted.CanAuthenticate() {
		t.Error("deleted should not authenticate")
	}
}

func TestExternalIdentity_MetadataJSON_Nil(t *testing.T) {
	e := &ExternalIdentity{}
	raw := e.MetadataJSON()
	if string(raw) != "{}" {
		t.Errorf("expected '{}', got %s", string(raw))
	}
}

func TestExternalIdentity_MetadataJSON_WithData(t *testing.T) {
	e := &ExternalIdentity{
		Metadata: map[string]any{"dn": "cn=user,dc=example,dc=com"},
	}
	raw := e.MetadataJSON()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["dn"] != "cn=user,dc=example,dc=com" {
		t.Errorf("expected dn, got %v", m["dn"])
	}
}

func TestCreateUserInput_Fields(t *testing.T) {
	input := CreateUserInput{
		TenantID:    uuid.New(),
		Username:    "testuser",
		Email:       "test@example.com",
		Phone:       "+1234567890",
		Password:    "secret",
		DisplayName: "Test User",
		Locale:      "en-US",
		Timezone:    "America/New_York",
		ExternalID:  "ext-123",
	}
	if input.Username != "testuser" {
		t.Error("unexpected username")
	}
	if input.Email != "test@example.com" {
		t.Error("unexpected email")
	}
}

func TestListUsersFilter_Defaults(t *testing.T) {
	f := ListUsersFilter{
		TenantID: uuid.New(),
		PageSize: 20,
		Offset:   0,
	}
	if f.PageSize != 20 {
		t.Error("expected PageSize=20")
	}
}

func TestListUsersResult_NextOffset(t *testing.T) {
	r := ListUsersResult{
		Users:      []*User{},
		Total:      50,
		NextOffset: 20,
	}
	if r.NextOffset != 20 {
		t.Error("expected NextOffset=20")
	}
}

func TestUser_StructCreation(t *testing.T) {
	u := &User{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Username:  "newuser",
		Email:     "new@example.com",
		Status:    UserStatusActive,
		Locale:    "en-US",
		Timezone:  "UTC",
	}
	if u.Status != UserStatusActive {
		t.Error("expected active")
	}
	if !u.Status.CanAuthenticate() {
		t.Error("should authenticate")
	}
}
