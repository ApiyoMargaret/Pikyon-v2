package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// MemoryPersistenceModel represents the raw database row structure.
type MemoryPersistenceModel struct {
	ID                string         `db:"id"`
	UserID            string         `db:"user_id"`
	Title             string         `db:"title"`
	Story             sql.NullString `db:"story"`
	Location          sql.NullString `db:"location"`
	MemoryDate        sql.NullTime   `db:"memory_date"`
	Visibility        string         `db:"visibility"`
	Status            string         `db:"status"`
	IsPinLocked       bool           `db:"is_pin_locked"`
	IsFavorite        bool           `db:"is_favorite"`
	AiCaption         sql.NullString `db:"ai_caption"`
	AiMood            sql.NullString `db:"ai_mood"`
	AiTags            pq.StringArray `db:"ai_tags"`
	CaptionTwitter    sql.NullString `db:"caption_twitter"`
	CaptionInstagram  sql.NullString `db:"caption_instagram"`
	CaptionLinkedin   sql.NullString `db:"caption_linkedin"`
	TrashedAt         sql.NullTime   `db:"trashed_at"`
	PermanentDeleteAt sql.NullTime   `db:"permanent_delete_at"`
	CreatedAt         time.Time      `db:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at"`
}

type Repository interface {
	Create(ctx context.Context, userID string, req *CreateMemoryRequest) (*MemoryPersistenceModel, error)
	GetByID(ctx context.Context, userID, memoryID string) (*MemoryPersistenceModel, error)
	ListByUser(ctx context.Context, userID string, status string, limit, offset int) ([]MemoryPersistenceModel, error)
	Update(ctx context.Context, userID, memoryID string, req *UpdateMemoryRequest) (*MemoryPersistenceModel, error)
	SoftDelete(ctx context.Context, userID, memoryID string) error
	Restore(ctx context.Context, userID, memoryID string) error
	BulkTrash(ctx context.Context, userID string, memoryIDs []string) error
	EmptyTrash(ctx context.Context, userID string) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// Helpers for safe null-handling in SQL drivers
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func scanMemory(row interface{ Scan(...any) error }) (*MemoryPersistenceModel, error) {
	var m MemoryPersistenceModel
	err := row.Scan(
		&m.ID, &m.UserID, &m.Title, &m.Story, &m.Location, &m.MemoryDate, &m.Visibility, &m.Status,
		&m.IsPinLocked, &m.IsFavorite, &m.AiCaption, &m.AiMood, &m.AiTags, &m.CaptionTwitter,
		&m.CaptionInstagram, &m.CaptionLinkedin, &m.TrashedAt, &m.PermanentDeleteAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *repository) Create(ctx context.Context, userID string, req *CreateMemoryRequest) (*MemoryPersistenceModel, error) {
	query := `
		INSERT INTO memories (user_id, title, story, location, memory_date, visibility, is_pin_locked, is_favorite)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, '')::DATE, COALESCE(NULLIF($6, ''), 'private'), $7, $8)
		RETURNING id, user_id, title, story, location, memory_date, visibility, status, is_pin_locked, is_favorite, 
		          ai_caption, ai_mood, ai_tags, caption_twitter, caption_instagram, caption_linkedin, 
		          trashed_at, permanent_delete_at, created_at, updated_at
	`
	return scanMemory(r.db.QueryRowContext(
		ctx, query,
		userID, req.Title, nullString(req.Story), nullString(req.Location), nullString(req.MemoryDate), nullString(req.Visibility), req.IsPinLocked, req.IsFavorite,
	))
}

func (r *repository) GetByID(ctx context.Context, userID, memoryID string) (*MemoryPersistenceModel, error) {
	query := `
		SELECT id, user_id, title, story, location, memory_date, visibility, status, is_pin_locked, is_favorite,
		       ai_caption, ai_mood, ai_tags, caption_twitter, caption_instagram, caption_linkedin,
		       trashed_at, permanent_delete_at, created_at, updated_at
		FROM memories
		WHERE id = $1 AND user_id = $2
	`
	model, err := scanMemory(r.db.QueryRowContext(ctx, query, memoryID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("memory not found")
	}
	return model, err
}

func (r *repository) ListByUser(ctx context.Context, userID string, status string, limit, offset int) ([]MemoryPersistenceModel, error) {
	query := `
		SELECT id, user_id, title, story, location, memory_date, visibility, status, is_pin_locked, is_favorite,
		       ai_caption, ai_mood, ai_tags, caption_twitter, caption_instagram, caption_linkedin,
		       trashed_at, permanent_delete_at, created_at, updated_at
		FROM memories
		WHERE user_id = $1 AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.QueryContext(ctx, query, userID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []MemoryPersistenceModel
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			return nil, err
		}
		models = append(models, *m)
	}
	return models, nil
}

func (r *repository) SoftDelete(ctx context.Context, userID, memoryID string) error {
	query := `
		UPDATE memories 
		SET status = 'trashed', trashed_at = NOW(), permanent_delete_at = NOW() + INTERVAL '30 days', updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status = 'active'
	`
	res, err := r.db.ExecContext(ctx, query, memoryID, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return errors.New("memory not found or already trashed")
	}
	return nil
}

func (r *repository) Restore(ctx context.Context, userID, memoryID string) error {
	query := `
		UPDATE memories 
		SET status = 'active', trashed_at = NULL, permanent_delete_at = NULL, updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status = 'trashed'
	`
	res, err := r.db.ExecContext(ctx, query, memoryID, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return errors.New("trashed memory not found")
	}
	return nil
}

func (r *repository) Update(ctx context.Context, userID, memoryID string, req *UpdateMemoryRequest) (*MemoryPersistenceModel, error) {
	// Pre-fetch existing record to safely handle omitted fields vs explicit erasures
	existing, err := r.GetByID(ctx, userID, memoryID)
	if err != nil {
		return nil, err
	}

	title := existing.Title
	if req.Title != nil {
		title = *req.Title
	}

	story := existing.Story
	if req.Story != nil {
		if *req.Story == "" {
			story = sql.NullString{Valid: false}
		} else {
			story = sql.NullString{String: *req.Story, Valid: true}
		}
	}

	location := existing.Location
	if req.Location != nil {
		if *req.Location == "" {
			location = sql.NullString{Valid: false}
		} else {
			location = sql.NullString{String: *req.Location, Valid: true}
		}
	}

	memoryDate := existing.MemoryDate
	if req.MemoryDate != nil {
		if *req.MemoryDate == "" {
			memoryDate = sql.NullTime{Valid: false}
		} else {
			parsedTime, parseErr := time.Parse("2006-01-02", *req.MemoryDate)
			if parseErr != nil {
				return nil, fmt.Errorf("invalid date format: %w", parseErr)
			}
			memoryDate = sql.NullTime{Time: parsedTime, Valid: true}
		}
	}

	visibility := existing.Visibility
	if req.Visibility != nil {
		visibility = *req.Visibility
	}

	isPinLocked := existing.IsPinLocked
	if req.IsPinLocked != nil {
		isPinLocked = *req.IsPinLocked
	}

	isFavorite := existing.IsFavorite
	if req.IsFavorite != nil {
		isFavorite = *req.IsFavorite
	}

	query := `
		UPDATE memories 
		SET title = $1,
		    story = $2,
		    location = $3,
		    memory_date = $4,
		    visibility = $5,
		    is_pin_locked = $6,
		    is_favorite = $7,
		    updated_at = NOW()
		WHERE id = $8 AND user_id = $9 AND status = 'active'
		RETURNING id, user_id, title, story, location, memory_date, visibility, status, is_pin_locked, is_favorite,
		          ai_caption, ai_mood, ai_tags, caption_twitter, caption_instagram, caption_linkedin,
		          trashed_at, permanent_delete_at, created_at, updated_at
	`

	model, err := scanMemory(r.db.QueryRowContext(
		ctx, query,
		title, story, location, memoryDate, visibility, isPinLocked, isFavorite,
		memoryID, userID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("memory not found or inactive")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update memory: %w", err)
	}

	return model, nil
}

func (r *repository) BulkTrash(ctx context.Context, userID string, memoryIDs []string) error {
	query := `
		UPDATE memories 
		SET status = 'trashed', trashed_at = NOW(), permanent_delete_at = NOW() + INTERVAL '30 days', updated_at = NOW()
		WHERE user_id = $1 AND id = ANY($2) AND status = 'active'
	`
	_, err := r.db.ExecContext(ctx, query, userID, pq.Array(memoryIDs))
	return err
}

func (r *repository) EmptyTrash(ctx context.Context, userID string) error {
	query := `DELETE FROM memories WHERE user_id = $1 AND status = 'trashed'`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}