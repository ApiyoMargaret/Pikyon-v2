package memory

import (
	"context"
	"time"
)

type Service interface {
	CreateMemory(ctx context.Context, userID string, req *CreateMemoryRequest) (*MemoryResponse, error)
	GetMemoryByID(ctx context.Context, userID, memoryID string) (*MemoryResponse, error)
	ListMemories(ctx context.Context, userID string, status string, limit, offset int) ([]MemoryResponse, error)
	UpdateMemory(ctx context.Context, userID, memoryID string, req *UpdateMemoryRequest) (*MemoryResponse, error)
	SoftDeleteMemory(ctx context.Context, userID, memoryID string) error
	RestoreMemory(ctx context.Context, userID, memoryID string) error
	BulkTrashMemories(ctx context.Context, userID string, memoryIDs []string) error
	EmptyTrashBin(ctx context.Context, userID string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// mapModelToDTO converts a persistence model to a DTO, applies the AiTags empty-slice safeguard,
// and safely allocates independent string pointers to prevent loop aliasing bugs.
func mapModelToDTO(m *MemoryPersistenceModel) MemoryResponse {
	tags := []string(m.AiTags)
	if tags == nil {
		tags = []string{} // Safeguard: prevents JSON serializer from outputting null instead of []
	}

	var story, location, memoryDate, aiCaption, aiMood, capTwitter, capInsta, capLink *string
	
	if m.Story.Valid {
		val := m.Story.String
		story = &val
	}
	if m.Location.Valid {
		val := m.Location.String
		location = &val
	}
	if m.MemoryDate.Valid {
		val := m.MemoryDate.Time.Format("2006-01-02")
		memoryDate = &val
	}
	if m.AiCaption.Valid {
		val := m.AiCaption.String
		aiCaption = &val
	}
	if m.AiMood.Valid {
		val := m.AiMood.String
		aiMood = &val
	}
	if m.CaptionTwitter.Valid {
		val := m.CaptionTwitter.String
		capTwitter = &val
	}
	if m.CaptionInstagram.Valid {
		val := m.CaptionInstagram.String
		capInsta = &val
	}
	if m.CaptionLinkedin.Valid {
		val := m.CaptionLinkedin.String
		capLink = &val
	}

	var trashedAt, permDeleteAt *time.Time
	if m.TrashedAt.Valid {
		t := m.TrashedAt.Time
		trashedAt = &t
	}
	if m.PermanentDeleteAt.Valid {
		t := m.PermanentDeleteAt.Time
		permDeleteAt = &t
	}

	return MemoryResponse{
		ID:                m.ID,
		UserID:            m.UserID,
		Title:             m.Title,
		Story:             story,
		Location:          location,
		MemoryDate:        memoryDate,
		Visibility:        m.Visibility,
		Status:            m.Status,
		IsPinLocked:       m.IsPinLocked,
		IsFavorite:        m.IsFavorite,
		AiCaption:         aiCaption,
		AiMood:            aiMood,
		AiTags:            tags,
		CaptionTwitter:    capTwitter,
		CaptionInstagram:  capInsta,
		CaptionLinkedin:   capLink,
		TrashedAt:         trashedAt,
		PermanentDeleteAt: permDeleteAt,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func (s *service) CreateMemory(ctx context.Context, userID string, req *CreateMemoryRequest) (*MemoryResponse, error) {
	model, err := s.repo.Create(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	dto := mapModelToDTO(model)
	return &dto, nil
}

func (s *service) GetMemoryByID(ctx context.Context, userID, memoryID string) (*MemoryResponse, error) {
	model, err := s.repo.GetByID(ctx, userID, memoryID)
	if err != nil {
		return nil, err
	}
	dto := mapModelToDTO(model)
	return &dto, nil
}

func (s *service) ListMemories(ctx context.Context, userID string, status string, limit, offset int) ([]MemoryResponse, error) {
	models, err := s.repo.ListByUser(ctx, userID, status, limit, offset)
	if err != nil {
		return nil, err
	}

	dtos := []MemoryResponse{}
	for _, m := range models {
		dtos = append(dtos, mapModelToDTO(&m))
	}
	return dtos, nil
}

func (s *service) UpdateMemory(ctx context.Context, userID, memoryID string, req *UpdateMemoryRequest) (*MemoryResponse, error) {
	model, err := s.repo.Update(ctx, userID, memoryID, req)
	if err != nil {
		return nil, err
	}
	dto := mapModelToDTO(model)
	return &dto, nil
}

func (s *service) SoftDeleteMemory(ctx context.Context, userID, memoryID string) error {
	return s.repo.SoftDelete(ctx, userID, memoryID)
}

func (s *service) RestoreMemory(ctx context.Context, userID, memoryID string) error {
	return s.repo.Restore(ctx, userID, memoryID)
}

func (s *service) BulkTrashMemories(ctx context.Context, userID string, memoryIDs []string) error {
	return s.repo.BulkTrash(ctx, userID, memoryIDs)
}

func (s *service) EmptyTrashBin(ctx context.Context, userID string) error {
	return s.repo.EmptyTrash(ctx, userID)
}