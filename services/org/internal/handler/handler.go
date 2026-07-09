// Package handler implements gRPC handlers for the Org Service.
package handler

import (
	"context"
	"encoding/json"

	pb "github.com/ggid/ggid/api/gen/org/v1"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/ggid/ggid/services/org/internal/repository"
	"github.com/ggid/ggid/services/org/internal/service"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// toGRPCError converts a GGIDError to a gRPC status error.
func toGRPCError(err error) error {
	if ge, ok := errors.AsGGIDError(err); ok {
		switch ge.Code {
		case errors.ErrNotFound:
			return status.Error(codes.NotFound, ge.Message)
		case errors.ErrAlreadyExists:
			return status.Error(codes.AlreadyExists, ge.Message)
		case errors.ErrInvalidArgument:
			return status.Error(codes.InvalidArgument, ge.Message)
		default:
			return status.Error(codes.Internal, ge.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}

func parseUUID(s string, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "invalid "+field)
	}
	return id, nil
}

func parseOptionalUUID(s string) (*uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func jsonToMap(s string) map[string]any {
	if s == "" {
		return nil
	}
	var m map[string]any
	json.Unmarshal([]byte(s), &m)
	return m
}

// --- TenantHandler ---

type TenantHandler struct {
	pb.UnimplementedTenantServiceServer
	svc *service.TenantService
}

func NewTenantHandler(svc *service.TenantService) *TenantHandler {
	return &TenantHandler{svc: svc}
}

func (h *TenantHandler) CreateTenant(ctx context.Context, req *pb.CreateTenantRequest) (*pb.Tenant, error) {
	t := &domain.Tenant{
		Name:     req.GetName(),
		Slug:     req.GetSlug(),
		Plan:     domain.Plan(req.GetPlan()),
		MaxUsers: int(req.GetMaxUsers()),
		Settings: jsonToMap(req.GetSettings()),
	}
	created, err := h.svc.Create(ctx, t)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return tenantToProto(created), nil
}

func (h *TenantHandler) GetTenant(ctx context.Context, req *pb.GetTenantRequest) (*pb.Tenant, error) {
	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}
	t, err := h.svc.Get(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return tenantToProto(t), nil
}

func (h *TenantHandler) UpdateTenant(ctx context.Context, req *pb.UpdateTenantRequest) (*pb.Tenant, error) {
	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}
	t := &domain.Tenant{ID: id}
	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Plan != nil {
		t.Plan = domain.Plan(*req.Plan)
	}
	if req.MaxUsers != nil {
		t.MaxUsers = int(*req.MaxUsers)
	}
	if req.Status != nil {
		t.Status = domain.TenantStatus(*req.Status)
	}
	if req.Settings != nil {
		t.Settings = jsonToMap(*req.Settings)
	}
	updated, err := h.svc.Update(ctx, t)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return tenantToProto(updated), nil
}

func (h *TenantHandler) DeleteTenant(ctx context.Context, req *pb.DeleteTenantRequest) (*pb.DeleteTenantResponse, error) {
	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.DeleteTenantResponse{}, nil
}

func tenantToProto(t *domain.Tenant) *pb.Tenant {
	settingsJSON, _ := json.Marshal(t.Settings)
	p := &pb.Tenant{
		Id:       t.ID.String(),
		Name:     t.Name,
		Slug:     t.Slug,
		Plan:     string(t.Plan),
		Status:   string(t.Status),
		Settings: string(settingsJSON),
		MaxUsers: int32(t.MaxUsers),
	}
	if !t.CreatedAt.IsZero() {
		p.CreatedAt = timestamppb.New(t.CreatedAt)
	}
	if !t.UpdatedAt.IsZero() {
		p.UpdatedAt = timestamppb.New(t.UpdatedAt)
	}
	return p
}

// --- OrgHandler ---

type OrgHandler struct {
	pb.UnimplementedOrganizationServiceServer
	svc *service.OrgService
}

func NewOrgHandler(svc *service.OrgService) *OrgHandler {
	return &OrgHandler{svc: svc}
}

func (h *OrgHandler) CreateOrganization(ctx context.Context, req *pb.CreateOrgRequest) (*pb.Organization, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, err
	}
	parentID, err := parseOptionalUUID(req.GetParentId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid parent_id")
	}
	org := &domain.Organization{
		TenantID: tenantID,
		ParentID: parentID,
		Name:     req.GetName(),
		Metadata: jsonToMap(req.GetMetadata()),
	}
	created, err := h.svc.Create(ctx, org)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return orgToProto(created), nil
}

func (h *OrgHandler) GetOrganization(ctx context.Context, req *pb.GetOrgRequest) (*pb.Organization, error) {
	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}
	org, err := h.svc.Get(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return orgToProto(org), nil
}

func (h *OrgHandler) ListOrganizations(ctx context.Context, req *pb.ListOrgsRequest) (*pb.ListOrgsResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, err
	}
	orgs, err := h.svc.List(ctx, tenantID, 1, int(req.GetPageSize()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbOrgs := make([]*pb.Organization, len(orgs))
	for i, o := range orgs {
		pbOrgs[i] = orgToProto(o)
	}
	return &pb.ListOrgsResponse{Organizations: pbOrgs}, nil
}

func (h *OrgHandler) GetSubTree(ctx context.Context, req *pb.GetSubTreeRequest) (*pb.ListOrgsResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, err
	}
	rootID, err := parseUUID(req.GetRootId(), "root_id")
	if err != nil {
		return nil, err
	}
	orgs, err := h.svc.GetSubTree(ctx, tenantID, rootID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbOrgs := make([]*pb.Organization, len(orgs))
	for i, o := range orgs {
		pbOrgs[i] = orgToProto(o)
	}
	return &pb.ListOrgsResponse{Organizations: pbOrgs}, nil
}

func (h *OrgHandler) UpdateOrganization(ctx context.Context, req *pb.UpdateOrgRequest) (*pb.Organization, error) {
	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}
	org := &domain.Organization{ID: id}
	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Metadata != nil {
		org.Metadata = jsonToMap(*req.Metadata)
	}
	updated, err := h.svc.Update(ctx, org)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return orgToProto(updated), nil
}

func (h *OrgHandler) DeleteOrganization(ctx context.Context, req *pb.DeleteOrgRequest) (*pb.DeleteOrgResponse, error) {
	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.DeleteOrgResponse{}, nil
}

func orgToProto(o *domain.Organization) *pb.Organization {
	metaJSON, _ := json.Marshal(o.Metadata)
	p := &pb.Organization{
		Id:       o.ID.String(),
		TenantId: o.TenantID.String(),
		Name:     o.Name,
		Path:     o.Path,
		Metadata: string(metaJSON),
	}
	if o.ParentID != nil {
		s := o.ParentID.String()
		p.ParentId = &s
	}
	return p
}

// --- DeptHandler ---

type DeptHandler struct {
	pb.UnimplementedDepartmentServiceServer
	svc *service.DeptService
}

func NewDeptHandler(svc *service.DeptService) *DeptHandler {
	return &DeptHandler{svc: svc}
}

func (h *DeptHandler) CreateDepartment(ctx context.Context, req *pb.CreateDeptRequest) (*pb.Department, error) {
	orgID, err := parseUUID(req.GetOrgId(), "org_id")
	if err != nil {
		return nil, err
	}
	parentID, err := parseOptionalUUID(req.GetParentId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid parent_id")
	}
	managerID, err := parseOptionalUUID(req.GetManagerId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid manager_id")
	}
	dept := &domain.Department{
		OrgID:     orgID,
		ParentID:  parentID,
		Name:      req.GetName(),
		ManagerID: managerID,
		Metadata:  jsonToMap(req.GetMetadata()),
	}
	created, err := h.svc.Create(ctx, dept)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return deptToProto(created), nil
}

func (h *DeptHandler) GetDepartment(ctx context.Context, req *pb.GetDeptRequest) (*pb.Department, error) {
	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}
	dept, err := h.svc.Get(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return deptToProto(dept), nil
}

func (h *DeptHandler) ListDepartments(ctx context.Context, req *pb.ListDeptsRequest) (*pb.ListDeptsResponse, error) {
	orgID, err := parseUUID(req.GetOrgId(), "org_id")
	if err != nil {
		return nil, err
	}
	depts, err := h.svc.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbDepts := make([]*pb.Department, len(depts))
	for i, d := range depts {
		pbDepts[i] = deptToProto(d)
	}
	return &pb.ListDeptsResponse{Departments: pbDepts}, nil
}

func (h *DeptHandler) UpdateDepartment(ctx context.Context, req *pb.UpdateDeptRequest) (*pb.Department, error) {
	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}
	dept := &domain.Department{ID: id}
	if req.Name != nil {
		dept.Name = *req.Name
	}
	if req.ManagerId != nil {
		mid, err := parseUUID(*req.ManagerId, "manager_id")
		if err != nil {
			return nil, err
		}
		dept.ManagerID = &mid
	}
	updated, err := h.svc.Update(ctx, dept)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return deptToProto(updated), nil
}

