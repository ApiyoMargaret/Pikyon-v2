// Package db owns the two PostgreSQL connection pools and the user-scoped transaction helper
// that together enforce database-level tenant isolation (ADR-018).
//
//   - Pools.App connects as the restricted pikyon_app role. Row level security applies to it.
//     Use it only through WithUser, which tells the database which user is making the request.
//   - Pools.Admin connects as the owner role and bypasses RLS. Use it only for work that has
//     no user context or is deliberately cross-tenant: login/register/refresh, cron jobs,
//     public share-link resolution, creating notifications for other users, AI job results.
//
// Both pools expect Supabase's SESSION pooler (port 5432). The transaction pooler (port 6543)
// does not support the prepared statements pgx uses by default.
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/config"
)

// ErrInvalidUserID is returned by WithUser when the user ID is not a valid UUID.
var ErrInvalidUserID = errors.New("db: invalid user id")

// Querier is the subset of pgx that repositories use. Both *pgxpool.Pool and pgx.Tx satisfy it,
// so a repository method works the same inside a user-scoped transaction or on the admin pool.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Pools holds the restricted application pool and the privileged admin pool.
type Pools struct {
	App   *pgxpool.Pool
	Admin *pgxpool.Pool
}

// Connect opens both pools, verifies connectivity, and verifies that the app pool really is a
// restricted role. It fails fast so a misconfiguration cannot silently bypass RLS.
func Connect(ctx context.Context, cfg *config.Config) (*Pools, error) {
	app, err := newPool(ctx, "app", cfg.AppDatabaseURL, cfg.DBAppMaxConns)
	if err != nil {
		return nil, err
	}
	admin, err := newPool(ctx, "admin", cfg.DatabaseURL, cfg.DBAdminMaxConns)
	if err != nil {
		app.Close()
		return nil, err
	}
	p := &Pools{App: app, Admin: admin}

	if err := verifyAppRole(ctx, app); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}

func newPool(ctx context.Context, name, url string, maxConns int32) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(url)
	if err != nil {
		// Deliberately not wrapped: the underlying error can contain parts of the URL.
		return nil, fmt.Errorf("db: invalid %s database URL", name)
	}
	pc.MaxConns = maxConns
	pc.MinConns = 0 // free tier: do not hold idle connections
	pc.MaxConnLifetime = 30 * time.Minute
	pc.MaxConnIdleTime = 5 * time.Minute
	pc.HealthCheckPeriod = time.Minute
	pc.ConnConfig.RuntimeParams["application_name"] = "pikyon-" + name

	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("db: create %s pool: %w", name, err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: connect %s pool: %w", name, err)
	}
	return pool, nil
}

// verifyAppRole refuses to run if the app pool's role can bypass row level security.
func verifyAppRole(ctx context.Context, pool *pgxpool.Pool) error {
	var role string
	var bypass, super bool
	err := pool.QueryRow(ctx,
		`SELECT r.rolname, r.rolbypassrls, r.rolsuper FROM pg_roles r WHERE r.rolname = current_user`,
	).Scan(&role, &bypass, &super)
	if err != nil {
		return fmt.Errorf("db: check app role: %w", err)
	}
	if bypass || super {
		return fmt.Errorf("db: APP_DATABASE_URL connects as role %q, which bypasses row level security; use the restricted pikyon_app role", role)
	}
	return nil
}

// Close closes both pools.
func (p *Pools) Close() {
	if p.App != nil {
		p.App.Close()
	}
	if p.Admin != nil {
		p.Admin.Close()
	}
}

// Ping checks both pools. Use it for readiness checks.
func (p *Pools) Ping(ctx context.Context) error {
	if err := p.App.Ping(ctx); err != nil {
		return fmt.Errorf("db: ping app pool: %w", err)
	}
	if err := p.Admin.Ping(ctx); err != nil {
		return fmt.Errorf("db: ping admin pool: %w", err)
	}
	return nil
}

// WithUser runs fn inside a transaction on the restricted pool with the database told which
// user is making the request. Row level security policies read that setting, so fn can only see
// and change that user's rows no matter what SQL it runs.
//
// The user ID must come from the verified JWT, never from the request body. The setting is
// transaction-local, so it cannot leak to the next request that reuses the connection. The
// transaction commits if fn returns nil and rolls back on error or panic.
func (p *Pools) WithUser(ctx context.Context, userID string, fn func(q Querier) error) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidUserID
	}

	tx, err := p.App.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("db: begin user transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			// WithoutCancel: still roll back if the request context was cancelled.
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", id.String()); err != nil {
		return fmt.Errorf("db: set user context: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("db: commit user transaction: %w", err)
	}
	committed = true
	return nil
}
