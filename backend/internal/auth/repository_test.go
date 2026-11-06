package auth_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/config"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/auth"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/db"
)

// These are integration tests. They need a migrated database (backend/migrations) and two
// logins. Set:
//
//	TEST_DATABASE_URL      owner/admin connection
//	TEST_APP_DATABASE_URL  pikyon_app connection
//
// They are skipped when these are not set. Every row they create is removed afterwards.

func testPools(t *testing.T) *db.Pools {
	t.Helper()
	admin, app := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_APP_DATABASE_URL")
	if admin == "" || app == "" {
		t.Skip("TEST_DATABASE_URL and TEST_APP_DATABASE_URL not set; skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	p, err := db.Connect(ctx, &config.Config{DatabaseURL: admin, AppDatabaseURL: app, DBAppMaxConns: 2, DBAdminMaxConns: 10})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(p.Close)
	return p
}

func newRepo(t *testing.T) (auth.Repository, *db.Pools) {
	t.Helper()
	p := testPools(t)
	return auth.NewRepository(p.Admin), p
}

// newUser creates a user through the repository and removes it (cascading to its sessions) later.
func newUser(t *testing.T, r auth.Repository, p *db.Pools) *auth.User {
	t.Helper()
	u, err := r.CreateUser(context.Background(), fmt.Sprintf("auth-%s@test.dev", uuid.NewString()), "Test User", "$2a$04$hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { _, _ = p.Admin.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, u.ID) })
	return u
}

func tokenHash() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b) // 64 hex characters, like a SHA-256
}

func sessionFor(userID string) *auth.Session {
	return &auth.Session{UserID: userID, TokenHash: tokenHash(), DeviceInfo: "test-agent", IPAddress: "203.0.113.9", ExpiresAt: time.Now().Add(time.Hour)}
}

func mustCreate(t *testing.T, r auth.Repository, s *auth.Session) *auth.Session {
	t.Helper()
	if err := r.CreateSession(context.Background(), s); err != nil {
		t.Fatalf("create session: %v", err)
	}
	return s
}

// --- Users ---

func TestRepo_CreateAndGetUser(t *testing.T) {
	r, p := newRepo(t)
	ctx := context.Background()
	u := newUser(t, r, p)

	if u.ID == "" || u.Name != "Test User" || !u.IsActive || u.EmailVerified || u.CreatedAt.IsZero() || u.PasswordChangedAt.IsZero() {
		t.Errorf("user = %+v", u)
	}
	if u.PasswordHash == nil || *u.PasswordHash != "$2a$04$hash" {
		t.Errorf("password hash = %v", u.PasswordHash)
	}
	if u.AvatarURL != nil {
		t.Errorf("avatar should be nil, got %v", *u.AvatarURL)
	}

	byEmail, err := r.GetUserByEmail(ctx, u.Email)
	if err != nil || byEmail.ID != u.ID {
		t.Errorf("GetUserByEmail = %+v, %v", byEmail, err)
	}
	byID, err := r.GetUserByID(ctx, u.ID)
	if err != nil || byID.Email != u.Email {
		t.Errorf("GetUserByID = %+v, %v", byID, err)
	}
}

func TestRepo_DuplicateEmailIsErrEmailTaken(t *testing.T) {
	r, p := newRepo(t)
	u := newUser(t, r, p)
	_, err := r.CreateUser(context.Background(), u.Email, "Other", "$2a$04$hash")
	if !errors.Is(err, auth.ErrEmailTaken) {
		t.Errorf("error = %v, want ErrEmailTaken", err)
	}
}

// The database refuses uppercase emails; the service must lowercase before it gets here.
func TestRepo_UppercaseEmailIsRejectedByTheDatabase(t *testing.T) {
	r, _ := newRepo(t)
	_, err := r.CreateUser(context.Background(), "Upper-"+uuid.NewString()+"@Test.dev", "X", "$2a$04$hash")
	if err == nil || errors.Is(err, auth.ErrEmailTaken) {
		t.Errorf("expected a constraint error, got %v", err)
	}
}

func TestRepo_NotFound(t *testing.T) {
	r, _ := newRepo(t)
	ctx := context.Background()
	if _, err := r.GetUserByEmail(ctx, "nobody-"+uuid.NewString()+"@test.dev"); !errors.Is(err, auth.ErrNotFound) {
		t.Errorf("GetUserByEmail err = %v", err)
	}
	if _, err := r.GetUserByID(ctx, uuid.NewString()); !errors.Is(err, auth.ErrNotFound) {
		t.Errorf("GetUserByID err = %v", err)
	}
	if _, err := r.GetSessionByTokenHash(ctx, tokenHash()); !errors.Is(err, auth.ErrNotFound) {
		t.Errorf("GetSessionByTokenHash err = %v", err)
	}
}

// A Google-only account has no password; scanning it must not fail.
func TestRepo_GoogleOnlyUserHasNilPasswordHash(t *testing.T) {
	r, p := newRepo(t)
	ctx := context.Background()
	var id string
	err := p.Admin.QueryRow(ctx,
		`INSERT INTO users (email, name, google_id, avatar_url) VALUES ($1, 'G', $2, 'https://x/a.png') RETURNING id`,
		fmt.Sprintf("g-%s@test.dev", uuid.NewString()), "google-"+uuid.NewString()).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = p.Admin.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id) })

	u, err := r.GetUserByID(ctx, id)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if u.PasswordHash != nil || u.AvatarURL == nil || *u.AvatarURL != "https://x/a.png" {
		t.Errorf("user = %+v", u)
	}
}

