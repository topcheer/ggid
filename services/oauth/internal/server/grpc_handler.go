package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"

	oauthv1 "github.com/ggid/ggid/api/gen/oauth/v1"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/ggid/ggid/services/oauth/internal/service"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OAuthGRPCHandler implements OAuthServiceServer by delegating to OAuthService.
type OAuthGRPCHandler struct {
	oauthv1.UnimplementedOAuthServiceServer
	svc *service.OAuthService
}

// NewOAuthGRPCHandler creates a new gRPC handler wrapping the given OAuthService.
func NewOAuthGRPCHandler(svc *service.OAuthService) *OAuthGRPCHandler {
	return &OAuthGRPCHandler{svc: svc}
}

// RegisterGRPC registers the OAuthServiceServer on the given gRPC server.
func (h *OAuthGRPCHandler) RegisterGRPC(s *grpc.Server) {
	oauthv1.RegisterOAuthServiceServer(s, h)
}

func (h *OAuthGRPCHandler) CreateClient(ctx context.Context, req *oauthv1.CreateClientRequest) (*oauthv1.OAuthClient, error) {
	tenantID := tenantIDFromContext(ctx)

	input := &service.CreateClientInput{
		TenantID:                tenantID,
		Name:                    req.GetName(),
		Type:                    pbToDomainClientType(req.GetType()),
		GrantTypes:              req.GetGrantTypes(),
		ResponseTypes:           req.GetResponseTypes(),
		RedirectURIs:            req.GetRedirectUris(),
		Scopes:                  req.GetScopes(),
		TokenEndpointAuthMethod: req.GetTokenEndpointAuthMethod(),
	}
	if len(req.GetMetadata()) > 0 {
		var md map[string]any
		if err := json.Unmarshal(req.GetMetadata(), &md); err == nil {
			input.Metadata = md
		}
	}

	result, err := h.svc.CreateClient(ctx, input)
	if err != nil {
		slog.Error("gRPC CreateClient error", "err", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create client: %v", err))
	}
	pbClient := domainToPbClient(result.Client)
	pbClient.ClientSecret = result.ClientSecret // plaintext secret — only returned on create
	return pbClient, nil
}

func (h *OAuthGRPCHandler) GetClient(ctx context.Context, req *oauthv1.GetClientRequest) (*oauthv1.OAuthClient, error) {
	client, err := h.svc.GetClient(ctx, req.GetClientId())
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("client not found: %v", err))
	}
	return domainToPbClient(client), nil
}

func (h *OAuthGRPCHandler) ListClients(ctx context.Context, req *oauthv1.ListClientsRequest) (*oauthv1.ListClientsResponse, error) {
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = 50
	}
	offset := 0
	if req.GetPageToken() != "" {
		if v, err := fmt.Sscanf(req.GetPageToken(), "%d", &offset); v == 1 && err != nil {
			// ignore parse error, use offset 0
		}
	}

	clients, total, err := h.svc.ListClients(ctx, pageSize, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list clients: %v", err))
	}

	pbClients := make([]*oauthv1.OAuthClient, 0, len(clients))
	for _, c := range clients {
		pbClients = append(pbClients, domainToPbClient(c))
	}

	nextToken := ""
	if offset+pageSize < total {
		nextToken = fmt.Sprintf("%d", offset+pageSize)
	}

	return &oauthv1.ListClientsResponse{
		Clients:       pbClients,
		NextPageToken: nextToken,
	}, nil
}

func (h *OAuthGRPCHandler) UpdateClient(ctx context.Context, req *oauthv1.UpdateClientRequest) (*oauthv1.OAuthClient, error) {
	updates := &service.ClientMetadataUpdate{
		RedirectURIs: req.GetRedirectUris(),
		Scopes:       req.GetScopes(),
	}
	if req.Name != nil {
		updates.Name = req.Name
	}

	if len(req.GetMetadata()) > 0 {
		var md map[string]any
		if err := json.Unmarshal(req.GetMetadata(), &md); err == nil {
			updates.Metadata = md
		}
	}

	client, err := h.svc.UpdateClientMetadata(ctx, req.GetClientId(), updates)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update client: %v", err))
	}
	return domainToPbClient(client), nil
}

