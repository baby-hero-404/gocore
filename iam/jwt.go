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

	// IDClaim names the claim holding the user id (string or number). Empty keeps the default
	// lookup: "sub", then "id".
	IDClaim string `mapstructure:"id_claim" json:"id_claim"`
	// RolesClaim names the claim holding roles (a list of strings or a single string). Empty keeps
	// the default lookup: "roles", then "role".
	RolesClaim string `mapstructure:"roles_claim" json:"roles_claim"`
	// TokenType, when set, makes JWTSigner write it as the "typ" claim and JWTValidator reject
	// tokens whose "typ" differs. Set it on every signer and validator of a service that issues
	// more than one kind of token (for example access and refresh), otherwise they are
	// interchangeable.
	TokenType string `mapstructure:"token_type" json:"token_type"`
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

	if v.cfg.TokenType != "" {
		if typ, _ := claims["typ"].(string); typ != v.cfg.TokenType {
			return Identity{}, stderr.NewUnauthorized("invalid token type").WithKey(KeyUnauthenticated)
		}
	}

	subject := v.subject(claims)
	if subject == "" && v.cfg.IDClaim != "" {
		// A configured id claim is part of the token contract: a correctly signed token without it
		// must not become an identity with an empty id.
		return Identity{}, stderr.NewUnauthorized("token has no user id").WithKey(KeyUnauthenticated)
	}

	var email string
	if emailVal, ok := claims["email"].(string); ok {
		email = emailVal
	}

	return Identity{
		ID:       subject,
		Email:    email,
		Roles:    v.roles(claims),
		Metadata: claims,
	}, nil
}

// subject returns the user id from IDClaim, or from "sub" then "id" when IDClaim is empty.
func (v *JWTValidator) subject(claims jwt.MapClaims) string {
	if v.cfg.IDClaim != "" {
		return claimString(claims[v.cfg.IDClaim])
	}
	subject, _ := claims.GetSubject()
	if subject == "" {
		if id, ok := claims["id"].(string); ok {
			return id
		}
		return claimString(claims["sub"])
	}
	return subject
}

// roles returns the roles from RolesClaim, or from "roles" then "role" when RolesClaim is empty.
func (v *JWTValidator) roles(claims jwt.MapClaims) []string {
	if v.cfg.RolesClaim != "" {
		return claimStrings(claims[v.cfg.RolesClaim])
	}
	if r := claimStrings(claims["roles"]); r != nil {
		return r
	}
	return claimStrings(claims["role"])
}

// claimString renders a string or numeric claim as a string; other types yield "".
func claimString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%.0f", t)
	default:
		return ""
	}
}

// claimStrings reads a list of strings or a single string claim, ignoring non-string entries.
func claimStrings(v any) []string {
	switch t := v.(type) {
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	case []any:
		var out []string
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
