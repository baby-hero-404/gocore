package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
)

// OAuth2Config defines settings for remote OAuth2 token validation.
type OAuth2Config struct {
	UserInfoURL string        `mapstructure:"user_info_url" json:"user_info_url"`
	Timeout     time.Duration `mapstructure:"timeout" json:"timeout"`
}

// DefaultOAuth2Config returns standard default configuration for OAuth2 token validation.
func DefaultOAuth2Config() OAuth2Config {
	return OAuth2Config{
		Timeout: 5 * time.Second,
	}
}

// OAuth2Validator verifies bearer tokens by querying the OAuth2 provider's UserInfo endpoint.
type OAuth2Validator struct {
	cfg        OAuth2Config
	httpClient *http.Client
}

// NewOAuth2Validator creates an OAuth2Validator instance.
func NewOAuth2Validator(cfg OAuth2Config, client ...*http.Client) *OAuth2Validator {
	timeout := 5 * time.Second
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}

	c := &http.Client{Timeout: timeout}
	if len(client) > 0 && client[0] != nil {
		c = client[0]
	}

	return &OAuth2Validator{
		cfg:        cfg,
		httpClient: c,
	}
}

// Validate calls the userinfo endpoint with the bearer token and parses identity.
func (v *OAuth2Validator) Validate(ctx context.Context, token string) (Identity, error) {
	if strings.TrimSpace(token) == "" {
		return Identity{}, stderr.NewUnauthorized("missing bearer token").WithKey(KeyUnauthenticated)
	}
	if v.cfg.UserInfoURL == "" {
		return Identity{}, stderr.NewInternal("oauth2 user_info_url is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.cfg.UserInfoURL, nil)
	if err != nil {
		return Identity{}, stderr.Wrap(stderr.CodeInternal, err, "failed to create userinfo request")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return Identity{}, stderr.Wrap(stderr.CodeUnavailable, err, "failed to reach oauth2 provider")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return Identity{}, stderr.NewUnauthorized(fmt.Sprintf("oauth2 token rejected: status %d", resp.StatusCode)).WithKey(KeyUnauthenticated)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Identity{}, stderr.Wrap(stderr.CodeInternal, err, "failed to parse userinfo response")
	}

	sub, _ := payload["sub"].(string)
	if sub == "" {
		sub, _ = payload["id"].(string)
	}

	email, _ := payload["email"].(string)

	return Identity{
		ID:       sub,
		Email:    email,
		Metadata: payload,
	}, nil
}
