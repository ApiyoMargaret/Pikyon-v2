package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the Repository interface for unit tests
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateUser(ctx context.Context, email, passwordHash, fullName string) (*User, error) {
	args := m.Called(ctx, email, passwordHash, fullName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRepository) CreateSession(ctx context.Context, session *Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockRepository) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}

func (m *MockRepository) UpdateSessionRotation(ctx context.Context, session *Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockRepository) DeleteSession(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
}

func (m *MockRepository) BlockUserSessions(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRepository) DeleteExpiredSessions(ctx context.Context, gracePeriod time.Duration) (int64, error) {
	args := m.Called(ctx, gracePeriod)
	return args.Get(0).(int64), args.Error(1)
}

// Helper function to setup testing instance
func setupTestService(mockRepo *MockRepository) Service {
	cfg := Config{
		JWTSecret:            "test-secret-key-32-bytes-minimum!!",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
		RotationGracePeriod:  30 * time.Second,
	}
	return NewService(mockRepo, cfg)
}

func TestRegister_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := setupTestService(mockRepo)
	ctx := context.Background()

	email := "test@example.com"
	password := "SecurePassword123"
	fullName := "Margaret Apiyo"

	mockRepo.On("GetUserByEmail", ctx, email).Return((*User)(nil), nil)
	mockRepo.On("CreateUser", ctx, email, mock.AnythingOfType("string"), fullName).
		Return(&User{
			ID:        "user-uuid-123",
			Email:     email,
			FullName:  fullName,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil)

	user, err := svc.Register(ctx, email, password, fullName)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, fullName, user.FullName)
	mockRepo.AssertExpectations(t)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := setupTestService(mockRepo)
	ctx := context.Background()

	email := "existing@example.com"

	mockRepo.On("GetUserByEmail", ctx, email).
		Return(&User{ID: "user-1", Email: email}, nil)

	user, err := svc.Register(ctx, email, "Password123", "User One")

	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
	assert.Nil(t, user)
	mockRepo.AssertExpectations(t)
}

func TestRefreshToken_Rotation_InGracePeriod(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := setupTestService(mockRepo)
	ctx := context.Background()

	oldToken := "old_refresh_token_123"
	replacementToken := "new_refresh_token_456"
	userID := "user-uuid-123"
	now := time.Now()
	rotatedAt := now.Add(-5 * time.Second) // Rotated 5s ago (within 30s grace period)

	session := &Session{
		ID:               "session-1",
		UserID:           userID,
		RefreshToken:     oldToken,
		IsBlocked:        false,
		ExpiresAt:        now.Add(24 * time.Hour),
		RotatedAt:        &rotatedAt,
		ReplacementToken: &replacementToken,
	}

	replacementSession := &Session{
		ID:           "session-2",
		UserID:       userID,
		RefreshToken: replacementToken,
		IsBlocked:    false,
		ExpiresAt:    now.Add(24 * time.Hour),
	}

	mockRepo.On("GetSessionByRefreshToken", ctx, oldToken).Return(session, nil)
	mockRepo.On("GetSessionByRefreshToken", ctx, replacementToken).Return(replacementSession, nil)

	tokens, err := svc.RefreshToken(ctx, oldToken, "TestAgent", "127.0.0.1")

	assert.NoError(t, err)
	assert.NotNil(t, tokens)
	assert.Equal(t, replacementToken, tokens.RefreshToken)
	mockRepo.AssertExpectations(t)
}

func TestRefreshToken_ReuseDetection_BlocksUser(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := setupTestService(mockRepo)
	ctx := context.Background()

	oldToken := "compromised_token"
	userID := "user-uuid-123"
	now := time.Now()
	rotatedAt := now.Add(-60 * time.Second) // Rotated 60s ago (exceeds 30s grace window)

	session := &Session{
		ID:           "session-1",
		UserID:       userID,
		RefreshToken: oldToken,
		IsBlocked:    false,
		ExpiresAt:    now.Add(24 * time.Hour),
		RotatedAt:    &rotatedAt,
	}

	mockRepo.On("GetSessionByRefreshToken", ctx, oldToken).Return(session, nil)
	mockRepo.On("BlockUserSessions", ctx, userID).Return(nil)

	tokens, err := svc.RefreshToken(ctx, oldToken, "AttackerAgent", "192.168.1.1")

	assert.ErrorIs(t, err, ErrSessionBlocked)
	assert.Nil(t, tokens)
	mockRepo.AssertExpectations(t)
}

func TestCleanExpiredSessions(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := setupTestService(mockRepo)
	ctx := context.Background()

	gracePeriod := 30 * time.Second
	mockRepo.On("DeleteExpiredSessions", ctx, gracePeriod).Return(int64(5), nil)

	deletedCount, err := svc.CleanExpiredSessions(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(5), deletedCount)
	mockRepo.AssertExpectations(t)
}