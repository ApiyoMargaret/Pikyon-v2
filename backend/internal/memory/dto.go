package memory

import (
	"time"
)

// CreateMemoryRequest represents the validated payload for creating a new memory.
type CreateMemoryRequest struct {
	Title       string `json:"title" binding:"required,min=1,max=255"`
	Story       string `json:"story"`
	Location    string `json:"location"`
	MemoryDate  string `json:"memory_date" binding:"omitempty,datetime=2006-01-02"`
	Visibility  string `json:"visibility" binding:"oneof=private public"`
	IsPinLocked bool   `json:"is_pin_locked"`
	IsFavorite  bool   `json:"is_favorite"`
}

// UpdateMemoryRequest represents the validated payload for updating an existing memory.
type UpdateMemoryRequest struct {
	Title       *string `json:"title" binding:"omitempty,min=1,max=255"`
	Story       *string `json:"story"`
	Location    *string `json:"location"`
	MemoryDate  *string `json:"memory_date" binding:"omitempty,datetime=2006-01-02"`
	Visibility  *string `json:"visibility" binding:"omitempty,oneof=private public"`
	IsPinLocked *bool   `json:"is_pin_locked"`
	IsFavorite  *bool   `json:"is_favorite"`
}

// MemoryResponse represents the pure network response DTO (decoupled from DB tags).
type MemoryResponse struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	Title             string     `json:"title"`
	Story             *string    `json:"story"`
	Location          *string    `json:"location"`
	MemoryDate        *string    `json:"memory_date"`
	Visibility        string     `json:"visibility"`
	Status            string     `json:"status"`
	IsPinLocked       bool       `json:"is_pin_locked"`
	IsFavorite        bool       `json:"is_favorite"`
	AiCaption         *string    `json:"ai_caption"`
	AiMood            *string    `json:"ai_mood"`
	AiTags            []string   `json:"ai_tags"`
	CaptionTwitter    *string    `json:"caption_twitter"`
	CaptionInstagram  *string    `json:"caption_instagram"`
	CaptionLinkedin   *string    `json:"caption_linkedin"`
	TrashedAt         *time.Time `json:"trashed_at,omitempty"`
	PermanentDeleteAt *time.Time `json:"permanent_delete_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// BulkActionRequest represents payloads for bulk trash or deletion.
type BulkActionRequest struct {
	MemoryIDs []string `json:"memory_ids" binding:"required,min=1"`
}