func (h *OAuthGRPCHandler) DeleteClient(ctx context.Context, req *oauthv1.DeleteClientRequest) (*oauthv1.DeleteClientResponse, error) {
	if err := h.svc.DeleteClient(ctx, req.GetClientId()); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete client: %v", err))
	}
	return &oauthv1.DeleteClientResponse{Success: true}, nil
}

// --- helpers ---

func pbToDomainClientType(t oauthv1.ClientType) domain.ClientType {
	switch t {
	case oauthv1.ClientType_CLIENT_TYPE_CONFIDENTIAL:
		return domain.ClientTypeConfidential
	case oauthv1.ClientType_CLIENT_TYPE_PUBLIC:
		return domain.ClientTypePublic
	default:
		return domain.ClientTypeConfidential
	}
}

func domainToPbClientType(t domain.ClientType) oauthv1.ClientType {
	switch t {
	case domain.ClientTypeConfidential:
		return oauthv1.ClientType_CLIENT_TYPE_CONFIDENTIAL
	case domain.ClientTypePublic:
		return oauthv1.ClientType_CLIENT_TYPE_PUBLIC
	default:
		return oauthv1.ClientType_CLIENT_TYPE_UNSPECIFIED
	}
}

func domainToPbClient(c *domain.OAuthClient) *oauthv1.OAuthClient {
	var metadataBytes []byte
	if c.Metadata != nil {
		metadataBytes, _ = json.Marshal(c.Metadata)
	}
	return &oauthv1.OAuthClient{
		Id:                      c.ID.String(),
		TenantId:                c.TenantID.String(),
		ClientId:                c.ClientID,
		Name:                    c.Name,
		Type:                    domainToPbClientType(c.Type),
		GrantTypes:              c.GrantTypes,
		ResponseTypes:           c.ResponseTypes,
		RedirectUris:            c.RedirectURIs,
		Scopes:                  c.Scopes,
		TokenEndpointAuthMethod: c.TokenEndpointAuthMethod,
		Metadata:                metadataBytes,
		Enabled:                 c.Enabled,
		CreatedAt:               timestamppb.New(c.CreatedAt),
		UpdatedAt:               timestamppb.New(c.UpdatedAt),
	}
}

func tenantIDFromContext(_ context.Context) uuid.UUID {
	// Default tenant for gRPC calls without explicit tenant context.
	// Configured via GGID_TENANT_ID (preferred) or DEFAULT_TENANT_ID;
	// uuid.Nil when unset — never a hardcoded tenant UUID.
	tenantStr := os.Getenv("GGID_TENANT_ID")
	if tenantStr == "" {
		tenantStr = os.Getenv("DEFAULT_TENANT_ID")
	}
	id, err := uuid.Parse(tenantStr)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// startGRPCServer starts a gRPC server for the OAuth service on the given address.
// It returns the server and listener; the caller should defer GracefulStop and close.
func (s *Server) startGRPCServer(addr string) (*grpc.Server, net.Listener, error) {
	grpcSrv := newOAuthGRPCServer()
	handler := NewOAuthGRPCHandler(s.oauthSvc)
	handler.RegisterGRPC(grpcSrv)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("grpc listen %s: %w", addr, err)
	}

	go func() {
		slog.Info("OAuth gRPC server listening", "addr", addr)
		if err := grpcSrv.Serve(lis); err != nil {
			slog.Info("OAuth gRPC server stopped", "error", err)
		}
	}()

	return grpcSrv, lis, nil
}

func newOAuthGRPCServer() *grpc.Server {
	// TLS support: when GRPC_TLS_ENABLED=true, attempt TLS credentials.
	if os.Getenv("GRPC_TLS_ENABLED") == "true" {
		certFile := os.Getenv("GRPC_TLS_CERT")
		keyFile := os.Getenv("GRPC_TLS_KEY")
		if certFile != "" && keyFile != "" {
			creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
			if err != nil {
				if os.Getenv("GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK") != "true" {
					log.Fatalf("GRPC_TLS_ENABLED but cert/key invalid: %v; refusing to start. Set GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true only in dev.", err)
				}
				slog.Info("GRPC TLS cert/key invalid, falling back to plaintext", "error", err)
			} else {
				return grpc.NewServer(grpc.Creds(creds))
			}
		}
	}
	return grpc.NewServer()
}

// unused but kept for reference
var _ = os.Getenv
