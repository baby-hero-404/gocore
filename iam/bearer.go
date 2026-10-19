package iam

import (
	"strings"

	"github.com/baby-hero-404/gocore/stderr"
)

// BearerToken extracts the token from an "Authorization: Bearer <token>" header value.
// It returns a 401 *stderr.Error when the header is missing or malformed.
func BearerToken(header string) (string, error) {
	if header == "" {
		return "", stderr.NewUnauthorized("authorization header is required")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", stderr.NewUnauthorized("invalid authorization format, expected 'Bearer <token>'")
	}
	return strings.TrimSpace(parts[1]), nil
}
