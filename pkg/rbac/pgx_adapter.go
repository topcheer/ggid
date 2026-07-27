package rbac

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxPool implements PoolExecutor using *pgxpool.Pool.
type PgxPool struct {
	Pool *pgxpool.Pool
}

func (p *PgxPool) Exec(ctx context.Context, sql string, args ...any) (CommandTag, error) {
	tag, err := p.Pool.Exec(ctx, sql, args...)
	return &pgxTag{tag}, err
}

type pgxTag struct {
	tag pgconn.CommandTag
}

func (t *pgxTag) RowsAffected() int64 {
	return t.tag.RowsAffected()
}

// NewPgxPool wraps a pgxpool.Pool for EnsureSystemPermissions.
func NewPgxPool(pool *pgxpool.Pool) *PgxPool {
	return &PgxPool{Pool: pool}
}
