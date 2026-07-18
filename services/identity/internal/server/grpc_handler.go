package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"

	identityv1 "github.com/ggid/ggid/api/gen/identity/v1"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/ggid/ggid/services/identity/internal/service"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// IdentityGRPCHandler implements IdentityServiceServer by delegating to IdentityService.
type IdentityGRPCHandler struct {
	identityv1.UnimplementedIdentityServiceServer
	svc *service.IdentityService
}

func NewIdentityGRPCHandler(svc *service.IdentityService) *IdentityGRPCHandler {
	return &IdentityGRPCHandler{svc: svc}
}

func (h *IdentityGRPCHandler) RegisterGRPC(s *grpc.Server) {
	identityv1.RegisterIdentityServiceServer(s, h)
}

func (h *IdentityGRPCHandler) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.User, error) {
	input := &domain.CreateUserInput{
		TenantID:    defaultTenantID(),
		Username:    req.GetUsername(),
		Email:       req.GetEmail(),
		Phone:       req.GetPhone(),
		Password:    req.GetPassword(),
		DisplayName: req.GetDisplayName(),
		Locale:      req.GetLocale(),
		Timezone:    req.GetTimezone(),
	}
	user, err := h.svc.CreateUser(ctx, input)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("create user: %v", err))
	}
	return domainToPbUser(user), nil
}

func (h *IdentityGRPCHandler) GetUser(ctx context.Context, req *identityv1.GetUserRequest) (*identityv1.User, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	user, err := h.svc.GetUser(ctx, id)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("user not found: %v", err))
	}
	return domainToPbUser(user), nil
}

func (h *IdentityGRPCHandler) ListUsers(ctx context.Context, req *identityv1.ListUsersRequest) (*identityv1.ListUsersResponse, error) {
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = 50
	}
	offset := 0
	if req.GetPageToken() != "" {
		if v, err := strconv.Atoi(req.GetPageToken()); err == nil {
			offset = v
		}
	}

	filter := &domain.ListUsersFilter{
		TenantID: defaultTenantID(),
		Search:   req.GetSearch(),
		PageSize: pageSize,
		Offset:   offset,
		SortBy:   req.GetSortBy(),
		SortDesc: req.GetSortDesc(),
	}

	result, err := h.svc.ListUsers(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("list users: %v", err))
	}

	pbUsers := make([]*identityv1.User, 0, len(result.Users))
	for _, u := range result.Users {
		pbUsers = append(pbUsers, domainToPbUser(u))
	}

	nextToken := ""
	if result.NextOffset > 0 {
		nextToken = strconv.Itoa(result.NextOffset)
	}

	return &identityv1.ListUsersResponse{
		Users:         pbUsers,
		NextPageToken: nextToken,
		Total:         int32(result.Total),
	}, nil
}

func (h *IdentityGRPCHandler) UpdateUser(ctx context.Context, req *identityv1.UpdateUserRequest) (*identityv1.User, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	input := &domain.UpdateUserInput{
		Phone:       req.Phone,
		AvatarURL:   req.AvatarUrl,
		Locale:      req.Locale,
		Timezone:    req.Timezone,
		DisplayName: req.DisplayName,
	}
	user, err := h.svc.UpdateUser(ctx, id, input)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("update user: %v", err))
	}
	return domainToPbUser(user), nil
}

func (h *IdentityGRPCHandler) DeleteUser(ctx context.Context, req *identityv1.DeleteUserRequest) (*identityv1.DeleteUserResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	if err := h.svc.DeleteUser(ctx, id); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("delete user: %v", err))
	}
	return &identityv1.DeleteUserResponse{}, nil
}

func (h *IdentityGRPCHandler) LockUser(ctx context.Context, req *identityv1.LockUserRequest) (*identityv1.User, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	user, err := h.svc.LockUser(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("lock user: %v", err))
	}
	return domainToPbUser(user), nil
}

