package iam

import (
	"context"
	"testing"
	"time"
)

func TestJWTSigner_RoundTripWithValidator(t *testing.T) {
	cfg := JWTConfig{Secret: "signer-secret"}
	signer, err := NewJWTSigner(cfg)
	if err != nil {
		t.Fatalf("NewJWTSigner: %v", err)
	}
	token, err := signer.Sign("user-1", time.Hour, map[string]any{"sid": "s-123", "sub": "ignored"})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	id, err := NewJWTValidator(cfg).Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if id.ID != "user-1" {
		t.Errorf("ID = %q, want user-1 (signer must override extra sub)", id.ID)
	}
	if id.Metadata["sid"] != "s-123" {
		t.Errorf("sid claim = %v, want s-123", id.Metadata["sid"])
	}
}

func TestJWTSigner_Errors(t *testing.T) {
	if _, err := NewJWTSigner(JWTConfig{}); err == nil {
		t.Error("expected error for empty secret")
	}
	if _, err := NewJWTSigner(JWTConfig{Secret: "s", SigningMethod: "RS256"}); err == nil {
		t.Error("expected error for non-HMAC method")
	}
	s, _ := NewJWTSigner(JWTConfig{Secret: "s"})
	if _, err := s.Sign("", time.Hour, nil); err == nil {
		t.Error("expected error for empty subject")
	}
	if _, err := s.Sign("u", 0, nil); err == nil {
		t.Error("expected error for non-positive ttl")
	}
}

func TestJWTSigner_ExpiredTokenRejected(t *testing.T) {
	cfg := JWTConfig{Secret: "signer-secret"}
	s, _ := NewJWTSigner(cfg)
	token, _ := s.Sign("user-1", time.Nanosecond, nil)
	time.Sleep(1100 * time.Millisecond)
	if _, err := NewJWTValidator(cfg).Validate(context.Background(), token); err == nil {
		t.Error("expected expired token to be rejected")
	}
}
