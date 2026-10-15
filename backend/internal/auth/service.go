package auth

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

var (
    ErrInvalidCredentials = errors.New("invalid email or password")
    ErrEmailAlreadyExists = errors.New("email is already registered")
    ErrSessionNotFound    = errors.New("session not found or expired")
    ErrSessionBlocked     = errors.New("session has been revoked or blocked")
    ErrInvalidToken       = errors.New("invalid authentication token")
)

const dummyBcryptHash = "$2a$10$7EqJtq986P2xXh73j9d6O.z7qG.8UvX5T2qJtq986P2xXh73j9d6O"

type Config struct {
    JWTSecret            string
    AccessTokenDuration  time.Duration
    RefreshTokenDuration time.Duration
    RotationGracePeriod  time.Duration
}

type TokenPair struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at"`
}

type Claims struct {
    UserID string `json:"user_id"`
    jwt.RegisteredClaims
}

type Service interface {
    Register(ctx context.Context, email, password, name string) (*UserResponse, error)
    Login(ctx context.Context, email, password, deviceInfo, ipAddress string) (*TokenPair, *UserResponse, error)
    RefreshToken(ctx context.Context, refreshToken, deviceInfo, ipAddress string) (*TokenPair, error)
    Logout(ctx context.Context, userID, refreshToken string) error // Updated to accept userID
    CleanExpiredSessions(ctx context.Context) (int64, error)
    ValidateAccessToken(tokenStr string) (*Claims, error)
}

type service struct {
    repo   Repository
    config Config
}

func NewService(repo Repository, config Config) Service {
    if config.RotationGracePeriod == 0 {
        config.RotationGracePeriod = 30 * time.Second
    }
    return &service{
        repo:   repo,
        config: config,
    }
}

func (s *service) Register(ctx context.Context, email, password, name string) (*UserResponse, error) {
    existingUser, err := s.repo.GetUserByEmail(ctx, email)
    if err != nil {
        return nil, fmt.Errorf("check email availability: %w", err)
    }
    if existingUser != nil {
        return nil, ErrEmailAlreadyExists
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, fmt.Errorf("hash password: %w", err)
    }

    user, err := s.repo.CreateUser(ctx, email, string(hashedPassword), name)
    if err != nil {
        return nil, fmt.Errorf("create user: %w", err)
    }

    return user.ToResponse(), nil
}

func (s *service) Login(ctx context.Context, email, password, deviceInfo, ipAddress string) (*TokenPair, *UserResponse, error) {
    user, err := s.repo.GetUserByEmail(ctx, email)
    if err != nil {
        return nil, nil, fmt.Errorf("fetch user: %w", err)
    }

    hashToCompare := dummyBcryptHash
    if user != nil {
        hashToCompare = user.PasswordHash
    }

    bcryptErr := bcrypt.CompareHashAndPassword([]byte(hashToCompare), []byte(password))
    if user == nil || bcryptErr != nil || !user.IsActive {
        return nil, nil, ErrInvalidCredentials
    }

    tokenPair, err := s.generateAndSaveTokens(ctx, user.ID, nil, deviceInfo, ipAddress)
    if err != nil {
        return nil, nil, fmt.Errorf("generate tokens: %w", err)
    }

    return tokenPair, user.ToResponse(), nil
}

func (s *service) RefreshToken(ctx context.Context, refreshToken, deviceInfo, ipAddress string) (*TokenPair, error) {
    // 1. Parse composite token (format: sessionID.rawToken)
    sessionID, rawToken, err := parseCompositeToken(refreshToken)
    if err != nil {
        return nil, ErrInvalidToken
    }

    // 2. O(1) direct index lookup by session ID
    session, err := s.repo.GetSessionByID(ctx, sessionID)
    if err != nil {
        return nil, fmt.Errorf("fetch session: %w", err)
    }
    if session == nil || session.IsRevoked {
        return nil, ErrSessionNotFound
    }

    // 3. Verify bcrypt hash of the raw token
    if err := bcrypt.CompareHashAndPassword([]byte(session.RefreshTokenHash), []byte(rawToken)); err != nil {
        // Potential tampering or invalid token hash match
        return nil, ErrInvalidCredentials
    }

    now := time.Now()
    if now.After(session.ExpiresAt) {
        _ = s.repo.DeleteSession(ctx, session.ID)
        return nil, ErrSessionNotFound
    }

    // 4. Handle Token Rotation & Replay Attack Defense
    if session.RotatedFrom != nil {
        // Replay attack detected on an already rotated session. Revoke ALL user sessions immediately.
        _ = s.repo.RevokeUserSessions(ctx, session.UserID)
        return nil, ErrSessionBlocked
    }

    // Generate fresh token pair for rotation
    tokenPair, err := s.generateAndSaveTokens(ctx, session.UserID, &session.ID, deviceInfo, ipAddress)
    if err != nil {
        return nil, fmt.Errorf("generate rotation tokens: %w", err)
    }

    // Mark current session as rotated and revoked atomically
    if err := s.repo.UpdateSessionRotation(ctx, session.ID, &session.ID, true); err != nil {
        return nil, fmt.Errorf("update session rotation state: %w", err)
    }

    return tokenPair, nil
}

