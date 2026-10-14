package media

import "time"

// PresignedURLRequest represents the incoming payload to request an upload URL.
type PresignedURLRequest struct {
	Filename    string `json:"filename" binding:"required,max=255"`
	ContentType string `json:"content_type" binding:"required,oneof=image/jpeg image/png image/webp video/mp4"`
	FileSize    int64  `json:"file_size" binding:"required,gt=0,max=104857600"` // Enforces max 100MB limit (104,857,600 bytes)
	MemoryID    string `json:"memory_id" binding:"required,uuid"`
}

// PresignedURLResponse represents the response containing the secure upload URL and file tracking ID.
type PresignedURLResponse struct {
	UploadID  string    `json:"upload_id"`
	UploadURL string    `json:"upload_url"`
	FileKey   string    `json:"file_key"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CompleteUploadRequest represents the request to finalize an upload after client-side transfer.
type CompleteUploadRequest struct {
	UploadID string `json:"upload_id" binding:"required,uuid"`
}

// MediaResponse represents the public JSON representation of an attached media item.
type MediaResponse struct {
	ID        string    `json:"id"`
	MemoryID  string    `json:"memory_id"`
	FileUrl   string    `json:"file_url"`
	FileType  string    `json:"file_type"`
	FileSize  int64     `json:"file_size"`
	CreatedAt time.Time `json:"created_at"`
}
