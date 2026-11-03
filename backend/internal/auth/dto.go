package auth

import "time"

// --- Requests (validated by pkg/validator using the `binding` tags) ---

// RegisterRequest is the body of POST /auth/register.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Name     string `json:"name" binding:"required,min=1,max=100"`
}

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,max=72"`
}

// RefreshRequest is the body of POST /auth/refresh and POST /auth/logout. The refresh token may
// instead be sent in the X-Refresh-Token header (06-api-design.md §6.4); the handler accepts
// either, so the field is not required here.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"omitempty,max=512"`
}

// --- Responses (the data part of the standard envelope) ---

// UserResponse is the public view of a user. It never contains secrets.
type UserResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Name          string    `json:"name"`
	AvatarURL     *string   `json:"avatar_url"`
	CreatedAt     time.Time `json:"created_at"`
}

// AuthResponse is returned by login: a token pair plus the user.
type AuthResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresAt    time.Time     `json:"expires_at"`
	User         *UserResponse `json:"user"`
}

// TokenResponse is returned by refresh: a new token pair.
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// --- Domain models (never serialised to clients directly) ---

// User is a row of the users table, as far as authentication needs it.
type User struct {
	ID                string
	Email             string
	EmailVerified     bool
	Name              string
	AvatarURL         *string
	PasswordHash      *string // nil for Google-only accounts
	IsActive          bool
	PasswordChangedAt time.Time
	CreatedAt         time.Time
}

// ToResponse converts a User into its public form.
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Name:          u.Name,
		AvatarURL:     u.AvatarURL,
		CreatedAt:     u.CreatedAt,
	}
}

// Session is a row of the sessions table. TokenHash is the hex SHA-256 of the refresh token; the
// raw token is never stored (ADR-019).
type Session struct {
	ID          string
	UserID      string
	TokenHash   string
	RotatedFrom *string
	RotatedAt   *time.Time
	DeviceInfo  string
	IPAddress   string
	ExpiresAt   time.Time
	IsRevoked   bool
	CreatedAt   time.Time
	LastUsedAt  time.Time
}

// TokenPair is a freshly issued access token and refresh token.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time // when the access token expires
}