func (s *service) Logout(ctx context.Context, userID, refreshToken string) error {
    sessionID, _, err := parseCompositeToken(refreshToken)
    if err != nil {
        return ErrInvalidToken
    }

    // Fetch the session to verify ownership
    session, err := s.repo.GetSessionByID(ctx, sessionID)
    if err != nil {
        return fmt.Errorf("fetch session for logout: %w", err)
    }
    if session == nil {
        return ErrSessionNotFound
    }

    // IDOR Protection: Ensure the session belongs strictly to the authenticated user
    if session.UserID != userID {
        return ErrSessionNotFound // Return not found to avoid leaking session existence across tenants
    }

    if err := s.repo.DeleteSession(ctx, sessionID); err != nil {
        return fmt.Errorf("delete session on logout: %w", err)
    }

    return nil
}
func (s *service) CleanExpiredSessions(ctx context.Context) (int64, error) {
    count, err := s.repo.DeleteExpiredSessions(ctx, s.config.RotationGracePeriod)
    if err != nil {
        return 0, fmt.Errorf("cleanup expired sessions: %w", err)
    }
    return count, nil
}

func (s *service) ValidateAccessToken(tokenStr string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(s.config.JWTSecret), nil
    })

    if err != nil {
        return nil, ErrInvalidToken
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }

    return claims, nil
}

func (s *service) generateAndSaveTokens(ctx context.Context, userID string, rotatedFrom *string, deviceInfo, ipAddress string) (*TokenPair, error) {
    now := time.Now()
    accessExpiresAt := now.Add(s.config.AccessTokenDuration)

    claims := &Claims{
        UserID: userID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
            IssuedAt:  jwt.NewNumericDate(now),
            NotBefore: jwt.NewNumericDate(now),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    accessToken, err := token.SignedString([]byte(s.config.JWTSecret))
    if err != nil {
        return nil, fmt.Errorf("sign access token: %w", err)
    }

    rawRefreshToken, err := generateRandomToken(32)
    if err != nil {
        return nil, fmt.Errorf("generate random token: %w", err)
    }

    tokenHashBytes, err := bcrypt.GenerateFromPassword([]byte(rawRefreshToken), bcrypt.DefaultCost)
    if err != nil {
        return nil, fmt.Errorf("hash refresh token: %w", err)
    }

    refreshExpiresAt := now.Add(s.config.RefreshTokenDuration)

    // Temporary placeholder session to acquire DB-generated session ID
    session := &Session{
        UserID:           userID,
        RefreshTokenHash: string(tokenHashBytes),
        RotatedFrom:      rotatedFrom,
        DeviceInfo:       deviceInfo,
        IPAddress:        ipAddress,
        IsRevoked:        false,
        ExpiresAt:        refreshExpiresAt,
    }

    if err := s.repo.CreateSession(ctx, session); err != nil {
        return nil, fmt.Errorf("persist session: %w", err)
    }

    // Construct composite refresh token: sessionID.rawToken
    compositeRefreshToken := fmt.Sprintf("%s.%s", session.ID, rawRefreshToken)

    return &TokenPair{
        AccessToken:  accessToken,
        RefreshToken: compositeRefreshToken,
        ExpiresAt:    accessExpiresAt,
    }, nil
}

func parseCompositeToken(token string) (string, string, error) {
    parts := strings.SplitN(token, ".", 2)
    if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
        return "", "", ErrInvalidToken
    }
    return parts[0], parts[1], nil
}

func generateRandomToken(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}