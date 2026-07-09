package handler

import (
	"context"
	"time"

	pb "github.com/ggid/ggid/api/gen/policy/v1"
	"github.com/ggid/ggid/pkg/audit"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/ggid/ggid/services/policy/internal/service"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RoleHandler implements the RoleService gRPC interface.
type RoleHandler struct {
	pb.UnimplementedRoleServiceServer
	roleSvc   *service.RoleService
	auditor   *audit.Publisher
}

func NewRoleHandler(roleSvc *service.RoleService, auditor *audit.Publisher) *RoleHandler {
	return &RoleHandler{roleSvc: roleSvc, auditor: auditor}
}

func (h *RoleHandler) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (*pb.Role, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	var parentID *uuid.UUID
	if req.ParentRoleId != nil && *req.ParentRoleId != "" {
		pid, err := uuid.Parse(*req.ParentRoleId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent_role_id")
		}
		parentID = &pid
	}
	role, err := h.roleSvc.CreateRole(ctx, tenantID, req.GetKey(), req.GetName(), req.GetDescription(), parentID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	h.emitAudit("role.create", "success", tenantID, uuid.Nil, "roles", role.ID.String())
	return roleToProto(role), nil
}

func (h *RoleHandler) GetRole(ctx context.Context, req *pb.GetRoleRequest) (*pb.Role, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	role, err := h.roleSvc.GetRole(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return roleToProto(role), nil
}

func (h *RoleHandler) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	roles, err := h.roleSvc.ListRoles(ctx, tenantID, 1, int(req.GetPageSize()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbRoles := make([]*pb.Role, len(roles))
	for i, r := range roles {
		pbRoles[i] = roleToProto(r)
	}
	return &pb.ListRolesResponse{Roles: pbRoles}, nil
}

func (h *RoleHandler) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (*pb.Role, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	var parentID *uuid.UUID
	if req.ParentRoleId != nil && *req.ParentRoleId != "" {
		pid, err := uuid.Parse(*req.ParentRoleId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent_role_id")
		}
		parentID = &pid
	}
	role, err := h.roleSvc.UpdateRole(ctx, id, req.Name, req.Description, parentID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return roleToProto(role), nil
}

func (h *RoleHandler) DeleteRole(ctx context.Context, req *pb.DeleteRoleRequest) (*pb.DeleteRoleResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	if err := h.roleSvc.DeleteRole(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.DeleteRoleResponse{}, nil
}

func (h *RoleHandler) AssignRole(ctx context.Context, req *pb.AssignRoleRequest) (*pb.AssignRoleResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	roleID, err := uuid.Parse(req.GetRoleId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid role_id")
	}
	scopeID, err := uuid.Parse(req.GetScopeId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid scope_id")
	}
	grantedBy, err := uuid.Parse(req.GetGrantedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid granted_by")
	}
	var expiresAt *time.Time
	if req.GetExpiresAt() != nil {
		t := req.GetExpiresAt().AsTime()
		expiresAt = &t
	}
	if err := h.roleSvc.AssignRole(ctx, userID, roleID, domain.ScopeType(req.GetScopeType()), scopeID, grantedBy, expiresAt); err != nil {
		return nil, toGRPCError(err)
	}
	h.emitAudit("role.assign", "success", tenantIDFromContext(ctx), grantedBy, "roles", roleID.String())
	return &pb.AssignRoleResponse{}, nil
}

func (h *RoleHandler) RevokeRole(ctx context.Context, req *pb.RevokeRoleRequest) (*pb.RevokeRoleResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	roleID, err := uuid.Parse(req.GetRoleId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid role_id")
	}
	scopeID, err := uuid.Parse(req.GetScopeId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid scope_id")
	}
	if err := h.roleSvc.RevokeRole(ctx, userID, roleID, domain.ScopeType(req.GetScopeType()), scopeID); err != nil {
		return nil, toGRPCError(err)
	}
	h.emitAudit("role.revoke", "success", tenantIDFromContext(ctx), uuid.Nil, "roles", roleID.String())
	return &pb.RevokeRoleResponse{}, nil
}

func (h *RoleHandler) ListUserRoles(ctx context.Context, req *pb.ListUserRolesRequest) (*pb.ListUserRolesResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	userRoles, err := h.roleSvc.ListUserRoles(ctx, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	entries := make([]*pb.UserRoleEntry, len(userRoles))
	for i, ur := range userRoles {
		entries[i] = &pb.UserRoleEntry{
			RoleId:    ur.RoleID.String(),
			ScopeType: string(ur.ScopeType),
			ScopeId:   ur.ScopeID.String(),
		}
		if ur.ExpiresAt != nil {
			entries[i].ExpiresAt = timestamppb.New(*ur.ExpiresAt)
		}
	}
	return &pb.ListUserRolesResponse{Roles: entries}, nil
}

// --- Helpers ---

func roleToProto(r *domain.Role) *pb.Role {
	p := &pb.Role{
		Id:          r.ID.String(),
		TenantId:    r.TenantID.String(),
		Key:         r.Key,
		Name:        r.Name,
		Description: r.Description,
		SystemRole:  r.SystemRole,
	}
	if r.ParentRoleID != nil {
		s := r.ParentRoleID.String()
		p.ParentRoleId = &s
	}
	if !r.CreatedAt.IsZero() {
		p.CreatedAt = timestamppb.New(r.CreatedAt)
	}
	if !r.UpdatedAt.IsZero() {
		p.UpdatedAt = timestamppb.New(r.UpdatedAt)
	}
	return p
}

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
		case errors.ErrPermissionDenied:
			return status.Error(codes.PermissionDenied, ge.Message)
		case errors.ErrFailedPrecondition:
			return status.Error(codes.FailedPrecondition, ge.Message)
		default:
			return status.Error(codes.Internal, ge.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}

// emitAudit publishes an audit event if the publisher is configured.
// It's best-effort — failures are silently dropped.
func (h *RoleHandler) emitAudit(action, result string, tenantID, actorID uuid.UUID, resourceType, resourceID string) {
	if h.auditor == nil {
		return
	}
	h.auditor.PublishAsync(audit.Event{
		TenantID:     tenantID,
		ActorType:    "user",
		ActorID:      actorID,
		Action:       action,
		Result:       result,
		ResourceType: resourceType,
		ResourceID:   uuid.MustParse(resourceID),
	})
}

// tenantIDFromContext is a placeholder — in production this would extract
// the tenant ID from gRPC metadata or context.
func tenantIDFromContext(_ context.Context) uuid.UUID {
	return uuid.Nil
}
