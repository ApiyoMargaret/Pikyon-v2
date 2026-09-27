package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash, fullName string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	CreateSession(ctx context.Context, session *Session) error
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*Session, error)
	UpdateSessionRotation(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, refreshToken string) error
	BlockUserSessions(ctx context.Context, userID string) error
	DeleteExpiredSessions(ctx context.Context, gracePeriod time.Duration) (int64, error)
}

type postgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateUser(ctx context.Context, email, passwordHash, fullName string) (*User, error) {
	query := `
		INSERT INTO users (email, password_hash, full_name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, full_name, created_at, updated_at
	`
	user := &User{}
	err := r.db.QueryRowContext(ctx, query, email, passwordHash, fullName).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

func (r *postgresRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, full_name, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	user := &User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

func (r *postgresRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, password_hash, full_name, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	user := &User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return user, nil
}

func (r *postgresRepository) CreateSession(ctx context.Context, session *Session) error {
	query := `
		INSERT INTO sessions (user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(
		ctx,
		query,
		session.UserID,
		session.RefreshToken,
		session.UserAgent,
		session.ClientIP,
		session.IsBlocked,
		session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	query := `
		SELECT id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at, rotated_at, replacement_token, created_at
		FROM sessions
		WHERE refresh_token = $1
	`
	session := &Session{}
	err := r.db.QueryRowContext(ctx, query, refreshToken).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshToken,
		&session.UserAgent,
		&session.ClientIP,
		&session.IsBlocked,
		&session.ExpiresAt,
		&session.RotatedAt,
		&session.ReplacementToken,
		&session.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session by refresh token: %w", err)
	}
	return session, nil
}

func (r *postgresRepository) UpdateSessionRotation(ctx context.Context, session *Session) error {
	query := `
		UPDATE sessions
		SET rotated_at = $1, replacement_token = $2, is_blocked = $3
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, session.RotatedAt, session.ReplacementToken, session.IsBlocked, session.ID)
	if err != nil {
		return fmt.Errorf("failed to update session rotation: %w", err)
	}
	return nil
}

func (r *postgresRepository) BlockUserSessions(ctx context.Context, userID string) error {
	query := `UPDATE sessions SET is_blocked = true WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to block user sessions: %w", err)
	}
	return nil
}

func (r *postgresRepository) DeleteSession(ctx context.Context, refreshToken string) error {
	query := `DELETE FROM sessions WHERE refresh_token = $1`
	_, err := r.db.ExecContext(ctx, query, refreshToken)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (r *postgresRepository) DeleteExpiredSessions(ctx context.Context, gracePeriod time.Duration) (int64, error) {
	// Prevents sweeping sessions actively relying on a rotation grace period
	query := `
		DELETE FROM sessions 
		WHERE expires_at < NOW() 
		  AND (rotated_at IS NULL OR rotated_at < NOW() - $1::interval)
	`
	intervalStr := fmt.Sprintf("%d seconds", int(gracePeriod.Seconds()))
	res, err := r.db.ExecContext(ctx, query, intervalStr)
	if err != nil {
		return 0, fmt.Errorf("failed to clean up expired sessions: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, nil
	}
	return rowsAffected, nil
}