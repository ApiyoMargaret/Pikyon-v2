package auth

import (
	"context"
	"errors"
	"fmt"
	"net/netip"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/db"
)

// Errors returned by the repository. The service maps them to API errors.
var (
	ErrNotFound   = errors.New("auth: not found")
	ErrEmailTaken = errors.New("auth: email already registered")
	// ErrNotRotated means RotateSession changed nothing: the session does not exist, is already
	// revoked or rotated, has expired, or its user is inactive.
	ErrNotRotated = errors.New("auth: session was not rotated")
)

// Repository is the data access layer for users and sessions.
//
// It runs on the ADMIN pool by design (ADR-018): registration, login and token refresh happen
// before any user is identified, and finding a session by its token hash needs a column the
// restricted role cannot read. Because RLS does not protect these queries, every query that
// belongs to a user carries an explicit user condition, and the tests verify each one.
type Repository interface {
	CreateUser(ctx context.Context, email, name, passwordHash string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)

	CreateSession(ctx context.Context, s *Session) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	// RotateSession atomically exchanges the session with oldTokenHash for next. See the method.
	RotateSession(ctx context.Context, oldTokenHash string, next *Session) error
	// RevokeSessionByTokenHash revokes the session only if it belongs to userID.
	RevokeSessionByTokenHash(ctx context.Context, tokenHash, userID string) (bool, error)
	RevokeAllUserSessions(ctx context.Context, userID string) (int64, error)
	DeleteExpiredSessions(ctx context.Context) (int64, error)
}

type repository struct {
	q db.Querier
}

// NewRepository returns a repository on q, which must be the admin pool (db.Pools.Admin).
func NewRepository(q db.Querier) Repository {
	return &repository{q: q}
}

const userColumns = `id, email, email_verified, name, avatar_url, password_hash, is_active, password_changed_at, created_at`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.EmailVerified, &u.Name, &u.AvatarURL, &u.PasswordHash,
		&u.IsActive, &u.PasswordChangedAt, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser inserts a user. email must already be lowercase (the database rejects anything
// else). It returns ErrEmailTaken if the email is registered.
func (r *repository) CreateUser(ctx context.Context, email, name, passwordHash string) (*User, error) {
	u, err := scanUser(r.q.QueryRow(ctx,
		`INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING `+userColumns,
		email, name, passwordHash))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("auth: create user: %w", err)
	}
	return u, nil
}

// GetUserByEmail looks a user up by (lowercase) email. It returns ErrNotFound if there is none.
func (r *repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u, err := scanUser(r.q.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email))
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("auth: get user by email: %w", err)
	}
	return u, err
}

// GetUserByID looks a user up by ID. It returns ErrNotFound if there is none.
func (r *repository) GetUserByID(ctx context.Context, id string) (*User, error) {
	u, err := scanUser(r.q.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("auth: get user by id: %w", err)
	}
	return u, err
}

// sessionInputs normalises the optional fields of a new session.
func sessionInputs(s *Session) (device, ip string) {
	// An unparseable address is stored as NULL rather than failing the login.
	if _, err := netip.ParseAddr(s.IPAddress); err == nil {
		ip = s.IPAddress
	}
	return s.DeviceInfo, ip
}

// CreateSession inserts a new session and fills in its ID and timestamps. TokenHash must be the
// 64-character hex SHA-256 of the refresh token; the database enforces that.
func (r *repository) CreateSession(ctx context.Context, s *Session) error {
	device, ip := sessionInputs(s)
	err := r.q.QueryRow(ctx, `
		INSERT INTO sessions (user_id, refresh_token_hash, device_info, ip_address, expires_at)
		VALUES ($1, $2, left(NULLIF($3, ''), 512), NULLIF($4, '')::inet, $5)
		RETURNING id, created_at, last_used_at`,
		s.UserID, s.TokenHash, device, ip, s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt, &s.LastUsedAt)
	if err != nil {
		return fmt.Errorf("auth: create session: %w", err)
	}
	return nil
}