// --- Sessions ---

func TestRepo_CreateAndGetSession(t *testing.T) {
	r, p := newRepo(t)
	u := newUser(t, r, p)
	s := mustCreate(t, r, sessionFor(u.ID))
	if s.ID == "" || s.CreatedAt.IsZero() || s.LastUsedAt.IsZero() {
		t.Errorf("IDs/timestamps not filled: %+v", s)
	}

	got, err := r.GetSessionByTokenHash(context.Background(), s.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != s.ID || got.UserID != u.ID || got.TokenHash != s.TokenHash || got.IsRevoked || got.RotatedAt != nil || got.RotatedFrom != nil {
		t.Errorf("session = %+v", got)
	}
	if got.DeviceInfo != "test-agent" || got.IPAddress != "203.0.113.9" {
		t.Errorf("device/ip = %q / %q (the IP must have no /32 suffix)", got.DeviceInfo, got.IPAddress)
	}
}

func TestRepo_SessionOptionalFields(t *testing.T) {
	r, p := newRepo(t)
	u := newUser(t, r, p)
	ctx := context.Background()

	for name, tc := range map[string]struct{ ip, wantIP string }{
		"empty ip":   {"", ""},
		"garbage ip": {"not-an-ip", ""},
		"ipv6":       {"2001:db8::1", "2001:db8::1"},
		"plain ipv4": {"203.0.113.7", "203.0.113.7"},
	} {
		s := sessionFor(u.ID)
		s.IPAddress = tc.ip
		mustCreate(t, r, s)
		got, err := r.GetSessionByTokenHash(ctx, s.TokenHash)
		if err != nil || got.IPAddress != tc.wantIP {
			t.Errorf("%s: ip = %q, err = %v, want %q", name, got.IPAddress, err, tc.wantIP)
		}
	}

	long := sessionFor(u.ID)
	long.DeviceInfo = strings.Repeat("x", 2000)
	mustCreate(t, r, long)
	got, _ := r.GetSessionByTokenHash(ctx, long.TokenHash)
	if len(got.DeviceInfo) != 512 {
		t.Errorf("device info length = %d, want it cut to 512", len(got.DeviceInfo))
	}

	none := sessionFor(u.ID)
	none.DeviceInfo = ""
	mustCreate(t, r, none)
	if got, _ := r.GetSessionByTokenHash(ctx, none.TokenHash); got.DeviceInfo != "" {
		t.Errorf("empty device info = %q", got.DeviceInfo)
	}
}

// Only a SHA-256 hex digest may be stored: the raw token must never reach the table.
func TestRepo_RefusesToStoreAnythingButAHash(t *testing.T) {
	r, p := newRepo(t)
	u := newUser(t, r, p)
	for _, bad := range []string{"raw-token-abc", strings.Repeat("g", 64), strings.Repeat("A", 64), tokenHash()[:63]} {
		s := sessionFor(u.ID)
		s.TokenHash = bad
		if err := r.CreateSession(context.Background(), s); err == nil {
			t.Errorf("token hash %q must be rejected", bad)
		}
	}
}

// --- RotateSession ---

func TestRepo_RotateSession(t *testing.T) {
	r, p := newRepo(t)
	ctx := context.Background()
	u := newUser(t, r, p)
	old := mustCreate(t, r, sessionFor(u.ID))

	next := sessionFor("")
	next.UserID = "" // must be taken from the old session, never from the caller
	if err := r.RotateSession(ctx, old.TokenHash, next); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if next.ID == "" || next.UserID != u.ID || next.RotatedFrom == nil || *next.RotatedFrom != old.ID {
		t.Errorf("new session = %+v", next)
	}

	oldNow, _ := r.GetSessionByTokenHash(ctx, old.TokenHash)
	if !oldNow.IsRevoked || oldNow.RotatedAt == nil {
		t.Errorf("old session must be revoked and stamped rotated_at: %+v", oldNow)
	}
	newNow, _ := r.GetSessionByTokenHash(ctx, next.TokenHash)
	if newNow.IsRevoked || newNow.RotatedAt != nil || newNow.UserID != u.ID {
		t.Errorf("new session must be active: %+v", newNow)
	}
}

func TestRepo_RotateTwiceFails(t *testing.T) {
	r, p := newRepo(t)
	ctx := context.Background()
	u := newUser(t, r, p)
	old := mustCreate(t, r, sessionFor(u.ID))
	if err := r.RotateSession(ctx, old.TokenHash, sessionFor(u.ID)); err != nil {
		t.Fatal(err)
	}
	if err := r.RotateSession(ctx, old.TokenHash, sessionFor(u.ID)); !errors.Is(err, auth.ErrNotRotated) {
		t.Errorf("second rotation err = %v, want ErrNotRotated", err)
	}
}

// Many refreshes of the same token at the same instant: exactly one may win.
func TestRepo_ConcurrentRotationHasExactlyOneWinner(t *testing.T) {
	r, p := newRepo(t)
	ctx := context.Background()
	u := newUser(t, r, p)
	old := mustCreate(t, r, sessionFor(u.ID))

	const workers = 20
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- r.RotateSession(ctx, old.TokenHash, sessionFor(u.ID))
		}()
	}
	wg.Wait()
	close(results)

	wins, losses := 0, 0
	for err := range results {
		switch {
		case err == nil:
			wins++
		case errors.Is(err, auth.ErrNotRotated):
			losses++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if wins != 1 || losses != workers-1 {
		t.Errorf("wins=%d losses=%d, want exactly 1 winner", wins, losses)
	}
	var children int
	_ = p.Admin.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE rotated_from = $1`, old.ID).Scan(&children)
	if children != 1 {
		t.Errorf("%d child sessions were created, want 1 (a fork would mean two valid tokens)", children)
	}
}

func TestRepo_RotateRefusesWhenNotAllowed(t *testing.T) {
	r, p := newRepo(t)
	ctx := context.Background()
	u := newUser(t, r, p)

	t.Run("unknown token", func(t *testing.T) {
		if err := r.RotateSession(ctx, tokenHash(), sessionFor(u.ID)); !errors.Is(err, auth.ErrNotRotated) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("expired", func(t *testing.T) {
		s := sessionFor(u.ID)
		s.ExpiresAt = time.Now().Add(-time.Minute)
		mustCreate(t, r, s)
		if err := r.RotateSession(ctx, s.TokenHash, sessionFor(u.ID)); !errors.Is(err, auth.ErrNotRotated) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("revoked", func(t *testing.T) {
		s := mustCreate(t, r, sessionFor(u.ID))
		if ok, err := r.RevokeSessionByTokenHash(ctx, s.TokenHash, u.ID); err != nil || !ok {
			t.Fatalf("revoke: %v %v", ok, err)
		}
		if err := r.RotateSession(ctx, s.TokenHash, sessionFor(u.ID)); !errors.Is(err, auth.ErrNotRotated) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("inactive user", func(t *testing.T) {
		other := newUser(t, r, p)
		s := mustCreate(t, r, sessionFor(other.ID))
		if _, err := p.Admin.Exec(ctx, `UPDATE users SET is_active = false WHERE id = $1`, other.ID); err != nil {
			t.Fatal(err)
		}
		if err := r.RotateSession(ctx, s.TokenHash, sessionFor(other.ID)); !errors.Is(err, auth.ErrNotRotated) {
			t.Errorf("a deactivated user must not be able to refresh, err = %v", err)
		}
	})
}

// If the new session cannot be inserted, the old one must stay usable.
func TestRepo_FailedRotationLeavesOldSessionUntouched(t *testing.T) {
	r, p := newRepo(t)
	ctx := context.Background()
	u := newUser(t, r, p)
	old := mustCreate(t, r, sessionFor(u.ID))
	existing := mustCreate(t, r, sessionFor(u.ID))

	clash := sessionFor(u.ID)
	clash.TokenHash = existing.TokenHash // duplicate: the insert will fail
	err := r.RotateSession(ctx, old.TokenHash, clash)
	if err == nil || errors.Is(err, auth.ErrNotRotated) {
		t.Fatalf("expected a database error, got %v", err)
	}
	got, _ := r.GetSessionByTokenHash(ctx, old.TokenHash)
	if got.IsRevoked || got.RotatedAt != nil {
		t.Errorf("the old session was revoked even though the rotation failed: %+v", got)
	}
}

// --- Revocation and cleanup ---

func TestRepo_RevokeSessionByTokenHashChecksOwner(t *testing.T) {
	r, p := newRepo(t)
	ctx := context.Background()
	owner, stranger := newUser(t, r, p), newUser(t, r, p)
	s := mustCreate(t, r, sessionFor(owner.ID))

	ok, err := r.RevokeSessionByTokenHash(ctx, s.TokenHash, stranger.ID)
	if err != nil || ok {
		t.Errorf("a stranger revoked someone else's session: ok=%v err=%v", ok, err)
	}
	if got, _ := r.GetSessionByTokenHash(ctx, s.TokenHash); got.IsRevoked {
		t.Fatal("the session must still be active after a stranger's attempt")
	}

	ok, err = r.RevokeSessionByTokenHash(ctx, s.TokenHash, owner.ID)
	if err != nil || !ok {
		t.Errorf("owner revoke: ok=%v err=%v", ok, err)
	}
	if ok, _ := r.RevokeSessionByTokenHash(ctx, s.TokenHash, owner.ID); ok {
		t.Error("revoking an already revoked session must report false")
	}
}

func TestRepo_RevokeAllUserSessions(t *testing.T) {
	r, p := newRepo(t)
	ctx := context.Background()
	a, b := newUser(t, r, p), newUser(t, r, p)
	a1, a2 := mustCreate(t, r, sessionFor(a.ID)), mustCreate(t, r, sessionFor(a.ID))
	b1 := mustCreate(t, r, sessionFor(b.ID))

	n, err := r.RevokeAllUserSessions(ctx, a.ID)
	if err != nil || n != 2 {
		t.Fatalf("revoked %d, err %v, want 2", n, err)
	}
	for _, s := range []*auth.Session{a1, a2} {
		if got, _ := r.GetSessionByTokenHash(ctx, s.TokenHash); !got.IsRevoked {
			t.Error("a's session should be revoked")
		}
	}
	if got, _ := r.GetSessionByTokenHash(ctx, b1.TokenHash); got.IsRevoked {
		t.Error("another user's session must not be touched")
	}
	if n, _ := r.RevokeAllUserSessions(ctx, a.ID); n != 0 {
		t.Errorf("second call revoked %d, want 0", n)
	}
}

func TestRepo_DeleteExpiredSessionsKeepsRevokedButLiveOnes(t *testing.T) {
	r, p := newRepo(t)
	ctx := context.Background()
	u := newUser(t, r, p)

	expired := sessionFor(u.ID)
	expired.ExpiresAt = time.Now().Add(-time.Minute)
	mustCreate(t, r, expired)
	live := mustCreate(t, r, sessionFor(u.ID))
	revokedLive := mustCreate(t, r, sessionFor(u.ID))
	if ok, _ := r.RevokeSessionByTokenHash(ctx, revokedLive.TokenHash, u.ID); !ok {
		t.Fatal("setup: revoke failed")
	}

	if _, err := r.DeleteExpiredSessions(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetSessionByTokenHash(ctx, expired.TokenHash); !errors.Is(err, auth.ErrNotFound) {
		t.Error("the expired session must be deleted")
	}
	if _, err := r.GetSessionByTokenHash(ctx, live.TokenHash); err != nil {
		t.Error("a live session must be kept")
	}
	if _, err := r.GetSessionByTokenHash(ctx, revokedLive.TokenHash); err != nil {
		t.Error("a revoked but unexpired session must be kept: it is the evidence for detecting token replay")
	}
}