func (h *IdentityGRPCHandler) UnlockUser(ctx context.Context, req *identityv1.UnlockUserRequest) (*identityv1.User, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	user, err := h.svc.UnlockUser(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("unlock user: %v", err))
	}
	return domainToPbUser(user), nil
}

func (h *IdentityGRPCHandler) RegisterUser(ctx context.Context, req *identityv1.RegisterUserRequest) (*identityv1.RegisterUserResponse, error) {
	input := &domain.CreateUserInput{
		TenantID: defaultTenantID(),
		Username: req.GetUsername(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
		Locale:   req.GetLocale(),
	}
	user, token, err := h.svc.RegisterUser(ctx, input)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("register user: %v", err))
	}
	return &identityv1.RegisterUserResponse{
		User:              domainToPbUser(user),
		VerificationToken: token,
	}, nil
}

func (h *IdentityGRPCHandler) VerifyEmail(ctx context.Context, req *identityv1.VerifyEmailRequest) (*identityv1.VerifyEmailResponse, error) {
	_, err := h.svc.VerifyEmail(ctx, req.GetToken())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("verify email: %v", err))
	}
	return &identityv1.VerifyEmailResponse{Success: true}, nil
}

func (h *IdentityGRPCHandler) ListUserEmails(ctx context.Context, req *identityv1.ListUserEmailsRequest) (*identityv1.ListUserEmailsResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	emails, err := h.svc.ListUserEmails(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("list emails: %v", err))
	}
	pbEmails := make([]*identityv1.UserEmail, 0, len(emails))
	for _, e := range emails {
		pbEmails = append(pbEmails, domainToPbUserEmail(e))
	}
	return &identityv1.ListUserEmailsResponse{Emails: pbEmails}, nil
}

func (h *IdentityGRPCHandler) AddUserEmail(ctx context.Context, req *identityv1.AddUserEmailRequest) (*identityv1.UserEmail, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	email, err := h.svc.AddUserEmail(ctx, userID, req.GetEmail())
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("add email: %v", err))
	}
	return domainToPbUserEmail(email), nil
}

func (h *IdentityGRPCHandler) RemoveUserEmail(ctx context.Context, req *identityv1.RemoveUserEmailRequest) (*identityv1.RemoveUserEmailResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	if err := h.svc.RemoveUserEmail(ctx, userID, req.GetEmail()); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("remove email: %v", err))
	}
	return &identityv1.RemoveUserEmailResponse{Success: true}, nil
}

func (h *IdentityGRPCHandler) SetPrimaryEmail(ctx context.Context, req *identityv1.SetPrimaryEmailRequest) (*identityv1.UserEmail, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	emailID, err := uuid.Parse(req.GetEmailId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid email id")
	}
	email, err := h.svc.SetPrimaryEmail(ctx, userID, emailID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("set primary email: %v", err))
	}
	return domainToPbUserEmail(email), nil
}

func (h *IdentityGRPCHandler) ListExternalIdentities(ctx context.Context, req *identityv1.ListExternalIdentitiesRequest) (*identityv1.ListExternalIdentitiesResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	identities, err := h.svc.ListExternalIdentities(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("list external identities: %v", err))
	}
	pbIdentities := make([]*identityv1.ExternalIdentity, 0, len(identities))
	for _, ei := range identities {
		pbIdentities = append(pbIdentities, domainToPbExternalIdentity(ei))
	}
	return &identityv1.ListExternalIdentitiesResponse{Identities: pbIdentities}, nil
}

func (h *IdentityGRPCHandler) LinkExternalIdentity(ctx context.Context, req *identityv1.LinkExternalIdentityRequest) (*identityv1.ExternalIdentity, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	ei := &domain.ExternalIdentity{
		UserID:     userID,
		TenantID:   defaultTenantID(),
		Provider:   req.GetProvider(),
		ExternalID: req.GetExternalId(),
	}
	if len(req.GetMetadata()) > 0 {
		var md map[string]any
		if err := json.Unmarshal(req.GetMetadata(), &md); err == nil {
			ei.Metadata = md
		}
	}
	result, err := h.svc.LinkExternalIdentity(ctx, ei)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("link external identity: %v", err))
	}
	return domainToPbExternalIdentity(result), nil
}

