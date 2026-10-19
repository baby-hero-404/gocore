// Package envmock provides test doubles for github.com/baby-hero-404/gocore/env.
package envmock

import (
	"github.com/baby-hero-404/gocore/env"
	"github.com/stretchr/testify/mock"
)

// Loader provides an in-memory mock of Loader for unit testing.
type Loader struct {
	mock.Mock
}

// NewLoader creates a new Loader instance.
func NewLoader(t mock.TestingT) *Loader {
	m := &Loader{}
	m.Test(t)
	return m
}

func (m *Loader) Load(prefix string, target any, opts ...env.Option) error {
	args := m.Called(prefix, target)
	return args.Error(0)
}
