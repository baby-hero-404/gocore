package iam

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/golang-jwt/jwt/v5"
)

func generateTestToken(secret string, sub string, email string, roles []string, exp time.Duration) string {
	claims := jwt.MapClaims{
		"sub":   sub,
		"email": email,
		"roles": roles,
		"exp":   time.Now().Add(exp).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestJWTValidator_ValidToken(t *testing.T) {
	secret := "my-secret-key-12345" // gitleaks:allow (test fixture, not a real secret)
	validator := NewJWTValidator(JWTConfig{Secret: secret})

	token := generateTestToken(secret, "user-100", "user@example.com", []string{"admin"}, time.Hour)

	identity, err := validator.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if identity.ID != "user-100" {
		t.Errorf("expected ID user-100, got %s", identity.ID)
	}
	if identity.Email != "user@example.com" {
		t.Errorf("expected email user@example.com, got %s", identity.Email)
	}
	if !identity.HasRole("admin") {
		t.Error("expected identity to have role admin")
	}
}

func TestJWTValidator_ExpiredOrInvalidToken(t *testing.T) {
	secret := "my-secret-key-12345" // gitleaks:allow (test fixture, not a real secret)
	validator := NewJWTValidator(JWTConfig{Secret: secret})

	// Expired token
	expiredToken := generateTestToken(secret, "user-100", "user@example.com", nil, -time.Hour)
	_, err := validator.Validate(context.Background(), expiredToken)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	var stdErr *stderr.Error
	if !errors.As(err, &stdErr) || stdErr.Code != stderr.CodeUnauthorized {
		t.Errorf("expected CodeUnauthorized, got %v", err)
	}

	// Invalid signature
	badSigToken := generateTestToken("different-secret", "user-100", "user@example.com", nil, time.Hour)
	_, err = validator.Validate(context.Background(), badSigToken)
	if err == nil {
		t.Fatal("expected error for bad signature, got nil")
	}

	// Empty token
	_, err = validator.Validate(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

func TestJWTConfig_SecretOmittedFromJSON(t *testing.T) {
	cfg := JWTConfig{
		Secret:        "super-secret-jwt-key-xyz", // gitleaks:allow
		SigningMethod: "HS256",
		Issuer:        "auth-service",
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	out := string(b)
	if strings.Contains(out, "super-secret-jwt-key-xyz") || strings.Contains(out, "secret") {
		t.Errorf("expected secret to be omitted from JSON, got %s", out)
	}
}
