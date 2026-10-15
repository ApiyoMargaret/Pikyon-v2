package auth

import "time"

// REQUEST DTOs (Incoming API Payloads)

// RegisterRequest represents the payload for creating a new user account.
type RegisterRequest struct {
    Email    string `json:"email" binding:"required,email,max=255"`
    Password string `json:"password" binding:"required,min=8,max=72"`
    Name     string `json:"name" binding:"required,min=1,max=100"`
}

// LoginRequest represents the payload for authenticating an existing user.
type LoginRequest struct {
    Email    string `json:"email" binding:"required,email,max=255"`
    Password string `json:"password" binding:"required,max=72"`
}

// RefreshTokenRequest represents the payload for rotating/refreshing access tokens.
type RefreshTokenRequest struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}

// RESPONSE DTOs (Public API Contracts)

// UserResponse represents the public-facing user profile safe for network transmission.
type UserResponse struct {
    ID            string    `json:"id"`
    Email         string    `json:"email"`
    EmailVerified bool      `json:"email_verified"`
    Name          string    `json:"name"`
    AvatarURL     *string   `json:"avatar_url,omitempty"`
    CreatedAt     time.Time `json:"created_at"`
}

// LoginResponse represents the response payload specifically for successful logins.
type LoginResponse struct {
    AccessToken  string        `json:"access_token"`
    RefreshToken string        `json:"refresh_token"`
    ExpiresAt    time.Time     `json:"expires_at"`
    User         *UserResponse `json:"user"`
}

// RefreshResponse represents the lightweight response payload for token renewals.
type RefreshResponse struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at"`
}

// User represents the internal database user model. Never exposed directly to the wire.
type User struct {
    ID                string    `db:"id"`
    Email             string    `db:"email"`
    EmailVerified     bool      `db:"email_verified"`
    Name              string    `db:"name"`
    AvatarURL         *string   `db:"avatar_url"`
    PasswordHash      string    `db:"password_hash"`
    GoogleID          *string   `db:"google_id"`
    TotpEnabled       bool      `db:"totp_enabled"`
    StorageUsedBytes  int64     `db:"storage_used_bytes"`
    PasswordChangedAt time.Time `db:"password_changed_at"`
    IsActive          bool      `db:"is_active"`
    CreatedAt         time.Time `db:"created_at"`
    UpdatedAt         time.Time `db:"updated_at"`
}

// Session represents the persisted session model mapped to the sessions table.
type Session struct {
    ID               string     `db:"id"`
    UserID           string     `db:"user_id"`
    RefreshTokenHash string     `db:"refresh_token_hash"`
    RotatedFrom      *string    `db:"rotated_from"`
    DeviceInfo       string     `db:"device_info"`
    IPAddress        string     `db:"ip_address"` // Scans natively from Postgres INET type as text
    IsRevoked        bool       `db:"is_revoked"`
    ExpiresAt        time.Time  `db:"expires_at"`
    CreatedAt        time.Time  `db:"created_at"`
    LastUsedAt       time.Time  `db:"last_used_at"`
}

// ToResponse converts an internal User database model into a safe public DTO.
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