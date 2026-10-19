// Package iammock provides test doubles for github.com/baby-hero-404/gocore/iam.
package iammock

import (
	"context"
	"github.com/baby-hero-404/gocore/iam"

	"github.com/stretchr/testify/mock"
)

// TokenValidator provides an in-memory mock of TokenValidator for unit testing.
type TokenValidator struct {
	mock.Mock
}

// NewTokenValidator creates a new TokenValidator instance.
func NewTokenValidator(t mock.TestingT) *TokenValidator {
	m := &TokenValidator{}
	m.Test(t)
	return m
}

func (m *TokenValidator) Validate(ctx context.Context, token string) (iam.Identity, error) {
	args := m.Called(ctx, token)
	if id, ok := args.Get(0).(iam.Identity); ok {
		return id, args.Error(1)
	}
	return iam.Identity{}, args.Error(1)
}

// Enforcer provides an in-memory mock of Enforcer for unit testing.
type Enforcer struct {
	mock.Mock
}

// NewEnforcer creates a new Enforcer instance.
func NewEnforcer(t mock.TestingT) *Enforcer {
	m := &Enforcer{}
	m.Test(t)
	return m
}

func (m *Enforcer) Enforce(rvals ...any) (bool, error) {
	args := m.Called(rvals...)
	return args.Bool(0), args.Error(1)
}
