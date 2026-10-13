package auth

import "time"

// RegisterRequest represents the payload for creating a new user account.
type RegisterRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8,max=72"`
    FullName string `json:"full_name" binding:"required"`
}

// LoginRequest represents the payload for authenticating an existing user.
type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,max=72"`
}

// RefreshTokenRequest represents the payload for rotating/refreshing access tokens.
type RefreshTokenRequest struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}

// LoginResponse represents the response payload specifically for successful logins.
type LoginResponse struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at"`
    User         *User     `json:"user"`
}

// RefreshResponse represents the lightweight response payload for token renewals.
type RefreshResponse struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at"`
}

// User represents the core user account model.
type User struct {
    ID           string    `json:"id" db:"id"`
    Email        string    `json:"email" db:"email"`
    PasswordHash string    `json:"-" db:"password_hash"`
    FullName     string    `json:"full_name" db:"full_name"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Session represents the persisted refresh token session model.
type Session struct {
    ID               string     `json:"id" db:"id"`
    UserID           string     `json:"user_id" db:"user_id"`
    RefreshToken     string     `json:"refresh_token" db:"refresh_token"`
    UserAgent        string     `json:"user_agent" db:"user_agent"`
    ClientIP         string     `json:"client_ip" db:"client_ip"`
    IsBlocked        bool       `json:"is_blocked" db:"is_blocked"`
    ExpiresAt        time.Time  `json:"expires_at" db:"expires_at"`
    RotatedAt        *time.Time `json:"rotated_at,omitempty" db:"rotated_at"`
    ReplacementToken *string    `json:"replacement_token,omitempty" db:"replacement_token"`
    CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}