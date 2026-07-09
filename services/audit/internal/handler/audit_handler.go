// Package handler implements the gRPC handler for the Audit Service.
package handler

import (
	"context"
	"time"

	pb "github.com/ggid/ggid/api/gen/audit/v1"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/ggid/ggid/services/audit/internal/service"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuditHandler implements the AuditService gRPC interface.
type AuditHandler struct {
	pb.UnimplementedAuditServiceServer
	svc *service.AuditService
}

func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

func (h *AuditHandler) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	filter := domain.ListFilter{
		TenantID:     tenantID,
		Action:       req.GetAction(),
		ResourceType: req.GetResourceType(),
		Result:       domain.EventResult(req.GetResult()),
		OrderBy:      req.GetOrderBy(),
		Descending:   req.GetDescending(),
	}

	if req.GetActorId() != "" {
		actorID, err := uuid.Parse(req.GetActorId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid actor_id")
		}
		filter.ActorID = &actorID
	}

	if req.GetStartTime() != nil {
		t := req.GetStartTime().AsTime()
		filter.StartTime = &t
	}
	if req.GetEndTime() != nil {
		t := req.GetEndTime().AsTime()
		filter.EndTime = &t
	}

	events, total, err := h.svc.ListEvents(ctx, filter, 1, int(req.GetPageSize()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	pbEvents := make([]*pb.AuditEvent, len(events))
	for i, e := range events {
		pbEvents[i] = eventToProto(e)
	}

	return &pb.ListEventsResponse{
		Events: pbEvents,
		Total:  int32(total),
	}, nil
}

func (h *AuditHandler) GetEvent(ctx context.Context, req *pb.GetEventRequest) (*pb.AuditEvent, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	event, err := h.svc.GetEvent(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return eventToProto(event), nil
}

func eventToProto(e *domain.AuditEvent) *pb.AuditEvent {
	p := &pb.AuditEvent{
		Id:           e.ID.String(),
		TenantId:     e.TenantID.String(),
		ActorType:    string(e.ActorType),
		ActorName:    e.ActorName,
		Action:       e.Action,
		ResourceType: e.ResourceType,
		ResourceName: e.ResourceName,
		Result:       string(e.Result),
		IpAddress:    e.IPAddress,
		UserAgent:    e.UserAgent,
		RequestId:    e.RequestID,
	}
	if e.ActorID != uuid.Nil {
		p.ActorId = e.ActorID.String()
	}
	if e.ResourceID != uuid.Nil {
		p.ResourceId = e.ResourceID.String()
	}
	if e.Metadata != nil {
		if s, err := structpb.NewStruct(e.Metadata); err == nil {
			p.Metadata = s
		}
	}
	if !e.CreatedAt.IsZero() {
		p.CreatedAt = timestamppb.New(e.CreatedAt)
	}
	return p
}

// toGRPCError converts a GGIDError to a gRPC status error.
func toGRPCError(err error) error {
	if ge, ok := errors.AsGGIDError(err); ok {
		switch ge.Code {
		case errors.ErrNotFound:
			return status.Error(codes.NotFound, ge.Message)
		case errors.ErrInvalidArgument:
			return status.Error(codes.InvalidArgument, ge.Message)
		case errors.ErrPermissionDenied:
			return status.Error(codes.PermissionDenied, ge.Message)
		default:
			return status.Error(codes.Internal, ge.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}

// Ensure time import is used (needed for potential future time parsing).
var _ = time.Now
