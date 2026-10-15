package auth

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "time"
)

type Repository interface {
    CreateUser(ctx context.Context, email, passwordHash, name string) (*User, error)
    GetUserByEmail(ctx context.Context, email string) (*User, error)
    GetUserByID(ctx context.Context, id string) (*User, error)
    CreateSession(ctx context.Context, session *Session) error
    GetSessionByID(ctx context.Context, id string) (*Session, error)
    GetActiveSessionsByUserID(ctx context.Context, userID string) ([]*Session, error)
    GetChildSessionByParentID(ctx context.Context, parentID string) (*Session, error)
    UpdateSessionRotation(ctx context.Context, sessionID string, rotatedFrom *string, isRevoked bool) error
    UpdateSessionUsage(ctx context.Context, sessionID string) error
    DeleteSession(ctx context.Context, id string) error
    RevokeUserSessions(ctx context.Context, userID string) error
    DeleteExpiredSessions(ctx context.Context, gracePeriod time.Duration) (int64, error)
}

type postgresRepository struct {
    db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
    return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateUser(ctx context.Context, email, passwordHash, name string) (*User, error) {
    query := `
        INSERT INTO users (email, password_hash, name)
        VALUES ($1, $2, $3)
        RETURNING id, email, email_verified, name, avatar_url, password_hash, google_id, totp_enabled, storage_used_bytes, password_changed_at, is_active, created_at, updated_at
    `
    user := &User{}
    err := r.db.QueryRowContext(ctx, query, email, passwordHash, name).Scan(
        &user.ID,
        &user.Email,
        &user.EmailVerified,
        &user.Name,
        &user.AvatarURL,
        &user.PasswordHash,
        &user.GoogleID,
        &user.TotpEnabled,
        &user.StorageUsedBytes,
        &user.PasswordChangedAt,
        &user.IsActive,
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
        SELECT id, email, email_verified, name, avatar_url, password_hash, google_id, totp_enabled, storage_used_bytes, password_changed_at, is_active, created_at, updated_at
        FROM users
        WHERE email = $1
    `
    user := &User{}
    err := r.db.QueryRowContext(ctx, query, email).Scan(
        &user.ID,
        &user.Email,
        &user.EmailVerified,
        &user.Name,
        &user.AvatarURL,
        &user.PasswordHash,
        &user.GoogleID,
        &user.TotpEnabled,
        &user.StorageUsedBytes,
        &user.PasswordChangedAt,
        &user.IsActive,
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
        SELECT id, email, email_verified, name, avatar_url, password_hash, google_id, totp_enabled, storage_used_bytes, password_changed_at, is_active, created_at, updated_at
        FROM users
        WHERE id = $1
    `
    user := &User{}
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &user.ID,
        &user.Email,
        &user.EmailVerified,
        &user.Name,
        &user.AvatarURL,
        &user.PasswordHash,
        &user.GoogleID,
        &user.TotpEnabled,
        &user.StorageUsedBytes,
        &user.PasswordChangedAt,
        &user.IsActive,
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
        INSERT INTO sessions (user_id, refresh_token_hash, rotated_from, device_info, ip_address, is_revoked, expires_at)
        VALUES ($1, $2, $3, $4, $5::INET, $6, $7)
        RETURNING id, created_at, last_used_at
    `
    err := r.db.QueryRowContext(
        ctx,
        query,
        session.UserID,
        session.RefreshTokenHash,
        session.RotatedFrom,
        session.DeviceInfo,
        session.IPAddress,
        session.IsRevoked,
        session.ExpiresAt,
    ).Scan(&session.ID, &session.CreatedAt, &session.LastUsedAt)
    if err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }
    return nil
}

func (r *postgresRepository) GetSessionByID(ctx context.Context, id string) (*Session, error) {
    query := `
        SELECT id, user_id, refresh_token_hash, rotated_from, device_info, ip_address::TEXT, is_revoked, expires_at, created_at, last_used_at
        FROM sessions
        WHERE id = $1
    `
    session := &Session{}
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &session.ID,
        &session.UserID,
        &session.RefreshTokenHash,
        &session.RotatedFrom,
        &session.DeviceInfo,
        &session.IPAddress,
        &session.IsRevoked,
        &session.ExpiresAt,
        &session.CreatedAt,
        &session.LastUsedAt,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, fmt.Errorf("failed to get session by id: %w", err)
    }
    return session, nil
}

func (r *postgresRepository) GetActiveSessionsByUserID(ctx context.Context, userID string) ([]*Session, error) {
    query := `
        SELECT id, user_id, refresh_token_hash, rotated_from, device_info, ip_address::TEXT, is_revoked, expires_at, created_at, last_used_at
        FROM sessions
        WHERE user_id = $1 AND is_revoked = false AND expires_at > NOW()
    `
    rows, err := r.db.QueryContext(ctx, query, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to query active sessions: %w", err)
    }
    defer rows.Close()

    sessions := make([]*Session, 0)
    for rows.Next() {
        s := &Session{}
        if err := rows.Scan(
            &s.ID,
            &s.UserID,
            &s.RefreshTokenHash,
            &s.RotatedFrom,
            &s.DeviceInfo,
            &s.IPAddress,
            &s.IsRevoked,
            &s.ExpiresAt,
            &s.CreatedAt,
            &s.LastUsedAt,
        ); err != nil {
            return nil, fmt.Errorf("failed to scan session row: %w", err)
        }
        sessions = append(sessions, s)
    }
    return sessions, nil
}

func (r *postgresRepository) GetChildSessionByParentID(ctx context.Context, parentID string) (*Session, error) {
    query := `
        SELECT id, user_id, refresh_token_hash, rotated_from, device_info, ip_address::TEXT, is_revoked, expires_at, created_at, last_used_at
        FROM sessions
        WHERE rotated_from = $1
        LIMIT 1
    `
    session := &Session{}
    err := r.db.QueryRowContext(ctx, query, parentID).Scan(
        &session.ID,
        &session.UserID,
        &session.RefreshTokenHash,
        &session.RotatedFrom,
        &session.DeviceInfo,
        &session.IPAddress,
        &session.IsRevoked,
        &session.ExpiresAt,
        &session.CreatedAt,
        &session.LastUsedAt,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, fmt.Errorf("failed to get child session: %w", err)
    }
    return session, nil
}

func (r *postgresRepository) UpdateSessionRotation(ctx context.Context, sessionID string, rotatedFrom *string, isRevoked bool) error {
    query := `
        UPDATE sessions
        SET rotated_from = $1, is_revoked = $2, last_used_at = NOW()
        WHERE id = $3 AND is_revoked = false
    `
    res, err := r.db.ExecContext(ctx, query, rotatedFrom, isRevoked, sessionID)
    if err != nil {
        return fmt.Errorf("failed to update session rotation: %w", err)
    }

    rowsAffected, err := res.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to verify rotation affected rows: %w", err)
    }
    if rowsAffected == 0 {
        return errors.New("concurrent rotation conflict or session already revoked")
    }

    return nil
}

func (r *postgresRepository) UpdateSessionUsage(ctx context.Context, sessionID string) error {
    query := `UPDATE sessions SET last_used_at = NOW() WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, sessionID)
    if err != nil {
        return fmt.Errorf("failed to update session usage timestamp: %w", err)
    }
    return nil
}

func (r *postgresRepository) DeleteSession(ctx context.Context, id string) error {
    query := `DELETE FROM sessions WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("failed to delete session: %w", err)
    }
    return nil
}

func (r *postgresRepository) RevokeUserSessions(ctx context.Context, userID string) error {
    query := `UPDATE sessions SET is_revoked = true WHERE user_id = $1`
    _, err := r.db.ExecContext(ctx, query, userID)
    if err != nil {
        return fmt.Errorf("failed to revoke user sessions: %w", err)
    }
    return nil
}

func (r *postgresRepository) DeleteExpiredSessions(ctx context.Context, gracePeriod time.Duration) (int64, error) {
    query := `
        DELETE FROM sessions 
        WHERE expires_at < NOW() 
           OR (is_revoked = true AND created_at < NOW() - CAST($1 AS interval))
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