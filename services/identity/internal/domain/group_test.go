package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestGroup_IsValid(t *testing.T) {
	tid := uuid.New()
	tests := []struct {
		name  string
		group Group
		want  bool
	}{
		{"valid", Group{TenantID: tid, DisplayName: "Admin"}, true},
		{"empty display", Group{TenantID: tid, DisplayName: ""}, false},
		{"nil tenant", Group{TenantID: uuid.Nil, DisplayName: "Admin"}, false},
		{"both empty", Group{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.group.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroup_String(t *testing.T) {
	g := Group{DisplayName: "Engineering"}
	if g.String() != "Engineering" {
		t.Errorf("String() = %q, want Engineering", g.String())
	}
}

func TestGroupMember_IsValid(t *testing.T) {
	gid, uid := uuid.New(), uuid.New()
	tests := []struct {
		name   string
		member GroupMember
		want   bool
	}{
		{"valid", GroupMember{GroupID: gid, UserID: uid}, true},
		{"nil group", GroupMember{GroupID: uuid.Nil, UserID: uid}, false},
		{"nil user", GroupMember{GroupID: gid, UserID: uuid.Nil}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.member.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupMember_String(t *testing.T) {
	m := GroupMember{Display: "john.doe"}
	if m.String() != "john.doe" {
		t.Errorf("String() = %q", m.String())
	}
}

func TestPatchOpType_Constants(t *testing.T) {
	if PatchOpAdd != "add" {
		t.Errorf("PatchOpAdd = %q", PatchOpAdd)
	}
	if PatchOpRemove != "remove" {
		t.Errorf("PatchOpRemove = %q", PatchOpRemove)
	}
}

func TestCreateGroupInput_Fields(t *testing.T) {
	tid := uuid.New()
	input := CreateGroupInput{
		TenantID:    tid,
		DisplayName: "Test",
		ExternalID:  "ext-1",
		Members:     []GroupMemberInput{{UserID: uuid.New(), Type: "User"}},
	}
	if input.DisplayName != "Test" {
		t.Error("DisplayName mismatch")
	}
	if len(input.Members) != 1 {
		t.Errorf("Members len = %d", len(input.Members))
	}
}

func TestUpdateGroupInput_Pointers(t *testing.T) {
	name := "NewName"
	input := UpdateGroupInput{DisplayName: &name}
	if input.DisplayName == nil || *input.DisplayName != "NewName" {
		t.Errorf("DisplayName = %v", input.DisplayName)
	}
}

func TestGroupListFilter_Defaults(t *testing.T) {
	f := GroupListFilter{TenantID: uuid.New()}
	if f.PageSize != 0 {
		t.Errorf("default PageSize should be 0 (handled by repo)")
	}
	if f.Search != "" {
		t.Errorf("default Search should be empty")
	}
}

func TestGroupListResult_Empty(t *testing.T) {
	r := GroupListResult{}
	if len(r.Groups) != 0 {
		t.Error("Groups should be nil/empty by default")
	}
	if r.Total != 0 {
		t.Error("Total should be 0")
	}
}

func TestPatchGroupResult(t *testing.T) {
	r := PatchGroupResult{Added: 3, Removed: 1}
	if r.Added != 3 || r.Removed != 1 {
		t.Error("PatchGroupResult mismatch")
	}
}

func TestMembershipPatchResult(t *testing.T) {
	r := MembershipPatchResult{Added: 2, Removed: 0}
	if r.Added != 2 {
		t.Errorf("Added = %d", r.Added)
	}
}

func TestPatchGroupMembersInput(t *testing.T) {
	uid := uuid.New()
	input := PatchGroupMembersInput{
		GroupID:       uuid.New(),
		AddMembers:    []GroupMemberInput{{UserID: uid, Type: "User"}},
		RemoveMembers: []uuid.UUID{uuid.New()},
	}
	if len(input.AddMembers) != 1 {
		t.Errorf("AddMembers len = %d", len(input.AddMembers))
	}
	if len(input.RemoveMembers) != 1 {
		t.Errorf("RemoveMembers len = %d", len(input.RemoveMembers))
	}
}
