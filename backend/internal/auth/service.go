package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailAlreadyExists = errors.New("email is already registered")
	ErrSessionNotFound    = errors.New("session not found or expired")
	ErrSessionBlocked     = errors.New("session has been blocked")
	ErrInvalidToken       = errors.New("invalid authentication token")
)

// Pre-computed dummy hash to enforce constant-time execution during failed logins.
const dummyBcryptHash = "$2a$10$7EqJtq986P2xXh73j9d6O.z7qG.8UvX5T2qJtq986P2xXh73j9d6O"

// Config holds timing, duration, and secret parameters for auth operations.
type Config struct {
	JWTSecret            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	RotationGracePeriod  time.Duration
}

// TokenPair represents generated access and refresh tokens.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Claims defines standard JWT claims containing strictly immutable user identity identifiers.
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// Service defines application business logic for authentication.
type Service interface {
	Register(ctx context.Context, email, password, fullName string) (*User, error)
	Login(ctx context.Context, email, password, userAgent, clientIP string) (*TokenPair, *User, error)
	RefreshToken(ctx context.Context, refreshToken, userAgent, clientIP string) (*TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	CleanExpiredSessions(ctx context.Context) (int64, error)
	ValidateAccessToken(tokenStr string) (*Claims, error)
}

type service struct {
	repo   Repository
	config Config
}

// NewService instantiates an authentication service with repository and config dependencies.
func NewService(repo Repository, config Config) Service {
	if config.RotationGracePeriod == 0 {
		config.RotationGracePeriod = 30 * time.Second
	}
	return &service{
		repo:   repo,
		config: config,
	}
}

func (s *service) Register(ctx context.Context, email, password, fullName string) (*User, error) {
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

	user, err := s.repo.CreateUser(ctx, email, string(hashedPassword), fullName)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *service) Login(ctx context.Context, email, password, userAgent, clientIP string) (*TokenPair, *User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch user: %w", err)
	}

	hashToCompare := dummyBcryptHash
	if user != nil {
		hashToCompare = user.PasswordHash
	}

	bcryptErr := bcrypt.CompareHashAndPassword([]byte(hashToCompare), []byte(password))

	if user == nil || bcryptErr != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokenPair, err := s.generateAndSaveTokens(ctx, user.ID, userAgent, clientIP)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	return tokenPair, user, nil
}

func (s *service) RefreshToken(ctx context.Context, refreshToken, userAgent, clientIP string) (*TokenPair, error) {
	session, err := s.repo.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("fetch session: %w", err)
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}

	if session.IsBlocked {
		return nil, ErrSessionBlocked
	}

	now := time.Now()

	if now.After(session.ExpiresAt) {
		_ = s.repo.DeleteSession(ctx, refreshToken)
		return nil, ErrSessionNotFound
	}

	if session.RotatedAt != nil {
		if now.Sub(*session.RotatedAt) <= s.config.RotationGracePeriod {
			if session.ReplacementToken != nil {
				repSession, err := s.repo.GetSessionByRefreshToken(ctx, *session.ReplacementToken)
				if err == nil && repSession != nil && !repSession.IsBlocked {
					accessExpiresAt := now.Add(s.config.AccessTokenDuration)
					claims := &Claims{
						UserID: session.UserID,
						RegisteredClaims: jwt.RegisteredClaims{
							ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
							IssuedAt:  jwt.NewNumericDate(now),
						},
					}
					token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
					accessToken, err := token.SignedString([]byte(s.config.JWTSecret))
					if err != nil {
						return nil, fmt.Errorf("sign access token: %w", err)
					}

					return &TokenPair{
						AccessToken:  accessToken,
						RefreshToken: *session.ReplacementToken,
						ExpiresAt:    accessExpiresAt,
					}, nil
				}
			}
		}

		_ = s.repo.BlockUserSessions(ctx, session.UserID)
		return nil, ErrSessionBlocked
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil || user == nil {
		return nil, ErrInvalidCredentials
	}

	tokenPair, err := s.generateAndSaveTokens(ctx, user.ID, userAgent, clientIP)
	if err != nil {
		return nil, fmt.Errorf("generate fresh tokens: %w", err)
	}

	session.RotatedAt = &now
	session.ReplacementToken = &tokenPair.RefreshToken
	if err := s.repo.UpdateSessionRotation(ctx, session); err != nil {
		return nil, fmt.Errorf("update session rotation state: %w", err)
	}

	return tokenPair, nil
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	if err := s.repo.DeleteSession(ctx, refreshToken); err != nil {
		return fmt.Errorf("delete session: %w", err)
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

func (s *service) generateAndSaveTokens(ctx context.Context, userID, userAgent, clientIP string) (*TokenPair, error) {
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

	refreshToken, err := generateRandomToken(16)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token string: %w", err)
	}

	refreshExpiresAt := now.Add(s.config.RefreshTokenDuration)

	session := &Session{
		UserID:       userID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIP:     clientIP,
		IsBlocked:    false,
		ExpiresAt:    refreshExpiresAt,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("persist session: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiresAt,
	}, nil
}

func generateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}