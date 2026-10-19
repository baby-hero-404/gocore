// Package stdresmock provides test doubles for github.com/baby-hero-404/gocore/stdres.
package stdresmock

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// Translator provides an in-memory mock of Translator for unit testing.
type Translator struct {
	mock.Mock
}

// NewTranslator creates a new Translator instance.
func NewTranslator(t mock.TestingT) *Translator {
	m := &Translator{}
	m.Test(t)
	return m
}

func (m *Translator) Translate(ctx context.Context, key string, args ...any) string {
	callArgs := m.Called(ctx, key, args)
	return callArgs.String(0)
}
