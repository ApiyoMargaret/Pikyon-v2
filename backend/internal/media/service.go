package media

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	GeneratePresignedURL(ctx context.Context, userID string, req *PresignedURLRequest) (*PresignedURLResponse, error)
	CompleteUpload(ctx context.Context, userID string, req *CompleteUploadRequest) (*MediaResponse, error)
	CleanupOrphanedUploads(ctx context.Context) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GeneratePresignedURL(ctx context.Context, userID string, req *PresignedURLRequest) (*PresignedURLResponse, error) {
	ext := filepath.Ext(req.Filename)
	if ext == "" {
		ext = ".bin"
	}
	fileKey := fmt.Sprintf("users/%s/memories/%s/%s%s", userID, req.MemoryID, uuid.New().String(), ext)

	// Set upload expiration window (e.g., 15 minutes)
	expiresAt := time.Now().Add(15 * time.Minute)

	pending, err := s.repo.CreatePendingUpload(ctx, userID, req, fileKey, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize pending upload: %w", err)
	}

	mockUploadURL := fmt.Sprintf("https://storage.googleapis.com/safedrive-uploads/%s?signed=true", fileKey)

	return &PresignedURLResponse{
		UploadID:  pending.ID,
		UploadURL: mockUploadURL,
		FileKey:   pending.FileKey,
		ExpiresAt: pending.ExpiresAt,
	}, nil
}

func (s *service) CompleteUpload(ctx context.Context, userID string, req *CompleteUploadRequest) (*MediaResponse, error) {
	pending, err := s.repo.ConsumeAndGetPendingUpload(ctx, req.UploadID)
	if err != nil {
		return nil, err
	}

	if pending.UserID != userID {
		return nil, fmt.Errorf("unauthorized access to pending upload session")
	}

	fileURL := fmt.Sprintf("https://storage.googleapis.com/safedrive-uploads/%s", pending.FileKey)

	mediaModel, err := s.repo.CreateMedia(ctx, pending.MemoryID, fileURL, pending.ContentType, pending.FileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to persist media record: %w", err)
	}

	return &MediaResponse{
		ID:        mediaModel.ID,
		MemoryID:  mediaModel.MemoryID,
		FileUrl:   mediaModel.FileUrl,
		FileType:  mediaModel.FileType,
		FileSize:  mediaModel.FileSize,
		CreatedAt: mediaModel.CreatedAt,
	}, nil
}

func (s *service) CleanupOrphanedUploads(ctx context.Context) error {
	orphans, err := s.repo.GetOrphanedUploads(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch orphaned uploads: %w", err)
	}

	if len(orphans) == 0 {
		return nil
	}

	// Iterate through expired uploads to prevent storage leaks by deleting physical objects
	for _, orphan := range orphans {
		// TODO: Invoke your Cloud Storage SDK client here to delete the physical object:
		// e.g., cloudStorageClient.DeleteObject(ctx, orphan.FileKey)

		// Safely remove the individual tracking row once storage object deletion is handled
		if err := s.repo.DeletePendingUpload(ctx, orphan.ID); err != nil {
			// Log error or continue cleanup for remaining items depending on resilience policy
			continue
		}
	}

	return nil
}