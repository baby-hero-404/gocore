// Package stderrmock provides test doubles for github.com/baby-hero-404/gocore/stderr.
package stderrmock

import (
	"github.com/baby-hero-404/gocore/stderr"
	"github.com/stretchr/testify/mock"
)

// Error provides a mock of ErrorInterface for unit testing.
type Error struct {
	mock.Mock
}

// NewError creates a new instance of Error.
func NewError(t mock.TestingT) *Error {
	m := &Error{}
	m.Test(t)
	return m
}

func (m *Error) Error() string {
	args := m.Called()
	return args.String(0)
}

func (m *Error) HTTPStatus() int {
	args := m.Called()
	return args.Int(0)
}

func (m *Error) StatusCode() stderr.Code {
	args := m.Called()
	return args.Get(0).(stderr.Code)
}

func (m *Error) ErrorMessage() string {
	args := m.Called()
	return args.String(0)
}
