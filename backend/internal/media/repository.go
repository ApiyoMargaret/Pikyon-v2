package media

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// PendingUploadModel represents a raw database row for tracking abandoned/unfinalized uploads.
type PendingUploadModel struct {
	ID          string    `db:"id"`
	UserID      string    `db:"user_id"`
	MemoryID    string    `db:"memory_id"`
	FileKey     string    `db:"file_key"`
	ContentType string    `db:"content_type"`
	FileSize    int64     `db:"file_size"`
	ExpiresAt   time.Time `db:"expires_at"`
	CreatedAt   time.Time `db:"created_at"`
}

// MediaPersistenceModel represents the raw database row for finalized media files.
type MediaPersistenceModel struct {
	ID        string    `db:"id"`
	MemoryID  string    `db:"memory_id"`
	FileUrl   string    `db:"file_url"`
	FileType  string    `db:"file_type"`
	FileSize  int64     `db:"file_size"`
	CreatedAt time.Time `db:"created_at"`
}

type Repository interface {
	CreatePendingUpload(ctx context.Context, userID string, req *PresignedURLRequest, fileKey string, expiresAt time.Time) (*PendingUploadModel, error)
	// ConsumeAndGetPendingUpload atomically deletes and returns the pending upload to prevent race conditions on completion.
	ConsumeAndGetPendingUpload(ctx context.Context, uploadID string) (*PendingUploadModel, error)
	DeletePendingUpload(ctx context.Context, uploadID string) error
	CreateMedia(ctx context.Context, memoryID, fileURL, fileType string, fileSize int64) (*MediaPersistenceModel, error)
	GetOrphanedUploads(ctx context.Context) ([]PendingUploadModel, error)
	DeleteOrphanedUploads(ctx context.Context, cutoff time.Time) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreatePendingUpload(ctx context.Context, userID string, req *PresignedURLRequest, fileKey string, expiresAt time.Time) (*PendingUploadModel, error) {
	query := `
		INSERT INTO pending_uploads (user_id, memory_id, file_key, content_type, file_size, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, memory_id, file_key, content_type, file_size, expires_at, created_at
	`
	var p PendingUploadModel
	err := r.db.QueryRowContext(
		ctx, query,
		userID, req.MemoryID, fileKey, req.ContentType, req.FileSize, expiresAt,
	).Scan(&p.ID, &p.UserID, &p.MemoryID, &p.FileKey, &p.ContentType, &p.FileSize, &p.ExpiresAt, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ConsumeAndGetPendingUpload performs an atomic DELETE ... RETURNING to prevent double-completion race conditions.
func (r *repository) ConsumeAndGetPendingUpload(ctx context.Context, uploadID string) (*PendingUploadModel, error) {
	query := `
		DELETE FROM pending_uploads
		WHERE id = $1
		RETURNING id, user_id, memory_id, file_key, content_type, file_size, expires_at, created_at
	`
	var p PendingUploadModel
	err := r.db.QueryRowContext(ctx, query, uploadID).Scan(
		&p.ID, &p.UserID, &p.MemoryID, &p.FileKey, &p.ContentType, &p.FileSize, &p.ExpiresAt, &p.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("pending upload not found, expired, or already completed")
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) DeletePendingUpload(ctx context.Context, uploadID string) error {
	query := `DELETE FROM pending_uploads WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, uploadID)
	return err
}

func (r *repository) CreateMedia(ctx context.Context, memoryID, fileURL, fileType string, fileSize int64) (*MediaPersistenceModel, error) {
	query := `
		INSERT INTO media (memory_id, file_url, file_type, file_size)
		VALUES ($1, $2, $3, $4)
		RETURNING id, memory_id, file_url, file_type, file_size, created_at
	`
	var m MediaPersistenceModel
	err := r.db.QueryRowContext(ctx, query, memoryID, fileURL, fileType, fileSize).Scan(
		&m.ID, &m.MemoryID, &m.FileUrl, &m.FileType, &m.FileSize, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *repository) GetOrphanedUploads(ctx context.Context) ([]PendingUploadModel, error) {
	query := `
		SELECT id, user_id, memory_id, file_key, content_type, file_size, expires_at, created_at
		FROM pending_uploads
		WHERE expires_at < NOW()
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Pre-initialize slice with reasonable capacity or safe default to avoid continuous reallocations
	uploads := make([]PendingUploadModel, 0, 16)
	for rows.Next() {
		var p PendingUploadModel
		if err := rows.Scan(&p.ID, &p.UserID, &p.MemoryID, &p.FileKey, &p.ContentType, &p.FileSize, &p.ExpiresAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		uploads = append(uploads, p)
	}
	return uploads, nil
}

func (r *repository) DeleteOrphanedUploads(ctx context.Context, cutoff time.Time) error {
	query := `DELETE FROM pending_uploads WHERE expires_at < $1`
	_, err := r.db.ExecContext(ctx, query, cutoff)
	return err
}
