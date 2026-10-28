package iam

import (
	"strings"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/golang-jwt/jwt/v5"
)

// JWTSigner issues HMAC-signed JWTs that JWTValidator (same JWTConfig) accepts.
type JWTSigner struct {
	cfg    JWTConfig
	method jwt.SigningMethod
}

// NewJWTSigner creates a JWTSigner. Only HMAC methods (HS256, HS384, HS512) are supported,
// matching JWTValidator; an empty SigningMethod defaults to HS256.
func NewJWTSigner(cfg JWTConfig) (*JWTSigner, error) {
	if strings.TrimSpace(cfg.Secret) == "" {
		return nil, stderr.NewInternal("jwt signer: secret is required")
	}
	name := cfg.SigningMethod
	if name == "" {
		name = "HS256"
	}
	method, ok := jwt.GetSigningMethod(name).(*jwt.SigningMethodHMAC)
	if !ok {
		return nil, stderr.NewInternal("jwt signer: unsupported signing method " + name)
	}
	return &JWTSigner{cfg: cfg, method: method}, nil
}

// Sign issues a token for subject with the given lifetime. Extra claims (e.g. a session ID)
// are added as-is; "sub", "iat" and "exp" (and "iss"/"aud", IDClaim and "typ" when configured) are
// always set by the signer.
func (s *JWTSigner) Sign(subject string, ttl time.Duration, extra map[string]any) (string, error) {
	if strings.TrimSpace(subject) == "" {
		return "", stderr.NewInternal("jwt signer: subject is required")
	}
	if ttl <= 0 {
		return "", stderr.NewInternal("jwt signer: ttl must be positive")
	}

	now := time.Now()
	claims := jwt.MapClaims{}
	for k, v := range extra {
		claims[k] = v
	}
	claims["sub"] = subject
	if s.cfg.IDClaim != "" {
		claims[s.cfg.IDClaim] = subject
	}
	if s.cfg.TokenType != "" {
		claims["typ"] = s.cfg.TokenType
	}
	claims["iat"] = now.Unix()
	claims["exp"] = now.Add(ttl).Unix()
	if s.cfg.Issuer != "" {
		claims["iss"] = s.cfg.Issuer
	}
	if s.cfg.Audience != "" {
		claims["aud"] = s.cfg.Audience
	}

	token, err := jwt.NewWithClaims(s.method, claims).SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return "", stderr.Wrap(stderr.CodeInternal, err, "jwt signer: failed to sign token")
	}
	return token, nil
}