// GetSessionByTokenHash returns the session with that token hash, or ErrNotFound.
func (r *repository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	var s Session
	err := r.q.QueryRow(ctx, `
		SELECT id, user_id, refresh_token_hash, rotated_from, rotated_at,
		       COALESCE(device_info, ''), COALESCE(host(ip_address), ''),
		       expires_at, is_revoked, created_at, last_used_at
		FROM sessions WHERE refresh_token_hash = $1`, tokenHash,
	).Scan(&s.ID, &s.UserID, &s.TokenHash, &s.RotatedFrom, &s.RotatedAt, &s.DeviceInfo, &s.IPAddress,
		&s.ExpiresAt, &s.IsRevoked, &s.CreatedAt, &s.LastUsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("auth: get session: %w", err)
	}
	return &s, nil
}

// RotateSession exchanges the session whose token hash is oldTokenHash for the new session next,
// in ONE SQL statement, so it is atomic and race-free:
//
//   - the old session is revoked and stamped rotated_at, and
//   - next is inserted with rotated_from pointing at it and the same user,
//
// but only if the old session is not revoked, has not expired, and its user is active. If two
// requests rotate the same token at once, exactly one succeeds and the other gets ErrNotRotated.
// If inserting next fails (for example a duplicate token hash) the old session is left untouched.
//
// On success next.ID, next.UserID, next.RotatedFrom and the timestamps are filled in.
func (r *repository) RotateSession(ctx context.Context, oldTokenHash string, next *Session) error {
	device, ip := sessionInputs(next)
	err := r.q.QueryRow(ctx, `
		WITH claimed AS (
			UPDATE sessions s
			SET is_revoked = true, rotated_at = now(), last_used_at = now()
			FROM users u
			WHERE s.refresh_token_hash = $1
			  AND u.id = s.user_id
			  AND u.is_active
			  AND s.is_revoked = false
			  AND s.expires_at > now()
			RETURNING s.id, s.user_id
		)
		INSERT INTO sessions (user_id, refresh_token_hash, rotated_from, device_info, ip_address, expires_at)
		SELECT user_id, $2, id, left(NULLIF($3, ''), 512), NULLIF($4, '')::inet, $5 FROM claimed
		RETURNING id, user_id, rotated_from, created_at, last_used_at`,
		oldTokenHash, next.TokenHash, device, ip, next.ExpiresAt,
	).Scan(&next.ID, &next.UserID, &next.RotatedFrom, &next.CreatedAt, &next.LastUsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotRotated
	}
	if err != nil {
		return fmt.Errorf("auth: rotate session: %w", err)
	}
	return nil
}

// RevokeSessionByTokenHash revokes an active session, but only if it belongs to userID. It
// reports whether a session was revoked; a session that is unknown, already revoked or owned by
// someone else all return false, so callers cannot learn which.
func (r *repository) RevokeSessionByTokenHash(ctx context.Context, tokenHash, userID string) (bool, error) {
	tag, err := r.q.Exec(ctx, `
		UPDATE sessions SET is_revoked = true
		WHERE refresh_token_hash = $1 AND user_id = $2 AND is_revoked = false`, tokenHash, userID)
	if err != nil {
		return false, fmt.Errorf("auth: revoke session: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// RevokeAllUserSessions revokes every active session of the user (used on token reuse and on
// password change) and returns how many it revoked.
func (r *repository) RevokeAllUserSessions(ctx context.Context, userID string) (int64, error) {
	tag, err := r.q.Exec(ctx, `UPDATE sessions SET is_revoked = true WHERE user_id = $1 AND is_revoked = false`, userID)
	if err != nil {
		return 0, fmt.Errorf("auth: revoke user sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}

// DeleteExpiredSessions removes sessions past their expiry and returns how many. Revoked
// sessions that have not expired are kept on purpose: they are the evidence that lets the service
// recognise a replayed, already-rotated refresh token (ADR-019).
func (r *repository) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	tag, err := r.q.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`)
	if err != nil {
		return 0, fmt.Errorf("auth: delete expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
