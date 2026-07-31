package handler

import (
	"context"
	"strconv"

	pb "github.com/ggid/ggid/api/gen/policy/v1"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/ggid/ggid/services/policy/internal/service"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PermissionHandler implements the PermissionService gRPC interface.
type PermissionHandler struct {
	pb.UnimplementedPermissionServiceServer
	roleSvc *service.RoleService
}

func NewPermissionHandler(roleSvc *service.RoleService) *PermissionHandler {
	return &PermissionHandler{roleSvc: roleSvc}
}

func (h *PermissionHandler) CreatePermission(ctx context.Context, req *pb.CreatePermissionRequest) (*pb.Permission, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	perm := &domain.Permission{
		TenantID:     tenantID,
		Key:          req.GetKey(),
		Name:         req.GetName(),
		ResourceType: req.GetResourceType(),
		Action:       req.GetAction(),
		Description:  req.GetDescription(),
	}
	created, err := h.roleSvc.CreatePermission(ctx, perm)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return permissionToProto(created), nil
}

func (h *PermissionHandler) ListPermissions(ctx context.Context, req *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	if err := validateTenantFromMetadata(ctx, req.GetTenantId()); err != nil {
		return nil, err
	}
	pageSize := int(req.GetPageSize())
	page := parsePageToken(req.GetPageToken())
	perms, err := h.roleSvc.ListPermissions(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbPerms := make([]*pb.Permission, len(perms))
	for i, p := range perms {
		pbPerms[i] = permissionToProto(p)
	}
	resp := &pb.ListPermissionsResponse{Permissions: pbPerms}
	if len(perms) == pageSize {
		resp.NextPageToken = strconv.Itoa(page + 1)
	}
	return resp, nil
}

func (h *PermissionHandler) GrantPermissions(ctx context.Context, req *pb.GrantPermissionsRequest) (*pb.GrantPermissionsResponse, error) {
	roleID, err := uuid.Parse(req.GetRoleId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid role_id")
	}
	permIDs := make([]uuid.UUID, len(req.GetPermissionIds()))
	for i, idStr := range req.GetPermissionIds() {
		pid, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid permission_id")
		}
		permIDs[i] = pid
	}
	if err := h.roleSvc.GrantPermissionsToRole(ctx, roleID, permIDs); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.GrantPermissionsResponse{}, nil
}

func (h *PermissionHandler) RevokePermissions(ctx context.Context, req *pb.RevokePermissionsRequest) (*pb.RevokePermissionsResponse, error) {
	roleID, err := uuid.Parse(req.GetRoleId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid role_id")
	}
	permIDs := make([]uuid.UUID, len(req.GetPermissionIds()))
	for i, idStr := range req.GetPermissionIds() {
		pid, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid permission_id")
		}
		permIDs[i] = pid
	}
	if err := h.roleSvc.RevokePermissionsFromRole(ctx, roleID, permIDs); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.RevokePermissionsResponse{}, nil
}

func permissionToProto(p *domain.Permission) *pb.Permission {
	return &pb.Permission{
		Id:           p.ID.String(),
		TenantId:     p.TenantID.String(),
		Key:          p.Key,
		Name:         p.Name,
		ResourceType: p.ResourceType,
		Action:       p.Action,
		Description:  p.Description,
		SystemPerm:   p.SystemPerm,
	}
}