func (h *IdentityGRPCHandler) UnlinkExternalIdentity(ctx context.Context, req *identityv1.UnlinkExternalIdentityRequest) (*identityv1.UnlinkExternalIdentityResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	identityID, err := uuid.Parse(req.GetIdentityId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid identity id")
	}
	if err := h.svc.UnlinkExternalIdentity(ctx, userID, identityID); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("unlink external identity: %v", err))
	}
	return &identityv1.UnlinkExternalIdentityResponse{Success: true}, nil
}

// --- helpers ---

func defaultTenantID() uuid.UUID {
	s := os.Getenv("DEFAULT_TENANT_ID")
	if s == "" {
		s = "00000000-0000-0000-0000-000000000001"
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}
	return id
}

func domainToPbUser(u *domain.User) *identityv1.User {
	pb := &identityv1.User{
		Id:            u.ID.String(),
		TenantId:      u.TenantID.String(),
		Username:      u.Username,
		Email:         u.Email,
		Phone:         u.Phone,
		EmailVerified: u.EmailVerified,
		PhoneVerified: u.PhoneVerified,
		AvatarUrl:     u.AvatarURL,
		Locale:        u.Locale,
		Timezone:      u.Timezone,
		CreatedAt:     timestamppb.New(u.CreatedAt),
		UpdatedAt:     timestamppb.New(u.UpdatedAt),
	}
	if u.PrimaryEmailID != nil {
		pb.PrimaryEmailId = u.PrimaryEmailID.String()
	}
	if u.LastLoginAt != nil {
		pb.LastLoginAt = timestamppb.New(*u.LastLoginAt)
	}
	if u.LastLoginIP.IsValid() {
		pb.LastLoginIp = u.LastLoginIP.String()
	}
	return pb
}

func domainToPbUserEmail(e *domain.UserEmail) *identityv1.UserEmail {
	pb := &identityv1.UserEmail{
		Id:        e.ID.String(),
		UserId:    e.UserID.String(),
		Email:     e.Email,
		IsPrimary: e.IsPrimary,
		CreatedAt: timestamppb.New(e.CreatedAt),
	}
	if e.VerifiedAt != nil {
		pb.VerifiedAt = timestamppb.New(*e.VerifiedAt)
	}
	return pb
}

func domainToPbExternalIdentity(ei *domain.ExternalIdentity) *identityv1.ExternalIdentity {
	var mdBytes []byte
	if ei.Metadata != nil {
		mdBytes, _ = json.Marshal(ei.Metadata)
	}
	return &identityv1.ExternalIdentity{
		Id:         ei.ID.String(),
		UserId:     ei.UserID.String(),
		Provider:   ei.Provider,
		ExternalId: ei.ExternalID,
		Metadata:   mdBytes,
		LinkedAt:   timestamppb.New(ei.LinkedAt),
	}
}

// startGRPCServer starts a gRPC server for the Identity service.
func (s *Server) startGRPCServer(addr string) (*grpc.Server, net.Listener, error) {
	grpcSrv := grpc.NewServer()
	// The IdentityService is created in New() and available via s.idSvc.
	// We retrieve it from the server struct if available.
	if s.idSvc != nil {
		handler := NewIdentityGRPCHandler(s.idSvc)
		handler.RegisterGRPC(grpcSrv)
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("grpc listen %s: %w", addr, err)
	}

	go func() {
		slog.Info("Identity gRPC server listening", "addr", addr)
		if err := grpcSrv.Serve(lis); err != nil {
			slog.Info("Identity gRPC server stopped", "error", err)
		}
	}()

	return grpcSrv, lis, nil
}
