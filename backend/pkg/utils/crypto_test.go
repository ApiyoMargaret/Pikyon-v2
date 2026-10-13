package utils

import (
	"testing"
)

func TestHashAndPasswordVerification(t *testing.T) {
	password := "SecretPass123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error hashing password, got: %v", err)
	}

	if hash == "" {
		t.Fatal("expected non-empty hash string")
	}

	if !CheckPasswordHash(password, hash) {
		t.Error("expected CheckPasswordHash to return true for correct password")
	}

	if CheckPasswordHash("WrongPassword!", hash) {
		t.Error("expected CheckPasswordHash to return false for incorrect password")
	}
}

func TestJWTGenerationAndValidation(t *testing.T) {
	userID := "123e4567-e89b-12d3-a456-426614174000"
	email := "test@pikyon.com"

	tokenStr, err := GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("expected no error generating access token, got: %v", err)
	}

	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	claims, err := ValidateAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("expected token validation to pass, got: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
	}

	if claims.Email != email {
		t.Errorf("expected Email %s, got %s", email, claims.Email)
	}

	_, err = ValidateAccessToken("invalid.token.string")
	if err == nil {
		t.Error("expected error validating malformed token, got nil")
	}
}