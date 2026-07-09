// Package tenant provides multi-tenant context propagation and resolution.
package tenant

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type contextKey struct{}

// IsolationLevel defines the data isolation strategy for a tenant.
type IsolationLevel string

const (
	// IsolationShared — all tenants share one DB, isolated by RLS.
	IsolationShared IsolationLevel = "shared"
	// IsolationSchema — tenant gets a dedicated PostgreSQL schema.
	IsolationSchema IsolationLevel = "schema"
	// IsolationDatabase — tenant gets a dedicated database instance.
	IsolationDatabase IsolationLevel = "database"
)

// Context carries tenant-specific information through the request lifecycle.
type Context struct {
	TenantID       uuid.UUID
	IsolationLevel IsolationLevel
	SchemaName     string          // for schema-level isolation
	Settings       map[string]any  // tenant-specific settings
}

// FromContext extracts the tenant context from a context.Context.
func FromContext(ctx context.Context) (*Context, error) {
	tc, ok := ctx.Value(contextKey{}).(*Context)
	if !ok || tc == nil {
		return nil, fmt.Errorf("no tenant context found")
	}
	return tc, nil
}

// WithContext returns a new context with the given tenant context attached.
func WithContext(ctx context.Context, tc *Context) context.Context {
	return context.WithValue(ctx, contextKey{}, tc)
}

// MustFromContext panics if no tenant context is found. Use only in tests.
func MustFromContext(ctx context.Context) *Context {
	tc, err := FromContext(ctx)
	if err != nil {
		panic(err)
	}
	return tc
}
