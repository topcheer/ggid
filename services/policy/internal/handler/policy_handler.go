package handler

import (
	"context"

	pb "github.com/ggid/ggid/api/gen/policy/v1"
	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/ggid/ggid/services/policy/internal/service"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PolicyHandler implements the PolicyService gRPC interface.
type PolicyHandler struct {
	pb.UnimplementedPolicyServiceServer
	policySvc *service.PolicyService
	evaluator *service.Evaluator
}

func NewPolicyHandler(policySvc *service.PolicyService, evaluator *service.Evaluator) *PolicyHandler {
	return &PolicyHandler{policySvc: policySvc, evaluator: evaluator}
}

func (h *PolicyHandler) CreatePolicy(ctx context.Context, req *pb.CreatePolicyRequest) (*pb.Policy, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	policy := &domain.Policy{
		TenantID:    tenantID,
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Effect:      domain.Effect(req.GetEffect()),
		Actions:     req.GetActions(),
		Resources:   req.GetResources(),
		Priority:    int(req.GetPriority()),
	}
	created, err := h.policySvc.CreatePolicy(ctx, policy)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return policyToProto(created), nil
}

func (h *PolicyHandler) GetPolicy(ctx context.Context, req *pb.GetPolicyRequest) (*pb.Policy, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	policy, err := h.policySvc.GetPolicy(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return policyToProto(policy), nil
}

func (h *PolicyHandler) ListPolicies(ctx context.Context, req *pb.ListPoliciesRequest) (*pb.ListPoliciesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	policies, err := h.policySvc.ListPolicies(ctx, tenantID, 1, int(req.GetPageSize()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbPolicies := make([]*pb.Policy, len(policies))
	for i, p := range policies {
		pbPolicies[i] = policyToProto(p)
	}
	return &pb.ListPoliciesResponse{Policies: pbPolicies}, nil
}

func (h *PolicyHandler) DeletePolicy(ctx context.Context, req *pb.DeletePolicyRequest) (*pb.DeletePolicyResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	if err := h.policySvc.DeletePolicy(ctx, id); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.DeletePolicyResponse{}, nil
}

func (h *PolicyHandler) AttachPolicy(ctx context.Context, req *pb.AttachPolicyRequest) (*pb.AttachPolicyResponse, error) {
	policyID, err := uuid.Parse(req.GetPolicyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid policy_id")
	}
	principalID, err := uuid.Parse(req.GetPrincipalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid principal_id")
	}
	if err := h.policySvc.AttachPolicy(ctx, policyID, domain.PrincipalType(req.GetPrincipalType()), principalID); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.AttachPolicyResponse{}, nil
}

func (h *PolicyHandler) DetachPolicy(ctx context.Context, req *pb.DetachPolicyRequest) (*pb.DetachPolicyResponse, error) {
	policyID, err := uuid.Parse(req.GetPolicyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid policy_id")
	}
	principalID, err := uuid.Parse(req.GetPrincipalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid principal_id")
	}
	if err := h.policySvc.DetachPolicy(ctx, policyID, domain.PrincipalType(req.GetPrincipalType()), principalID); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.DetachPolicyResponse{}, nil
}

// Evaluate performs a full permission evaluation with context.
func (h *PolicyHandler) Evaluate(ctx context.Context, req *pb.EvaluateRequest) (*pb.EvaluateResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	result, err := h.evaluator.Check(ctx, &domain.CheckRequest{
		UserID:       userID,
		ResourceType: req.GetResourceType(),
		Action:       req.GetAction(),
		Resource:     req.GetResource(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.EvaluateResponse{
		Allowed:   result.Allowed,
		Reason:    result.Reason,
		MatchedBy: result.MatchedBy,
	}, nil
}

// Check is a simple boolean permission check.
func (h *PolicyHandler) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	result, err := h.evaluator.Check(ctx, &domain.CheckRequest{
		UserID:       userID,
		ResourceType: req.GetResourceType(),
		Action:       req.GetAction(),
		Resource:     req.GetResource(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.CheckResponse{Allowed: result.Allowed}, nil
}

func policyToProto(p *domain.Policy) *pb.Policy {
	return &pb.Policy{
		Id:          p.ID.String(),
		TenantId:    p.TenantID.String(),
		Name:        p.Name,
		Description: p.Description,
		Effect:      string(p.Effect),
		Actions:     p.Actions,
		Resources:   p.Resources,
		Priority:    int32(p.Priority),
	}
}
