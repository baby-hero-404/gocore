package iam

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig defines verification parameters for JWTs.
type JWTConfig struct {
	Secret        string        `mapstructure:"secret" json:"-"`
	SigningMethod string        `mapstructure:"signing_method" json:"signing_method"` // default "HS256"
	Issuer        string        `mapstructure:"issuer" json:"issuer"`
	Audience      string        `mapstructure:"audience" json:"audience"`
	ClockSkew     time.Duration `mapstructure:"clock_skew" json:"clock_skew"`
}

// DefaultJWTConfig returns conservative default settings for JWT validation.
func DefaultJWTConfig() JWTConfig {
	return JWTConfig{
		SigningMethod: "HS256",
		ClockSkew:     5 * time.Second,
	}
}

// JWTValidator validates JWTs using HMAC or RSA keys with golang-jwt/jwt/v5.
type JWTValidator struct {
	keyFunc jwt.Keyfunc
	parser  *jwt.Parser
	cfg     JWTConfig
}

// NewJWTValidator creates a JWTValidator configured with a secret key. When set, SigningMethod,
// Issuer, Audience and ClockSkew are enforced (the issuer and audience must match, the signing
// method must equal SigningMethod, and ClockSkew is the leeway for exp/nbf/iat checks).
func NewJWTValidator(cfg JWTConfig) *JWTValidator {
	var opts []jwt.ParserOption
	if cfg.SigningMethod != "" {
		opts = append(opts, jwt.WithValidMethods([]string{cfg.SigningMethod}))
	}
	if cfg.Issuer != "" {
		opts = append(opts, jwt.WithIssuer(cfg.Issuer))
	}
	if cfg.Audience != "" {
		opts = append(opts, jwt.WithAudience(cfg.Audience))
	}
	if cfg.ClockSkew > 0 {
		opts = append(opts, jwt.WithLeeway(cfg.ClockSkew))
	}

	return &JWTValidator{
		cfg:    cfg,
		parser: jwt.NewParser(opts...),
		keyFunc: func(token *jwt.Token) (any, error) {
			// Ensure token signing method matches expected
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.Secret), nil
		},
	}
}

// Validate parses and validates a raw JWT string into an Identity.
func (v *JWTValidator) Validate(ctx context.Context, tokenStr string) (Identity, error) {
	if strings.TrimSpace(tokenStr) == "" {
		return Identity{}, stderr.NewUnauthorized("missing authentication token").WithKey(KeyUnauthenticated)
	}

	claims := jwt.MapClaims{}
	token, err := v.parser.ParseWithClaims(tokenStr, claims, v.keyFunc)
	if err != nil || !token.Valid {
		return Identity{}, stderr.Wrap(stderr.CodeUnauthorized, err, "invalid or expired token").WithKey(KeyUnauthenticated)
	}

	// Extract standard claims
	subject, _ := claims.GetSubject()
	if subject == "" {
		if id, ok := claims["id"].(string); ok {
			subject = id
		} else if subFloat, ok := claims["sub"].(float64); ok {
			subject = fmt.Sprintf("%.0f", subFloat)
		}
	}

	var email string
	if emailVal, ok := claims["email"].(string); ok {
		email = emailVal
	}

	var roles []string
	if rolesVal, ok := claims["roles"].([]any); ok {
		for _, r := range rolesVal {
			if strRole, ok := r.(string); ok {
				roles = append(roles, strRole)
			}
		}
	} else if roleVal, ok := claims["role"].(string); ok {
		roles = append(roles, roleVal)
	}

	return Identity{
		ID:       subject,
		Email:    email,
		Roles:    roles,
		Metadata: claims,
	}, nil
}
