package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	authv1 "github.com/ggid/ggid/api/gen/auth/v1"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/ggid/ggid/services/auth/internal/service"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthGRPCHandler implements AuthServiceServer by delegating to AuthService.
type AuthGRPCHandler struct {
	authv1.UnimplementedAuthServiceServer
	svc *service.AuthService
}

func NewAuthGRPCHandler(svc *service.AuthService) *AuthGRPCHandler {
	return &AuthGRPCHandler{svc: svc}
}

func (h *AuthGRPCHandler) RegisterGRPC(s *grpc.Server) {
	authv1.RegisterAuthServiceServer(s, h)
}

func (h *AuthGRPCHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	tokens, err := h.svc.Login(ctx, req.GetUsername(), req.GetPassword(), "", "")
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, fmt.Sprintf("login failed: %v", err))
	}

	resp := &authv1.LoginResponse{
		Tokens:      domainToPbTokenSet(tokens),
		MfaRequired: tokens.MFARequired,
	}
	return resp, nil
}

func (h *AuthGRPCHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	tenantID := defaultAuthTenantID()
	userID := uuid.New()

	if err := h.svc.Register(ctx, tenantID, userID, req.GetUsername(), req.GetPassword()); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("register failed: %v", err))
	}

	return &authv1.RegisterResponse{UserId: userID.String()}, nil
}

func (h *AuthGRPCHandler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if err := h.svc.Logout(ctx, req.GetRefreshToken()); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("logout failed: %v", err))
	}
	return &authv1.LogoutResponse{}, nil
}

func (h *AuthGRPCHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.TokenSet, error) {
	tokens, err := h.svc.Refresh(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, fmt.Sprintf("refresh failed: %v", err))
	}
	return domainToPbTokenSet(tokens), nil
}

func (h *AuthGRPCHandler) ForgotPassword(ctx context.Context, req *authv1.ForgotPasswordRequest) (*authv1.ForgotPasswordResponse, error) {
	tenantID := defaultAuthTenantID()
	if err := h.svc.ForgotPassword(ctx, tenantID, req.GetEmail()); err != nil {
		// Always return success to prevent email enumeration.
		return &authv1.ForgotPasswordResponse{ResetInitiated: true}, nil
	}
	return &authv1.ForgotPasswordResponse{ResetInitiated: true}, nil
}

func (h *AuthGRPCHandler) ResetPassword(ctx context.Context, req *authv1.ResetPasswordRequest) (*authv1.ResetPasswordResponse, error) {
	if err := h.svc.ResetPassword(ctx, req.GetToken(), req.GetNewPassword()); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("reset password failed: %v", err))
	}
	return &authv1.ResetPasswordResponse{}, nil
}

func (h *AuthGRPCHandler) ChangePassword(ctx context.Context, req *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error) {
	tenantID := defaultAuthTenantID()
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	if err := h.svc.ChangePassword(ctx, tenantID, userID, req.GetOldPassword(), req.GetNewPassword()); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("change password failed: %v", err))
	}
	return &authv1.ChangePasswordResponse{}, nil
}

func (h *AuthGRPCHandler) ListSessions(ctx context.Context, req *authv1.ListSessionsRequest) (*authv1.ListSessionsResponse, error) {
	tenantID := defaultAuthTenantID()
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	sessions, err := h.svc.ListSessions(ctx, tenantID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("list sessions failed: %v", err))
	}
	pbSessions := make([]*authv1.SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		pbSessions = append(pbSessions, domainToPbSessionInfo(s))
	}
	return &authv1.ListSessionsResponse{Sessions: pbSessions}, nil
}

func (h *AuthGRPCHandler) RevokeSession(ctx context.Context, req *authv1.RevokeSessionRequest) (*authv1.RevokeSessionResponse, error) {
	sessionID, err := uuid.Parse(req.GetSessionId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid session id")
	}
	if err := h.svc.RevokeSession(ctx, sessionID); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("revoke session failed: %v", err))
	}
	return &authv1.RevokeSessionResponse{}, nil
}

// --- helpers ---

func domainToPbTokenSet(t *domain.TokenSet) *authv1.TokenSet {
	return &authv1.TokenSet{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
		ExpiresIn:    int32(t.ExpiresIn),
		SessionId:    t.SessionID,
	}
}

func domainToPbSessionInfo(s *domain.Session) *authv1.SessionInfo {
	return &authv1.SessionInfo{
		Id:        s.ID.String(),
		IpAddress: s.IPAddress,
		UserAgent: s.UserAgent,
		CreatedAt: timestamppb.New(s.CreatedAt),
		ExpiresAt: timestamppb.New(s.ExpiresAt),
	}
}

func defaultAuthTenantID() uuid.UUID {
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

// AuthGRPCServer starts a gRPC server for the Auth service on the given address.
// It returns the server so the caller can GracefulStop on shutdown.
func StartAuthGRPCServer(addr string, svc *service.AuthService) (*grpc.Server, net.Listener, error) {
	grpcSrv := grpc.NewServer()
	handler := NewAuthGRPCHandler(svc)
	handler.RegisterGRPC(grpcSrv)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("auth grpc listen %s: %w", addr, err)
	}

	go func() {
		log.Printf("Auth gRPC server listening on %s", addr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Printf("Auth gRPC server stopped: %v", err)
		}
	}()

	return grpcSrv, lis, nil
}
