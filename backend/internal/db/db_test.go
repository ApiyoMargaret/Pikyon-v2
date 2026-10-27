package db_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/config"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/db"
)

// --- Unit tests (no database needed) ---

func TestWithUser_RejectsInvalidUserID(t *testing.T) {
	p := &db.Pools{} // pools are never touched: validation happens first
	for _, bad := range []string{"", "not-a-uuid", "123", "' OR 1=1 --"} {
		called := false
		err := p.WithUser(context.Background(), bad, func(db.Querier) error {
			called = true
			return nil
		})
		if !errors.Is(err, db.ErrInvalidUserID) {
			t.Errorf("WithUser(%q) error = %v, want ErrInvalidUserID", bad, err)
		}
		if called {
			t.Errorf("WithUser(%q) must not run the callback", bad)
		}
	}
}

func TestConnect_InvalidURLDoesNotLeakPassword(t *testing.T) {
	const pw = "Sup3rSecretPassw0rd"
	cfg := &config.Config{
		AppDatabaseURL:  "postgresql://pikyon_app:" + pw + "@host.example.com:notaport/postgres",
		DatabaseURL:     "postgresql://postgres:" + pw + "@host.example.com:5432/postgres",
		DBAppMaxConns:   1,
		DBAdminMaxConns: 1,
	}
	_, err := db.Connect(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), pw) {
		t.Errorf("error leaks the password: %v", err)
	}
}

// --- Integration tests ---
// They need a migrated database (backend/migrations) and two logins. Set:
//   TEST_DATABASE_URL      owner/admin connection
//   TEST_APP_DATABASE_URL  pikyon_app connection
// They are skipped when these are not set.

func integrationConfig(t *testing.T, appConns int32) *config.Config {
	t.Helper()
	admin, app := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_APP_DATABASE_URL")
	if admin == "" || app == "" {
		t.Skip("TEST_DATABASE_URL and TEST_APP_DATABASE_URL not set; skipping integration test")
	}
	return &config.Config{DatabaseURL: admin, AppDatabaseURL: app, DBAppMaxConns: appConns, DBAdminMaxConns: 2}
}

func connect(t *testing.T, appConns int32) *db.Pools {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	p, err := db.Connect(ctx, integrationConfig(t, appConns))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(p.Close)
	return p
}

// newUser creates a user on the admin pool and removes it (and its memories) afterwards.
func newUser(t *testing.T, p *db.Pools) string {
	t.Helper()
	ctx := context.Background()
	var id string
	email := fmt.Sprintf("dbtest-%s@test.dev", uuid.NewString())
	err := p.Admin.QueryRow(ctx,
		`INSERT INTO users (email, name, password_hash) VALUES ($1, 'T', 'x') RETURNING id`, email).Scan(&id)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { _, _ = p.Admin.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id) })
	return id
}

func addMemory(ctx context.Context, q db.Querier, userID, title string) error {
	_, err := q.Exec(ctx, `INSERT INTO memories (user_id, title) VALUES ($1, $2)`, userID, title)
	return err
}

func countMemories(ctx context.Context, q db.Querier) (n int, err error) {
	err = q.QueryRow(ctx, `SELECT count(*) FROM memories`).Scan(&n)
	return
}