func (h *DeptHandler) DeleteDepartment(ctx context.Context, req *pb.DeleteDeptRequest) (*pb.DeleteDeptResponse, error) {
	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.DeleteDeptResponse{}, nil
}

func deptToProto(d *domain.Department) *pb.Department {
	metaJSON, _ := json.Marshal(d.Metadata)
	p := &pb.Department{
		Id:       d.ID.String(),
		OrgId:    d.OrgID.String(),
		Name:     d.Name,
		Path:     d.Path,
		Metadata: string(metaJSON),
	}
	if d.ParentID != nil {
		s := d.ParentID.String()
		p.ParentId = &s
	}
	if d.ManagerID != nil {
		s := d.ManagerID.String()
		p.ManagerId = &s
	}
	return p
}

// --- TeamHandler ---

type TeamHandler struct {
	pb.UnimplementedTeamServiceServer
	svc *service.TeamService
}

func NewTeamHandler(svc *service.TeamService) *TeamHandler {
	return &TeamHandler{svc: svc}
}

func (h *TeamHandler) CreateTeam(ctx context.Context, req *pb.CreateTeamRequest) (*pb.Team, error) {
	orgID, err := parseUUID(req.GetOrgId(), "org_id")
	if err != nil {
		return nil, err
	}
	createdBy, err := parseUUID(req.GetCreatedBy(), "created_by")
	if err != nil {
		return nil, err
	}
	team := &domain.Team{
		OrgID:       orgID,
		Name:        req.GetName(),
		Description: req.GetDescription(),
		CreatedBy:   createdBy,
	}
	created, err := h.svc.Create(ctx, team)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return teamToProto(created), nil
}

