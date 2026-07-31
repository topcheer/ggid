// Package handler provides shared gRPC security utilities for policy handlers.
package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// tenantFromMetadata extracts the caller's tenant ID from gRPC metadata.
// This is set by the gateway (from verified JWT claims) and is the trustworthy
// source of the caller's tenant — request body tenant fields are not.
func tenantFromMetadata(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-tenant-id"); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

// validateTenantFromMetadata cross-checks the request body tenant_id against
// the gRPC metadata tenant. If metadata is present and differs from the
// request, it returns PermissionDenied. If metadata is absent (backward
// compatibility), it returns nil (the request tenant is trusted).
func validateTenantFromMetadata(ctx context.Context, reqTenantID string) error {
	mdTenant := tenantFromMetadata(ctx)
	if mdTenant != "" && mdTenant != reqTenantID {
		return status.Error(codes.PermissionDenied, "tenant mismatch: request tenant does not match caller tenant")
	}
	return nil
}