func TestIntegration_ConnectVerifiesRestrictedRole(t *testing.T) {
	p := connect(t, 2)
	var role string
	if err := p.App.QueryRow(context.Background(), `SELECT current_user`).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(role, "pikyon_app") {
		t.Errorf("app pool role = %q, want pikyon_app", role)
	}
	if err := p.Ping(context.Background()); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestIntegration_ConnectRefusesOwnerRoleForAppPool(t *testing.T) {
	cfg := integrationConfig(t, 2)
	cfg.AppDatabaseURL = cfg.DatabaseURL // owner credentials used for the app pool: dangerous
	_, err := db.Connect(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "bypasses row level security") {
		t.Fatalf("expected a bypass-RLS refusal, got: %v", err)
	}
}

func TestIntegration_WithUserIsolatesTenants(t *testing.T) {
	p := connect(t, 2)
	ctx := context.Background()
	a, b := newUser(t, p), newUser(t, p)

	if err := p.WithUser(ctx, a, func(q db.Querier) error { return addMemory(ctx, q, a, "A's memory") }); err != nil {
		t.Fatalf("A insert: %v", err)
	}
	if err := p.WithUser(ctx, b, func(q db.Querier) error { return addMemory(ctx, q, b, "B's memory") }); err != nil {
		t.Fatalf("B insert: %v", err)
	}

	for name, id := range map[string]string{"A": a, "B": b} {
		err := p.WithUser(ctx, id, func(q db.Querier) error {
			n, err := countMemories(ctx, q)
			if err != nil {
				return err
			}
			if n != 1 {
				return fmt.Errorf("%s sees %d memories, want exactly 1 (their own)", name, n)
			}
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	}

	// A user cannot forge ownership even by writing SQL that names someone else.
	err := p.WithUser(ctx, a, func(q db.Querier) error { return addMemory(ctx, q, b, "forged") })
	if err == nil {
		t.Error("A inserting a memory owned by B must fail")
	}

	// The admin pool bypasses RLS and sees both.
	var total int
	if err := p.Admin.QueryRow(ctx, `SELECT count(*) FROM memories WHERE user_id = ANY($1)`, []string{a, b}).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Errorf("admin sees %d of the two test memories, want 2", total)
	}
}

func TestIntegration_UserContextDoesNotLeakAcrossReusedConnections(t *testing.T) {
	p := connect(t, 1) // a single connection forces reuse
	ctx := context.Background()
	a := newUser(t, p)
	if err := p.WithUser(ctx, a, func(q db.Querier) error { return addMemory(ctx, q, a, "A's memory") }); err != nil {
		t.Fatal(err)
	}
	// Same connection, no user context: must see nothing, and must not error.
	n, err := countMemories(ctx, p.App)
	if err != nil {
		t.Fatalf("query without user context errored: %v", err)
	}
	if n != 0 {
		t.Errorf("connection without user context sees %d memories, want 0", n)
	}
}

func TestIntegration_WithUserRollsBackOnErrorAndPanic(t *testing.T) {
	p := connect(t, 1)
	ctx := context.Background()
	a := newUser(t, p)
	count := func() int {
		var n int
		if err := p.Admin.QueryRow(ctx, `SELECT count(*) FROM memories WHERE user_id = $1`, a).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	boom := errors.New("boom")
	err := p.WithUser(ctx, a, func(q db.Querier) error {
		if err := addMemory(ctx, q, a, "will be rolled back"); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want boom", err)
	}
	if n := count(); n != 0 {
		t.Errorf("after error: %d rows persisted, want 0", n)
	}

	func() {
		defer func() { _ = recover() }()
		_ = p.WithUser(ctx, a, func(q db.Querier) error {
			_ = addMemory(ctx, q, a, "will be rolled back too")
			panic("handler bug")
		})
	}()
	if n := count(); n != 0 {
		t.Errorf("after panic: %d rows persisted, want 0", n)
	}

	// The single connection must still be usable afterwards.
	if err := p.WithUser(ctx, a, func(q db.Querier) error { return addMemory(ctx, q, a, "works") }); err != nil {
		t.Errorf("pool unusable after panic: %v", err)
	}
	if n := count(); n != 1 {
		t.Errorf("final count = %d, want 1", n)
	}
}

func TestIntegration_AppRoleCannotReadSecrets(t *testing.T) {
	p := connect(t, 2)
	ctx := context.Background()
	a := newUser(t, p)
	err := p.WithUser(ctx, a, func(q db.Querier) error {
		var h string
		return q.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, a).Scan(&h)
	})
	if err == nil {
		t.Error("app role must not be able to read password_hash")
	}
}
