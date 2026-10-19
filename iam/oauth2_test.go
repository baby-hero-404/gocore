package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuth2Validator(t *testing.T) {
	// Mock OAuth2 UserInfo server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != "Bearer valid-oauth2-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"oauth2-user-99","email":"oauth2@example.com"}`))
	}))
	defer server.Close()

	validator := NewOAuth2Validator(OAuth2Config{
		UserInfoURL: server.URL,
	})

	// Case 1: Valid token
	identity, err := validator.Validate(context.Background(), "valid-oauth2-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if identity.ID != "oauth2-user-99" {
		t.Errorf("expected ID oauth2-user-99, got %s", identity.ID)
	}
	if identity.Email != "oauth2@example.com" {
		t.Errorf("expected email oauth2@example.com, got %s", identity.Email)
	}

	// Case 2: Invalid token
	_, err = validator.Validate(context.Background(), "invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}

	// Case 3: Empty token
	_, err = validator.Validate(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}