func (h *TeamHandler) GetTeam(ctx context.Context, req *pb.GetTeamRequest) (*pb.Team, error) {
	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}
	team, err := h.svc.Get(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return teamToProto(team), nil
}

func (h *TeamHandler) ListTeams(ctx context.Context, req *pb.ListTeamsRequest) (*pb.ListTeamsResponse, error) {
	orgID, err := parseUUID(req.GetOrgId(), "org_id")
	if err != nil {
		return nil, err
	}
	teams, err := h.svc.List(ctx, orgID, 1, int(req.GetPageSize()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbTeams := make([]*pb.Team, len(teams))
	for i, t := range teams {
		pbTeams[i] = teamToProto(t)
	}
	return &pb.ListTeamsResponse{Teams: pbTeams}, nil
}

func (h *TeamHandler) DeleteTeam(ctx context.Context, req *pb.DeleteTeamRequest) (*pb.DeleteTeamResponse, error) {
	id, err := parseUUID(req.GetId(), "id")
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.DeleteTeamResponse{}, nil
}

func teamToProto(t *domain.Team) *pb.Team {
	p := &pb.Team{
		Id:          t.ID.String(),
		OrgId:       t.OrgID.String(),
		Name:        t.Name,
		Description: t.Description,
		CreatedBy:   t.CreatedBy.String(),
	}
	if !t.CreatedAt.IsZero() {
		p.CreatedAt = timestamppb.New(t.CreatedAt)
	}
	return p
}

// --- MembershipHandler ---

type MembershipHandler struct {
	pb.UnimplementedMembershipServiceServer
	svc *service.MembershipService
}

func NewMembershipHandler(svc *service.MembershipService) *MembershipHandler {
	return &MembershipHandler{svc: svc}
}

func (h *MembershipHandler) InviteMember(ctx context.Context, req *pb.InviteMemberRequest) (*pb.Membership, error) {
	userID, err := parseUUID(req.GetUserId(), "user_id")
	if err != nil {
		return nil, err
	}
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, err
	}
	orgID, err := parseUUID(req.GetOrgId(), "org_id")
	if err != nil {
		return nil, err
	}
	m := &domain.Membership{
		UserID:   userID,
		TenantID: tenantID,
		OrgID:    orgID,
		Title:    req.GetTitle(),
		Metadata: jsonToMap(req.GetMetadata()),
	}
	if req.GetDeptId() != "" {
		deptID, err := parseUUID(req.GetDeptId(), "dept_id")
		if err != nil {
			return nil, err
		}
		m.DeptID = &deptID
	}
	if req.GetTeamId() != "" {
		teamID, err := parseUUID(req.GetTeamId(), "team_id")
		if err != nil {
			return nil, err
		}
		m.TeamID = &teamID
	}
	created, err := h.svc.Invite(ctx, m)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return membershipToProto(created), nil
}

func (h *MembershipHandler) AcceptInvitation(ctx context.Context, req *pb.AcceptInvitationRequest) (*pb.Membership, error) {
	id, err := parseUUID(req.GetMembershipId(), "membership_id")
	if err != nil {
		return nil, err
	}
	if err := h.svc.AcceptInvitation(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.Membership{Id: req.GetMembershipId()}, nil
}

func (h *MembershipHandler) RemoveMember(ctx context.Context, req *pb.RemoveMemberRequest) (*pb.RemoveMemberResponse, error) {
	id, err := parseUUID(req.GetMembershipId(), "membership_id")
	if err != nil {
		return nil, err
	}
	if err := h.svc.Remove(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.RemoveMemberResponse{}, nil
}

func (h *MembershipHandler) ListMembers(ctx context.Context, req *pb.ListMembersRequest) (*pb.ListMembersResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, err
	}
	filter := repository.ListMembersFilter{
		TenantID: tenantID,
		Status:   domain.MembershipStatus(req.GetStatus()),
	}
	if req.GetOrgId() != "" {
		orgID, err := parseUUID(req.GetOrgId(), "org_id")
		if err != nil {
			return nil, err
		}
		filter.OrgID = &orgID
	}
	members, err := h.svc.List(ctx, filter, 1, int(req.GetPageSize()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbMembers := make([]*pb.Membership, len(members))
	for i, m := range members {
		pbMembers[i] = membershipToProto(m)
	}
	return &pb.ListMembersResponse{Memberships: pbMembers}, nil
}

func membershipToProto(m *domain.Membership) *pb.Membership {
	metaJSON, _ := json.Marshal(m.Metadata)
	p := &pb.Membership{
		Id:       m.ID.String(),
		UserId:   m.UserID.String(),
		TenantId: m.TenantID.String(),
		OrgId:    m.OrgID.String(),
		Title:    m.Title,
		Status:   string(m.Status),
		Metadata: string(metaJSON),
	}
	if m.DeptID != nil {
		s := m.DeptID.String(); p.DeptId = &s
	}
	if m.TeamID != nil {
		s2 := m.TeamID.String(); p.TeamId = &s2
	}
	return p
}