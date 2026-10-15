package auth

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "golang.org/x/crypto/bcrypt"
)

// MockRepository is a mock implementation of the auth.Repository interface.
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) CreateUser(ctx context.Context, email, passwordHash, name string) (*User, error) {
    args := m.Called(ctx, email, passwordHash, name)
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
    if session != nil && session.ID == "" {
        session.ID = "mock-session-id"
    }
    return args.Error(0)
}

func (m *MockRepository) GetSessionByID(ctx context.Context, id string) (*Session, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Session), args.Error(1)
}

func (m *MockRepository) GetActiveSessionsByUserID(ctx context.Context, userID string) ([]*Session, error) {
    args := m.Called(ctx, userID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).([]*Session), args.Error(1)
}

func (m *MockRepository) GetChildSessionByParentID(ctx context.Context, parentID string) (*Session, error) {
    args := m.Called(ctx, parentID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Session), args.Error(1)
}

func (m *MockRepository) UpdateSessionRotation(ctx context.Context, sessionID string, rotatedFrom *string, isRevoked bool) error {
    args := m.Called(ctx, sessionID, rotatedFrom, isRevoked)
    return args.Error(0)
}

func (m *MockRepository) UpdateSessionUsage(ctx context.Context, sessionID string) error {
    args := m.Called(ctx, sessionID)
    return args.Error(0)
}

func (m *MockRepository) DeleteSession(ctx context.Context, id string) error {
    args := m.Called(ctx, id)
    return args.Error(0)
}

func (m *MockRepository) RevokeUserSessions(ctx context.Context, userID string) error {
    args := m.Called(ctx, userID)
    return args.Error(0)
}

func (m *MockRepository) DeleteExpiredSessions(ctx context.Context, gracePeriod time.Duration) (int64, error) {
    args := m.Called(ctx, gracePeriod)
    return args.Get(0).(int64), args.Error(1)
}

func TestRegister_Success(t *testing.T) {
    mockRepo := new(MockRepository)
    cfg := Config{
        JWTSecret:            "test-secret",
        AccessTokenDuration:  15 * time.Minute,
        RefreshTokenDuration: 24 * time.Hour,
        RotationGracePeriod:  30 * time.Second,
    }
    svc := NewService(mockRepo, cfg)

    ctx := context.Background()
    email := "test@example.com"
    password := "Password123!"
    name := "Test User"

    mockRepo.On("GetUserByEmail", ctx, email).Return(nil, nil)
    mockRepo.On("CreateUser", ctx, email, mock.Anything, name).Return(&User{
        ID:        "user-1",
        Email:     email,
        Name:      name,
        IsActive:  true,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }, nil)

    resp, err := svc.Register(ctx, email, password, name)

    assert.NoError(t, err)
    assert.NotNil(t, resp)
    assert.Equal(t, email, resp.Email)
    assert.Equal(t, name, resp.Name)
    mockRepo.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
    mockRepo := new(MockRepository)
    cfg := Config{
        JWTSecret:            "test-secret",
        AccessTokenDuration:  15 * time.Minute,
        RefreshTokenDuration: 24 * time.Hour,
        RotationGracePeriod:  30 * time.Second,
    }
    svc := NewService(mockRepo, cfg)

    ctx := context.Background()
    email := "test@example.com"
    password := "Password123!"
    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

    mockRepo.On("GetUserByEmail", ctx, email).Return(&User{
        ID:           "user-1",
        Email:        email,
        PasswordHash: string(hashedPassword),
        Name:         "Test User",
        IsActive:     true,
    }, nil)

    mockRepo.On("CreateSession", ctx, mock.AnythingOfType("*auth.Session")).Return(nil)

    tokens, userResp, err := svc.Login(ctx, email, password, "Mozilla", "127.0.0.1")

    assert.NoError(t, err)
    assert.NotNil(t, tokens)
    assert.NotNil(t, userResp)
    assert.NotEmpty(t, tokens.AccessToken)
    assert.NotEmpty(t, tokens.RefreshToken)
    mockRepo.AssertExpectations(t)
}